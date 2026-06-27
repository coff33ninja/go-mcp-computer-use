# ADR-001: MCP SDK Selection

## Status

Accepted

## Context

We need an MCP SDK for Go to build the computer-use server. Two options exist:

1. **modelcontextprotocol/go-sdk** — Official SDK maintained with Google. Supports MCP spec up to 2026-07-28. v1.7.0+.
2. **mark3labs/mcp-go** — Community SDK with 8.5k stars, 190 contributors, 73 releases. Supports MCP 2025-11-25.

Key considerations:
- We need stable, widely-adopted MCP spec version that agents support
- API ergonomics for tool registration and transport setup
- Ongoing maintenance and spec compatibility

## Decision

Use **modelcontextprotocol/go-sdk** for the following reasons:
- Maintained by the protocol authors — guaranteed spec alignment
- Clean API design with `mcp.NewServer` + `mcp.AddTool` pattern
- Stdio transport built in
- v1.7.0 supports MCP 2025-11-25 and 2026-07-28 — we target 2025-11-25 for broad agent compatibility
- Google collaboration provides long-term maintenance guarantee

## Consequences

- Easier: spec upgrades come from the same team that writes the spec
- Easier: less risk of SDK falling behind protocol changes
- Harder: fewer community examples than mark3labs/mcp-go (newer)
- Harder: API may change during v1.x development (mitigated by pinning to v1.6.1 for now)
