//go:build vtable && windows

package actions

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// vtbl: 5
func TestVtable_IUIAutomation_GetRootElement(t *testing.T) {
	windows.CoInitializeEx(0, COINIT_MULTITHREADED)
	defer windows.CoUninitialize()

	au, err := newUIA()
	if err != nil {
		t.Fatalf("newUIA: %v", err)
	}
	defer au.release()

	root, err := au.getRootElement()
	if err != nil {
		t.Fatalf("GetRootElement (vtbl 5): %v", err)
	}
	defer root.release()
	if root.p == nil {
		t.Fatal("GetRootElement returned nil")
	}
}

// vtbl: 43
func TestVtable_IUIAutomationElement_BoundingRect(t *testing.T) {
	windows.CoInitializeEx(0, COINIT_MULTITHREADED)
	defer windows.CoUninitialize()

	au, err := newUIA()
	if err != nil {
		t.Fatalf("newUIA: %v", err)
	}
	defer au.release()

	root, err := au.getRootElement()
	if err != nil {
		t.Fatalf("GetRootElement: %v", err)
	}
	defer root.release()

	rect, err := root.getBoundingRect()
	if err != nil {
		t.Fatalf("get_CurrentBoundingRectangle (vtbl 43): %v", err)
	}
	if rect.Right <= rect.Left || rect.Bottom <= rect.Top {
		t.Fatalf("unreasonable rect: %+v", rect)
	}
	t.Logf("desktop rect: %+v", rect)
}

// vtbl: 5, 21
func TestVtable_IUIAutomationElement_FindFirst(t *testing.T) {
	windows.CoInitializeEx(0, COINIT_MULTITHREADED)
	defer windows.CoUninitialize()

	au, err := newUIA()
	if err != nil {
		t.Fatalf("newUIA: %v", err)
	}
	defer au.release()

	root, err := au.getRootElement()
	if err != nil {
		t.Fatalf("GetRootElement: %v", err)
	}
	defer root.release()

	cond, err := au.createTrueCondition()
	if err != nil {
		t.Fatalf("CreateTrueCondition (vtbl 21): %v", err)
	}
	defer cond.release()

	found, err := root.findFirst(TreeScope_Children, uintptr(cond.p))
	if err != nil {
		t.Fatalf("FindFirst (vtbl 5): %v", err)
	}
	if found != nil {
		found.release()
	}
}

// vtbl: 21, 23, 25
func TestVtable_IUIAutomation_Conditions(t *testing.T) {
	windows.CoInitializeEx(0, COINIT_MULTITHREADED)
	defer windows.CoUninitialize()

	au, err := newUIA()
	if err != nil {
		t.Fatalf("newUIA: %v", err)
	}
	defer au.release()

	trueCond, err := au.createTrueCondition()
	if err != nil {
		t.Fatalf("CreateTrueCondition (vtbl 21): %v", err)
	}
	defer trueCond.release()
	if trueCond.p == nil {
		t.Fatal("CreateTrueCondition returned nil")
	}

	propCond, err := au.createPropertyCondition(UIA_NamePropertyId, varString("Desktop"))
	if err != nil {
		t.Fatalf("CreatePropertyCondition (vtbl 23): %v", err)
	}
	defer propCond.release()
	if propCond.p == nil {
		t.Fatal("CreatePropertyCondition returned nil")
	}

	// vtbl 25 = CreateAndCondition (hit when both Name + AutomationID provided)
	nameCond, err := au.createPropertyCondition(UIA_NamePropertyId, varString("Desktop"))
	if err != nil {
		t.Fatalf("CreatePropertyCondition (vtbl 23): %v", err)
	}
	defer nameCond.release()
	propCond2, err := au.createPropertyCondition(UIA_AutomationIdPropertyId, varString("TitleBar"))
	if err != nil {
		t.Fatalf("CreatePropertyCondition (vtbl 23): %v", err)
	}
	defer propCond2.release()
	nameCondFake := nameCond
	propCond2Fake := propCond2

	andCond, err := buildCondition(au, UIAFindOpts{Name: "Desktop", AutomationID: "TitleBar"})
	if err != nil {
		t.Fatalf("buildCondition (vtbl 25): %v", err)
	}
	if andCond != nil {
		andCond.release()
	}
	_ = nameCondFake
	_ = propCond2Fake
}

// vtbl: 10
func TestVtable_IUIAutomationElement_GetPropValue(t *testing.T) {
	windows.CoInitializeEx(0, COINIT_MULTITHREADED)
	defer windows.CoUninitialize()

	au, err := newUIA()
	if err != nil {
		t.Fatalf("newUIA: %v", err)
	}
	defer au.release()

	root, err := au.getRootElement()
	if err != nil {
		t.Fatalf("GetRootElement: %v", err)
	}
	defer root.release()

	v, err := root.getPropValue(UIA_NamePropertyId)
	if err != nil {
		t.Fatalf("GetCurrentPropertyValue (vtbl 10): %v", err)
	}
	if v == nil {
		t.Fatal("GetCurrentPropertyValue returned nil")
	}
}

