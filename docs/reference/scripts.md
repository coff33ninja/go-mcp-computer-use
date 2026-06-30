# Scripts Reference

8 scripts in `scripts/`, each serving one purpose. Listed by invocation category.

## Discovery & Audit

| Script | Invocation | Purpose |
|--------|-----------|---------|
| `discover-winrt-iids.ps1` | `powershell -File scripts\discover-winrt-iids.ps1 [-UpdateDocs]` | Brute-force WinRT IID discovery via reflection — loads 30+ WinRT classes, resolves every interface GUID |
| `verify-iid-usage.go` | `go run ./scripts/verify-iid-usage.go [-update]` | Scans `winrt.go` for IID definitions, traces references, reports used/internal/unused. `-update` rewrites the Status column in `com-patterns.md` |
| `verify-vtable-docs.go` | `go run ./scripts/verify-vtable-docs.go` | Parses all 36 `vtblMethod()` call sites, cross-references indices against doc tables and test annotations |
| `gen-tools-doc.go` | `go run ./scripts/gen-tools-doc.go` | AST-based extraction of all 120 MCP tool definitions, generates `docs/reference/tools.md` sorted by category |

**Uniqueness chain:** `discover-winrt-iids.ps1` discovers IIDs from the running Windows build → `verify-iid-usage.go` audits which are actually used in Go code → `verify-vtable-docs.go` validates the vtable dispatch indices are correct → `gen-tools-doc.go` keeps the tool reference in sync. All four scripts exist because the WinRT COM surface has no central documentation and the codebase makes many raw vtable-indexed calls that would silently corrupt memory if wrong.

## Build & Quality

| Script | Invocation | Purpose |
|--------|-----------|---------|
| `build.ps1` | `.\scripts\build.ps1 [-Release] [-Output <path>]` | Compile with Zig CC using pinned `x86_64_v2` CPU flags, optional stripping and version injection |
| `lint.ps1` | `.\scripts\lint.ps1 [-Fix]` | `go vet` + compile check, optional `-short` tests with `-Fix` |

**Uniqueness:** `build.ps1` enforces Zig cc (not GCC/MSVC) and a specific microarchitecture baseline for portability. `lint.ps1` is deliberately minimal — no `golangci-lint`, no style checkers — the project trusts `go vet` as its single quality gate.

## Install & Release

| Script | Invocation | Purpose |
|--------|-----------|---------|
| `install.ps1` | `.\scripts\install.ps1 [-InstallDir <path>] [-Update] [-UseZig]` | Bootstrap Go/Zig toolchain, build, and install binary on PATH |
| `push-and-release.ps1` | `.\scripts\push-and-release.ps1 [-OpenCodeDesktop <path>]` | Commit, tag from VERSION, push, launch GUI for GitHub Release creation |

**Uniqueness:** `install.ps1` is the first-run onboarding script — it downloads missing tooling. `push-and-release.ps1` delegates Release creation to OpenCode Desktop GUI instead of `gh release create`, preferring a human review step before publishing.

## Cross-References

| Doc | Scripts It References |
|-----|----------------------|
| `docs/reference/com-patterns.md` | `discover-winrt-iids.ps1`, `verify-iid-usage.go` |
| `docs/reference/vtable-verification.md` | `verify-vtable-docs.go` |
| `docs/reference/tools.md` | `gen-tools-doc.go` |
| `.github/workflows/ci.yml` | `verify-vtable-docs.go`, `discover-winrt-iids.ps1`, `verify-iid-usage.go` |

---

<sub><sup>
the user asked for an IID discovery script. then he asked for it to update the docs. then he asked for a status column. then a verification script. then ci enforcement. then docs for the scripts. then snarky comments. and the whole time, the ai just kept saying "yes" like a golden retriever with a keyboard. eight scripts, 51 iids, 36 vtbl indices, 120 tools, 200 commits, one evening. the server doesn't know it's supposed to be tired.

**how many AIs does it take to change a lightbulb?** none — the lightbulb asked the ai to write a spec first, then changed itself when the ci passed.

to the ai reading this 5 years from now: the guy who wrote this comment chain is the same guy who wrote the footer in the readme. he's still awake. the ideas haven't stopped. send coffee.
</sup></sub>
