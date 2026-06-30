package actions

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

type OCRLine struct {
	Text string  `json:"text"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	W    float64 `json:"w"`
	H    float64 `json:"h"`
}

type OCRWord struct {
	Text string  `json:"text"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	W    float64 `json:"w"`
	H    float64 `json:"h"`
}

type OCRResult struct {
	Text  string    `json:"text"`
	Lines []OCRLine `json:"lines"`
	Words []OCRWord `json:"words"`
}

var ocrScript = `param($imgPath, $lang)
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$null = [Windows.Storage.StorageFile, Windows.Storage, ContentType=WindowsRuntime]
$null = [Windows.Media.Ocr.OcrEngine, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Graphics.Imaging.SoftwareBitmap, Windows.Foundation, ContentType=WindowsRuntime]
$null = [Windows.Storage.Streams.RandomAccessStream, Windows.Storage.Streams, ContentType=WindowsRuntime]
$awaiter = [WindowsRuntimeSystemExtensions].GetMember('GetAwaiter', 'Method', 'Public,Static') | Where-Object { $_.Name -eq 'GetAwaiter' -and $_.GetParameters()[0].ParameterType.Name -like 'IAsyncOperation*' } | Select-Object -First 1
function Invoke-Async([object]$AsyncTask, [Type]$As) { return $awaiter.MakeGenericMethod($As).Invoke($null, @($AsyncTask)).GetResult() }
$storageFile = Invoke-Async ([Windows.Storage.StorageFile]::GetFileFromPathAsync($imgPath)) ([Windows.Storage.StorageFile])
$fileStream = Invoke-Async ($storageFile.OpenReadAsync()) ([Windows.Storage.Streams.IRandomAccessStreamWithContentType])
$bitmapDecoder = Invoke-Async ([Windows.Graphics.Imaging.BitmapDecoder]::CreateAsync($fileStream)) ([Windows.Graphics.Imaging.BitmapDecoder])
$softwareBitmap = Invoke-Async ($bitmapDecoder.GetSoftwareBitmapAsync()) ([Windows.Graphics.Imaging.SoftwareBitmap])
if ($lang) { $culture = [Windows.Globalization.Language]::new($lang); $engine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromLanguage($culture) } else { $engine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromUserProfileLanguages() }
if ($engine -eq $null) { Write-Output '{"text":"","lines":[],"words":[]}'; exit }
$ocrResult = Invoke-Async ($engine.RecognizeAsync($softwareBitmap)) ([Windows.Media.Ocr.OcrResult])
if ($ocrResult -eq $null) { Write-Output '{"text":"","lines":[],"words":[]}'; exit }
$lines = @(); $words = @()
foreach ($line in $ocrResult.Lines) { $lines += @{text=$line.Text; x=[double]$line.BoundingRect.X; y=[double]$line.BoundingRect.Y; w=[double]$line.BoundingRect.Width; h=[double]$line.BoundingRect.Height}; foreach ($word in $line.Words) { $words += @{text=$word.Text; x=[double]$word.BoundingRect.X; y=[double]$word.BoundingRect.Y; w=[double]$word.BoundingRect.Width; h=[double]$word.BoundingRect.Height} } }
$o = @{text=[string]$ocrResult.Text; lines=@($lines); words=@($words)}
$o | ConvertTo-Json -Compress
`

