# Deployment shape: compose-everywhere with split dev runtime

The system runs in containers in production and uses containers only for sidecar state in local development. The Go app runs on the host in dev; everything (Go app + reverse proxy) runs in `docker compose` in production. A single multi-stage `Dockerfile` produces the binary; SQLite lives on a host bind-mount so backups are host-side tooling.

## Context

The deployment target is a single Hetzner VPS (or a homelab box), not a multi-host cluster — k8s, ECS, and friends are out of scope. The codebase is a single Go binary serving an htmx/Alpine frontend (`html/template` files embedded via `embed.FS`), backed by SQLite. The stack is locked: Chi, `coder/websocket`, SQLite + `modernc.org/sqlite`, htmx + Alpine, Resend + Mailpit, hexagonal architecture. Cross-platform dev (Linux + Windows + macOS) is a hard destination constraint.

Standing preferences require container-first deployment and cross-platform dev. The dev setup guide (W21) already shipped an implicit shape — `docker compose up -d` for Mailpit, then `go test` on the host — but the shape had not been formally decided. This ticket ratifies that shape and pins down the rest: prod runtime, reverse proxy, build, data persistence, bootstrap, and update.

## Decision

1. **Local dev: split runtime.** `docker compose -f deploy/docker-compose.dev.yml up -d` starts Mailpit only; `cd backend && go run ./cmd/server` runs the Go app on the host. SQLite database file lives at `./data/ludo.db` on the host. No Caddy in dev.
2. **Production: `docker compose` on the VPS.** A single `deploy/docker-compose.yml` runs the `app` (the static Go binary) and `caddy` services. Both services share the same image, env (loaded from `.env` at the repo root), and bind-mounts.
3. **Reverse proxy / TLS: Caddy.** A single `deploy/Caddyfile` is bind-mounted into the caddy container. ACME auto-cert handles Let's Encrypt; HTTP→HTTPS redirect is on by default. Dev runs HTTP only (no TLS, no Caddy).
4. **Container build: single static binary in `alpine`.** Multi-stage `Dockerfile` (Go builder → alpine runtime) produces a CGO-disabled binary with `html/template` files embedded via `//go:embed`. Base is `alpine` (~10 MB) rather than `scratch` so `docker exec` debugging is possible without a second image.
5. **SQLite persistence: bind-mount `./data:/data`.** The SQLite file is a real file on the host, so backups run as host-side tooling (cron, `sqlite3 .backup`, Litestream) without `docker cp` indirection. The `data/` directory is gitignored.
6. **First-time bootstrap: separate `cmd/seed/main.go` invoked via `docker compose -f deploy/docker-compose.yml run --rm --build seed --email <email> --password <password>`.** The compose file declares both `app` and `seed` services sharing image + env + bind-mount. Email + password come from CLI args (with `.env` fallback), not a wizard. The seed binary is re-runnable for disaster recovery.
7. **Update flow: `git pull && docker compose -f deploy/docker-compose.yml up -d --build`.** Two commands, no helper script, no Makefile target, no Ansible playbook.

## Considered options

- **Full docker compose for dev (Go + Mailpit + SQLite in containers)** — rejected. Every code change requires an image rebuild, slowing the edit/test loop by an order of magnitude. The "mirrors prod" benefit is real but is achieved more cheaply by sharing the Dockerfile between dev and prod.
- **Host-only dev (no Docker at all)** — rejected. Mailpit still wants to run in a container for parity with prod; installing it natively on every dev platform is friction. Cross-platform pain (Windows service vs macOS brew vs Linux systemd) outweighs the small speed benefit over the split-runtime option.
- **`systemd` unit + Caddy-as-system-package in prod** — rejected. Diverges from the "container-first" standing preference; doubles the operational vocabulary (Docker locally, systemd in place); recovery-from-snapshot on a new VPS requires installing Go, configuring systemd, configuring Caddy, configuring the firewall — versus one `git clone` + `docker compose up` with compose.
- **Caddy auto-TLS via `caddy-docker-proxy` plugin (no Caddyfile)** — rejected. The Caddyfile is the audit artefact for "what does prod see?" — losing it to plugin magic makes the deployment harder to read and harder to debug.
- **`nginx` + certbot in prod** — rejected. Caddyfile is shorter than an nginx config + certbot cron + renewal hook; ACME renewal is automatic in Caddy and a manual ceremony in nginx. For a single VPS, the saved complexity is decisive.
- **`traefik` in prod** — rejected. Docker-aware and elegant, but heavier and the auto-discovery pattern obscures the route table. For a one-app VPS, Caddy is the right size.
- **Distroless/static base image** — rejected. ~20 MB savings is irrelevant; losing the shell makes `docker exec` debugging impossible without a sidecar. Alpine is the safer default.
- **Named `docker volume` for SQLite** — rejected. Forces backup tooling to use `docker run --rm -v ludo_data:/data alpine sqlite3 ...` instead of `sqlite3 data/ludo.db .backup` on the host. Host-side cron / Litestream is materially simpler.
- **Web-based `/setup` wizard for the first manager** — rejected. Adds a state machine to the binary (the wizard is not idempotent and must delete itself after first use); the CLI-seed approach is stateless and re-runnable. The cost of a wizard for a single first-time setup is not justified.
- **Ansible playbook / helper script / Makefile target for updates** — rejected. Two commands don't earn an abstraction layer. A Makefile target can be added later without changing the underlying commands.

## Consequences

- **Application code unchanged.** Hexagonal architecture (ADR 0001) keeps Go, SQLite, SMTP, and WebSocket behind ports. Swapping any component above (Caddy → Traefik, compose → systemd, host cron → Litestream) is a `deploy/` change only.
- **`deploy/` is a hard boundary.** It is the only place that knows about containers, bind-mounts, Caddy, or `.env`. Application code under `backend/` is environment-agnostic.
- **Backup tooling (W19) is host-side.** Cadence, retention, and restore are entirely host concerns (`cron`, `sqlite3 .backup`, or Litestream reading `./data/ludo.db`). The `seed` binary doubles as a disaster-recovery tool: `git clone` on a fresh box + the seed binary + the latest backup restores the system.
- **Cross-platform dev is preserved.** Windows + Linux + macOS all run Go natively and Docker Desktop. The bind-mount `./data:/data` works because Docker Desktop normalises host paths.
- **First-run is a real script.** Documented in `docs/deploy.md` (to be written with the `deploy/` directory): `git clone` → `cp .env.example .env && $EDITOR .env` → `docker compose -f deploy/docker-compose.yml run --rm --build seed --email ... --password ...` → `docker compose -f deploy/docker-compose.yml up -d --build`.
- **Reversibility is high.** The reverse-proxy choice is one Caddyfile + one compose service block. The container-runtime choice is one compose service definition (or a systemd unit pointing at the same static binary). The backup tool is a host-side cron line or a Litestream config. The dev/prod split is reversible at any time by adding/removing the `caddy` and `app` services from dev's compose file.