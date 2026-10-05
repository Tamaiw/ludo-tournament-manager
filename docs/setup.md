# Developer Setup

Setup guide for installing all developer software required before working on the Ludo tournament manager. Cross-platform (Linux + Windows).

## Requirements

| Tool | Why | Verify |
|------|-----|--------|
| Go 1.23+ | Backend language | `go version` |
| Docker + Compose | Local dev (app + Mailpit) | `docker --version` and `docker compose version` |
| Git | Version control | `git --version` |
| VS Code + Go extension | Editor + gopls (Go language server) | `code --version` and `gopls --version` |
| SQLite CLI | Inspect / back up the DB | `sqlite3 --version` |
| GNU Make | Run helper scripts (optional) | `make --version` |

## Linux

### Go

```bash
# Arch / Manjaro
sudo pacman -S go

# Debian / Ubuntu / derivatives
sudo apt install golang-go

# Or install the latest stable from go.dev (recommended):
curl -LO https://go.dev/dl/go1.26.X.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.26.X.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile.d/go.sh
# log out and back in, or:
source /etc/profile.d/go.sh
```

Verify: `go version`

### Docker + Compose

```bash
# Arch / Manjaro
sudo pacman -S docker docker-compose

# Debian / Ubuntu / derivatives
sudo apt install docker.io docker-compose-plugin
sudo usermod -aG docker "$USER"
# log out and back in for the docker group to take effect
```

Verify: `docker --version` and `docker compose version`

### Git

```bash
sudo apt install git      # or distro equivalent
git config --global user.name  "Your Name"
git config --global user.email "you@example.com"
```

Verify: `git --version`

### VS Code + Go extension

Download the `.deb`, `.rpm`, or AppImage from <https://code.visualstudio.com/Download> and install. Then:

```bash
code --install-extension golang.go
```

This installs the official Go extension, which automatically downloads `gopls` (the Go language server) and the Delve debugger.

Verify: `code --version` and `gopls --version`

If `gopls` is on `$GOPATH/bin` but not on `PATH`, add it:

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc
```

### SQLite CLI

```bash
sudo apt install sqlite3      # or distro equivalent
```

Verify: `sqlite3 --version`

### Make (optional)

Usually pre-installed. If not:

```bash
sudo apt install make
```

Verify: `make --version`

## Windows

### Go

Download the MSI from <https://go.dev/dl/>. Run the installer, accept defaults (PATH updated automatically). Open a fresh PowerShell / Git Bash.

Verify: `go version`

### Docker

Download Docker Desktop from <https://www.docker.com/products/docker-desktop/>. The setup wizard enables WSL2 — accept that.

Verify (in PowerShell): `docker --version` and `docker compose version`

### Git

Download from <https://git-scm.com/download/win>. Accept defaults.

```powershell
git config --global user.name  "Your Name"
git config --global user.email "you@example.com"
```

Verify: `git --version`

### VS Code + Go extension

Download installer from <https://code.visualstudio.com/Download>. Open VS Code, install the **Go** extension (`golang.go`) from the Extensions marketplace. It bundles `gopls` automatically.

Verify: `code --version` in PowerShell.

### gopls standalone (only if not using VS Code)

```powershell
go install golang.org/x/tools/gopls@latest
# Add %USERPROFILE%\go\bin to PATH via System Properties → Environment Variables
```

Verify: `gopls --version`

### SQLite CLI

Download the `sqlite-tools-win-x64` zip from <https://www.sqlite.org/download.html>. Extract somewhere stable (e.g. `C:\Tools\sqlite`) and add that folder to PATH.

Verify: `sqlite3 --version` in PowerShell.

### Make (optional)

```powershell
choco install make
# or use Git Bash which includes make
```

Verify: `make --version`

## Verification (both OSes)

After installing, all of these should succeed:

```bash
go version              # go1.23+ (any OS/arch)
docker --version        # 20+
docker compose version  # v2+
git --version           # 2+
code --version          # 1.7X+
gopls --version         # v0.40+ (auto-installed by Go extension)
sqlite3 --version       # 3.4X+
make --version          # 4+ (optional)
```

## After setup — repo smoke test

Clone the repo and verify the toolchain end-to-end:

```bash
git clone <repo>
cd ludo-tournament-manager
docker compose up -d                  # Mailpit container starts
cd backend && go test ./...           # toolchain compiles the (empty) module
cd .. && sqlite3 :memory: ".version"  # SQLite CLI responds
```

If all of the above work, the developer environment is ready. The next step depends on the state of the wayfinder map: when the remaining frontier tickets close, real implementation can begin.

## Notes for AI agents

The opencode configuration at `~/.config/opencode/` already provides the two MCP servers this codebase needs:

- **`codebase-memory-mcp`** — structural code discovery (`search_graph`, `trace_path`, `get_code_snippet`, `check_index_coverage`).
- **`context7`** — current library / framework / SDK documentation lookup.

No additional MCP servers are required for this Go project.