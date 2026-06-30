# Security Policy

This project can fully control a Windows desktop. That's the point. But it's designed to be local-first and transparent about what it does.

## What you should know

- Everything runs locally. No telemetry, no phone-home, no cloud dependency.
- Every action can be logged. Training capture can be disabled at runtime.
- The server does not expose network endpoints by default. MCP communicates over stdio.
- If you're using it with a remote MCP transport (e.g. SSE), secure that connection yourself.

## Reporting a vulnerability

If you find something that bypasses the privacy controls or allows unauthorized access, open an issue with the label `security`. If it's sensitive, reach out to the repo owner directly.

This is a hobby project by a single developer. Response times won't be instant, but legitimate reports will be taken seriously.
