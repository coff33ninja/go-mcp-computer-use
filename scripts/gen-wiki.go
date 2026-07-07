//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root := "."
	docsDir := filepath.Join(root, "docs")
	metaDir := filepath.Join(docsDir, "meta")
	refDir := filepath.Join(docsDir, "reference")
	guidesDir := filepath.Join(docsDir, "guides")

	var b strings.Builder

	b.WriteString("# go-mcp-computer-use Wiki\n\n")
	b.WriteString("Auto-generated from project docs. Run `go run ./scripts/gen-wiki.go` to regenerate.\n\n")

	b.WriteString("## Table of Contents\n\n")
	b.WriteString("1. [Project Overview](#project-overview)\n")
	b.WriteString("2. [Features](#features)\n")
	b.WriteString("3. [Architecture](#architecture)\n")
	b.WriteString("4. [Tools Reference](#tools-reference)\n")
	b.WriteString("5. [Build & Usage](#build--usage)\n")
	b.WriteString("6. [Configuration](#configuration)\n")
	b.WriteString("7. [Project Plan](#project-plan)\n")
	b.WriteString("8. [Backlog](#backlog)\n")
	b.WriteString("9. [Changelog](#changelog)\n")
	b.WriteString("10. [Known Issues](#known-issues)\n")
	b.WriteString("11. [Security](#security)\n")
	b.WriteString("12. [Reference Docs](#reference-docs)\n")
	b.WriteString("13. [Guides](#guides)\n")
	b.WriteString("14. [CI/CD Pipeline](#cicd-pipeline)\n\n")

	b.WriteString("---\n\n")

	b.WriteString("## Project Overview\n\n")
	readSection(root, "README.md", &b, 20, 80)
	b.WriteString("\n---\n\n")

	b.WriteString("## Features\n\n")
	readSection(root, "README.md", &b, 22, 47)
	b.WriteString("\n---\n\n")

	b.WriteString("## Architecture\n\n")
	if data, err := os.ReadFile(filepath.Join(docsDir, "architecture.md")); err == nil {
		text := string(data)
		lines := strings.SplitN(text, "\n", 3)
		if len(lines) > 2 {
			b.WriteString(lines[2])
		} else {
			b.WriteString(string(data))
		}
		b.WriteString("\n\n")
	}
	b.WriteString("See [`docs/architecture.md`](docs/architecture.md) for the full architecture document.\n\n")
	b.WriteString("---\n\n")

	b.WriteString("## Tools Reference\n\n")
	if data, err := os.ReadFile(filepath.Join(refDir, "tools.md")); err == nil {
		b.WriteString(string(data))
	} else {
		b.WriteString("_Run `go run ./scripts/gen-tools-doc.go` to generate._\n\n")
	}
	b.WriteString("\n---\n\n")

	b.WriteString("## Build & Usage\n\n")
	readSection(root, "README.md", &b, 66, 77)
	b.WriteString("\n---\n\n")

	b.WriteString("## Configuration\n\n")
	if data, err := os.ReadFile(filepath.Join(refDir, "configuration.md")); err == nil {
		text := string(data)
		lines := strings.SplitN(text, "\n", 3)
		if len(lines) > 2 {
			b.WriteString(lines[2])
		} else {
			b.WriteString(string(data))
		}
		b.WriteString("\n\n")
	}
	b.WriteString("See [`docs/reference/configuration.md`](docs/reference/configuration.md) for the full config reference.\n\n")
	b.WriteString("---\n\n")

	b.WriteString("## Project Plan\n\n")
	if data, err := os.ReadFile(filepath.Join(metaDir, "plan.md")); err == nil {
		text := string(data)
		lines := strings.SplitN(text, "\n", 4)
		if len(lines) > 3 {
			for _, l := range lines[3:] {
				b.WriteString(l + "\n")
				if strings.HasPrefix(l, "## ") && l != "## Goal\n" {
					break
				}
			}
		}
		b.WriteString("\n\n")
	}
	b.WriteString("See [`docs/meta/plan.md`](docs/meta/plan.md) for the full project plan.\n\n")
	b.WriteString("---\n\n")

	b.WriteString("## Backlog\n\n")
	if data, err := os.ReadFile(filepath.Join(metaDir, "backlog.md")); err == nil {
		text := string(data)
		lines := strings.SplitN(text, "\n", 5)
		if len(lines) > 4 {
			for _, l := range lines[4:] {
				b.WriteString(l + "\n")
				if strings.HasPrefix(l, "## Summary") {
					break
				}
			}
		}
		b.WriteString("\n\n")
	}
	b.WriteString("See [`docs/meta/backlog.md`](docs/meta/backlog.md) for the full 385-item backlog.\n\n")
	b.WriteString("---\n\n")

	b.WriteString("## Changelog\n\n")
	if data, err := os.ReadFile(filepath.Join(metaDir, "CHANGELOG.md")); err == nil {
		text := string(data)
		versionHeadings := 0
		written := false
		for _, line := range strings.Split(text, "\n") {
			if strings.HasPrefix(line, "## [") {
				versionHeadings++
			}
			if versionHeadings <= 5 || !written {
				b.WriteString(line + "\n")
				if versionHeadings > 5 && !written {
					written = true
					b.WriteString("\n_... older versions truncated. See [`docs/meta/CHANGELOG.md`](docs/meta/CHANGELOG.md) for full history._\n")
				}
			}
		}
		b.WriteString("\n\n")
	}
	b.WriteString("---\n\n")

	b.WriteString("## Known Issues\n\n")
	if data, err := os.ReadFile(filepath.Join(metaDir, "known-issues.md")); err == nil {
		text := string(data)
		lines := strings.SplitN(text, "\n", 4)
		if len(lines) > 3 {
			b.WriteString(strings.Join(lines[3:], "\n"))
		}
		b.WriteString("\n\n")
	}
	b.WriteString("See [`docs/meta/known-issues.md`](docs/meta/known-issues.md) for the full list.\n\n")
	b.WriteString("---\n\n")

	b.WriteString("## Security\n\n")
	readSection(root, "README.md", &b, 54, 61)
	b.WriteString("\n")
	if data, err := os.ReadFile(filepath.Join(docsDir, "security.md")); err == nil {
		text := string(data)
		lines := strings.SplitN(text, "\n", 4)
		if len(lines) > 3 {
			b.WriteString(strings.Join(lines[3:], "\n"))
		}
		b.WriteString("\n\n")
	}
	b.WriteString("See [`docs/security.md`](docs/security.md) for the full security document.\n\n")
	b.WriteString("---\n\n")

	b.WriteString("## Reference Docs\n\n")
	entries, _ := os.ReadDir(refDir)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			name := strings.TrimSuffix(e.Name(), ".md")
			title := strings.ReplaceAll(name, "-", " ")
			title = strings.ReplaceAll(title, "_", " ")
			title = strings.Title(title)
			b.WriteString(fmt.Sprintf("- [%s](docs/reference/%s) — ", title, e.Name()))
			if data, err := os.ReadFile(filepath.Join(refDir, e.Name())); err == nil {
				text := string(data)
				for _, line := range strings.Split(text, "\n") {
					if strings.HasPrefix(line, "> ") {
						b.WriteString(strings.TrimPrefix(line, "> "))
						break
					}
				}
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\n---\n\n")

	b.WriteString("## Guides\n\n")
	entries, _ = os.ReadDir(guidesDir)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			name := strings.TrimSuffix(e.Name(), ".md")
			title := strings.ReplaceAll(name, "-", " ")
			title = strings.ReplaceAll(title, "_", " ")
			title = strings.Title(title)
			b.WriteString(fmt.Sprintf("- [%s](docs/guides/%s)\n", title, e.Name()))
		}
	}
	b.WriteString("\n---\n\n")

	b.WriteString("## CI/CD Pipeline\n\n")
	if data, err := os.ReadFile(filepath.Join(docsDir, "ci-cd-pipeline.md")); err == nil {
		text := string(data)
		lines := strings.SplitN(text, "\n", 4)
		if len(lines) > 3 {
			b.WriteString(strings.Join(lines[3:], "\n"))
		}
		b.WriteString("\n\n")
	}
	b.WriteString("See [`docs/ci-cd-pipeline.md`](docs/ci-cd-pipeline.md) for the full CI/CD documentation.\n\n")

	b.WriteString("<!--\n")
	b.WriteString("Generated by scripts/gen-wiki.go\n")
	b.WriteString("-->\n")

	if err := os.WriteFile("WIKI.md", []byte(b.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing WIKI.md: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("wrote WIKI.md")
}

func readSection(root, file string, b *strings.Builder, start, end int) {
	data, err := os.ReadFile(filepath.Join(root, file))
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if i < start {
			continue
		}
		if i >= end {
			break
		}
		b.WriteString(line + "\n")
	}
}
