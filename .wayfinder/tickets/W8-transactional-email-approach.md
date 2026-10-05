# W8: Transactional email approach

**Type:** research
**State:** open
**Assignee:** (unclaimed)
**Blocked by:** (none)
**Blocks:** W6

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

## What "good" looks like

A clear single recommendation with reasoning. Cover: dev experience (how is email tested locally), prod delivery (provider choice, why), and how the chosen approach is wired into the Go backend. Suggested filename `research/W8-transactional-email.md`.