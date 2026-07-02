package actions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecycleDelete(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(fpath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := recycleDelete(fpath); err != nil {
		t.Fatalf("recycleDelete failed: %v", err)
	}
}

func TestFilePreCheck(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(fpath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := FilePreCheck(&ExpConfig{Text: fpath}, ""); err != nil {
		t.Fatalf("pre-check existing file should pass: %v", err)
	}
	if err := FilePreCheck(&ExpConfig{NotText: filepath.Join(dir, "nope.txt")}, ""); err != nil {
		t.Fatalf("pre-check non-existing file (NotText) should pass: %v", err)
	}
	if err := FilePreCheck(&ExpConfig{Text: filepath.Join(dir, "nope.txt")}, ""); err == nil {
		t.Fatal("pre-check non-existing (Text) should fail")
	}
}

func TestFilePostVerify(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "test.txt")

	r := FilePostVerify(&ExpConfig{NotText: fpath}, fpath)
	if !r.Passed {
		t.Fatalf("post-verify non-existing file (NotText) should pass: %s", r.Reason)
	}

	if err := os.WriteFile(fpath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	r = FilePostVerify(&ExpConfig{Text: fpath}, fpath)
	if !r.Passed {
		t.Fatalf("post-verify existing file (Text) should pass: %s", r.Reason)
	}

	r = FilePostVerify(&ExpConfig{NotText: fpath}, fpath)
	if r.Passed {
		t.Fatal("post-verify existing file (NotText) should fail")
	}
}

func TestFilePostVerifyChange(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(fpath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	change := true
	r := FilePostVerify(&ExpConfig{Change: &change}, fpath)
	if !r.Passed {
		t.Fatalf("post-verify existing file (Change) should pass: %s", r.Reason)
	}
}

func TestWriteReadDocx(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test.docx")
	content := "Hello\nWorld\nDocx"

	r, err := WriteFile(p, content, false)
	if err != nil {
		t.Fatalf("WriteFile docx: %v", err)
	}
	if !r.WasNew {
		t.Fatal("expected new file")
	}

	result, err := ReadFile(p, 1, 8000)
	if err != nil {
		t.Fatalf("ReadFile docx: %v", err)
	}
	got := strings.TrimSpace(result.Content)
	if got != content {
		t.Fatalf("content mismatch:\n want: %q\n got:  %q", content, got)
	}
}

func TestWriteReadXlsx(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test.xlsx")
	content := "Name\tAge\tCity\nAlice\t30\tNYC\nBob\t25\tLA"

	r, err := WriteFile(p, content, false)
	if err != nil {
		t.Fatalf("WriteFile xlsx: %v", err)
	}
	if !r.WasNew {
		t.Fatal("expected new file")
	}

	result, err := ReadFile(p, 1, 8000)
	if err != nil {
		t.Fatalf("ReadFile xlsx: %v", err)
	}
	if !strings.Contains(result.Content, "Alice") {
		t.Fatalf("xlsx content missing 'Alice':\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "30") {
		t.Fatalf("xlsx content missing '30':\n%s", result.Content)
	}
}

func TestWriteOverwriteDocx(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "overwrite.docx")

	_, err := WriteFile(p, "first draft", false)
	if err != nil {
		t.Fatalf("first write: %v", err)
	}

	_, err = WriteFile(p, "second draft", true)
	if err != nil {
		t.Fatalf("overwrite write: %v", err)
	}

	result, err := ReadFile(p, 1, 8000)
	if err != nil {
		t.Fatalf("read after overwrite: %v", err)
	}
	got := strings.TrimSpace(result.Content)
	if !strings.Contains(got, "second") {
		t.Fatalf("expected 'second draft', got: %q", got)
	}
}

func TestReadImageFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test.png")

	_, err := os.Stat(p)
	if os.IsNotExist(err) {
		t.Skip("no image file to read")
	}
	_, err = ReadFile(p, 1, 8000)
	if err != nil {
		t.Fatalf("ReadFile png: %v", err)
	}
}

func TestWriteReadPdf(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test.pdf")
	content := "Hello World\nPDF Test"

	r, err := WriteFile(p, content, false)
	if err != nil {
		t.Fatalf("WriteFile pdf: %v", err)
	}
	if !r.WasNew {
		t.Fatal("expected new file")
	}

	result, err := ReadFile(p, 1, 8000)
	if err != nil {
		t.Fatalf("ReadFile pdf: %v", err)
	}
	if !strings.Contains(result.Content, "Hello World") {
		t.Fatalf("pdf content missing expected text:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "PDF Test") {
		t.Fatalf("pdf content missing 'PDF Test':\n%s", result.Content)
	}
}

func TestWriteReadTxt(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test.txt")
	content := "Hello\nWorld\nText"

	r, err := WriteFile(p, content, false)
	if err != nil {
		t.Fatalf("WriteFile txt: %v", err)
	}
	if !r.WasNew {
		t.Fatal("expected new file")
	}

	result, err := ReadFile(p, 1, 8000)
	if err != nil {
		t.Fatalf("ReadFile txt: %v", err)
	}
	if strings.TrimSpace(result.Content) != content {
		t.Fatalf("content mismatch:\n want: %q\n got:  %q", content, result.Content)
	}
}

func TestReadFilePagination(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "long.txt")
	content := ""
	for i := 0; i < 100; i++ {
		content += fmt.Sprintf("line %04d: %s\n", i, strings.Repeat("abcdefghij", 100))
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	r1, err := ReadFile(p, 1, 500)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if r1.Page != 1 {
		t.Fatalf("expected page 1, got %d", r1.Page)
	}
	if r1.TotalPages < 2 {
		t.Fatalf("expected >1 page, got %d", r1.TotalPages)
	}
	if !r1.Truncated {
		t.Fatal("expected truncated=true for page 1")
	}
	if len(r1.Content) > 520 {
		t.Fatalf("page too large: %d", len(r1.Content))
	}

	result, err := ReadFile(p, r1.TotalPages, 500)
	if err != nil {
		t.Fatalf("last page: %v", err)
	}
	if result.Truncated {
		t.Fatal("last page should not be truncated")
	}
}


