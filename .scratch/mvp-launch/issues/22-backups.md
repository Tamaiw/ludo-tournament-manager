# 22: Backups (host-side nightly, integrity check, restore doc)

**What to build:** A host-side backup script at `deploy/scripts/backup.sh` that the operator installs as a nightly cron job. The script: (1) calls `sqlite3 ./data/ludo.db ".backup ./backups/ludo-$(date +%F).db"` to take a consistent snapshot (SQLite's `.backup` is safe under concurrent writes); (2) runs `sqlite3 ./backups/ludo-$(date +%F).db "PRAGMA integrity_check;"` and aborts (with an email alert via the configured SMTP credentials, or just stdout in dev) if the result is not `ok`; (3) deletes backups older than 30 days with `find ./backups -name 'ludo-*.db' -mtime +30 -delete`. The script is ~25 lines of bash, no Python, no extra dependencies beyond the `sqlite3` CLI (already on the operator's machine per the setup doc). The restore procedure is documented in `docs/deploy.md`: `cp ./backups/ludo-YYYY-MM-DD.db ./data/ludo.db && docker compose -f deploy/docker-compose.yml restart app`. The cron line is documented too: `0 3 * * * /path/to/ludo-tournament-manager/deploy/scripts/backup.sh >> /var/log/ludo-backup.log 2>&1` (run at 03:00 daily). The script is upgradable to restic + B2 or Litestream by swapping the script and the cron line — no application-code change (per W19).

**Blocked by:** 21 (Production deployment)

**Status:** ready-for-agent

- [ ] `deploy/scripts/backup.sh`: a bash script with `set -euo pipefail`, the three steps above, an `EMAIL_ALERT_TO` env var (optional) that pipes the integrity-check failure to `mail` or `sendmail` (or to the SMTP relay via a one-liner)
- [ ] `docs/deploy.md` adds a "Backups" section: install the cron, verify the script runs, restore procedure
- [ ] `docs/deploy.md` adds a "Disaster recovery" section: `git clone` on a fresh box → `cp .env.example .env && $EDITOR .env` → `docker compose -f deploy/docker-compose.yml run --rm --build seed --email ... --password ...` (re-running the seed on a fresh DB is fine — it errors out if the user already exists, but that's the desired behaviour during recovery: seed a new first manager, then restore the data file with the real users)
- [ ] A `Makefile` target at the repo root (optional): `make backup` (runs the script locally) and `make restore DATE=YYYY-MM-DD` (does the `cp` + restart) — keep them as thin wrappers over the script
- [ ] Manual test: run the script against the dev SQLite file at `./data/ludo.db`; verify a backup file appears in `./backups/`; verify the integrity check passes; verify old backups get cleaned up after 30 days (use `touch -d` to fake the mtime for the test)
- [ ] Verification: the script is idempotent (running it twice in a row produces two snapshot files with the same content modulo timestamps); a corrupt backup is detected and the operator is alerted
- [ ] Documentation note: explain the upgrade path to restic + B2 or Litestream — what to change in the script and the cron line, what stays the same
