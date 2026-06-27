package actions

import (
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type WinRect struct {
	X, Y, W, H float32
}

var (
	roOnce   sync.Once
	roErr    error
	roInited bool
)

func ensureRo() error {
	roOnce.Do(func() {
		err := roInitialize(RO_INIT_MULTITHREADED)
		if err != nil {
			roErr = err
			return
		}
		roInited = true
	})
	return roErr
}

func newHString(s string) (HSTRING, error) {
	return windowsCreateString(s)
}

func freeHString(h HSTRING) {
	windowsDeleteString(h)
}

func createLanguage(language string) (unsafe.Pointer, error) {
	hLangClass, err := newHString("Windows.Globalization.Language")
	if err != nil {
		return nil, fmt.Errorf("Language HSTRING: %w", err)
	}
	defer freeHString(hLangClass)

	factory, err := roGetActivationFactory(hLangClass, IID_ILanguageFactory)
	if err != nil {
		return nil, fmt.Errorf("LanguageFactory: %w", err)
	}
	defer comRelease(factory)

	hTag, err := newHString(language)
	if err != nil {
		return nil, fmt.Errorf("lang tag HSTRING: %w", err)
	}
	defer freeHString(hTag)

	var langObj unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(factory, 6), uintptr(factory), uintptr(hTag), uintptr(unsafe.Pointer(&langObj)))
	if r != 0 {
		return nil, fmt.Errorf("CreateLanguage 0x%X", r)
	}
	return langObj, nil
}

func openStorageFile(path string) (unsafe.Pointer, error) {
	hClass, err := newHString("Windows.Storage.StorageFile")
	if err != nil {
		return nil, err
	}
	defer freeHString(hClass)

	factory, err := roGetActivationFactory(hClass, IID_IStorageFileStatics)
	if err != nil {
		return nil, fmt.Errorf("StorageFileStatics: %w", err)
	}
	defer comRelease(factory)

	hPath, err := newHString(path)
	if err != nil {
		return nil, err
	}
	defer freeHString(hPath)

	var asyncOp unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(factory, 6), uintptr(factory), uintptr(hPath), uintptr(unsafe.Pointer(&asyncOp)))
	if r != 0 {
		return nil, fmt.Errorf("GetFileFromPathAsync 0x%X", r)
	}
	defer comRelease(asyncOp)

	result, err := getAsyncObj(asyncOp, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("get storage file: %w", err)
	}
	defer comRelease(result)

	storageFile, err := qei(result, IID_IStorageFile)
	if err != nil {
		return nil, fmt.Errorf("QI IStorageFile: %w", err)
	}
	return storageFile, nil
}

func openStream(storageFile unsafe.Pointer) (unsafe.Pointer, error) {
	var asyncOp unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(storageFile, 8), uintptr(storageFile), uintptr(FileAccessModeRead), uintptr(unsafe.Pointer(&asyncOp)))
	if r != 0 {
		return nil, fmt.Errorf("OpenAsync 0x%X", r)
	}
	defer comRelease(asyncOp)

	stream, err := getAsyncObj(asyncOp, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("get stream: %w", err)
	}
	return stream, nil
}

func createDecoder(stream unsafe.Pointer) (unsafe.Pointer, error) {
	hClass, err := newHString("Windows.Graphics.Imaging.BitmapDecoder")
	if err != nil {
		return nil, err
	}
	defer freeHString(hClass)

	factory, err := roGetActivationFactory(hClass, IID_IBitmapDecoderStatics)
	if err != nil {
		return nil, fmt.Errorf("BitmapDecoderStatics: %w", err)
	}
	defer comRelease(factory)

	var asyncOp unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(factory, 14), uintptr(factory), uintptr(stream), uintptr(unsafe.Pointer(&asyncOp)))
	if r != 0 {
		return nil, fmt.Errorf("CreateAsync 0x%X", r)
	}
	defer comRelease(asyncOp)

	result, err := getAsyncObj(asyncOp, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("get decoder: %w", err)
	}
	defer comRelease(result)

	swBitmapPtr, err := qei(result, IID_IBitmapFrameWithSoftwareBitmap)
	if err != nil {
		return nil, fmt.Errorf("QI SWBitmap: %w", err)
	}
	return swBitmapPtr, nil
}

