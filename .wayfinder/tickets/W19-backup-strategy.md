# W19: Backup strategy for the SQLite database

**Type:** grilling
**State:** open
**Assignee:** (unclaimed)
**Blocked by:** W3 ✓, W11 ✓
**Blocks:** (none — implementation feeds directly)

## Question

How is the SQLite database backed up on the deployment box, and how is it restored?

## Specific things to settle

- **Backup cadence (Hetzner VPS / homelab)**: nightly cron? On every write (overkill for SQLite WAL)? A sidecar like **Litestream** which streams WAL to S3 / Backblaze / local?
- **Backup destination**: local disk only, or off-host (Backblaze B2 is the obvious cheap choice, ~$0.005/GB/month)?
- **Mechanism**: `sqlite3 .backup` (online, safe under WAL), `VACUUM INTO` (compacted copy), or filesystem-level copy of the SQLite file while the app holds a lock
- **Restore procedure**: a one-command documented restore; covers both "restore yesterday's file" and "restore from 3 weeks ago via Litestream replay"
- **Retention**: how many backups, how long

## Scope tightened by W11

W11 (deployment shape) locks the SQLite file as a host bind-mount at `./data:/data`. Backups therefore run as **host-side tooling** (cron / Litestream / `restic` against `./data/ludo.db`), not as a container or another compose service. The host has the file; the `app` and `seed` services share it; the backup job is just another cron on the same box. This narrows the "mechanism" choice to host-side options and removes the `docker cp` / `docker run --rm -v ...` indirection from consideration.

## Constraints

- Engine is SQLite (resolved by W3); WAL mode is on
- Single-machine deployment
- The chosen mechanism must not require downtime; backup must not race with active writes

## What "good" looks like

A short written strategy naming the tool(s), the cadence, the destination, and the restore command(s). Lives as a resolution comment and feeds the deployment-shape (W11) and setup-guide tickets downstream.