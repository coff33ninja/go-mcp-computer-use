package actions

import "testing"

func TestKeyNameToVK(t *testing.T) {
	cases := []struct {
		name string
		want uint16
		ok   bool
	}{
		// Modifiers
		{"WIN", 0x5B, true},
		{"META", 0x5B, true},
		{"LWIN", 0x5B, true},
		{"SUPER", 0x5B, true},
		{"CMD", 0x5B, true},
		{"COMMAND", 0x5B, true},
		{"RWIN", 0x5C, true},
		{"LCTRL", 0xA2, true},
		{"LCONTROL", 0xA2, true},
		{"RCTRL", 0xA3, true},
		{"LALT", 0xA4, true},
		{"RALT", 0xA5, true},
		{"LSHIFT", 0xA0, true},
		{"RSHIFT", 0xA1, true},
		// Apps / snapshot
		{"APPS", 0x5D, true},
		{"CONTEXT", 0x5D, true},
		{"SNAPSHOT", 0x2C, true},
		{"CLEAR", 0x0C, true},
		{"EXECUTE", 0x2B, true},
		{"SLEEP", 0x5F, true},
		// Numpad
		{"NUMPAD0", 0x60, true},
		{"NUMPAD5", 0x65, true},
		{"NUMPAD9", 0x69, true},
		{"NUMPAD_MULTIPLY", 0x6A, true},
		{"NUMPAD_ADD", 0x6B, true},
		{"NUMPAD_SEPARATOR", 0x6C, true},
		{"NUMPAD_SUBTRACT", 0x6D, true},
		{"NUMPAD_DECIMAL", 0x6E, true},
		{"NUMPAD_DIVIDE", 0x6F, true},
		// Media / volume
		{"VOLUME_MUTE", 0xAD, true},
		{"VOLUME_DOWN", 0xAE, true},
		{"VOLUME_UP", 0xAF, true},
		{"MEDIA_NEXT_TRACK", 0xB0, true},
		{"MEDIA_PREV_TRACK", 0xB1, true},
		{"MEDIA_STOP", 0xB2, true},
		{"MEDIA_PLAY_PAUSE", 0xB3, true},
		// Single-char punctuation via charToVK
		{".", 0xBE, true},
		{"-", 0xBD, true},
		{",", 0xBC, true},
		{"/", 0xBF, true},
		{"[", 0xDB, true},
		// Unknown keys
		{"NOTAKEY", 0, false},
		{"", 0, false},
	}

	for _, c := range cases {
		got, ok := keyNameToVK(c.name)
		if ok != c.ok {
			t.Errorf("keyNameToVK(%q) ok = %v, want %v", c.name, ok, c.ok)
			continue
		}
		if ok && got != c.want {
			t.Errorf("keyNameToVK(%q) = 0x%X, want 0x%X", c.name, got, c.want)
		}
	}
}

func TestKeyNameToVKCaseInsensitive(t *testing.T) {
	if got, ok := keyNameToVK("win"); !ok || got != 0x5B {
		t.Errorf("keyNameToVK(\"win\") = 0x%X, %v; want 0x5B, true", got, ok)
	}
	if got, ok := keyNameToVK("Meta"); !ok || got != 0x5B {
		t.Errorf("keyNameToVK(\"Meta\") = 0x%X, %v; want 0x5B, true", got, ok)
	}
}