func getSoftwareBitmap(swBitmapPtr unsafe.Pointer) (unsafe.Pointer, error) {
	var asyncOp unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(swBitmapPtr, 6), uintptr(swBitmapPtr), uintptr(unsafe.Pointer(&asyncOp)))
	if r != 0 {
		return nil, fmt.Errorf("GetSoftwareBitmapAsync 0x%X", r)
	}
	defer comRelease(asyncOp)

	swBitmap, err := getAsyncObj(asyncOp, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("get software bitmap: %w", err)
	}
	return swBitmap, nil
}

func createOcrEngine(language string) (unsafe.Pointer, error) {
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

	if language != "" {
		langObj, err := createLanguage(language)
		if err != nil {
			return nil, fmt.Errorf("create language: %w", err)
		}
		defer comRelease(langObj)

		var engine unsafe.Pointer
		r, _, _ := syscall.SyscallN(vtblMethod(factory, 9), uintptr(factory), uintptr(langObj), uintptr(unsafe.Pointer(&engine)))
		if r != 0 {
			return nil, fmt.Errorf("TryCreateFromLanguage 0x%X", r)
		}
		if engine == nil {
			return nil, fmt.Errorf("language %q not supported by OCR", language)
		}
		return engine, nil
	}

	var engine unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(factory, 10), uintptr(factory), uintptr(unsafe.Pointer(&engine)))
	if r != 0 {
		return nil, fmt.Errorf("TryCreateFromUserProfileLanguages 0x%X", r)
	}
	if engine == nil {
		return nil, fmt.Errorf("no OCR engine available")
	}
	return engine, nil
}

func recognizeOcr(engine, swBitmap unsafe.Pointer) (unsafe.Pointer, error) {
	var asyncOp unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(engine, 6), uintptr(engine), uintptr(swBitmap), uintptr(unsafe.Pointer(&asyncOp)))
	if r != 0 {
		return nil, fmt.Errorf("RecognizeAsync 0x%X", r)
	}
	defer comRelease(asyncOp)

	result, err := getAsyncObj(asyncOp, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("get OCR result: %w", err)
	}
	return result, nil
}

func getOcrLines(ocrResult unsafe.Pointer) (unsafe.Pointer, uint32, error) {
	lines, err := callObjectGetter(ocrResult, 6)
	if err != nil {
		return nil, 0, fmt.Errorf("get_Lines: %w", err)
	}
	var count uint32
	r, _, _ := syscall.SyscallN(vtblMethod(lines, 7), uintptr(lines), uintptr(unsafe.Pointer(&count)))
	if r != 0 {
		comRelease(lines)
		return nil, 0, fmt.Errorf("get_Size 0x%X", r)
	}
	return lines, count, nil
}

func getLineAt(lines unsafe.Pointer, index uint32) (unsafe.Pointer, error) {
	var line unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(lines, 6), uintptr(lines), uintptr(index), uintptr(unsafe.Pointer(&line)))
	if r != 0 {
		return nil, fmt.Errorf("GetAt[%d] 0x%X", index, r)
	}
	return line, nil
}

func getLineText(line unsafe.Pointer) (string, error) {
	return callStringGetter(line, 7)
}

func getLineWords(line unsafe.Pointer) (unsafe.Pointer, uint32, error) {
	words, err := callObjectGetter(line, 6)
	if err != nil {
		return nil, 0, fmt.Errorf("get_Words: %w", err)
	}
	var count uint32
	r, _, _ := syscall.SyscallN(vtblMethod(words, 7), uintptr(words), uintptr(unsafe.Pointer(&count)))
	if r != 0 {
		comRelease(words)
		return nil, 0, fmt.Errorf("get_Size 0x%X", r)
	}
	return words, count, nil
}

func getWordAt(words unsafe.Pointer, index uint32) (unsafe.Pointer, error) {
	var word unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(words, 6), uintptr(words), uintptr(index), uintptr(unsafe.Pointer(&word)))
	if r != 0 {
		return nil, fmt.Errorf("GetAt[%d] 0x%X", index, r)
	}
	return word, nil
}

