# W12: Apply hexagonal architecture to backend

**Type:** grilling
**State:** resolved
**Assignee:** opencode
**Blocked by:** (none)
**Blocks:** W6, W9, W10, W11

## Question

Apply hexagonal (ports & adapters) architecture to the Go backend: lock down the layer boundaries, the directory layout under `backend/`, and which concerns are interfaces (ports) vs in-process helpers.

## Resolution

**Architecture chosen:** hexagonal (ports & adapters). Application core under `internal/core/{domain,ports,services}`; adapters under `internal/adapters/{inbound,outbound}`; composition root at `cmd/server/main.go`.

**Layout** (locked):

```
backend/
├── go.mod, go.sum
├── cmd/server/main.go                       (composition root — the only file that imports everything)
└── internal/
    │
    ├── core/                                [CORE wrapper — domain logic + contracts]
    │   ├── domain/                          (entities, value types, pure logic; no project imports)
    │   │   ├── tournament.go, match.go, bracket.go, user.go, invite.go
    │   │   └── bracket_gen.go               (PURE single-elim algorithm — domain logic, not a port)
    │   ├── ports/                           (interfaces only; depends on domain only)
    │   │   ├── repository.go                (TournamentRepo, MatchRepo, UserRepo, InviteRepo, SessionRepo)
    │   │   ├── auth.go                      (PasswordHasher, SessionStore, InviteSigner)
    │   │   ├── realtime.go                  (Broadcaster)
    │   │   └── clock.go                     (Clock — for testability)
    │   └── services/                        (use case services; depends on ports + domain)
    │       ├── create_tournament.go
    │       ├── register_player.go
    │       ├── record_match_result.go
    │       ├── invite_user.go
    │       ├── accept_invite.go
    │       ├── sign_in.go
    │       └── sign_out.go
    │
    └── adapters/                            [ADAPTERS — implement the ports]
        ├── inbound/                         (driving adapters)
        │   ├── http/
        │   │   ├── server.go                (Chi setup, route registration)
        │   │   ├── handlers/                (per-resource)
        │   │   ├── middleware/              (logging, request id, auth)
        │   │   └── templates/               (embedded via embed.FS)
        │   └── ws/                          (coder/websocket hub — implements Broadcaster)
        └── outbound/                        (driven adapters)
            ├── sqlite/                      (one repo file per entity)
            ├── smtp/                        (Resend / Mailpit)
            ├── bcrypt/
            └── session/                     (cookie sessions)
```

**Dependency direction** (the rule that makes it hexagonal):

```
cmd/server    →  all  (composition root only)
adapters      →  core  (implements ports, orchestrates services)
core/services →  core/ports, core/domain
core/ports    →  core/domain
core/domain   →  (nothing — no project imports)
```

**Specific decisions locked (Q1–Q5):**
- **Layout** — `core/` wrapper + `services/` naming adopted; brackets generator in `core/domain/` (domain logic, not a port)
- **Templates** — `internal/adapters/inbound/http/templates/`; `embed.FS` declared in the http package
- **Tests** — white-box for `core/domain/` (internal access for invariants); mostly black-box (`package services_test`) for `core/services/` (test through the public method); white-box where invariants are easier to set up
- **Logging** — at the HTTP boundary via middleware in `adapters/inbound/http/middleware/`; domain stays pure

**Compile-time check** (the rule that makes the architecture actually hexagonal — build breaks if an adapter drifts from its port):

```go
// In every adapter file:
var _ ports.TournamentRepository = (*sqlite.TournamentRepo)(nil)
```

**Full architecture writeup:** [`docs/architecture/hexagonal.md`](../../docs/architecture/hexagonal.md).

**ADR:** [`docs/adr/0001-hexagonal-backend.md`](../../docs/adr/0001-hexagonal-backend.md).