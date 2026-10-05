# W11: Decide deployment shape

**Type:** grilling
**State:** open
**Assignee:** (unclaimed)
**Blocked by:** W5 ✓, W1 ✓, W2 ✓, W3 ✓, W4 ✓, W12
**Blocks:** W21

## Question

How is the system deployed — both locally for dev and to the Hetzner VPS / homelab box for prod — and how does local mirror prod closely enough that the "works on my machine" gap is closed?

## Specific things to settle

- **Local dev shape**: single `docker compose` (Go app + Mailpit) with a SQLite file on a bind-mount, OR `docker compose` + a local Go toolchain running the app outside the container
- **Production shape**: same `docker compose` on the VPS, OR a systemd unit running the binary directly with a reverse proxy (Caddy / nginx / Traefik) terminating HTTPS
- **HTTPS termination**: Caddy is the obvious default (auto-cert via ACME), but the choice matters for setup guides
- **Container build strategy**: single static binary (scratch or alpine), with `html/template` files embedded via `embed.FS`
- **Volume strategy for SQLite**: bind-mount on the host for backup simplicity, OR `docker volume` + `docker cp` for backup
- **First-time setup flow**: `git clone` → `docker compose run seed` → creates the first manager account; what env vars are required at first run
- **Update flow**: `git pull` + `docker compose up -d --build` on the VPS, OR an Ansible playbook / helper script

## Constraints

- W5 (monorepo vs polyrepo) is unresolved — this ticket runs after W5 lands
- Stack is locked: Go + Chi + `coder/websocket` + SQLite (modernc) + htmx + Alpine + Resend SMTP + Mailpit
- Deployment target is a single VPS / homelab box — no k8s, no multi-host
- Fresh-computer setup guides are a destination requirement; the deployment shape chosen here directly drives those guides

## What "good" looks like

A written deployment shape with a sketched file layout, plus reasoning for each choice. Lives as a resolution comment and a directory sketch under `deploy/` once decided.