// ── WinRT OCR vtable ──

// vtbl: 7, 6
func TestVtable_OcrEngineStatics_GetLanguages(t *testing.T) {
	if err := ensureRo(); err != nil {
		t.Skipf("WinRT not available: %v", err)
	}

	hClass, err := newHString("Windows.Media.Ocr.OcrEngine")
	if err != nil {
		t.Fatalf("newHString: %v", err)
	}
	defer freeHString(hClass)

	factory, err := roGetActivationFactory(hClass, IID_IOcrEngineStatics)
	if err != nil {
		t.Fatalf("RoGetActivationFactory: %v", err)
	}
	defer comRelease(factory)
	if factory == nil {
		t.Fatal("factory is nil")
	}

	var view unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(factory, 7), uintptr(factory), uintptr(unsafe.Pointer(&view)))
	// 7 = get_AvailableRecognizerLanguages
	if r != 0 {
		t.Fatalf("get_AvailableRecognizerLanguages (vtbl 7) failed: 0x%X", r)
	}
	if view == nil {
		t.Fatal("languages view is nil")
	}
	defer comRelease(view)

	var count uint32
	r, _, _ = syscall.SyscallN(vtblMethod(view, 7), uintptr(view), uintptr(unsafe.Pointer(&count)))
	// 7 = IVectorView::get_Size
	if r != 0 {
		t.Fatalf("get_Size (vtbl 7) failed: 0x%X", r)
	}
	t.Logf("OCR languages available: %d", count)
}

// vtbl: 10
func TestVtable_OcrEngine_TryCreate(t *testing.T) {
	if err := ensureRo(); err != nil {
		t.Skipf("WinRT not available: %v", err)
	}

	hClass, err := newHString("Windows.Media.Ocr.OcrEngine")
	if err != nil {
		t.Fatalf("newHString: %v", err)
	}
	defer freeHString(hClass)

	factory, err := roGetActivationFactory(hClass, IID_IOcrEngineStatics)
	if err != nil {
		t.Fatalf("RoGetActivationFactory: %v", err)
	}
	defer comRelease(factory)

	var engine unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(factory, 10), uintptr(factory), uintptr(unsafe.Pointer(&engine)))
	// 10 = TryCreateFromUserProfileLanguages
	if r != 0 {
		t.Fatalf("TryCreateFromUserProfileLanguages (vtbl 10) failed: 0x%X", r)
	}
	if engine == nil {
		t.Fatal("engine is nil — no OCR languages installed?")
	}
	defer comRelease(engine)
}

// ── WinRT async + StorageFile vtable ──

// vtbl: 6, 8, 14, 0, 7
func TestVtable_StorageFile_Async(t *testing.T) {
	if err := ensureRo(); err != nil {
		t.Skipf("WinRT not available: %v", err)
	}

	tmpDir := os.Getenv("TEMP")
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}
	testFile := filepath.Join(tmpDir, "vtable_test_storage.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	defer os.Remove(testFile)

	hClass, err := newHString("Windows.Storage.StorageFile")
	if err != nil {
		t.Fatalf("newHString: %v", err)
	}
	defer freeHString(hClass)

	factory, err := roGetActivationFactory(hClass, IID_IStorageFileStatics)
	if err != nil {
		t.Fatalf("RoGetActivationFactory: %v", err)
	}
	defer comRelease(factory)

	hPath, err := newHString(testFile)
	if err != nil {
		t.Fatalf("newHString path: %v", err)
	}
	defer freeHString(hPath)

	var asyncOp unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(factory, 6), uintptr(factory), uintptr(hPath), uintptr(unsafe.Pointer(&asyncOp)))
	// 6 = GetFileFromPathAsync
	if r != 0 {
		t.Fatalf("GetFileFromPathAsync (vtbl 6) failed: 0x%X", r)
	}
	defer comRelease(asyncOp)

	if err := waitForAsync(asyncOp, 30*time.Second); err != nil {
		t.Fatalf("waitForAsync: %v", err)
	}

	result, err := getAsyncObj(asyncOp, 30*time.Second)
	if err != nil {
		t.Fatalf("getAsyncObj: %v", err)
	}

	sf, err := qei(result, IID_IStorageFile)
	if err != nil {
		t.Fatalf("QI IStorageFile: %v", err)
	}
	defer comRelease(sf)
}

// ── HSTRING lifecycle ──

