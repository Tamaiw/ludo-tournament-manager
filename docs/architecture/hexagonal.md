# Backend architecture: hexagonal (ports & adapters)

The Go backend follows hexagonal architecture (also known as ports and adapters). The application core has no I/O; concrete adapters translate between the core and the outside world (HTTP, SQLite, SMTP, WebSocket, …).

This document is the canonical reference. The ADR [`../adr/0001-hexagonal-backend.md`](../adr/0001-hexagonal-backend.md) is the short, dated record of the decision.

## Layout

```
backend/
├── go.mod, go.sum
├── cmd/server/main.go                       (composition root — the only file that imports everything)
└── internal/
    │
    ├── core/                                [CORE wrapper — domain logic + contracts]
    │   ├── domain/                          (entities, value types, pure logic; no project imports)
    │   │   ├── tournament.go, match.go, bracket.go, user.go, invite.go
    │   │   └── bracket_gen.go               (PURE single-elim algorithm)
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

## Dependency direction

```
cmd/server    →  all  (composition root only)
adapters      →  core  (implements ports, orchestrates services)
core/services →  core/ports, core/domain
core/ports    →  core/domain
core/domain   →  (nothing — no project imports)
```

The single arrow from `cmd/server` to everything is the only place in the codebase where all layers are visible at once. Inside the application, dependencies flow *inward*: adapters know about the core; the core knows nothing about adapters.

## What lives where

### `internal/core/domain/` — entities and pure logic

Plain Go structs, with methods that operate on themselves or on other domain types. No DB tags, no JSON tags, no `time.Sleep`, no `os.Getenv`. Domain errors are defined here.

```go
// internal/core/domain/tournament.go
package domain

type Tournament struct {
    ID        TournamentID
    Name      string
    Status    TournamentStatus
    CreatedAt time.Time
}

func NewTournament(name string, managerID UserID, now time.Time) (Tournament, error) {
    if strings.TrimSpace(name) == "" {
        return Tournament{}, ErrEmptyTournamentName
    }
    return Tournament{
        ID:        newTournamentID(),
        Name:      name,
        Status:    StatusDraft,
        CreatedAt: now,
        ManagerID: managerID,
    }, nil
}

func (t Tournament) OpenRegistration() (Tournament, error) {
    if t.Status != StatusDraft {
        return t, ErrTournamentNotDraft
    }
    return Tournament{...t, Status: StatusRegistrationOpen}, nil
}
```

`bracket_gen.go` lives here too — bracket shape is part of what a Tournament *is*, not an external concern:

```go
// internal/core/domain/bracket_gen.go
package domain

func GenerateSingleEliminationBracket(playerIDs []UserID, now time.Time) (Bracket, error) {
    // pure algorithm: N players → N-1 matches with parent/child edges
    // ...
}
```

### `internal/core/ports/` — interfaces

Interfaces only. No concrete implementations. Each file groups related ports by concern.

```go
// internal/core/ports/repository.go
package ports

import (
    "context"
    "ludo-tournament-manager/backend/internal/core/domain"
)

type TournamentRepository interface {
    Save(ctx context.Context, t domain.Tournament) error
    Find(ctx context.Context, id domain.TournamentID) (domain.Tournament, error)
    List(ctx context.Context) ([]domain.Tournament, error)
    // ...
}
```

Other port files: `auth.go` (PasswordHasher, SessionStore, InviteSigner), `realtime.go` (Broadcaster), `clock.go` (Clock).

### `internal/core/services/` — use case services

Each service is a struct with the ports it needs as fields, plus one or more methods (typically `Handle` taking a typed command and returning `(domain.X, error)`). Services never do I/O directly; they call ports.

```go
// internal/core/services/create_tournament.go
package services

import (
    "context"
    "ludo-tournament-manager/backend/internal/core/domain"
    "ludo-tournament-manager/backend/internal/core/ports"
)

type CreateTournament struct {
    Repo  ports.TournamentRepository
    Clock ports.Clock
}

type CreateTournamentCmd struct {
    Name      string
    ManagerID domain.UserID
}

func (c CreateTournament) Handle(ctx context.Context, cmd CreateTournamentCmd) (domain.Tournament, error) {
    t, err := domain.NewTournament(cmd.Name, cmd.ManagerID, c.Clock.Now())
    if err != nil {
        return domain.Tournament{}, err
    }
    return t, c.Repo.Save(ctx, t)
}
```

### `internal/adapters/inbound/` — driving adapters (HTTP, WebSocket)

Translate external requests (HTTP, WS upgrade) into service calls. Return rendered HTML or JSON depending on the request.

```go
// internal/adapters/inbound/http/handlers/tournament_handler.go
package handlers

import (
    "net/http"
    "ludo-tournament-manager/backend/internal/core/services"
)

type TournamentHandler struct {
    Create  services.CreateTournament
    List    services.ListTournaments
    // ...
}

func (h TournamentHandler) CreateTournament(w http.ResponseWriter, r *http.Request) {
    // parse form, build CreateTournamentCmd, call h.Create.Handle(...)
    // render template
}
```

Templates live alongside the http adapter — `embed.FS` is declared in this package:

```go
// internal/adapters/inbound/http/server.go
package http

