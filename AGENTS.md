# AGENTS.md

## Overview

`eyep` is a POSIX-compliant shell CLI that inspects geographical and network details for an IP address or the local machine. Core stack: `sh`, `curl`, `jq`, and `bats` for testing.

## Setup

- Runtime dependencies: `curl`, `jq`
- Test dependency: `bats` (`apt-get install -y bats` on Ubuntu)
- No build step or package install — the single file `eyep.sh` is the project.

## Commands

- Run all tests: `bats test/`
- Validate script syntax: `sh -n eyep.sh`
- Run the CLI directly: `./eyep.sh -h`

## Conventions

- **POSIX `sh` only** — no bashisms. Always run via `sh eyep.sh` or with a `#!/bin/sh` shebang. Verify with `sh -n`.
- **Tabs for indentation** — enforced by `.editorconfig`. Spaces only in `.md` and `.json`.
- **`set -eu` at the top of every script** — exit on error and undefined variables.
- **Every user-facing error goes to stderr** with a clear message; exit non-zero.
- **Tests use stubs, never real network calls** — see `test/stubs/curl` and the env vars `CURL_STUB_IP`, `CURL_STUB_JSON`, `CURL_STUB_EXIT`, `CURL_LOG`.
- **`VERSION` is the single source of truth** for the release tag; it's read by tests and the release workflow.

## Quality Gate

Before declaring a task complete, run in order and confirm each exits 0:

1. `sh -n eyep.sh` — syntax check
2. `bats test/` — full test suite
