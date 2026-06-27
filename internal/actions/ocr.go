package actions

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
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

var ocrScript = `Add-Type -AssemblyName System.Runtime.WindowsRuntime
$imgPath = $args[0]
$file = Get-Item -LiteralPath $imgPath -ErrorAction Stop
$stream = [Windows.Storage.Streams.FileRandomAccessStream]::OpenAsync($file, [Windows.Storage.FileAccessMode]::Read).GetAwaiter().GetResult()
$decoder = [Windows.Graphics.Imaging.BitmapDecoder]::CreateAsync($stream).GetAwaiter().GetResult()
$sb = $decoder.GetSoftwareBitmapAsync().GetAwaiter().GetResult()
$engine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromUserProfileLanguages()
if ($engine -eq $null) { Write-Output '{"text":"","lines":[],"words":[]}'; $stream.Dispose(); exit }
$result = $engine.RecognizeAsync($sb).GetAwaiter().GetResult()
if ($result -eq $null) { Write-Output '{"text":"","lines":[],"words":[]}'; $stream.Dispose(); exit }
$lines = @(); $words = @()
foreach ($line in $result.Lines) {
  $lines += @{text=$line.Text; x=[double]$line.BoundingRect.X; y=[double]$line.BoundingRect.Y; w=[double]$line.BoundingRect.Width; h=[double]$line.BoundingRect.Height}
  foreach ($word in $line.Words) {
    $words += @{text=$word.Text; x=[double]$word.BoundingRect.X; y=[double]$word.BoundingRect.Y; w=[double]$word.BoundingRect.Width; h=[double]$word.BoundingRect.Height}
  }
}
$o = @{text=[string]$result.Text; lines=@($lines); words=@($words)}
$o | ConvertTo-Json -Compress
$stream.Dispose()
`

func ocrExec(imgPath string) (*OCRResult, error) {
	tmpDir := os.Getenv("TEMP")
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}
	scriptPath := filepath.Join(tmpDir, fmt.Sprintf("ocr_%d.ps1", time.Now().UnixNano()))
	if err := os.WriteFile(scriptPath, []byte(ocrScript), 0); err != nil {
		return nil, fmt.Errorf("write ocr script: %w", err)
	}
	defer os.Remove(scriptPath)

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath, imgPath)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("ocr exec: %w (stderr: %s)", err, string(ee.Stderr))
		}
		return nil, fmt.Errorf("ocr exec: %w", err)
	}

	var result OCRResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("ocr parse: %w", err)
	}
	return &result, nil
}

func OCRScreen() (*OCRResult, error) {
	b64, err := CaptureScreen()
	if err != nil {
		return nil, fmt.Errorf("ocr screenshot: %w", err)
	}
	return ocrFromBase64(b64)
}

func OCRRegion(x, y, w, h int32) (*OCRResult, error) {
	b64, err := CaptureRegion(x, y, w, h)
	if err != nil {
		return nil, fmt.Errorf("ocr region shot: %w", err)
	}
	return ocrFromBase64(b64)
}

func ocrFromBase64(b64 string) (*OCRResult, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("ocr decode b64: %w", err)
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

	return ocrExec(imgPath)
}
