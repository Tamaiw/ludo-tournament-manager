# W1: Pick Go web framework

**Type:** research
**State:** open
**Assignee:** (unclaimed)
**Blocked by:** (none)
**Blocks:** W9, W10, W11, W21

## Question

Which Go web framework fits the MVP — Gin, Chi, Echo, or Fiber — and why?

## Constraints

- Cross-platform: runs identically on Linux + Windows native (no Linux-only assumptions)
- Container-friendly: small image, fast startup, easy to debug
- Mature and actively maintained as of 2026
- Good WebSocket integration (or doesn't fight it)
- Middleware ecosystem: auth helpers, CORS, session, request logging, recovery
- Comes from a Java / .NET background — easy to read, no steep idiomatic curve
- Used with a relational database (decided separately by W3)
- Will host ~100 concurrent users per tournament, low thousands total

## What "good" looks like

A clear single recommendation with reasoning, plus the rejected alternatives and *why* they were rejected. The answer lands as a research artifact (suggested filename `research/W1-go-web-frameworks.md`) and the ticket closes with a pointer to it.

## Out of scope

- Microframework-vs-stdlib net/http comparison — the project will use a framework
- Picking a router-only library (the chosen framework's router is fine)