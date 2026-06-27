//go:build windows

package actions

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

var (
	comOnce    sync.Once
	comInitErr error
)

func ensureCOM() {
	comOnce.Do(func() {
		hr, _, _ := procCoInitializeEx.Call(0, COINIT_MULTITHREADED)
		if hr != S_OK && hr != 1 {
			comInitErr = fmt.Errorf("CoInitializeEx: 0x%X", hr)
		}
	})
}

type UIAElement struct {
	Name          string  `json:"name"`
	AutomationID  string  `json:"automation_id,omitempty"`
	ControlType   string  `json:"control_type,omitempty"`
	IsEnabled     bool    `json:"is_enabled"`
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
	Width         float64 `json:"width"`
	Height        float64 `json:"height"`
	ProcessID     int     `json:"process_id,omitempty"`
}

type UIAFindOpts struct {
	Name          string `json:"name,omitempty"`
	AutomationID  string `json:"automation_id,omitempty"`
	ControlType   string `json:"control_type,omitempty"`
	WaitMs        int    `json:"wait_ms,omitempty"`
}

type UIATextResult struct {
	Text string `json:"text"`
}

func ctrlTypeFromName(name string) *int32 {
	m := map[string]int32{
		"Button": 50000, "Calendar": 50001, "CheckBox": 50002,
		"ComboBox": 50003, "Edit": 50004, "Hyperlink": 50005,
		"Image": 50006, "ListItem": 50007, "List": 50008,
		"Menu": 50009, "MenuBar": 50010, "MenuItem": 50011,
		"ProgressBar": 50012, "RadioButton": 50013, "ScrollBar": 50014,
		"Slider": 50015, "Spinner": 50016, "StatusBar": 50017,
		"Tab": 50018, "TabItem": 50019, "Text": 50020,
		"ToolBar": 50021, "ToolTip": 50022, "Tree": 50023,
		"TreeItem": 50024, "Custom": 50025, "Group": 50026,
		"Thumb": 50027, "DataGrid": 50028, "DataItem": 50029,
		"Document": 50030, "SplitButton": 50031, "Window": 50032,
		"Pane": 50033, "Header": 50034, "HeaderItem": 50035,
		"Table": 50036, "TitleBar": 50037, "Separator": 50038,
		"SemanticZoom": 50039, "AppBar": 50040,
	}
	if v, ok := m[name]; ok {
		return &v
	}
	return nil
}

// Build a combined condition from UIAFindOpts
func buildCondition(au *uiaAuto, opts UIAFindOpts) (*uiaCondition, error) {
	var conds []unsafe.Pointer
	if opts.Name != "" {
		v := varString(opts.Name)
		c, err := au.createPropertyCondition(UIA_NamePropertyId, v)
		varFree(v)
		if err != nil {
			return nil, err
		}
		conds = append(conds, c.p)
	}
	if opts.AutomationID != "" {
		v := varString(opts.AutomationID)
		c, err := au.createPropertyCondition(UIA_AutomationIdPropertyId, v)
		varFree(v)
		if err != nil {
			return nil, err
		}
		conds = append(conds, c.p)
	}
	if opts.ControlType != "" {
		ct := ctrlTypeFromName(opts.ControlType)
		if ct == nil {
			return nil, fmt.Errorf("unknown control type %q", opts.ControlType)
		}
		c, err := au.createPropertyCondition(UIA_ControlTypePropertyId, varInt(*ct))
		if err != nil {
			return nil, err
		}
		conds = append(conds, c.p)
	}

	switch len(conds) {
	case 0:
		return nil, fmt.Errorf("no conditions specified")
	case 1:
		return &uiaCondition{p: conds[0]}, nil
	default:
		var andC unsafe.Pointer
		r, _, _ := syscall.SyscallN(vtblMethod(au.p, 25), uintptr(au.p), uintptr(conds[0]), uintptr(conds[1]),
			uintptr(unsafe.Pointer(&andC)))
		if r != S_OK {
			return nil, fmt.Errorf("CreateAndCondition: 0x%X", r)
		}
		// Release individual conds since AndCondition holds refs
		for _, c := range conds {
			comRelease(c)
		}
		return &uiaCondition{p: andC}, nil
	}
}

