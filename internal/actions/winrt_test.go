package actions

import (
	"testing"
)

func TestWinRT_HSTRING(t *testing.T) {
	h, err := newHString("test-string")
	if err != nil {
		t.Fatalf("newHString failed: %v", err)
	}
	defer freeHString(h)

	s, err := hstringToString(h)
	if err != nil {
		t.Fatalf("hstringToString failed: %v", err)
	}
	if s != "test-string" {
		t.Fatalf("got %q, want %q", s, "test-string")
	}
}

func TestWinRT_HSTRING_Empty(t *testing.T) {
	h, err := newHString("")
	if err != nil {
		t.Fatalf("newHString empty failed: %v", err)
	}
	defer freeHString(h)

	s, err := hstringToString(h)
	if err != nil {
		t.Fatalf("hstringToString empty failed: %v", err)
	}
	if s != "" {
		t.Fatalf("got %q, want empty", s)
	}
}

func TestWinRT_Initialize(t *testing.T) {
	err := ensureRo()
	if err != nil {
		t.Fatalf("ensureRo failed: %v", err)
	}
}

func TestWinRT_ActivationFactory(t *testing.T) {
	if err := ensureRo(); err != nil {
		t.Skipf("WinRT not available: %v", err)
	}

	h, err := newHString("Windows.Media.Ocr.OcrEngine")
	if err != nil {
		t.Fatalf("newHString failed: %v", err)
	}
	defer freeHString(h)

	factory, err := roGetActivationFactory(h, IID_IActivationFactory)
	if err != nil {
		t.Fatalf("roGetActivationFactory failed: %v", err)
	}
	t.Logf("factory ptr: %v", factory)
	defer comRelease(factory)
}