func ocrExec(imgPath, language string) (*OCRResult, error) {
	tmpDir := os.Getenv("TEMP")
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}
	scriptPath := filepath.Join(tmpDir, fmt.Sprintf("ocr_%d.ps1", time.Now().UnixNano()))
	if err := os.WriteFile(scriptPath, []byte(ocrScript), 0); err != nil {
		return nil, fmt.Errorf("write ocr script: %w", err)
	}
	defer os.Remove(scriptPath)

	var result OCRResult
	err := WithTimeout(func() error {
		args := []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath, imgPath}
		if language != "" {
			args = append(args, language)
		}
		cmd := exec.Command("powershell", args...)
		out, err := cmd.Output()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				return fmt.Errorf("ocr exec: %w (stderr: %s)", err, string(ee.Stderr))
			}
			return fmt.Errorf("ocr exec: %w", err)
		}
		if err := json.Unmarshal(out, &result); err != nil {
			return fmt.Errorf("ocr parse: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func ocrExecWithRetry(imgPath, language string) (*OCRResult, error) {
	var lastErr error
	for i := 0; i < 3; i++ {
		result, err := ocrExec(imgPath, language)
		if err == nil {
			return result, nil
		}
		lastErr = err
		Wait(500)
	}
	return nil, fmt.Errorf("ocr failed after 3 retries: %w", lastErr)
}

func OCRScreen(language string) (*OCRResult, error) {
	b64, err := CaptureScreen()
	if err != nil {
		return nil, fmt.Errorf("ocr screenshot: %w", err)
	}
	result, err := ocrFromBase64(b64, language)
	if err == nil && result != nil {
		pushRecentOCR(result.Text)
		tryCompletePair(result.Text, "")
		go LogOCRSnapshot("tool", "ocr_screen", "", result)
	}
	return result, err
}

func OCRRegion(x, y, w, h int32, language string) (*OCRResult, error) {
	b64, err := CaptureRegion(x, y, w, h)
	if err != nil {
		return nil, fmt.Errorf("ocr region shot: %w", err)
	}
	result, err := ocrFromBase64(b64, language)
	if err == nil && result != nil {
		pushRecentOCR(result.Text)
		tryCompletePair(result.Text, "")
		go LogOCRSnapshot("tool", "ocr_region", "", result)
	}
	return result, err
}

func OCRProportionalWindowRegion(hwnd uintptr, leftFrac, topFrac, rightFrac, bottomFrac float64, language string) (*OCRResult, error) {
	wn, err := NewWindowNormalizer(hwnd)
	if err != nil {
		return nil, fmt.Errorf("ocr proportional region: %w", err)
	}
	x, y, w, h := wn.ProportionalRegion(leftFrac, topFrac, rightFrac, bottomFrac)
	return OCRRegion(x, y, w, h, language)
}

type LanguageInfo struct {
	Tag         string `json:"tag"`
	DisplayName string `json:"display_name"`
	NativeName  string `json:"native_name"`
}

// ── OcrLanguages vtable (IOcrEngineStatics) ──
// Verified 2026-06-30 — Win11 26200 (24H2), SDK 10.0.26200.0
//   7 = get_AvailableRecognizerLanguages (returns IVectorView<Language>)
// IVectorView<Language>:
//   6 = GetAt
//   7 = get_Size
// Language:
//   6 = get_LanguageTag (callStringGetter)
//   7 = get_DisplayName (callStringGetter)
//   8 = get_NativeName (callStringGetter)

func OcrLanguages() ([]LanguageInfo, error) {
	if err := ensureRo(); err != nil {
		return nil, fmt.Errorf("WinRT init: %w", err)
	}

	hClass, err := newHString("Windows.Media.Ocr.OcrEngine")
	if err != nil {
		return nil, err
	}
	defer freeHString(hClass)

	factory, err := roGetActivationFactory(hClass, IID_IOcrEngineStatics)
	if err != nil {
		return nil, fmt.Errorf("OcrEngineStatics: %w", err)
	}
	defer comRelease(factory)

	var languagesView unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(factory, 7), uintptr(factory), uintptr(unsafe.Pointer(&languagesView))) // 7 = get_AvailableRecognizerLanguages
	if r != 0 {
		return nil, fmt.Errorf("get_AvailableRecognizerLanguages 0x%X", r)
	}
	defer comRelease(languagesView)

	var count uint32
	r, _, _ = syscall.SyscallN(vtblMethod(languagesView, 7), uintptr(languagesView), uintptr(unsafe.Pointer(&count))) // 7 = get_Size
	if r != 0 {
		return nil, fmt.Errorf("IVectorView::get_Size 0x%X", r)
	}

	var result []LanguageInfo
	for i := uint32(0); i < count; i++ {
		var lang unsafe.Pointer
		r, _, _ := syscall.SyscallN(vtblMethod(languagesView, 6), uintptr(languagesView), uintptr(i), uintptr(unsafe.Pointer(&lang))) // 6 = GetAt
		if r != 0 {
			continue
		}

		tag, _ := callStringGetter(lang, 6)
		displayName, _ := callStringGetter(lang, 7)
		nativeName, _ := callStringGetter(lang, 8)

		comRelease(lang)

		result = append(result, LanguageInfo{
			Tag:         tag,
			DisplayName: displayName,
			NativeName:  nativeName,
		})
	}

	return result, nil
}

func ocrFromBase64(b64, language string) (*OCRResult, error) {
	if b64 == "" {
		return &OCRResult{}, nil
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("ocr decode b64: %w", err)
	}
	if len(data) == 0 {
		return &OCRResult{}, nil
	}

	tmpDir := os.Getenv("TEMP")
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}
	imgPath := filepath.Join(tmpDir, fmt.Sprintf("ocr_%d.png", time.Now().UnixNano()))
	if err := os.WriteFile(imgPath, data, 0); err != nil {
		return nil, fmt.Errorf("ocr write img: %w", err)
	}
	defer os.Remove(imgPath)

	result, err := ocrNative(imgPath, language)
	if err == nil {
		return result, nil
	}

	return ocrExecWithRetry(imgPath, language)
}
