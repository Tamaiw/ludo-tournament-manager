# W8: Transactional email approach

**Type:** research
**State:** resolved
**Assignee:** opencode-research
**Blocked by:** (none)
**Blocks:** (none)

## Question

How does the system send transactional email (invite emails, password resets, future notifications)? Pick one of:

- **Direct SMTP** from the Go app using `net/smtp` or a Go SMTP library, talking to a third-party SMTP relay
- **HTTP API** to a transactional-email provider (Mailgun, Postmark, SendGrid, Brevo, Resend)
- **Self-hosted** (Mailhog for dev, real SMTP relay in prod) — usually overkill for v1

## Constraints

- Cross-platform dev (Docker-friendly, no Linux-only SMTP dependencies)
- Hetzner VPS IPs have neutral reputation that needs warming for direct SMTP — direct-SMTP-on-Hetzner is a footgun
- Must reliably deliver invite emails (the system depends on this for user onboarding)
- Cost: free tier or very low cost preferred for v1
- Simplicity: the chosen path should be ~one integration, not a saga

## Resolution

**Chosen approach: Resend (shared-IP SMTP relay) for production; Mailpit (Docker container) for local development.** Both wired through Go's stdlib `net/smtp`, so dev/prod parity is a config flip — same code path, different `MAIL_HOST`.

**Why:** Resend's free tier (3,000 emails/month, 100/day cap) covers the entire invite flow through v1 without a paid plan, the integration is one credential (`RESEND_API_KEY`), and Resend's shared IPs are pre-warmed — sidestepping the Hetzner-IP-reputation footgun entirely. Mailpit is the only actively-maintained local SMTP capture server in 2026 (v1.31.4, 2026-10-03) and runs in a single Docker container with a web UI and REST API.

Detailed comparison, alternatives considered (direct SMTP, Mailgun, Postmark, Brevo, SendGrid) and source links: `research/W8-transactional-email.md` on branch `research/W8`.

## What "good" looks like

A clear single recommendation with reasoning. Cover: dev experience (how is email tested locally), prod delivery (provider choice, why), and how the chosen approach is wired into the Go backend. Suggested filename `research/W8-transactional-email.md`.