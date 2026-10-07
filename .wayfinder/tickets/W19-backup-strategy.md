# W19: Backup strategy for the SQLite database

**Type:** grilling
**State:** resolved
**Assignee:** tamas
**Blocked by:** W3 ✓, W11 ✓
**Blocks:** (none — implementation feeds directly)

## Question

How is the SQLite database backed up on the deployment box, and how is it restored?

## Resolution

**Strategy locked: Package A — Simple local nightly.** One host cron, one bash script, one restore command. ~25 lines of bash in total.

### Mechanism

`sqlite3 .backup` against the live database. The online backup API acquires a shared lock briefly, copies the main DB + committed WAL frames into a consistent snapshot file, then releases. Safe under WAL mode; does not block writers; produces a single self-contained file that opens with no extra tooling.

### Cadence

Nightly at **03:00** (host local time). One file per day; if a backup runs twice in one calendar day, the second overwrites the first — no rotation needed for the same-day case.

### Destination

`./backups/` at the repo root on the VPS (e.g. `/opt/ludo-tournament-manager/backups/`). Gitignored (alongside `data/` and `.env`). The directory lives next to `data/` so the restore command is a single relative `cp`.

### File naming

`ludo-YYYY-MM-DD.db` (e.g. `ludo-2026-10-06.db`). ISO date makes `ls` order chronologically and the restore command trivial.

### Verification

`PRAGMA integrity_check;` runs against the new backup file immediately after `.backup`. Catches a corrupt DB at backup time rather than at restore time. Cost is ~10 ms on a SQLite file of this size.

### Retention

Keep 30 days; prune older. Single `find ... -mtime +30 -delete` line in the same script.

### Script

`deploy/scripts/backup.sh` ships in the repo. Runnable by hand for a manual snapshot. The host cron entry is the only thing to add on a fresh VPS.

```bash
#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="${1:-$(pwd)}"
DB="$REPO_ROOT/data/ludo.db"
BACKUP_DIR="$REPO_ROOT/backups"
DATE="$(date +%F)"

mkdir -p "$BACKUP_DIR"
sqlite3 "$DB" ".backup '$BACKUP_DIR/ludo-$DATE.db'"
sqlite3 "$BACKUP_DIR/ludo-$DATE.db" "PRAGMA integrity_check;"

find "$BACKUP_DIR" -name 'ludo-*.db' -mtime +30 -delete
```

```cron
0 3 * * * /opt/ludo-tournament-manager/deploy/scripts/backup.sh /opt/ludo-tournament-manager
```

### Restore

```bash
# 1. Pick a backup file
ls -1 backups/ludo-*.db

# 2. Restore (overwrites the live DB; do this with the app stopped)
docker compose -f deploy/docker-compose.yml stop app
cp backups/ludo-2026-10-06.db data/ludo.db
docker compose -f deploy/docker-compose.yml up -d
```

Documented in `docs/deploy.md` (to be written with the deploy/ directory).

### Reversibility

The whole stack sits outside the application binary. Switching to Package B (restic + Backblaze B2) or Package D (Litestream) is a swap of `deploy/scripts/backup.sh` + the cron entry, with no application-code change. The seed binary from W11 stays as the disaster-recovery path of last resort: `git clone` on a fresh box + the seed binary + the latest backup restores the system.

### Considered options

- **Package B (restic + Backblaze B2)**: rejected for v1 — off-host + encryption + dedup are real wins, but the user opted for simpler local-first on the basis that off-host can be added later without rewriting the application. Noted in the ticket as the likely next step when VPS-die risk becomes salient.
- **Package D (Litestream + B2/S3)**: rejected — near-zero RPO is nice but the daemon + systemd unit + B2 account are more moving parts than this deployment currently warrants.
- **`VACUUM INTO` as the mechanism**: rejected — `VACUUM` rewrites the entire database and is write-blocking for the duration; on a live tournament app this is unacceptable. `.backup` produces a comparable file without the lock.
- **Filesystem-level copy of `data/ludo.db`**: rejected — without `sqlite3 .backup`, the file may be in the middle of a WAL checkpoint and inconsistent. `.backup` is the safe path; the script's `PRAGMA integrity_check` proves it.

### Effects on adjacent tickets

- `docs/deploy.md` (to be created) needs a **Restore from backup** section that mirrors the command block above.
- `data/`, `backups/`, `.env` need a `.gitignore` entry. The first two already need gitignore for W11; `.env` likewise; `backups/` is added now.
- Nothing else on the wayfinder map blocks on W19. Implementation can begin once W20 closes.