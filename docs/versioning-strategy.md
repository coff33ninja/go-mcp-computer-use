# Versioning Strategy

## Status

Accepted (2026-06-27)

## Canonical Version Source

The single source of truth is the `Version` field in `internal/server/server.go`:

```go
mcp.NewServer(&mcp.Implementation{
    Name:    "go-mcp-computer-use",
    Version: "0.1.1",
}, nil)
```

This is what MCP clients see via `server.info`. All other references (CHANGELOG, git tags, release artifacts) must match this value.

## SemVer Convention (pre-1.0)

Since we are on major version 0, use the `0.x.y` scheme:

| Bump | When | Example |
|------|------|---------|
| `y` (patch) | Bug fixes, performance improvements, docs, refactors, new non-breaking tools | `0.1.0` → `0.1.1` |
| `x` (minor) | Breaking API changes, major feature additions that change the tool contract, significant architectural changes | `0.1.x` → `0.2.0` |

Breaking changes at 0.x require a minor bump (not major), per SemVer spec §4.

## Git Tagging

Every release must have an annotated or lightweight tag matching the version:

```
v0.1.0  ← tagged on the release commit
v0.1.1  ← tagged on the release commit
```

Tags are immutable once pushed. If a release is faulty, bump the patch and re-tag. Never delete and recreate a pushed tag.

## Changelog Convention

CHANGELOG.md follows [Keep a Changelog](https://keepachangelog.com) with sections:

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
[2] Bump version in internal/server/server.go
[3] Update CHANGELOG.md with the new version heading
[4] Run pre-release gates:
      - go build ./cmd/mcp-server/     (compiles)
      - go vet ./...                   (static analysis)
      - go run ./cmd/benchmark/        (update benchmark-results.txt)
[5] Commit: "release: vX.Y.Z"
[6] Tag:   git tag vX.Y.Z
[7] Build: go build -o mcp-server.exe ./cmd/mcp-server/
[8] Push:  git push && git push origin vX.Y.Z
```

## Commit Strategy

Use squash-merges into `master`/`main` — each release is a single commit on the default branch. This keeps the release history clean and makes cherry-picks straightforward. Feature branches with incremental commits are preserved in the branch history but collapsed into one commit on merge to default.

## Pre-Release Gates (mandatory)

| Gate | Command | Fail action |
|------|---------|-------------|
| Build | `go build ./cmd/mcp-server/` | Fix compilation |
| Vet | `go vet ./...` | Fix warnings |
| Benchmarks | `go run ./cmd/benchmark/` | Update results if numbers changed materially |
| Version consistency | grep for old version number in code and docs | Fix stale references |

## Example: Patch Release

```bash
# Edit internal/server/server.go: "0.1.0" → "0.1.1"
# Edit CHANGELOG.md: add ## [0.1.1] section
go build ./cmd/mcp-server/ && go vet ./... && go run ./cmd/benchmark/
git add -A && git commit -m "release: v0.1.1"
git tag v0.1.1
go build -o mcp-server.exe ./cmd/mcp-server/
git push && git push origin v0.1.1
```

## Cross-References

- `CHANGELOG.md` — release history
- `docs/adr-001-mcp-sdk-selection.md` — SDK choice that defines the version field location
- `benchmark-results.txt` — performance data updated per release
- `internal/server/server.go` — canonical version source
