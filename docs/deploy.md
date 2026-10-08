# Deployment

The Ludo Tournament Manager ships as a single Go binary plus an SQLite file,
deployed via Docker Compose. One `git clone` plus a one-shot seed command
brings up a working tournament server on a fresh VPS.

## Layout

```
deploy/
├── Dockerfile               multi-stage build (Go 1.23 → alpine:3.20)
├── docker-compose.yml       app + caddy + seed services
├── docker-compose.dev.yml    Mailpit only (run locally for email capture)
├── Caddyfile                HTTPS termination + ACME
└── scripts/
    └── backup.sh            host-side nightly SQLite backup
```

The SQLite database lives at `./data/ludo.db` on the host and bind-mounts into
the `app` container at `/data/ludo.db`. Backups land at `./backups/` and are
created by the host-side `backup.sh` script.

## First-run flow (production)

```bash
git clone git@github.com:Tamaiw/ludo-tournament-manager.git
cd ludo-tournament-manager

cp .env.example .env
$EDITOR .env                  # at minimum: SESSION_KEY, SMTP_*, PUBLIC_URL

# Generate SESSION_KEY:
head -c 32 /dev/urandom | base64

# Initialise the first manager (creates a users row; idempotent — fails
# cleanly if it already exists):
docker compose -f deploy/docker-compose.yml run --rm --build seed \
  --email manager@example.com --password 'correct horse battery staple'

# Bring the app + Caddy up:
docker compose -f deploy/docker-compose.yml up -d --build
```

Caddy auto-requests a cert via ACME on first request to `PUBLIC_URL`. The app
serves on port 8080 internally; Caddy listens on 80/443.

## Update flow

```bash
git pull
docker compose -f deploy/docker-compose.yml up -d --build
```

The `app` container is recreated; Caddy stays up. SQLite on disk is preserved.

## Dev loop

```bash
docker compose -f deploy/docker-compose.dev.yml up -d
# Mailpit SMTP at 127.0.0.1:1025, UI at http://localhost:8025

cd backend
DB_PATH=./data/ludo.db \
SESSION_KEY=$(head -c 32 /dev/urandom | base64) \
SMTP_HOST=localhost \
SMTP_PORT=1025 \
SMTP_FROM=dev@example.com \
PUBLIC_URL=http://localhost:8080 \
COOKIE_SECURE=false \
  go run ./cmd/server
```

The Go binary runs on the host; Mailpit captures outgoing email.

## Backups

A nightly snapshot of `./data/ludo.db` is taken by `deploy/scripts/backup.sh`,
configured as a host-side cron entry:

```
0 3 * * * /opt/ludo-tournament-manager/deploy/scripts/backup.sh >> /var/log/ludo-backup.log 2>&1
```

Each run:

1. Calls `sqlite3 .backup` (safe under concurrent writes).
2. Runs `PRAGMA integrity_check` on the snapshot.
3. Emits an email when the integrity check fails (set `EMAIL_ALERT_TO`).
4. Deletes snapshots older than 30 days.

### Restore

```bash
cp ./backups/ludo-2026-10-07.db ./data/ludo.db
docker compose -f deploy/docker-compose.yml restart app
```

### Disaster recovery

On a fresh box:

```bash
git clone … && cd ludo-tournament-manager
cp .env.example .env && $EDITOR .env

# Create a *new* first manager (or skip if you're restoring the old DB
# which has the real user):
docker compose -f deploy/docker-compose.yml run --rm --build seed \
  --email newadmin@example.com --password '…'

# Restore the snapshot ABOVE creating any new users:
cp ./backups/ludo-YYYY-MM-DD.db ./data/ludo.db
docker compose -f deploy/docker-compose.yml up -d --build
```

The seed command errors out if the email already exists — the desired
behaviour during recovery (seed a new admin first, then restore the snapshot
that contains the real players).

## Troubleshooting

| Symptom                                    | Cause / fix                                                                 |
|--------------------------------------------|-----------------------------------------------------------------------------|
| Sign-in submits succeed but cookie ignored | `COOKIE_SECURE=true` while serving over HTTP — flip to `false` for dev.    |
| Email never arrives                       | Check Mailpit UI at :8025 in dev. In prod, verify `SMTP_*` + `SMTP_FROM`.   |
| ACME cert never issued                     | Make sure `PUBLIC_URL` matches the hostname clients use. Caddy can't issue for localhost. |
| 500 on every request                       | `SESSION_KEY` is too short — must be ≥ 32 bytes.                             |
| Audit log page is empty                    | Audit rows are written best-effort. Check stderr logs for `db.Insert(...): …`. |
| `sqlite3: not found` from backup.sh        | `apt install sqlite3` (or distro equivalent).                                 |
| Bracket doesn't refresh                    | WebSocket events may be dropped — check Caddy config doesn't strip `Upgrade`.|