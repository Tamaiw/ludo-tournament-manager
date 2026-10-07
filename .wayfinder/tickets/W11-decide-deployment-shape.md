# W11: Decide deployment shape

**Type:** grilling
**State:** resolved
**Assignee:** tamas
**Blocked by:** W5 ✓, W1 ✓, W2 ✓, W3 ✓, W4 ✓, W12 ✓
**Blocks:** (W21 ✓ — dev setup assumed this shape and closed ahead of W11)

## Question

How is the system deployed — both locally for dev and to the Hetzner VPS / homelab box for prod — and how does local mirror prod closely enough that the "works on my machine" gap is closed?

## Resolution

**Shape locked:** `docker compose` in both local dev and prod, with **Go on the host for dev** (compose hosts Mailpit only) and the full app stack (app + Caddy) in compose for prod. Single static binary in an `alpine` image, `embed.FS` for templates, `cmd/seed` for first-time bootstrap, bind-mounted SQLite on the host.

### File layout (locked)

```
deploy/
├── Dockerfile                          (multi-stage: Go builder → alpine runtime; single static binary)
├── Caddyfile                           (reverse_proxy app:8080, auto-TLS via ACME, HTTP→HTTPS by default)
├── docker-compose.yml                  (PROD: app + caddy; bind-mounts ./data:/data and ./deploy/Caddyfile:/etc/caddy/Caddyfile)
└── docker-compose.dev.yml              (DEV: mailpit only; no app, no caddy)

data/                                    (bind-mount target; gitignored; holds ludo.db + ludo.db-wal + ludo.db-shm)
.env                                     (at repo root; gitignored; .env.example ships with placeholders)
backend/cmd/seed/main.go                 (bootstrap binary; invoked once for first-time setup + re-runnable for DR)
```

### Specific decisions (Q1–Q7)

1. **Local dev shape — split (compose for state, Go on host).** Compose runs Mailpit only; `go run ./cmd/server` runs the app on the host. Fastest edit/test loop; cross-platform because native Go + Docker work the same on Windows + Linux. The "works on my machine" gap is closed by **artefact identity** (`deploy/Dockerfile` and `.env` schema are identical locally and in prod), not runtime identity.
2. **Production shape — `docker compose` on the VPS.** App + Caddy in a single compose file. Same compose vocabulary as dev; only env vars and bind-mounted paths differ. Matches the "container-first deployment" standing preference.
3. **HTTPS termination — Caddy.** Auto-cert via ACME; single `Caddyfile`; HTTP→HTTPS by default. Dev uses HTTP only (no Caddy container locally); the Caddyfile is therefore exercised in prod only, with iteration via `git pull && docker compose restart caddy`.
4. **Container build strategy — single static binary in `alpine` (~10 MB).** Multi-stage `Dockerfile` produces a CGO-disabled binary that includes `html/template` files via `//go:embed`. Alpine over `scratch` so `docker exec` debugging (permissions check, `wget` against `localhost`) works without adding a second image; alpine over `distroless/static` so the shell stays reachable when something goes wrong in prod.
5. **Volume strategy for SQLite — bind-mount `./data:/data`.** The SQLite file is a real file on the host, so backups run as host-side tooling (cron / `sqlite3 .backup` / Litestream) without `docker cp` indirection. Cross-platform via Docker Desktop's path mapping on Windows. Tightens W19: the backup job is a host cron, not another compose service.
6. **First-time setup flow — `cmd/seed/main.go` invoked via `docker compose -f deploy/docker-compose.yml run --rm --build seed --email <email> --password <password>`.** The compose file declares both `app` and `seed` services sharing image + env + bind-mount. Email + password come from CLI args (or `.env` fallback), not a wizard — keeps the seed binary stateless and re-runnable for disaster recovery.
7. **Update flow — `git pull && docker compose -f deploy/docker-compose.yml up -d --build`.** Two commands. No Makefile, no Ansible, no helper script — they're one layer of indirection too many for two commands. Documented in `docs/deploy.md` (to be added with the deploy/ directory).

### Operational commands

```bash
# Local dev loop
docker compose -f deploy/docker-compose.dev.yml up -d   # Mailpit
cd backend && go run ./cmd/server                       # Go on host

# First-time prod bootstrap (once)
git clone <repo> && cd ludo-tournament-manager
cp .env.example .env && $EDITOR .env                    # Resend token, public domain, admin email
docker compose -f deploy/docker-compose.yml run --rm --build seed --email <email> --password <password>
docker compose -f deploy/docker-compose.yml up -d --build

# Prod update (after every release)
git pull
docker compose -f deploy/docker-compose.yml up -d --build
```

### Reversibility

Every decision above sits at the deployment boundary, not in the application code. The hexagonal split (W12) keeps Go, SQL, SMTP, and WebSocket behind ports, so the following swaps are non-breaking to application code:

- **Reverse proxy**: Caddy → Traefik or nginx means rewriting `deploy/Caddyfile` and the `caddy` service block. Application untouched.
- **Container runtime**: compose → `systemd` means replacing the `app` service definition with a systemd unit; the binary is already static and self-contained.
- **Backup tool**: cron → Litestream is a host-side swap. W19 decides.
- **CI/CD / Ansible / Tailscale**: added as a layer above compose (`make deploy`, GitHub Actions SSH step, etc.). Zero application-code change.

### Effects on adjacent tickets

- **W19 (backup strategy)** scope tightens: backup is a host-side cron or Litestream against `./data/ludo.db`, not a container. The chosen mechanism still has to be settled (which sidecar, what cadence, what retention).
- **W21 (developer setup)** was already resolved and assumed this shape (compose for Mailpit + Go on host). No retroactive change needed.
- **Implementation can begin** once W19 + W20 (real-time scope) close; nothing else on the wayfinder map blocks building.

### ADR

[`docs/adr/0004-deployment-shape.md`](../../docs/adr/0004-deployment-shape.md).