import "embed"

//go:embed templates/*.html
var templatesFS embed.FS
```

### `internal/adapters/outbound/` — driven adapters (SQLite, SMTP, bcrypt, cookie sessions)

Implement ports. Each adapter file declares a compile-time assertion that it satisfies the relevant port:

```go
// internal/adapters/outbound/sqlite/tournament_repo.go
package sqlite

import (
    "context"
    "database/sql"
    "ludo-tournament-manager/backend/internal/core/domain"
    "ludo-tournament-manager/backend/internal/core/ports"
)

type TournamentRepo struct{ db *sql.DB }

// compile-time assertion: this struct must satisfy the port
var _ ports.TournamentRepository = (*TournamentRepo)(nil)

func (r TournamentRepo) Save(ctx context.Context, t domain.Tournament) error { /* SQL */ }
func (r TournamentRepo) Find(ctx context.Context, id domain.TournamentID) (domain.Tournament, error) { /* SQL */ }
func (r TournamentRepo) List(ctx context.Context) ([]domain.Tournament, error) { /* SQL */ }
```

The compile-time check is the rule that keeps the architecture actually hexagonal: if a port changes its signature, every adapter that needs updating fails to build until it's fixed.

### `cmd/server/main.go` — composition root

The only file that imports every layer. Reads configuration, opens DB, instantiates adapters, wires them into services, starts the server.

```go
// cmd/server/main.go
package main

func main() {
    cfg := config.Load()

    db := sqlite.Open(cfg.DBPath)
    defer db.Close()

    tournamentRepo := sqlite.TournamentRepo{db: db}
    userRepo := sqlite.UserRepo{db: db}
    passwordHasher := bcrypt.PasswordHasher{}
    emailSender := smtp.EmailSender{Host: cfg.SMTPHost, ...}
    clock := realClock{}
    broadcaster := ws.NewBroadcaster()

    createTour := services.CreateTournament{
        Repo:  tournamentRepo,
        Clock: clock,
    }
    // ... wire every service

    server := httpsrv.Server{
        CreateTournament: createTour,
        // ... every handler takes the services it needs
    }

    httpsrv.ListenAndServe(cfg.Addr, server.Routes())
}
```

## Tests

### `core/domain/` — white-box

Tests live in the same package (`package domain`). Internal access is needed to test invariants and lifecycle methods directly.

### `core/services/` — mostly black-box

Tests live in `package services_test`. They use only the exported API of services, with fake/mock implementations of the ports:

```go
// internal/core/services/create_tournament_test.go
package services_test

import (
    "context"
    "testing"
    "ludo-tournament-manager/backend/internal/core/domain"
    "ludo-tournament-manager/backend/internal/core/ports"
    "ludo-tournament-manager/backend/internal/core/services"
)

type fakeRepo struct{ saved []domain.Tournament }

func (f *fakeRepo) Save(_ context.Context, t domain.Tournament) error {
    f.saved = append(f.saved, t)
    return nil
}
func (f *fakeRepo) Find(_ context.Context, id domain.TournamentID) (domain.Tournament, error) {
    return domain.Tournament{}, nil
}
func (f *fakeRepo) List(_ context.Context) ([]domain.Tournament, error) {
    return nil, nil
}

type fakeClock struct{}

func (fakeClock) Now() time.Time { return time.Unix(0, 0) }

func TestCreateTournament(t *testing.T) {
    repo := &fakeRepo{}
    cmd := services.CreateTournament{Repo: repo, Clock: fakeClock{}}

    got, err := cmd.Handle(context.Background(), services.CreateTournamentCmd{
        Name: "Friday Night Ludo",
    })

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if got.Name != "Friday Night Ludo" {
        t.Errorf("name = %q, want %q", got.Name, "Friday Night Ludo")
    }
    if len(repo.saved) != 1 {
        t.Errorf("expected 1 saved tournament, got %d", len(repo.saved))
    }
}
```

White-box tests (`package services`) are added where they're easier than black-box for a particular case.

### `adapters/` — tests live next to the adapter

Each adapter file has a `*_test.go` next to it. Adapter tests verify that the port is correctly implemented (round-trip through the adapter) — they're integration-style, often with a real SQLite DB and a Mailpit container.

## Adding a new use case

1. If new domain types are needed, add them to `core/domain/`.
2. Add new ports to `core/ports/` if the use case needs new I/O.
3. Create `core/services/<name>.go` with the use case struct + `Handle` method.
4. Add a test in `core/services/<name>_test.go` (black-box) using a fake port.
5. Wire the use case in `cmd/server/main.go`.
6. Add an HTTP handler in `adapters/inbound/http/handlers/` if needed.
7. Add a compile-time assertion (`var _ ports.X = (*Y)(nil)`) in every adapter that satisfies a new port.

## Migration & schema: future

Migrations are not part of this architecture — they belong to whichever database adapter is chosen (SQLite in v1). The migration tool is picked separately when W3's chosen engine needs one.