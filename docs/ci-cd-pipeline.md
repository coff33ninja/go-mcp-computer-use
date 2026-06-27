# CI/CD Pipeline

## Overview

Windows-only Go project. CI builds + vets on every push/PR. Release workflow cuts a GitHub Release with the binary + changelog when tagged.

## Version File

`VERSION` at repository root — single plain-text file containing the semver string (e.g. `0.1.10`). This is the canonical source:

- `go build -ldflags="-X main.Version=$(cat VERSION)"` injects it into the binary
- CI reads it for artifact naming
- Release workflow validates the git tag matches `VERSION` before building
- CHANGELOG.md headings must match

## Workflows

### CI (`.github/workflows/ci.yml`)

| Trigger | Action |
|---------|--------|
| Push to `main`, `v0.2.x` | Build + vet + upload artifact |
| PR to `main`, `v0.2.x` | Build + vet |

Artifact name: `mcp-server-windows-<version>`

### Release (`.github/workflows/release.yml`)

| Trigger | Action |
|---------|--------|
| Push tag `v*` | Build + SHA256 + GitHub Release |

Validates tag matches VERSION file. Extracts the corresponding section from CHANGELOG.md as release body. Uploads `mcp-server.exe` + `mcp-server.exe.sha256`.

## Branching Strategy

```
main  ───────────────────────────────────●─── (stable releases)
                                         │
v0.2.0-alpha ──●──●──●──●──●──●──●──────┘
               (feature work, chain/memory/ML)
```

| Branch | Purpose |
|--------|---------|
| `main` | Stable — release-ready. CI runs. Tags cut here. |
| `v0.2.0-alpha` | Feature branch for v0.2.0 work (chain tool, SQLite memory store, layout validation, template library, ONNX ML). CI runs. PRs merge into `main`. |
| Feature branches | Short-lived forks from `v0.2.0-alpha`. Squash-merged. |

### Release Cycle

```
[1] Feature work on alpha branch
[2] Bump VERSION (e.g. 0.1.10 -> 0.2.0)
[3] Update CHANGELOG.md with release notes
[4] Open PR: alpha -> main
[5] CI runs on PR — must pass build + vet
[6] Squash-merge to main
[7] Tag the merge commit: git tag v0.2.0 && git push origin v0.2.0
[8] Release workflow auto-builds + publishes GitHub Release
```

## Running CI Locally

```powershell
# Full lint (vet + build)
.\scripts\lint.ps1

# Just vet
go vet ./...

# Build with version injection
$ver = (Get-Content VERSION -Raw).Trim()
go build -ldflags="-X main.Version=$ver" -o mcp-server.exe ./cmd/mcp-server/

# Benchmark
go run ./cmd/benchmark/
```

## Cross-References

- `VERSION` — canonical version source
- `CHANGELOG.md` — release notes per version
- `.govetallow` — vet allowance conventions for COM/WinRT interop
- `scripts/lint.ps1` — local CI runner (vet + build)
- `.github/workflows/ci.yml` — CI workflow
- `.github/workflows/release.yml` — release workflow
- `docs/versioning-strategy.md` — version bump rules