// ── UIAFindElement ──
// Strategy:
//  1. If name/automation_id is specified, use FindFirst(Descendants) — fast (~2ms)
//  2. If only control_type, use FindAll(Children) — 275ms for root
//  3. As last resort, walk children breadth-first
func UIAFindElement(opts UIAFindOpts) ([]UIAElement, error) {
	if opts.Name == "" && opts.AutomationID == "" && opts.ControlType == "" {
		return nil, fmt.Errorf("uia_find: at least one of name, automation_id, control_type required")
	}

	ensureCOM()
	if comInitErr != nil {
		return nil, fmt.Errorf("uia_find com: %w", comInitErr)
	}

	au, err := newUIA()
	if err != nil {
		return nil, fmt.Errorf("uia_find: %w", err)
	}
	defer au.release()

	root, err := au.getRootElement()
	if err != nil {
		return nil, fmt.Errorf("uia_find root: %w", err)
	}
	defer root.release()

	cond, err := buildCondition(au, opts)
	if err != nil {
		return nil, fmt.Errorf("uia_find cond: %w", err)
	}
	defer cond.release()

	hasExact := opts.Name != "" || opts.AutomationID != ""

	// Case 1: Has name/automation_id — use FindFirst (very fast)
	if hasExact {
		elem, err := root.findFirst(TreeScope_Descendants, uintptr(cond.p))
		if err != nil {
			return nil, fmt.Errorf("uia_find find: %w", err)
		}
		if elem == nil {
			return []UIAElement{}, nil
		}
		defer elem.release()
		return []UIAElement{elem.toElement()}, nil
	}

	// Case 2: Control-type only — search Children first (fast)
	arr, err := root.findAll(TreeScope_Children, uintptr(cond.p))
	if err != nil {
		return nil, fmt.Errorf("uia_find find: %w", err)
	}
	defer arr.release()

	n := arr.length()
	results := make([]UIAElement, 0, n)
	for i := 0; i < n; i++ {
		e, err := arr.get(i)
		if err != nil || e == nil {
			continue
		}
		results = append(results, e.toElement())
		e.release()
	}
	return results, nil
}

// ── UIAGetText ──
func UIAGetText(name, automationID string) (string, error) {
	if name == "" && automationID == "" {
		return "", fmt.Errorf("uia_get_text: name or automation_id required")
	}

	ensureCOM()
	if comInitErr != nil {
		return "", fmt.Errorf("uia_get_text com: %w", comInitErr)
	}

	au, err := newUIA()
	if err != nil {
		return "", fmt.Errorf("uia_get_text: %w", err)
	}
	defer au.release()

	root, err := au.getRootElement()
	if err != nil {
		return "", fmt.Errorf("uia_get_text root: %w", err)
	}
	defer root.release()

	var cond *uiaCondition
	if automationID != "" {
		v := varString(automationID)
		c, err := au.createPropertyCondition(UIA_AutomationIdPropertyId, v)
		varFree(v)
		if err != nil {
			return "", fmt.Errorf("uia_get_text cond: %w", err)
		}
		cond = c
	} else {
		v := varString(name)
		c, err := au.createPropertyCondition(UIA_NamePropertyId, v)
		varFree(v)
		if err != nil {
			return "", fmt.Errorf("uia_get_text cond: %w", err)
		}
		cond = c
	}
	defer cond.release()

	elem, err := root.findFirst(TreeScope_Descendants, uintptr(cond.p))
	if err != nil {
		return "", fmt.Errorf("uia_get_text find: %w", err)
	}
	if elem == nil {
		return "", nil
	}
	defer elem.release()

	text, err := elem.getValue()
	if err != nil {
		return "", nil
	}
	return text, nil
}

// ── UIAInvoke ──
func UIAInvoke(name, automationID string) (bool, error) {
	if name == "" && automationID == "" {
		return false, fmt.Errorf("uia_invoke: name or automation_id required")
	}

	ensureCOM()
	if comInitErr != nil {
		return false, fmt.Errorf("uia_invoke com: %w", comInitErr)
	}

	au, err := newUIA()
	if err != nil {
		return false, fmt.Errorf("uia_invoke: %w", err)
	}
	defer au.release()

	root, err := au.getRootElement()
	if err != nil {
		return false, fmt.Errorf("uia_invoke root: %w", err)
	}
	defer root.release()

	var cond *uiaCondition
	if automationID != "" {
		v := varString(automationID)
		c, err := au.createPropertyCondition(UIA_AutomationIdPropertyId, v)
		varFree(v)
		if err != nil {
			return false, fmt.Errorf("uia_invoke cond: %w", err)
		}
		cond = c
	} else {
		v := varString(name)
		c, err := au.createPropertyCondition(UIA_NamePropertyId, v)
		varFree(v)
		if err != nil {
			return false, fmt.Errorf("uia_invoke cond: %w", err)
		}
		cond = c
	}
	defer cond.release()

	elem, err := root.findFirst(TreeScope_Descendants, uintptr(cond.p))
	if err != nil {
		return false, fmt.Errorf("uia_invoke find: %w", err)
	}
	if elem == nil {
		return false, nil
	}
	defer elem.release()

	if err := elem.invoke(); err != nil {
		return false, nil
	}
	return true, nil
}
