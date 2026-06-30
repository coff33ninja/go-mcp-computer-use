//go:build windows

package actions

import (
	"testing"
)

func TestCtrlTypeMap_AllConstantsCovered(t *testing.T) {
	names := []string{
		"Button", "Calendar", "CheckBox", "ComboBox", "Edit", "Hyperlink",
		"Image", "ListItem", "List", "Menu", "MenuBar", "MenuItem",
		"ProgressBar", "RadioButton", "ScrollBar", "Slider", "Spinner", "StatusBar",
		"Tab", "TabItem", "Text", "ToolBar", "ToolTip", "Tree",
		"TreeItem", "Custom", "Group", "Thumb", "DataGrid", "DataItem",
		"Document", "SplitButton", "Window", "Pane", "Header", "HeaderItem",
		"Table", "TitleBar", "Separator", "SemanticZoom", "AppBar",
	}
	covered := map[int32]bool{}
	for _, name := range names {
		v := ctrlTypeFromName(name)
		if v == nil {
			t.Errorf("ctrlTypeFromName(%q) returned nil", name)
			continue
		}
		if *v < 50000 || *v > 50040 {
			t.Errorf("%q = %d, outside [50000,50040]", name, *v)
		}
		if covered[*v] {
			t.Errorf("duplicate id %d for %q", *v, name)
		}
		covered[*v] = true
	}
	for id := int32(50000); id <= 50040; id++ {
		if !covered[id] {
			t.Errorf("missing UIA ControlTypeId %d", id)
		}
	}
}

func TestCtrlTypeFromName_Known(t *testing.T) {
	tests := []struct {
		name string
		id   int32
	}{
		{"Button", 50000},
		{"AppBar", 50040},
		{"Custom", 50025},
	}
	for _, tc := range tests {
		v := ctrlTypeFromName(tc.name)
		if v == nil || *v != tc.id {
			t.Errorf("ctrlTypeFromName(%q) = %v, want %d", tc.name, v, tc.id)
		}
	}
}

func TestCtrlTypeFromName_Unknown(t *testing.T) {
	if v := ctrlTypeFromName("NonExistent"); v != nil {
		t.Errorf("expected nil, got %d", *v)
	}
}
