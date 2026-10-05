# W1: Pick Go web framework

**Type:** research
**State:** resolved
**Assignee:** opencode-research
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

## Resolution

**Chosen framework: Chi v5** (`github.com/go-chi/chi/v5`).

**Why, in one line:** stdlib-shaped handlers (`func(http.ResponseWriter, *http.Request)`) and plain `net/http` middleware — closest to a Servlet/Spring mental model — on top of the smallest dependency tree, with three releases and a coordinated CVE hardening (v5.3.0, 2026-05-22) shipping in 2026.

Detailed comparison, rejected alternatives (Gin, Echo, Fiber) and source links: `research/W1-go-web-frameworks.md` on branch `research/W1`.