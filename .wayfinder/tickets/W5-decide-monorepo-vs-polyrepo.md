# W5: Decide monorepo vs polyrepo

**Type:** grilling
**State:** open
**Assignee:** (unclaimed)
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