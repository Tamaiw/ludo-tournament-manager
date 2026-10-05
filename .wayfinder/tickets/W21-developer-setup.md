# W21: Set up developer environment

**Type:** task
**State:** resolved
**Assignee:** opencode
**Blocked by:** (none)
**Blocks:** (gates implementation — no real code can be written until this is done)

## Question

Install and verify all developer software required before any real code is written, on both Windows and Linux dev machines. The user has a fresh computer setup; this ticket results in a setup guide they (and any future contributors) can follow.

## Specific things to install

- **Go** — latest stable (1.23+)
  - Windows: MSI installer from `go.dev`, default options add Go to `PATH`
  - Linux: download tarball OR use distro package (`apt install golang`, `pacman -S go`, etc.)
  - Verify: `go version`
- **Docker** (Docker Desktop on Windows, Docker Engine on Linux)
  - Windows: Docker Desktop installer; requires WSL2 enabled
  - Linux: Docker Engine + Docker Compose plugin (`docker compose`, v2)
  - Verify: `docker --version`, `docker compose version`
- **Git** (likely already present, but verify)
  - Verify: `git --version`, `git config user.name` / `user.email` set
- **A code editor** — VS Code recommended (cross-platform)
  - Install the official Go extension — bundles `gopls` (the Go language server) automatically, giving you real-time errors, autocomplete, go-to-definition, and rename refactoring in-editor
  - Optionally: editorconfig extension for consistent formatting
- **gopls** (if not using VS Code) — install standalone with `go install golang.org/x/tools/gopls@latest` for other editors (Neovim, Helix, etc.)
- **SQLite CLI** (for inspecting and backing up the DB during dev)
  - Cross-platform installer / package (`apt install sqlite3`, etc.)
  - Verify: `sqlite3 --version`
- **Make** (optional — for running helper scripts)
  - Windows: install via Chocolatey / WSL / Git Bash
  - Linux: usually pre-installed
  - Verify: `make --version`

## What "good" looks like

A `docs/setup.md` (with Windows + Linux sections, or two files: `docs/setup-windows.md` and `docs/setup-linux.md`) listing each tool, the install command(s) per OS, and a verification step. After following the guide, the user can:

1. `git clone <repo>` and `cd` into it
2. `docker compose up -d` — Mailpit container starts cleanly
3. `cd backend && go test ./...` — compiles without errors (even before any tests exist, this exercises `go mod tidy` + the toolchain)
4. `sqlite3 :memory: ".version"` — SQLite CLI responds

The deliverable lives in `docs/setup*.md`. The ticket closes with the path to the guide.

## Out of scope

- Production deployment setup — that's W11 (deployment shape, currently blocked)
- CI / GitHub Actions setup
- Production secrets management
- Domain / TLS provisioning for the Hetzner VPS
- Additional MCP servers for AI agents — `codebase-memory-mcp` (structural code discovery) and `context7` (library docs) are already configured at `~/.config/opencode/` and are exactly what AI agents need on this codebase. No more are required.