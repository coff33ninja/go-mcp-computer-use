# Versioning Strategy

## Status

Accepted (2026-06-27)

## Canonical Version Source

The canonical version is the `VERSION` file at repository root. The versioning policy is defined in [`plan.md`](../plan.md#versioning).

The `VERSION` file is read at build time and injected via ldflags:

```bash
go build -ldflags="-X main.Version=$(cat VERSION)" -o mcp-server.exe ./cmd/mcp-server/
```

This is what MCP clients see via `server.info`. All other references (CHANGELOG, git tags, release artifacts) must match `VERSION`. See [`ci-cd-pipeline.md`](ci-cd-pipeline.md) for the automated workflow.

## Versioning Scheme

```
v<major>.<minor>.<patch>
```

| Bump | When | Examples |
|------|------|----------|
| `+0.0.1` (patch) | Bug fixes, tool tweaks, doc updates, minor refactors | Fixing UIPI detection, adjusting OCR timing, renaming a tool parameter |
| `+0.1.0` (minor) | New tools, new capabilities, architecture changes, dependency adds | Adding native COM OCR, adding UIA layer, adding `chain` tool, introducing SQLite memory store |
| `+1.0.0` (major) | Stable release with proven architecture, all planned slices complete, field-tested | Full automation pipeline working, memory store battle-tested, ONNX integration verified |

**Current trajectory:** v0.1.x (archived, bug-fix cycle) → v0.2.x (stable, pipeline + memory + ML + introspection) → v0.3.x (active development, skill library + cross-platform) → v1.0.0 (stable release)

Breaking changes at 0.x require a minor bump (not major), per SemVer spec §4.

## Git Tagging

Every release must have an annotated or lightweight tag matching the version:

```
v0.1.0  ← tagged on the release commit
v0.1.1  ← tagged on the release commit
```

Tags are immutable once pushed. If a release is faulty, bump the patch and re-tag. Never delete and recreate a pushed tag.

## Changelog Convention

`docs/CHANGELOG.md` follows [Keep a Changelog](https://keepachangelog.com) with sections:

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
[3] Update `docs/CHANGELOG.md` with the new version heading
[4] Run pre-release gates:
      - go build ./cmd/mcp-server/     (compiles)
      - go vet ./...                   (static analysis)
      - go run ./cmd/benchmark/        (update benchmark-results.txt)
[5] Commit: "release: vX.Y.Z"
[6] Tag:   git tag vX.Y.Z
[7] Push:  git push && git push origin vX.Y.Z
[8] CI/CD auto-builds and creates GitHub Release (see ci-cd-pipeline.md)
```

## Commit Strategy

Use squash-merges into the release branch (`v0.2.x` stable, `v0.3.x` active development) — each release is a single commit on the default branch. This keeps the release history clean and makes cherry-picks straightforward. Feature branches with incremental commits are preserved in the branch history but collapsed into one commit on merge to default.

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
# Edit docs/CHANGELOG.md: add ## [0.1.1] section
$ver = (Get-Content VERSION -Raw).Trim()
go build -ldflags="-X main.Version=$ver" ./cmd/mcp-server/ && go vet ./...
go run ./cmd/benchmark/
git add -A && git commit -m "release: v0.1.1"
git tag v0.1.1
git push && git push origin v0.1.1  # triggers release workflow
```

## Cross-References

- `docs/CHANGELOG.md` — release history
- `VERSION` — canonical version source (replaces hardcoded string in server.go)
- `docs/adr-001-mcp-sdk-selection.md` — SDK choice that defines the version field location
- `docs/ci-cd-pipeline.md` — CI/CD workflows for automated build + release
- `benchmark-results.txt` — performance data updated per release

---

<sub><sup>
an entire document dedicated to versioning strategy for a project that's been at v0.x for 72 hours. "breaking changes at 0.x require a minor bump (not major), per SemVer spec §4" — we are citing sections of the SemVer spec. for a v0 project. the release process has 8 steps, 4 pre-release gates, and a commit convention. all for a changelog that grows faster than our ability to tag releases. "tags are immutable once pushed" — famous last words from someone who definitely pushed a bad tag at 3am.
</sup></sub>
