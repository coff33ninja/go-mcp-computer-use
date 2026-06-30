# Contributing

This project was built by pairing a human (who knows what they want) with AI (who knows how to write it). That workflow is the norm here, not the exception.

## How to contribute

- **Issues** — Bug reports and feature requests are welcome. Open an issue before spending time on a PR if you're not sure the change fits.
- **PRs** — If you're fixing a bug or adding a small feature, go for it. For larger changes, open an issue first so we don't duplicate effort.
- **AI-generated code** is fine. This repo was built with it and expects it. Just make sure you understand what the code does before submitting.

## Guidelines

- Keep the existing code style — look at nearby files before writing new ones.
- If you add a COM/WinRT vtable call, annotate the index and add a test. The CI will check.
- If you add a new tool, run `go run ./scripts/gen-tools-doc.go` to regenerate the docs.
- Run `go vet ./...` before committing.
- Keep the snark. It's part of the charm.

## Code of Conduct

Don't be a jerk. That's it.
