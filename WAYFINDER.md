# Ludo Tournament Manager — Wayfinder Map

## Destination

A working MVP for a Ludo tournament manager (Go backend + web frontend), containerized for local development and Hetzner/homelab deployment, with real-time bracket updates, single-elimination format, invite-only user onboarding, and support for tournaments of up to ~100 players.

## Notes

- **Domain**: tournament management
- **Skills every session should consult**: `grilling`, `domain-modeling` (default for any design work)
- **Standing preferences**:
  - Cross-platform: must work identically on Windows + Linux dev machines
  - Container-first: deployment runs in containers (docker compose)
  - No TypeScript anywhere (user excluded it explicitly)
  - Backend language: Go
  - Single-elimination brackets only; data model flexible enough to add other formats later
  - Open spectator view (browse without account); login only for actions
  - Invite-based user onboarding: manager enters email → token-link invite → user sets password
  - First manager bootstrapped via `docker compose run seed`
  - Local-only auth (email + password, no OAuth for v1)

## Frontier

Open, unblocked tickets, ready for a session to claim. Each is the body of `.wayfinder/tickets/<id>-*.md`.

- [W1: Pick Go web framework](./.wayfinder/tickets/W1-pick-go-web-framework.md) *(research, fired)*
- [W2: Pick Go WebSocket library](./.wayfinder/tickets/W2-pick-go-websocket-library.md) *(research, fired)*
- [W3: Pick database engine](./.wayfinder/tickets/W3-pick-database-engine.md) *(research, fired)*
- [W4: Pick frontend approach](./.wayfinder/tickets/W4-pick-frontend-approach.md) *(research, fired)*
- [W5: Decide monorepo vs polyrepo](./.wayfinder/tickets/W5-decide-monorepo-vs-polyrepo.md) *(grilling)*
- [W6: Sketch core domain model](./.wayfinder/tickets/W6-sketch-core-domain-model.md) *(grilling)*
- [W8: Transactional email approach](./.wayfinder/tickets/W8-transactional-email-approach.md) *(research, fired)*

## Blocked

Open tickets, waiting on the frontier. Discovered by the dependency graph; close after resolve.

- [W7: User roles + permissions matrix](./.wayfinder/tickets/W7-user-roles-permissions-matrix.md) *(blocked by W6)*

## Decisions so far

<!-- the index: one line per closed ticket, enough to judge relevance, then zoom the link for the detail the ticket holds -->

_(none yet — this is the first chart)_

## Not yet specified

<!-- see "Fog of war": in-scope fog you can't ticket yet; graduates as the frontier advances -->

- **Tournament lifecycle states**: how do we model the lifecycle (draft → registration_open → in_progress → completed → cancelled)? What transitions are allowed and who triggers them?
- **Match result recording**: who is allowed to record a match outcome — only the manager, any user, or the two players themselves with mutual confirmation?
- **Player withdrawal mid-tournament**: how does the bracket behave when a registered player drops out after the bracket is generated?
- **Real-time presence model**: do we surface "who is currently viewing this tournament", or is real-time used only for bracket state changes?
- **Public tournament discovery**: how are tournaments listed/found (browse all, search, filter by status, filter by date)?
- **Tournament audit trail**: is there a record of who did what when on a tournament (manager edits, match results, registrations)?
- **Bracket render strategy**: how is the bracket drawn on screen — DOM / SVG / canvas, client-side or server-rendered?
- **Tournament time zones**: how are scheduled times displayed when the organizer and players are in different zones?
- **Notifications**: do players/managers get email when a tournament starts, when their match is ready, when the bracket advances?

## Out of scope

<!-- see "Out of scope": work ruled beyond the destination; closed, never graduates -->

- Mobile-native apps — web frontend only
- Public cloud deployment (AWS/GCP/Azure) — local + Hetzner only
- Multi-format brackets beyond single elimination — single elimination only for v1
- OAuth / social login — email + password only for v1
- Account self-service creation — invites are manager-only by destination design