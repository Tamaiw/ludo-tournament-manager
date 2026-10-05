# W5: Decide monorepo vs polyrepo

**Type:** grilling
**State:** resolved
**Assignee:** opencode
**Blocked by:** (none)
**Blocks:** W11, W21

## Question

One repo with `/backend` and `/frontend`, or two separate repos?

## Constraints

- Single developer
- Containerized deploy (docker compose from one repo is simpler than coordinating two)
- Setup guides that work the same on Windows + Linux
- Backend is Go (decided); frontend approach decided by W4
- The MVP scope is small; the decision is reversible but it shapes setup guides and CI

## What "good" looks like

A written choice with reasoning. If monorepo: what does the directory layout look like (`/backend`, `/frontend`, `/deploy`, `/docs`)? If polyrepo: how is the frontend wired to the backend at deploy time? The answer lives as a resolution comment on this ticket.

## Resolution

**Monorepo.** Single repo at the Git root. W4's choice of server-rendered htmx + Go `html/template` means there's no separate frontend codebase to put in a second repo.

**Layout:**

```
/                                          (Git root)
├── README.md
├── WAYFINDER.md, CONTEXT.md
├── docker-compose.yml                     (dev: app + Mailpit)
├── docker-compose.prod.yml                (prod: app only, behind reverse proxy)
├── .gitignore
├── docs/
│   ├── adr/
│   └── architecture/
├── deploy/                                (Caddy/nginx config, prod compose reference)
├── scripts/                               (seed.sh, backup.sh, restore.sh)
└── backend/
    ├── go.mod, go.sum
    ├── cmd/server/main.go                 (composition root)
    ├── internal/
    │   ├── domain/                        (per W12)
    │   ├── ports/
    │   ├── app/
    │   └── adapters/
    │       ├── http_handler/
    │       │   └── templates/             (embedded via embed.FS)
    │       └── ...
    └── ...
```

**No `/frontend` placeholder** — YAGNI; the destination is a single binary serving server-rendered HTML. If a separate frontend arrives later, we restructure then.

**Future-split fallback (user preference, soft):** if v1 turns out to need a real JS frontend in its own service, the cost of splitting is contained because the domain sits behind ports (W12): only the inbound HTTP handler adapter gets replaced by a JSON API adapter; domain, ports, app remain unchanged. Tracks as a destination guardrail — *do not let business logic leak into adapters*; the hexagonal split is what makes B-later tractable.