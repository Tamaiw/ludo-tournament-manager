#!/usr/bin/env bash
#
# Host-side nightly backup of the SQLite database.
# Run from cron:
#   0 3 * * * /path/to/ludo-tournament-manager/deploy/scripts/backup.sh >> /var/log/ludo-backup.log 2>&1
#
# Required environment (set in /etc/ludo-backup.env or similar):
#   DB_PATH     absolute path to the live SQLite file (e.g. ./data/ludo.db)
#   BACKUP_DIR  where snapshots land (e.g. ./backups)
#   EMAIL_ALERT_TO  optional — if set, the integrity-check failure alert is piped here
#
# Restore:
#   cp ./backups/ludo-YYYY-MM-DD.db ./data/ludo.db
#   docker compose -f deploy/docker-compose.yml restart app

set -euo pipefail

DB_PATH="${DB_PATH:-$(dirname "$0")/../../data/ludo.db}"
BACKUP_DIR="${BACKUP_DIR:-$(dirname "$0")/../../backups}"
DATE="$(date -u +%F)"
DEST="${BACKUP_DIR}/ludo-${DATE}.db"

mkdir -p "${BACKUP_DIR}"

if ! command -v sqlite3 >/dev/null 2>&1; then
  echo "$(date -Iseconds) ERROR: sqlite3 CLI not found in PATH" >&2
  exit 1
fi

if [[ ! -f "${DB_PATH}" ]]; then
  echo "$(date -Iseconds) ERROR: DB file ${DB_PATH} not found" >&2
  exit 1
fi

# Take a consistent snapshot. SQLite's .backup is safe under concurrent writes.
sqlite3 "${DB_PATH}" ".backup '${DEST}'"

# Verify integrity.
RESULT="$(sqlite3 "${DEST}" 'PRAGMA integrity_check;')"
if [[ "${RESULT}" != "ok" ]]; then
  echo "$(date -Iseconds) ERROR: integrity_check failed for ${DEST}: ${RESULT}" >&2
  if [[ -n "${EMAIL_ALERT_TO:-}" ]]; then
    printf 'Ludo backup integrity failure\n\nDB: %s\nBackup: %s\nResult: %s\n' \
      "${DB_PATH}" "${DEST}" "${RESULT}" \
      | mail -s "[ludo] backup integrity FAILED on $(hostname)" "${EMAIL_ALERT_TO}" \
      || echo "warning: mail delivery failed" >&2
  fi
  exit 1
fi

# 30-day retention.
find "${BACKUP_DIR}" -name 'ludo-*.db' -mtime +30 -delete

echo "$(date -Iseconds) backup OK: ${DEST}"