func getWordText(word unsafe.Pointer) (string, error) {
	return callStringGetter(word, 7)
}

func getWordRect(word unsafe.Pointer) (WinRect, error) {
	var r WinRect
	ret, _, _ := syscall.SyscallN(vtblMethod(word, 6), uintptr(word), uintptr(unsafe.Pointer(&r)))
	if ret != 0 {
		return r, fmt.Errorf("get_BoundingRect 0x%X", ret)
	}
	return r, nil
}

func getOcrText(ocrResult unsafe.Pointer) (string, error) {
	return callStringGetter(ocrResult, 8)
}

func ocrNative(imgPath, language string) (*OCRResult, error) {
	if err := ensureRo(); err != nil {
		return nil, fmt.Errorf("WinRT init: %w", err)
	}

	storageFile, err := openStorageFile(imgPath)
	if err != nil {
		return nil, fmt.Errorf("open storage file: %w", err)
	}
	defer comRelease(storageFile)

	stream, err := openStream(storageFile)
	if err != nil {
		return nil, fmt.Errorf("open stream: %w", err)
	}
	defer comRelease(stream)

	swBitmapPtr, err := createDecoder(stream)
	if err != nil {
		return nil, fmt.Errorf("create decoder: %w", err)
	}
	defer comRelease(swBitmapPtr)

	swBitmap, err := getSoftwareBitmap(swBitmapPtr)
	if err != nil {
		return nil, fmt.Errorf("get software bitmap: %w", err)
	}
	defer comRelease(swBitmap)

	engine, err := createOcrEngine(language)
	if err != nil {
		return nil, fmt.Errorf("create OCR engine: %w", err)
	}
	defer comRelease(engine)

	ocrResult, err := recognizeOcr(engine, swBitmap)
	if err != nil {
		return nil, fmt.Errorf("recognize: %w", err)
	}
	defer comRelease(ocrResult)

	text, err := getOcrText(ocrResult)
	if err != nil {
		return nil, fmt.Errorf("get text: %w", err)
	}

	lines, lineCount, err := getOcrLines(ocrResult)
	if err != nil {
		return nil, fmt.Errorf("get lines: %w", err)
	}
	defer comRelease(lines)

	result := &OCRResult{Text: text}

	for i := uint32(0); i < lineCount; i++ {
		line, err := getLineAt(lines, i)
		if err != nil {
			continue
		}

		lineText, err := getLineText(line)
		if err != nil {
			comRelease(line)
			continue
		}

		words, wordCount, err := getLineWords(line)
		if err != nil {
			comRelease(line)
			continue
		}

		ocrLine := OCRLine{Text: lineText}

		for j := uint32(0); j < wordCount; j++ {
			word, err := getWordAt(words, j)
			if err != nil {
				continue
			}

			wordText, err := getWordText(word)
			if err != nil {
				comRelease(word)
				continue
			}

			rect, err := getWordRect(word)
			if err != nil {
				comRelease(word)
				continue
			}

			ocrWord := OCRWord{
				Text: wordText,
				X:    float64(rect.X),
				Y:    float64(rect.Y),
				W:    float64(rect.W),
				H:    float64(rect.H),
			}
			result.Words = append(result.Words, ocrWord)

			if j == 0 {
				ocrLine.X = float64(rect.X)
				ocrLine.Y = float64(rect.Y)
				ocrLine.W = float64(rect.W)
				ocrLine.H = float64(rect.H)
			} else {
				if rect.X < float32(ocrLine.X) {
					ocrLine.X = float64(rect.X)
				}
				if rect.Y < float32(ocrLine.Y) {
					ocrLine.Y = float64(rect.Y)
				}
				endX := rect.X + rect.W
				if endX > float32(ocrLine.X+ocrLine.W) {
					ocrLine.W = float64(endX - float32(ocrLine.X))
				}
				endY := rect.Y + rect.H
				if endY > float32(ocrLine.Y+ocrLine.H) {
					ocrLine.H = float64(endY - float32(ocrLine.Y))
				}
			}
			comRelease(word)
		}

		comRelease(words)
		result.Lines = append(result.Lines, ocrLine)
		comRelease(line)
	}

	return result, nil
}
