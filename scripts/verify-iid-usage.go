//go:build ignore

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type iidInfo struct {
	Name    string
	Status  string // "used" or "unused"
	Purpose string
	Line    int
}

func main() {
	updateMode := false
	for _, arg := range os.Args[1:] {
		if arg == "-update" {
			updateMode = true
		}
	}

	repoRoot := findRepoRoot()
	if repoRoot == "" {
		fmt.Fprintln(os.Stderr, "ERROR: cannot find repo root (no VERSION file)")
		os.Exit(1)
	}

	actionsDir := filepath.Join(repoRoot, "internal", "actions")
	winrtFile := filepath.Join(actionsDir, "winrt.go")
	comPatternsDoc := filepath.Join(repoRoot, "docs", "reference", "com-patterns.md")

	// Step 1: Parse all IID definitions from winrt.go
	iids := parseIIDDefs(winrtFile)
	if len(iids) == 0 {
		fmt.Fprintln(os.Stderr, "ERROR: no IID definitions found in winrt.go")
		os.Exit(1)
	}
	fmt.Printf("Found %d IID definitions in winrt.go\n", len(iids))

	// Step 2: Scan all .go files in internal/actions/ for IID usage (excluding winrt.go and test files)
	usedIIDs := scanIIDUsage(actionsDir, iids)
	internalOnlyIIDs := scanIIDInternalUsage(winrtFile, iids)

	// Step 3: Determine effective status
	iidStatus := make(map[string]string)
	for name := range iids {
		if usedIIDs[name] {
			iidStatus[name] = "used"
		} else if internalOnlyIIDs[name] {
			iidStatus[name] = "internal"
		} else {
			iidStatus[name] = "unused"
		}
	}

	// Print summary
	var usedNames, internalNames, unusedNames []string
	for name, status := range iidStatus {
		switch status {
		case "used":
			usedNames = append(usedNames, name)
		case "internal":
			internalNames = append(internalNames, name)
		default:
			unusedNames = append(unusedNames, name)
		}
	}
	sort.Strings(usedNames)
	sort.Strings(internalNames)
	sort.Strings(unusedNames)

	fmt.Printf("\n── IID Usage Summary ──\n")
	fmt.Printf("  USED (%d):      %s\n", len(usedNames), strings.Join(usedNames, ", "))
	fmt.Printf("  INTERNAL (%d):  %s\n", len(internalNames), strings.Join(internalNames, ", "))
	fmt.Printf("  UNUSED (%d):    %s\n", len(unusedNames), strings.Join(unusedNames, ", "))

	// Step 4: Parse doc table and update/validate statuses
	docBody := readFile(comPatternsDoc)
	docEntries := parseDocTable(docBody)

	fmt.Printf("\n── Cross-referencing against docs/reference/com-patterns.md ──\n")

	missingFromDoc := 0
	statusMismatch := 0
	updateLines := make(map[int]string) // line number -> replacement row

	for name, info := range iids {
		expectedStatus := iidStatus[name]
		entry, found := docEntries[info.Name]
		if !found {
			fmt.Printf("  MISSING FROM DOC: %s (status: %s)\n", info.Name, expectedStatus)
			missingFromDoc++
			continue
		}
		if entry.Status != expectedStatus {
			fmt.Printf("  STATUS MISMATCH: %s doc says '%s' but code says '%s'\n", info.Name, entry.Status, expectedStatus)
			statusMismatch++
			updateLines[entry.Line] = buildRow(info.Name, entry.Value, entry.Purpose, expectedStatus)
		}
	}

	if updateMode {
		// Apply updates
		if len(updateLines) > 0 {
			lines := strings.Split(docBody, "\n")
			for lineIdx, replacement := range updateLines {
				if lineIdx-1 < len(lines) {
					lines[lineIdx-1] = replacement
				}
			}
			newBody := strings.Join(lines, "\n")
			if err := os.WriteFile(comPatternsDoc, []byte(newBody), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: writing %s: %v\n", comPatternsDoc, err)
				os.Exit(1)
			}
			fmt.Printf("\nUpdated %d status rows in docs/reference/com-patterns.md\n", len(updateLines))
		} else {
			fmt.Println("\nAll statuses already correct ✓")
		}
	} else if statusMismatch > 0 || missingFromDoc > 0 {
		fmt.Printf("\nFAILED: %d missing from doc, %d status mismatches. Run with -update to fix.\n", missingFromDoc, statusMismatch)
		os.Exit(1)
	} else {
		fmt.Println("\nAll IID statuses are correct in docs ✓")
	}
}