func TestVtable_HSTRING_CreateExtract(t *testing.T) {
	if err := ensureRo(); err != nil {
		t.Skipf("WinRT not available: %v", err)
	}

	cases := []string{"hello", "", "Windows.Media.Ocr.OcrEngine"}
	for _, c := range cases {
		h, err := newHString(c)
		if err != nil {
			t.Fatalf("newHString(%q): %v", c, err)
		}

		s, err := hstringToString(h)
		freeHString(h)
		if err != nil {
			t.Fatalf("hstringToString(%q): %v", c, err)
		}
		if s != c {
			t.Fatalf("round-trip: got %q, want %q", s, c)
		}
	}
}

// vtbl: 16
func TestVtable_IUIAutomationElement_GetCurrentPattern(t *testing.T) {
	windows.CoInitializeEx(0, COINIT_MULTITHREADED)
	defer windows.CoUninitialize()

	au, err := newUIA()
	if err != nil {
		t.Fatalf("newUIA: %v", err)
	}
	defer au.release()

	root, err := au.getRootElement()
	if err != nil {
		t.Fatalf("GetRootElement: %v", err)
	}
	defer root.release()

	p, err := root.getCurrentPattern(10002) // ValuePattern
	if err != nil {
		// "not supported" is expected for plain desktop; it still proves vtbl 16 works
		t.Logf("GetCurrentPattern (vtbl 16) ValuePattern: %v (expected on desktop)", err)
	} else {
		defer comRelease(p)
	}

	// TextPattern (10018) is also commonly available on desktop
	p2, err := root.getCurrentPattern(10018) // TextPattern
	if err != nil {
		t.Logf("GetCurrentPattern (vtbl 16) TextPattern: %v (expected on desktop)", err)
	} else {
		defer comRelease(p2)
	}
}

// vtbl: 9
func TestVtable_OcrEngine_TryCreateFromLanguage(t *testing.T) {
	if err := ensureRo(); err != nil {
		t.Skipf("WinRT not available: %v", err)
	}

	langObj, err := createLanguage("en-US")
	if err != nil {
		t.Fatalf("createLanguage: %v", err)
	}
	defer comRelease(langObj)

	hClass, err := newHString("Windows.Media.Ocr.OcrEngine")
	if err != nil {
		t.Fatalf("newHString: %v", err)
	}
	defer freeHString(hClass)

	factory, err := roGetActivationFactory(hClass, IID_IOcrEngineStatics)
	if err != nil {
		t.Fatalf("RoGetActivationFactory: %v", err)
	}
	defer comRelease(factory)

	var engine unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(factory, 9), uintptr(factory), uintptr(langObj), uintptr(unsafe.Pointer(&engine)))
	if r != 0 {
		t.Fatalf("TryCreateFromLanguage (vtbl 9) failed: 0x%X", r)
	}
	if engine != nil {
		comRelease(engine)
	}
}

// vtbl: 3, 4
func TestVtable_IUIAutomation_ValuePattern_Invoke_Array(t *testing.T) {
	windows.CoInitializeEx(0, COINIT_MULTITHREADED)
	defer windows.CoUninitialize()

	au, err := newUIA()
	if err != nil {
		t.Fatalf("newUIA: %v", err)
	}
	defer au.release()

	// Try ValuePattern.get_Value (vtbl 3) on desktop — will fail with E_NOTIMPL
	// but that confirms the index doesn't crash; success = S_OK (editable control)
	root, err := au.getRootElement()
	if err != nil {
		t.Fatalf("GetRootElement: %v", err)
	}
	defer root.release()

	// Bounding rect (vtbl 43) is already tested; we just need to verify
	// that calling ValuePattern/InvokePattern methods doesn't AV.
	// getValue() returns error if no ValuePattern — that's expected.
	_, _ = root.getValue()
	_ = root.isEnabled()

	// Element array Length (vtbl 3) and GetElement (vtbl 4) via FindAll on desktop root.
	// This walks all top-level windows via cross-process UIA COM — expect 10-30s on first run.
	// We use the full desktop scope intentionally: every vtbl index that exists in production
	// must be exercised by at least one test, even if slow. This suite only runs with
	// `-tags=vtable`, not in normal builds or CI on every push.
	cond, err := au.createTrueCondition()
	if err != nil {
		t.Fatalf("CreateTrueCondition (vtbl 21): %v", err)
	}
	defer cond.release()

	arr, err := root.findAll(TreeScope_Children, uintptr(cond.p))
	if err != nil {
		t.Fatalf("FindAll (vtbl 6): %v", err)
	}
	defer arr.release()
	n := arr.length()
	t.Logf("desktop children count: %d", n)
	if n > 0 {
		child, err := arr.get(0)
		if err != nil {
			t.Fatalf("GetElement(0) (vtbl 4): %v", err)
		}
		defer child.release()
	}
}

// ── Internal dispatch helper vtblMethod ──

func TestVtable_vtblMethod_rejectsNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("vtblMethod(nil, 0) should panic")
		}
	}()
	vtblMethod(nil, 0)
}
