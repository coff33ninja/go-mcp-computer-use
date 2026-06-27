//go:build windows

package actions

import (
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func initCOM() {
	windows.CoInitializeEx(0, COINIT_MULTITHREADED)
}

var _ = syscall.LoadLibrary // ensure syscall import is used

// ═══════════════════════════════════════════════════════════
// INVESTIGATION: 6s overhead on first FindAll
// ═══════════════════════════════════════════════════════════

func TestUIA100_ColdStartProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cold-start profile in short mode")
	}
	initCOM()
	defer windows.CoUninitialize()

	// Measure each step to see where the 28s goes
	t0 := time.Now()
	au, _ := newUIA()
	t.Logf("  CoCreateInstance:     %v", time.Since(t0))

	t0 = time.Now()
	root, _ := au.getRootElement()
	t.Logf("  GetRootElement:       %v", time.Since(t0))

	t0 = time.Now()
	cond, _ := au.createTrueCondition()
	t.Logf("  CreateTrueCondition:  %v", time.Since(t0))

	t0 = time.Now()
	arr, _ := root.findAll(TreeScope_Children, cond.p)
	t1 := time.Since(t0)
	t.Logf("  FIRST FindAll(Chld):  %v", t1)

	t0 = time.Now()
	arr2, _ := root.findAll(TreeScope_Children, cond.p)
	t2 := time.Since(t0)
	t.Logf("  SECOND FindAll(Chld): %v  (%.0fx)", t2, float64(t1)/float64(t2))

	arr.release()
	arr2.release()
	cond.release()
	root.release()
	au.release()
}

// ═══════════════════════════════════════════════════════════
// ElementFromHandle
// ═══════════════════════════════════════════════════════════

func TestUIA110_ElementFromHandle(t *testing.T) {
	initCOM()
	defer windows.CoUninitialize()

	au, _ := newUIA()
	defer au.release()

	modUser32 := windows.NewLazySystemDLL("user32.dll")
	procGetDesktopWindow := modUser32.NewProc("GetDesktopWindow")
	hwnd, _, _ := procGetDesktopWindow.Call()
	t.Logf("  Desktop HWND: 0x%X", hwnd)

	start := time.Now()
	elem, err := au.elementFromHandle(hwnd)
	t.Logf("  ElementFromHandle: %v", time.Since(start))
	if err != nil {
		t.Fatalf("ElementFromHandle: %v", err)
	}
	defer elem.release()
	t.Logf("  Name=%q  PID=%d  Enabled=%v", elem.getName(), elem.getPid(), elem.isEnabled())
	r, err := elem.getBoundingRect()
	if err != nil {
		v, _ := elem.getPropValue(UIA_BoundingRectanglePropertyId)
		t.Logf("  Root BoundingRect err=%v (raw VT=%d)", err, v.VT)
	} else {
		t.Logf("  Root BoundingRect: %.0f x %.0f at (%.0f, %.0f)", r.Right-r.Left, r.Bottom-r.Top, r.Left, r.Top)
	}

	// Try FindFirst on this element
	cond, _ := au.createTrueCondition()
	defer cond.release()
	start = time.Now()
	first, err := elem.findFirst(TreeScope_Descendants, cond.p)
	t.Logf("  FindFirst(Descendants, true): %v", time.Since(start))
	if err != nil {
		t.Fatalf("FindFirst: %v", err)
	}
	if first != nil {
		defer first.release()
		t.Logf("  First child: Name=%q  PID=%d", first.getName(), first.getPid())
	} else {
		t.Log("  First child: nil")
	}

	// Try with HWND of a known window (e.g. Taskbar)
	// Get Shell_TrayWnd
	procFindWindow := modUser32.NewProc("FindWindowW")
	cls, _ := syscall.UTF16PtrFromString("Shell_TrayWnd")
	trayHwnd, _, _ := procFindWindow.Call(uintptr(unsafe.Pointer(cls)), 0)
	if trayHwnd != 0 {
		tray, err := au.elementFromHandle(trayHwnd)
		if err == nil {
			tray.release()
		}
	}

	// Test bounding rect on a regular window (Program Manager / Progman)
	progman, _ := syscall.UTF16PtrFromString("Progman")
	progHwnd, _, _ := procFindWindow.Call(uintptr(unsafe.Pointer(progman)), 0)
	if progHwnd != 0 {
		prog, err := au.elementFromHandle(progHwnd)
		if err == nil {
			t.Logf("  Progman: Name=%q", prog.getName())
			r, err := prog.getBoundingRect()
			if err == nil {
				t.Logf("    BoundingRect: %.0f x %.0f at (%.0f, %.0f)", r.Right-r.Left, r.Bottom-r.Top, r.Left, r.Top)
			} else {
				v, _ := prog.getPropValue(UIA_BoundingRectanglePropertyId)
				t.Logf("    BoundingRect err=%v (raw VT=%d)", err, v.VT)
			}
			prog.release()
		}
	}
}

// ═══════════════════════════════════════════════════════════
// BENCHMARKS
// ═══════════════════════════════════════════════════════════

func BenchmarkUIA_FindAll_TrueCondition(b *testing.B) {
	initCOM()
	defer windows.CoUninitialize()

	au, _ := newUIA()
	root, _ := au.getRootElement()
	cond, _ := au.createTrueCondition()

	// warmup
	root.findAll(TreeScope_Children, cond.p)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		arr, err := root.findAll(TreeScope_Children, cond.p)
		if err != nil {
			b.Fatal(err)
		}
		arr.release()
	}
}

