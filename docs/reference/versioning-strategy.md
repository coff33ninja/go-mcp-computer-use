# Versioning Strategy

## Status

Accepted (2026-06-27)

## Canonical Version Source

The canonical version is the `VERSION` file at repository root. The versioning policy is defined in this document.

The `VERSION` file is read at build time and injected via ldflags:

```bash
go build -ldflags="-X main.Version=$(cat VERSION)" -o mcp-server.exe ./cmd/mcp-server/
```

This is what MCP clients see via `server.info`. All other references (CHANGELOG, git tags, release artifacts) must match `VERSION`. See [`../ci-cd-pipeline.md`](../ci-cd-pipeline.md) for the automated workflow.

## Versioning Scheme

```
v<major>.<minor>.<patch>
```

| Bump | When | Examples |
|------|------|----------|
| `+0.0.1` (patch) | Bug fixes, tool tweaks, doc updates, minor refactors | Fixing UIPI detection, adjusting OCR timing, renaming a tool parameter |
| `+0.1.0` (minor) | New tools, new capabilities, architecture changes, dependency adds | Adding native COM OCR, adding UIA layer, adding `chain` tool, introducing SQLite memory store |
| `+1.0.0` (major) | Stable release with proven architecture, all planned slices complete, field-tested | Full automation pipeline working, memory store battle-tested, ONNX integration verified |

**Current trajectory:** v0.1.x (bug-fix cycle on initial tools) → v0.2.x (automation pipeline + memory + ML + priors + keylogger) → v0.3.x (iterative improvements) → v1.0.0 (stable release)

Breaking changes at 0.x require a minor bump (not major), per SemVer spec §4.

## Git Tagging

Every release must have an annotated or lightweight tag matching the version:

```
v0.1.0  ← tagged on the release commit
v0.1.1  ← tagged on the release commit
```

Tags are immutable once pushed. If a release is faulty, bump the patch and re-tag. Never delete and recreate a pushed tag.

## Changelog Convention

`../meta/CHANGELOG.md` follows [Keep a Changelog](https://keepachangelog.com) with sections:

- `### Added` — new tools, new capabilities
- `### Changed` — modifications to existing tools or behavior
- `### Fixed` — bug fixes
- `### Removed` — removed tools or features
- `### Security` — security-related changes
- `### Performance Improvements` — perf changes

A changelog entry is required for every release. Entries are written in present-tense imperative mood.

## Release Process

```
[1] Code complete — all changes for the release are merged
[2] Bump version in VERSION file
[3] Update `../meta/CHANGELOG.md` with the new version heading
[4] Run pre-release gates:
      - go build ./cmd/mcp-server/     (compiles)
      - go vet ./...                   (static analysis)
      - go run ./cmd/benchmark/        (update benchmark-results.txt)
[5] Commit: "release: vX.Y.Z"
[6] Tag:   git tag vX.Y.Z
[7] Push:  git push && git push origin vX.Y.Z
[8] CI/CD auto-builds and creates GitHub Release (see ../ci-cd-pipeline.md)
```

## Commit Strategy

Use squash-merges into `master`/`main` — each release is a single commit on the default branch. This keeps the release history clean and makes cherry-picks straightforward. Feature branches with incremental commits are preserved in the branch history but collapsed into one commit on merge to default.

## Pre-Release Gates (mandatory)

| Gate | Command | Fail action |
|------|---------|-------------|
| Lint | `go vet ./...` | Fix warnings — see `.govetallow` for COM conventions |
| Build | `go build ./cmd/mcp-server/` | Fix compilation |
| Benchmarks | `go run ./cmd/benchmark/` | Update results if numbers changed materially |
| Version consistency | `git grep "0\.1\.10" -- ':!VERSION' ':!.git'` | Fix stale references |

## Example: Patch Release

```bash
# Edit VERSION: "0.1.0" → "0.1.1"
# Edit ../meta/CHANGELOG.md: add ## [0.1.1] section
$ver = (Get-Content VERSION -Raw).Trim()
go build -ldflags="-X main.Version=$ver" ./cmd/mcp-server/ && go vet ./...
go run ./cmd/benchmark/
git add -A && git commit -m "release: v0.1.1"
git tag v0.1.1
git push && git push origin v0.1.1  # triggers release workflow
```

## Cross-References

- `../meta/CHANGELOG.md` — release history
- `VERSION` — canonical version source (replaces hardcoded string in server.go)
- `../adr/adr-001-mcp-sdk-selection.md` — SDK choice that defines the version field location
- `../ci-cd-pipeline.md` — CI/CD workflows for automated build + release
- `benchmark-results.txt` — performance data updated per release

## Adding a New MCP Tool

Every new tool requires updates in multiple files. Skip one and the count goes stale or the tool silently breaks.

### Step-by-step

1. **`internal/server/server.go`** — add `mcp.AddTool` call. Do NOT manually update the hardcoded `"tools", N` count in the startup log — `gen-tools-doc.go` patches it automatically.

2. **`scripts/gen-tools-doc.go`** — add the tool name to `categoryForTool` map with its category. If creating a new category, also add it to `categoryOrder`. Missing from this map = "Uncategorized" in generated docs.

3. **`internal/actions/chain.go`** (if chain-executable) — add to `toolDispatch` map + implement handler. Register in `inputTools` or `trainingTools`.

4. **`internal/actions/adaptive.go`** (if ML-tracked) — add to `coordTools` or `coordIndex`.

5. **Run `go run ./scripts/gen-tools-doc.go`** — auto-patches tool count in:
   - `internal/server/server.go` (startup log)
   - `docs/reference/tools.md` (regenerated)
   - `README.md`, `docs/architecture.md`, `docs/comparison-vs-alternatives.md`, `docs/guides/agent-guides.md`, `docs/guides/computer-use-guide-for-ai-agents.md`, `docs/meta/known-issues.md`, `docs/reference/scripts.md`, `docs/meta/backlog.md`

6. **Tests** — add in relevant `*_test.go`, run `go test ./internal/actions/...`

7. **CHANGELOG** — add entry under `## [Unreleased]`

### What push-and-release.ps1 does automatically

1. Reads `VERSION` file
2. Extracts changelog section for that version
3. Runs `go run ./scripts/gen-tools-doc.go` (patches all counts)
4. Commits + tags + pushes
5. Waits for CI release workflow
6. Downloads release binary, kills old processes, replaces, restarts OpenCode

### Common mistakes

- Forgetting `gen-tools-doc.go` → tool count stays stale in server.go and all docs
- Adding tool to server.go but not `categoryForTool` → shows as "Uncategorized"
- Adding chain tool but not `toolDispatch` → registered but not chain-executable
- Editing `VERSION` without running `gen-tools-doc.go` → CI rebuilds but counts don't update
- Building locally without `-ldflags` → binary shows "dev" (expected, CI handles it)
