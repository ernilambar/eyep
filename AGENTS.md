# AGENTS.md

## Overview

`eyep` is a CLI that inspects geographical and network details for an IP address or the local machine. Core stack: Go (standard library only), tested with `go test`.

## Setup

- Runtime dependencies: none (single static binary)
- Build dependency: Go 1.27+
- No package install; the module has no third-party dependencies

## Commands

- Run all tests: `go test ./...`
- Vet: `go vet ./...`
- Format check: `gofmt -l .`
- Run the CLI directly: `go run . -h`
- Build: `go build -o eyep .`

## Conventions

- **Standard library only** — no third-party modules.
- **gofmt** — tabs for indentation; `gofmt -l .` must print nothing.
- **Every user-facing error goes to stderr** with a clear message and a non-zero exit code.
- **Tests use `httptest` and injected backends, never real network calls** — see `runWith` in `main.go` and `testBackend` in `api_test.go`.
- **Keep `CHANGELOG.md` current with [flatlog](https://github.com/ernilambar/flatlog)** — `npx @nilambar/flatlog bullet "Added: ..."` for each user-facing change, `npx @nilambar/flatlog release <version>` when tagging. Prefixes: `Added:`, `Changed:`, `Fixed:`.
- **`VERSION` is the single source of truth** for the release tag. It is embedded into the binary and the release workflow rejects a tag that disagrees with it.

## Quality Gate

Before declaring a task complete, run in order and confirm each exits 0:

1. `gofmt -l .` — formatting check (must print nothing)
2. `go vet ./...` — static analysis
3. `go test ./...` — full test suite
4. `npx @nilambar/flatlog validate` — changelog structure (see `CHANGELOG.md`)