func BenchmarkUIA_FindFirst_ByName(b *testing.B) {
	initCOM()
	defer windows.CoUninitialize()

	au, _ := newUIA()
	root, _ := au.getRootElement()
	names := []string{
		"Taskbar",
		"",
		"Program Manager",
	}
	conds := make([]*uiaCondition, len(names))
	for i, n := range names {
		conds[i], _ = au.createPropertyCondition(UIA_NamePropertyId, varString(n))
	}
	// warmup
	for _, c := range conds {
		root.findFirst(TreeScope_Descendants, c.p)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e, err := root.findFirst(TreeScope_Descendants, conds[i%len(conds)].p)
		if err != nil {
			b.Fatal(err)
		}
		if e != nil {
			e.release()
		}
	}
}

// ═══════════════════════════════════════════════════════════
// LEGACY: old test indices (keep for regression)
// ═══════════════════════════════════════════════════════════

func TestUIA999_LegacySanity(t *testing.T) {
	initCOM()
	defer windows.CoUninitialize()

	au, err := newUIA()
	if err != nil {
		t.Fatalf("newUIA: %v", err)
	}
	defer au.release()

	_ = vtblMethod(au.p, 5)
	_ = vtblMethod(au.p, 23)
	_ = vtblMethod(au.p, 21)

	root, err := au.getRootElement()
	if err != nil {
		t.Fatalf("getRootElement: %v", err)
	}
	defer root.release()

	name := root.getName()
	pid := root.getPid()
	t.Logf("Root: name=%q pid=%d enabled=%v", name, pid, root.isEnabled())

	cond, err := au.createPropertyCondition(UIA_NamePropertyId, varString(""))
	if err != nil {
		t.Fatalf("CreatePropertyCondition (vtbl[23]): %v", err)
	}
	defer cond.release()

	tc, err := au.createTrueCondition()
	if err != nil {
		t.Fatalf("CreateTrueCondition (vtbl[21]): %v", err)
	}
	tc.release()

	arr, err := root.findAll(TreeScope_Children, cond.p)
	if err != nil {
		t.Fatalf("FindAll (element vtbl[6]): %v", err)
	}
	defer arr.release()

	n := arr.length()
	t.Logf("Root children: %d", n)

	found, err := root.findFirst(TreeScope_Descendants, cond.p)
	if err != nil {
		t.Fatalf("FindFirst (element vtbl[5]): %v", err)
	}
	if found != nil {
		t.Logf("First descendant: %q", found.getName())
		found.release()
	}

	// ElementFromHandle
	modUser32 := windows.NewLazySystemDLL("user32.dll")
	procFindWindow := modUser32.NewProc("FindWindowW")
	cls, _ := syscall.UTF16PtrFromString("Shell_TrayWnd")
	trayHwnd, _, _ := procFindWindow.Call(uintptr(unsafe.Pointer(cls)), 0)
	if trayHwnd != 0 {
		elem, err := au.elementFromHandle(trayHwnd)
		if err != nil {
			t.Fatalf("ElementFromHandle (vtbl[6] on auto): %v", err)
		}
		elem.release()
	}

	// Enum edits (children only for speed)
	ec, _ := au.createPropertyCondition(UIA_ControlTypePropertyId, varInt(50004))
	edits, _ := root.findAll(TreeScope_Children, ec.p)
	t.Logf("Edit fields (children): %d", edits.length())
	edits.release()
	ec.release()
}

// ═══════════════════════════════════════════════════════════
// Comparison benchmark
// ═══════════════════════════════════════════════════════════

func TestUIA200_ComparePSandCOM(t *testing.T) {
	initCOM()
	defer windows.CoUninitialize()

	// PowerShell: find "Taskbar"
	t.Log("--- PowerShell ---")
	start := time.Now()
	ps1, err1 := UIAFindElement(UIAFindOpts{Name: "Taskbar"})
	t.Logf("  UIAFindElement(Name=Taskbar): %v  (err=%v)", time.Since(start), err1)
	t.Logf("  Result: %+v", ps1)

	start = time.Now()
	ps2, err2 := UIAFindElement(UIAFindOpts{Name: "Taskbar"})
	t.Logf("  Second call: %v  (err=%v)  count=%d", time.Since(start), err2, len(ps2))

	// COM: find "Taskbar"
	t.Log("--- COM ---")
	au, _ := newUIA()
	root, _ := au.getRootElement()
	cond, _ := au.createPropertyCondition(UIA_NamePropertyId, varString("Taskbar"))

	start = time.Now()
	found, err := root.findFirst(TreeScope_Descendants, cond.p)
	dur := time.Since(start)
	if err != nil {
		t.Fatalf("COM FindFirst: %v", err)
	}
	if found != nil {
		t.Logf("  FindFirst: %v", dur)
		t.Logf("  Result: Name=%q PID=%d Enabled=%v", found.getName(), found.getPid(), found.isEnabled())
		found.release()
	} else {
		t.Logf("  FindFirst: %v (not found)", dur)
	}

	// Second call
	start = time.Now()
	found2, _ := root.findFirst(TreeScope_Descendants, cond.p)
	t.Logf("  Second FindFirst: %v", time.Since(start))
	if found2 != nil {
		found2.release()
	}

	cond.release()
	root.release()
	au.release()
}