func parseIIDDefs(winrtPath string) map[string]*iidInfo {
	// Regex: matches IID_* = &windows.GUID{ lines in var blocks
	// Catches both: `IID_Foo = &windows.GUID{` and inline comments
	iidDefRe := regexp.MustCompile(`^\s+(IID_\w+)\s*=\s*&windows\.GUID{`)

	iids := make(map[string]*iidInfo)
	f, err := os.Open(winrtPath)
	if err != nil {
		return iids
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if m := iidDefRe.FindStringSubmatch(line); m != nil {
			name := m[1]
			// Extract purpose from inline comment: // ... Purpose text
			purpose := ""
			if ci := strings.Index(line, "//"); ci >= 0 {
				comment := strings.TrimSpace(line[ci+2:])
				// Skip standard annotation patterns like "Discovered via..."
				if !strings.HasPrefix(comment, "Discovered") && !strings.HasPrefix(comment, "verified") {
					purpose = comment
				}
			}
			iids[name] = &iidInfo{
				Name:    name,
				Line:    lineNum,
				Purpose: purpose,
			}
		}
	}
	return iids
}

func scanIIDUsage(actionsDir string, iids map[string]*iidInfo) map[string]bool {
	used := make(map[string]bool)

	entries, err := os.ReadDir(actionsDir)
	if err != nil {
		return used
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if entry.Name() == "winrt.go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		path := filepath.Join(actionsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)

		for name := range iids {
			if strings.Contains(content, name) {
				used[name] = true
			}
		}
	}
	return used
}

func scanIIDInternalUsage(winrtPath string, iids map[string]*iidInfo) map[string]bool {
	internal := make(map[string]bool)
	data, err := os.ReadFile(winrtPath)
	if err != nil {
		return internal
	}
	content := string(data)

	for name := range iids {
		// Count occurrences: first is the definition, subsequent are internal usages
		count := strings.Count(content, name)
		if count > 1 {
			internal[name] = true
		}
	}
	return internal
}

type docEntry struct {
	Name   string
	Value  string
	Purpose string
	Status string
	Line   int
}

func parseDocTable(body string) map[string]*docEntry {
	entries := make(map[string]*docEntry)

	lines := strings.Split(body, "\n")
	inTable := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "<!-- IID_TABLE_START -->") {
			inTable = true
			continue
		}
		if strings.Contains(trimmed, "<!-- IID_TABLE_END -->") {
			break
		}
		if !inTable {
			continue
		}
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		if strings.Contains(trimmed, "|---|---|---") || strings.Contains(trimmed, "|---|---|----") {
			continue // header separator
		}
		if strings.HasPrefix(trimmed, "| IID |") {
			continue // header row
		}

		// Parse table row: | `IID_Name` | `{GUID}` | Purpose | Status |
		parts := splitRow(trimmed)
		if len(parts) < 4 {
			continue
		}
		name := strings.Trim(parts[0], "` ")
		value := strings.Trim(parts[1], "` ")
		purpose := strings.TrimSpace(parts[2])
		status := strings.TrimSpace(parts[3])

		entries[name] = &docEntry{
			Name:    name,
			Value:   value,
			Purpose: purpose,
			Status:  status,
			Line:    i + 1,
		}
	}
	return entries
}

func splitRow(row string) []string {
	// Split on | but handle backtick-delimited content
	row = strings.Trim(row, "|")
	parts := strings.Split(row, "|")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		result = append(result, strings.TrimSpace(p))
	}
	return result
}

func buildRow(name, value, purpose, status string) string {
	return fmt.Sprintf("| `%s` | `%s` | %s | %s |", name, value, purpose, status)
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
