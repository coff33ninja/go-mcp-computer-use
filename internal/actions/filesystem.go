package actions

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-pdf/fpdf"
	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/xuri/excelize/v2"
)

const DefaultPageSize = 8000

var (
	shell32FO              = syscall.NewLazyDLL("shell32.dll")
	shFileOperationW       = shell32FO.NewProc("SHFileOperationW")
	workingDir     string
	workingDirMu   sync.RWMutex
)

func init() {
	wd, _ := os.Getwd()
	workingDir = wd
}

func SetWorkingDirectory(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("set_working_directory: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("set_working_directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("set_working_directory: %q is not a directory", abs)
	}
	workingDirMu.Lock()
	workingDir = abs
	workingDirMu.Unlock()
	return nil
}

func GetWorkingDirectory() string {
	workingDirMu.RLock()
	defer workingDirMu.RUnlock()
	return workingDir
}

func resolvePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	if !filepath.IsAbs(path) {
		workingDirMu.RLock()
		wd := workingDir
		workingDirMu.RUnlock()
		path = filepath.Join(wd, path)
	}
	clean := filepath.Clean(path)
	return clean, nil
}

const (
	FO_DELETE          = 3
	FOF_ALLOWUNDO      = 0x0040
	FOF_NOCONFIRMATION = 0x0010
	FOF_SILENT         = 0x0004
	FOF_NOERRORUI      = 0x0400
)

type SHFILEOPSTRUCTW struct {
	hwnd                  uintptr
	wFunc                 uint32
	pFrom                 uintptr
	pTo                   uintptr
	fFlags                uint16
	fAnyOperationsAborted uint32
	hNameMappings         uintptr
	lpszProgressTitle     uintptr
}

func recycleDelete(path string) error {
	utf16 := syscall.StringToUTF16(path)
	utf16 = append(utf16, 0)
	op := &SHFILEOPSTRUCTW{
		wFunc: FO_DELETE,
		pFrom: uintptr(unsafe.Pointer(&utf16[0])),
		fFlags: FOF_ALLOWUNDO | FOF_NOCONFIRMATION | FOF_SILENT | FOF_NOERRORUI,
	}
	ret, _, _ := shFileOperationW.Call(uintptr(unsafe.Pointer(op)))
	if ret != 0 {
		return fmt.Errorf("recycle bin operation failed: error code %d", ret)
	}
	return nil
}

type FileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"is_dir"`
	ModTime string `json:"mod_time"`
	Mode    string `json:"mode"`
}

func ListDirectory(path string) ([]FileInfo, error) {
	p, err := resolvePath(path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(p)
	if err != nil {
		return nil, fmt.Errorf("list_directory: %w", err)
	}
	var result []FileInfo
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, FileInfo{
			Name:    e.Name(),
			Size:    info.Size(),
			IsDir:   e.IsDir(),
			ModTime: info.ModTime().Format("2006-01-02T15:04:05"),
			Mode:    info.Mode().String(),
		})
	}
	return result, nil
}

type ReadFileResult struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	Size       int64  `json:"size"`
	MimeType   string `json:"mime_type"`
	Page       int    `json:"page"`
	TotalPages int    `json:"total_pages"`
	Truncated  bool   `json:"truncated"`
}

var textExts = map[string]bool{
	".txt": true, ".json": true, ".csv": true, ".tsv": true,
	".xml": true, ".yaml": true, ".yml": true, ".toml": true,
	".md": true, ".log": true, ".env": true,
	".ini": true, ".cfg": true, ".conf": true,
	".go": true, ".py": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
	".rs": true, ".c": true, ".h": true, ".cpp": true, ".hpp": true,
	".java": true, ".kt": true, ".swift": true, ".rb": true, ".php": true,
	".sh": true, ".bat": true, ".ps1": true, ".psm1": true,
	".css": true, ".scss": true, ".less": true, ".html": true, ".htm": true,
	".sql": true, ".r": true, ".m": true, ".mm": true,
	".makefile": true, ".dockerfile": true, ".editorconfig": true,
	".gitignore": true, ".gitattributes": true,
	".mod": true, ".sum": true, ".lock": true,
	".pl": true, ".pm": true, ".t": true,
	".lua": true, ".hs": true, ".clj": true, ".cljs": true,
	".zig": true, ".v": true, ".vsh": true,
	".gradle": true, ".properties": true,
	".tex": true, ".bib": true,
	".cnf": true,
}

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".bmp": true, ".tiff": true, ".tif": true,
	".webp": true, ".ico": true, ".svg": true,
}

func detectMimeType(path string, header []byte) string {
	ct := http.DetectContentType(header)
	if ct != "application/octet-stream" {
		return ct
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".doc":
		return "application/msword"
	case ".pdf":
		return "application/pdf"
	case ".csv":
		return "text/csv"
	case ".tsv":
		return "text/tab-separated-values"
	case ".yaml", ".yml":
		return "text/yaml"
	case ".toml":
		return "text/toml"
	case ".md":
		return "text/markdown"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".svg":
		return "image/svg+xml"
	default:
		if textExts[ext] {
			return "text/plain"
		}
		return "application/octet-stream"
	}
}

func chunkContent(content string, page, pageSize int) (string, int, int, bool) {
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	total := len(content)
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	return content[start:end], page, totalPages, end < total
}

func ReadFile(path string, page, pageSize int) (*ReadFileResult, error) {
	p, err := resolvePath(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("read_file: %w", err)
	}
	header := make([]byte, 512)
	n, _ := io.ReadFull(f, header)
	if n < 512 {
		header = header[:n]
	}
	f.Close()



	mimeType := detectMimeType(p, header)
	ext := strings.ToLower(filepath.Ext(p))

	var raw string
	switch {
	case ext == ".docx":
		raw, err = readDocx(p)
	case ext == ".xlsx":
		raw, err = readXlsx(p)
	case ext == ".pdf":
		raw, err = readPdf(p)
	case imageExts[ext] || strings.HasPrefix(mimeType, "image/"):
		raw, err = readImage(p)
	default:
		raw, err = readText(p)
	}
	if err != nil {
		return nil, err
	}

	content, curPage, totalPages, truncated := chunkContent(raw, page, pageSize)

	finfo, _ := os.Stat(p)
	var size int64
	if finfo != nil {
		size = finfo.Size()
	}

	return &ReadFileResult{
		Path:       p,
		Content:    content,
		Size:       size,
		MimeType:   mimeType,
		Page:       curPage,
		TotalPages: totalPages,
		Truncated:  truncated,
	}, nil
}

func readText(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func readDocx(p string) (string, error) {
	r, err := docx.ReadDocxFile(p)
	if err != nil {
		return "", fmt.Errorf("read_file (docx): %w", err)
	}
	defer r.Close()
	return extractDocxText(r.Editable().GetContent()), nil
}

func extractDocxText(xmlContent string) string {
	var buf strings.Builder
	d := xml.NewDecoder(strings.NewReader(xmlContent))
	inT := false
	for {
		tok, err := d.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "t" {
				inT = true
			}
		case xml.CharData:
			if inT {
				buf.Write(t)
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inT = false
			}
			if t.Name.Local == "p" {
				buf.WriteString("\n")
			}
		}
	}
	return strings.TrimSpace(buf.String())
}

func readXlsx(p string) (string, error) {
	f, err := excelize.OpenFile(p)
	if err != nil {
		return "", fmt.Errorf("read_file (xlsx): %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	for _, sheet := range f.GetSheetList() {
		buf.WriteString("=== " + sheet + " ===\n")
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}
		for _, row := range rows {
			buf.WriteString(strings.Join(row, "\t") + "\n")
		}
		buf.WriteString("\n")
	}
	return buf.String(), nil
}

func readPdf(p string) (string, error) {
	rc, r, err := pdf.Open(p)
	if err != nil {
		return "", fmt.Errorf("read_file (pdf): %w", err)
	}
	defer rc.Close()

	var buf bytes.Buffer
	totalPage := r.NumPage()
	for i := 1; i <= totalPage; i++ {
		page := r.Page(i)
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		if i > 1 {
			buf.WriteString("\n--- page " + fmt.Sprintf("%d", i) + " ---\n")
		}
		buf.WriteString(text)
	}
	return buf.String(), nil
}

func readImage(p string) (string, error) {
	text, err := ocrNative(p, "")
	if err != nil {
		return "", fmt.Errorf("read_file (image): %w", err)
	}
	if text == nil {
		return "", nil
	}
	return text.Text, nil
}

type WriteFileResult struct {
	Path     string `json:"path"`
	Size     int    `json:"size"`
	WasNew   bool   `json:"was_new"`
}

func WriteFile(path, content string, overwrite bool) (*WriteFileResult, error) {
	p, err := resolvePath(path)
	if err != nil {
		return nil, err
	}
	wasNew := false
	if _, err := os.Stat(p); os.IsNotExist(err) {
		wasNew = true
	} else if err != nil {
		return nil, fmt.Errorf("write_file: cannot stat %q: %w", p, err)
	} else if !overwrite {
		return nil, fmt.Errorf("write_file: %q already exists (set overwrite=true to replace)", p)
	}

	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return nil, fmt.Errorf("write_file: cannot create parent directories: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(p))
	var written int
	switch ext {
	case ".docx":
		err = writeDocx(p, content)
	case ".xlsx":
		err = writeXlsx(p, content)
	case ".pdf":
		err = writePdf(p, content, wasNew)
	default:
		err = os.WriteFile(p, []byte(content), 0644)
		if err == nil {
			written = len(content)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("write_file: %w", err)
	}
	if written == 0 {
		finfo, _ := os.Stat(p)
		if finfo != nil {
			written = int(finfo.Size())
		}
	}

	return &WriteFileResult{
		Path:   p,
		Size:   written,
		WasNew: wasNew,
	}, nil
}

func writeDocx(p, content string) error {
	if _, err := os.Stat(p); err != nil {
		return createDocx(p, content)
	}
	r, err := docx.ReadDocxFile(p)
	if err != nil {
		return createDocx(p, content)
	}

	ed := r.Editable()
	raw := ed.GetContent()

	var buf strings.Builder
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	buf.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	buf.WriteString("<w:body>")
	for _, para := range strings.Split(content, "\n") {
		buf.WriteString("<w:p><w:r><w:t xml:space=\"preserve\">")
		xmlEscape(&buf, para)
		buf.WriteString("</w:t></w:r></w:p>")
	}
	buf.WriteString("</w:body></w:document>")

	newXML := buf.String()
	bodyStart := strings.Index(newXML, "<w:body>")
	bodyEnd := strings.Index(newXML, "</w:body>")
	if bodyStart == -1 || bodyEnd == -1 {
		r.Close()
		return createDocx(p, content)
	}
	newBody := newXML[bodyStart : bodyEnd+len("</w:body>")]

	oldBodyStart := strings.Index(raw, "<w:body>")
	oldBodyEnd := strings.Index(raw, "</w:body>")
	if oldBodyStart == -1 || oldBodyEnd == -1 {
		r.Close()
		return createDocx(p, content)
	}

	ed.SetContent(raw[:oldBodyStart] + newBody + raw[oldBodyEnd+len("</w:body>"):])

	tmpPath := p + ".tmp"
	if err := ed.WriteToFile(tmpPath); err != nil {
		r.Close()
		return err
	}
	r.Close()
	return os.Rename(tmpPath, p)
}

func xmlEscape(w *strings.Builder, s string) {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&':
			w.WriteString("&amp;")
		case '<':
			w.WriteString("&lt;")
		case '>':
			w.WriteString("&gt;")
		case '"':
			w.WriteString("&quot;")
		case '\'':
			w.WriteString("&apos;")
		default:
			w.WriteByte(s[i])
		}
	}
}

func writeXlsx(p, content string) error {
	var f *excelize.File
	if _, err := os.Stat(p); err != nil {
		f = excelize.NewFile()
	} else {
		f, err = excelize.OpenFile(p)
		if err != nil {
			f = excelize.NewFile()
		}
	}
	defer f.Close()

	sheet := f.GetSheetName(0)

	for i, row := range strings.Split(content, "\n") {
		cells := strings.Split(row, "\t")
		for j, cell := range cells {
			cellRef, cErr := excelize.CoordinatesToCellName(j+1, i+1)
			if cErr != nil {
				continue
			}
			f.SetCellValue(sheet, cellRef, cell)
		}
	}
	return f.SaveAs(p)
}

func writePdf(p, content string, wasNew bool) error {
	if !wasNew {
		if err := fillPdfForm(p, content); err == nil {
			return nil
		}
	}
	return createPdf(p, content)
}

func createPdf(p, content string) error {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 12)

	for _, line := range strings.Split(content, "\n") {
		tr := pdf.UnicodeTranslatorFromDescriptor("")
		pdf.Write(6, tr(line))
		pdf.Ln(6)
	}
	return pdf.OutputFileAndClose(p)
}

func fillPdfForm(p, content string) error {
	tmpDir, err := os.MkdirTemp("", "pdfform")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	jsonPath := filepath.Join(tmpDir, "form.json")
	if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("pdf form: cannot write temp json: %w", err)
	}

	conf := model.NewDefaultConfiguration()
	return api.FillFormFile(p, jsonPath, p, conf)
}

type docxContentTypes struct {
	XMLNs    string              `xml:"xmlns,attr"`
	Defaults []docxDefault       `xml:"Default"`
	Overrides []docxOverride     `xml:"Override"`
}
type docxDefault struct {
	Extension   string `xml:"Extension,attr"`
	ContentType string `xml:"ContentType,attr"`
}
type docxOverride struct {
	PartName    string `xml:"PartName,attr"`
	ContentType string `xml:"ContentType,attr"`
}

type docxRelationships struct {
	XMLNs         string               `xml:"xmlns,attr"`
	Relationships []docxRelationship   `xml:"Relationship"`
}
type docxRelationship struct {
	ID     string `xml:"Id,attr"`
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
}

type docxDocument struct {
	XMLNs string    `xml:"xmlns:w,attr"`
	Body  docxBody  `xml:"w:body"`
}
type docxBody struct {
	Paragraphs []docxParagraph `xml:"w:p"`
}
type docxParagraph struct {
	Runs []docxRun `xml:"w:r"`
}
type docxRun struct {
	Text string `xml:"w:t"`
}

func createDocx(p, content string) error {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	types := docxContentTypes{XMLNs: "http://schemas.openxmlformats.org/package/2006/content-types"}
	types.Defaults = []docxDefault{
		{Extension: "rels", ContentType: "application/vnd.openxmlformats-package.relationships+xml"},
		{Extension: "xml", ContentType: "application/xml"},
	}
	types.Overrides = []docxOverride{
		{PartName: "/word/document.xml", ContentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"},
	}
	writeXMLInZip(zw, "[Content_Types].xml", &types)

	rels := docxRelationships{XMLNs: "http://schemas.openxmlformats.org/package/2006/relationships"}
	rels.Relationships = []docxRelationship{
		{ID: "rId1", Type: "http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument", Target: "word/document.xml"},
	}
	writeXMLInZip(zw, "_rels/.rels", &rels)

	wordRels := docxRelationships{XMLNs: "http://schemas.openxmlformats.org/package/2006/relationships"}
	writeXMLInZip(zw, "word/_rels/document.xml.rels", &wordRels)

	doc := docxDocument{XMLNs: "http://schemas.openxmlformats.org/wordprocessingml/2006/main"}
	for _, para := range strings.Split(content, "\n") {
		doc.Body.Paragraphs = append(doc.Body.Paragraphs, docxParagraph{
			Runs: []docxRun{{Text: para}},
		})
	}
	writeXMLInZip(zw, "word/document.xml", &doc)

	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(p, buf.Bytes(), 0644)
}

func writeXMLInZip(zw *zip.Writer, name string, v any) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	return enc.Encode(v)
}

type FileMatch struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func FindFiles(root, pattern string) ([]FileMatch, error) {
	p, err := resolvePath(root)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(p)
	if err != nil {
		return nil, fmt.Errorf("find_files: %w", err)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("find_files: %q is not a directory", p)
	}

	var matches []FileMatch
	err = filepath.WalkDir(p, func(walkPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		match, err := filepath.Match(pattern, d.Name())
		if err != nil {
			return err
		}
		if match {
			info, err := d.Info()
			if err != nil {
				return nil
			}
			matches = append(matches, FileMatch{
				Path: walkPath,
				Size: info.Size(),
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("find_files: %w", err)
	}
	return matches, nil
}

func CopyFile(source, destination string) error {
	src, err := resolvePath(source)
	if err != nil {
		return err
	}
	dst, err := resolvePath(destination)
	if err != nil {
		return err
	}

	srcStat, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("copy_file: %w", err)
	}

	if srcStat.IsDir() {
		return copyDir(src, dst, srcStat)
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("copy_file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("copy_file: %w", err)
	}
	if err := os.WriteFile(dst, data, srcStat.Mode()); err != nil {
		return fmt.Errorf("copy_file: %w", err)
	}
	return nil
}

func copyDir(src, dst string, srcStat os.FileInfo) error {
	if err := os.MkdirAll(dst, srcStat.Mode()); err != nil {
		return fmt.Errorf("copy_file: %w", err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("copy_file: %w", err)
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			info, err := e.Info()
			if err != nil {
				return fmt.Errorf("copy_file: %w", err)
			}
			if err := copyDir(srcPath, dstPath, info); err != nil {
				return err
			}
		} else {
			if err := CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func MoveFile(source, destination string) error {
	src, err := resolvePath(source)
	if err != nil {
		return err
	}
	dst, err := resolvePath(destination)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("move_file: %w", err)
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("move_file: %w", err)
	}
	return nil
}

type DeleteFileResult struct {
	Path       string `json:"path"`
	ToRecycle  bool   `json:"to_recycle_bin"`
}

func DeleteFile(path string) (*DeleteFileResult, error) {
	p, err := resolvePath(path)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil, fmt.Errorf("delete_file: %q does not exist", p)
	}
	if err := recycleDelete(p); err != nil {
		return nil, fmt.Errorf("delete_file: %w", err)
	}
	return &DeleteFileResult{Path: p, ToRecycle: true}, nil
}

func CreateDirectory(path string) error {
	p, err := resolvePath(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(p, 0755); err != nil {
		return fmt.Errorf("create_directory: %w", err)
	}
	return nil
}

func GetFileInfo(path string) (*FileInfo, error) {
	p, err := resolvePath(path)
	if err != nil {
		return nil, err
	}
	stat, err := os.Stat(p)
	if err != nil {
		return nil, fmt.Errorf("get_file_info: %w", err)
	}
	return &FileInfo{
		Name:    stat.Name(),
		Size:    stat.Size(),
		IsDir:   stat.IsDir(),
		ModTime: stat.ModTime().Format("2006-01-02T15:04:05"),
		Mode:    stat.Mode().String(),
	}, nil
}

type FileVerifyResult struct {
	Passed bool   `json:"passed"`
	Method string `json:"method"`
	Reason string `json:"reason"`
	Path   string `json:"path"`
}

func FilePreCheck(ec *ExpConfig, path string) error {
	if ec == nil {
		return nil
	}
	if ec.Text != "" {
		if _, err := os.Stat(ec.Text); os.IsNotExist(err) {
			return fmt.Errorf("precondition: path %q does not exist", ec.Text)
		} else if err != nil {
			return fmt.Errorf("precondition: cannot stat %q: %w", ec.Text, err)
		}
	}
	if ec.NotText != "" {
		if _, err := os.Stat(ec.NotText); err == nil {
			return fmt.Errorf("precondition: path %q still exists", ec.NotText)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("precondition: cannot stat %q: %w", ec.NotText, err)
		}
	}
	if ec.Change != nil && *ec.Change {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("precondition: path %q does not exist (cannot detect change)", path)
		}
	}
	return nil
}

func FilePostVerify(ec *ExpConfig, path string) *FileVerifyResult {
	r := &FileVerifyResult{Path: path}
	if ec == nil {
		r.Passed = true
		r.Method = "no_check"
		return r
	}
	if ec.WaitMs > 0 {
		time.Sleep(time.Duration(ec.WaitMs) * time.Millisecond)
	} else {
		time.Sleep(200 * time.Millisecond)
	}

	if ec.Text != "" {
		r.Method = "path_exists"
		if _, err := os.Stat(ec.Text); err == nil {
			r.Passed = true
			r.Reason = fmt.Sprintf("path %q exists", ec.Text)
		} else if os.IsNotExist(err) {
			r.Reason = fmt.Sprintf("path %q does not exist", ec.Text)
		} else {
			r.Reason = fmt.Sprintf("cannot stat %q: %v", ec.Text, err)
		}
		return r
	}
	if ec.NotText != "" {
		r.Method = "path_gone"
		if _, err := os.Stat(ec.NotText); os.IsNotExist(err) {
			r.Passed = true
			r.Reason = fmt.Sprintf("path %q is gone", ec.NotText)
		} else if err == nil {
			r.Reason = fmt.Sprintf("path %q still exists", ec.NotText)
		} else {
			r.Reason = fmt.Sprintf("cannot stat %q: %v", ec.NotText, err)
		}
		return r
	}
	if ec.Change != nil && *ec.Change {
		r.Method = "path_changed"
		_, err := os.Stat(path)
		if err == nil {
			r.Passed = true
			r.Reason = "path exists after action"
		} else if os.IsNotExist(err) {
			r.Passed = true
			r.Reason = "path removed after action"
		} else {
			r.Reason = fmt.Sprintf("cannot stat %q: %v", path, err)
		}
		return r
	}
	r.Passed = true
	r.Method = "no_check"
	return r
}
