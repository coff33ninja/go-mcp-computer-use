//go:build ignore

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type CallSite struct {
	Index      int
	File       string
	Line       int
	Annotation string
}

func main() {
	repoRoot := findRepoRoot()
	if repoRoot == "" {
		fmt.Fprintln(os.Stderr, "ERROR: cannot find repo root (no VERSION file)")
		os.Exit(1)
	}

	actionDir := filepath.Join(repoRoot, "internal", "actions")
	sourceFiles := []string{
		filepath.Join(actionDir, "uia_com.go"),
		filepath.Join(actionDir, "uia.go"),
		filepath.Join(actionDir, "ocr_com.go"),
		filepath.Join(actionDir, "ocr.go"),
		filepath.Join(actionDir, "winrt.go"),
		filepath.Join(actionDir, "keyboard.go"),
		filepath.Join(actionDir, "chained.go"),
	}

	comPatternsDoc := filepath.Join(repoRoot, "docs", "reference", "com-patterns.md")
	vtableVerifyDoc := filepath.Join(repoRoot, "docs", "reference", "vtable-verification.md")
	vtableTest := filepath.Join(repoRoot, "internal", "actions", "vtable_test.go")

	exitCode := 0

	// Parse vtblMethod call sites from source code
	callSites := parseCallSites(sourceFiles)

	if len(callSites) == 0 {
		fmt.Fprintln(os.Stderr, "ERROR: no vtblMethod call sites found")
		os.Exit(1)
	}

	// Cross-check: each call site index must be documented in docs
	fmt.Printf("Found %d vtblMethod call sites across %d files:\n", len(callSites), len(sourceFiles))
	for _, cs := range callSites {
		ann := cs.Annotation
		if ann == "" {
			ann = "(no annotation)"
		}
		fmt.Printf("  %s:%d  vtbl[%d]  %s\n", cs.File, cs.Line, cs.Index, ann)
	}
	fmt.Println()

	// Build unique index set
	usedIndices := make(map[int]bool)
	for _, cs := range callSites {
		usedIndices[cs.Index] = true
	}

	// Check com-patterns.md
	fmt.Println("── Checking docs/reference/com-patterns.md ──")
	comBody := readFile(comPatternsDoc)
	missingInCom := 0
	for _, cs := range callSites {
		needles := []string{
			fmt.Sprintf("| %d |", cs.Index),
			fmt.Sprintf("`%d`", cs.Index),
		}
		if !containsAny(comBody, needles) {
			fmt.Printf("  MISSING: vtbl[%d] at %s:%d not documented in com-patterns.md\n", cs.Index, cs.File, cs.Line)
			missingInCom++
		}
	}
	if missingInCom == 0 {
		fmt.Println("  All used indices documented ✓")
	} else {
		fmt.Printf("  %d used indices missing from com-patterns.md\n", missingInCom)
		exitCode = 1
	}
	fmt.Println()

	// Check vtable-verification.md
	fmt.Println("── Checking docs/reference/vtable-verification.md ──")
	verifyBody := readFile(vtableVerifyDoc)
	// Extract all backtick-quoted numbers from the doc
	missingInVerify := 0
	for idx := range usedIndices {
		idxStr := strconv.Itoa(idx)
		found := strings.Contains(verifyBody, "vtbl "+idxStr)
		// Also check inside backtick-quoted groups like `5` or `6, 8, 14, 0, 7`
		if !found {
			parts := strings.Split(verifyBody, "`")
			for i := 1; i < len(parts); i += 2 {
				for _, field := range strings.FieldsFunc(parts[i], func(r rune) bool { return r == ',' || r == ' ' }) {
					if strings.TrimSpace(field) == idxStr {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
		if !found && !exemptIndices[idx] {
			fmt.Printf("  MISSING: vtbl[%d] not documented in vtable-verification.md\n", idx)
			missingInVerify++
		}
	}
	if missingInVerify == 0 {
		fmt.Println("  All used indices documented ✓")
	} else {
		fmt.Printf("  %d used indices missing from vtable-verification.md\n", missingInVerify)
		exitCode = 1
	}
	fmt.Println()

	// Check vtable_test.go
	fmt.Println("── Checking internal/actions/vtable_test.go ──")
	testBody := readFile(vtableTest)
	testedIndices := parseTestAnnotations(testBody)
	missingInTests := 0
	for idx := range usedIndices {
		if exemptIndices[idx] {
			continue
		}
		if !testedIndices[idx] {
			fmt.Printf("  MISSING TEST: vtbl[%d] has no test or annotation in vtable_test.go\n", idx)
			missingInTests++
		}
	}
	if missingInTests == 0 {
		fmt.Println("  All used indices covered by tests ✓")
	} else {
		fmt.Printf("  %d indices missing test coverage\n", missingInTests)
		exitCode = 1
	}
	fmt.Println()

	if exitCode != 0 {
		fmt.Println("FAILED: vtable docs/tests are out of sync with source")
		os.Exit(exitCode)
	}
	fmt.Println("PASSED: all used vtblMethod indices are documented and tested")
}

func findRepoRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "VERSION")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN: cannot read %s: %v\n", path, err)
		return ""
	}
	return string(data)
}

// IUnknown base indices that don't need individual tests
var exemptIndices = map[int]bool{
	0: true, // QueryInterface
	1: true, // AddRef
	2: true, // Release
}

// vtblMethod(p, N) — captures the second argument (the vtable index)
var vtblCallRe = regexp.MustCompile(`vtblMethod\([^,]+,\s*(\d+)\)`)

// Look for nearby annotation comment like: // N = MethodName
// Also look for preceding single-line comments mentioning the index
var annotRe = regexp.MustCompile(`//\s*(\d+)\s*=\s*(\w[\w.]*)`)

// Parse // vtbl: N, N, N test annotations
var vtblTestAnnotRe = regexp.MustCompile(`//\s*vtbl:\s*(\d+(?:\s*,\s*\d+)*)`)
var vtblTestAnnotRe2 = regexp.MustCompile(`//\s*Exercises vtbl:\s*(\d+(?:\s*,\s*\d+)*)`)


func parseCallSites(files []string) []CallSite {
	var sites []CallSite

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}
		defer f.Close()

		rel := filepath.ToSlash(file)
		if idx := strings.Index(rel, "internal/"); idx >= 0 {
			rel = rel[idx:]
		}

		scanner := bufio.NewScanner(f)
		lines := make([]string, 0, 500)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		for lineIdx, line := range lines {
			matches := vtblCallRe.FindAllStringSubmatch(line, -1)
			if matches == nil {
				continue
			}
			for _, m := range matches {
				idx, _ := strconv.Atoi(m[1])

				// Look for annotation: scan the 5 preceding lines AND the current line for // N = MethodName
				annotation := ""
				for look := max(0, lineIdx-5); look <= lineIdx; look++ {
					line := lines[look]
					// Check for // comment anywhere on the line (inline or preceding)
					if ci := strings.Index(line, "//"); ci >= 0 {
						comment := line[ci:]
						if !strings.Contains(comment, "verified") && !strings.Contains(comment, "Verified") {
							if am := annotRe.FindStringSubmatch(comment); am != nil {
								annIdx, _ := strconv.Atoi(am[1])
								if annIdx == idx {
									annotation = fmt.Sprintf("// %s = %s", am[1], am[2])
									break
								}
							}
						}
					}
				}

				sites = append(sites, CallSite{
					Index:      idx,
					File:       rel,
					Line:       lineIdx + 1,
					Annotation: annotation,
				})
			}
		}
	}
	return sites
}

func parseTestAnnotations(body string) map[int]bool {
	indices := make(map[int]bool)
	// Parse "// vtbl: N, N, N" format
	for _, m := range vtblTestAnnotRe.FindAllStringSubmatch(body, -1) {
		parts := strings.Split(m[1], ",")
		for _, p := range parts {
			if idx, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
				indices[idx] = true
			}
		}
	}
	// Parse "// Exercises vtbl: N, N, N" format
	for _, m := range vtblTestAnnotRe2.FindAllStringSubmatch(body, -1) {
		parts := strings.Split(m[1], ",")
		for _, p := range parts {
			if idx, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
				indices[idx] = true
			}
		}
	}
	// Also collect from (vtbl N) inline annotations
	inlineRe := regexp.MustCompile(`\(vtbl (\d+)\)`)
	for _, m := range inlineRe.FindAllStringSubmatch(body, -1) {
		if idx, err := strconv.Atoi(m[1]); err == nil {
			indices[idx] = true
		}
	}
	return indices
}

func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
