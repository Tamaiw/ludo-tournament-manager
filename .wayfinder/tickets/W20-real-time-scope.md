# W20: Decide real-time scope

**Type:** grilling
**State:** resolved
**Assignee:** tamas
**Blocked by:** (none)
**Blocks:** (none)

## Question

What exactly does the WebSocket push to clients?

## Resolution

**Scope locked: state-only for v1.** The WebSocket pushes bracket state changes (match status transitions, recorded results, bracket rebuilds, tournament lifecycle transitions) and nothing else. Spectators and players see the bracket update live; no presence, no per-match claim lock, no viewer count.

### Event set (closed; v1)

| Event | Trigger | Client effect |
|---|---|---|
| `match_status_changed` | `pending → in_progress → finalised` (and manager-edit reverts) | Re-fetch bracket fragment |
| `result_recorded` | A match outcome is committed | Re-fetch bracket fragment (and standings if separate) |
| `bracket_rebuilt` | Seeds changed, player removed, bracket regenerated | Re-fetch bracket fragment |
| `tournament_lifecycle_changed` | `draft → registration_open → in_progress → completed` (also `cancelled`) | Re-fetch tournament header / status pill |

No chat, no reactions, no notifications-in-app, no per-match claim, no viewer count. Out of v1 scope per the destination.

### Wire format (server → client)

Every event carries a `schema_version` field (currently `1`) so a future mobile client can branch on `(schema_version, type)` instead of guessing. JSON, snake_case fields.

```json
{ "schema_version": 1, "type": "match_status_changed", "tournament_id": "T123", "match_id": "M456" }
```

The set of `type` strings is a closed enum defined in `core/ports/realtime.go`. New types can be introduced in `v1` (additive); breaking changes bump `schema_version` to `2`.

### Client pattern (htmx on the bracket page)

```html
<div hx-ext="ws" ws-connect="/ws/tournaments/T123"
     hx-trigger="ws:message from:body"
     hx-get="/fragments/tournaments/T123/bracket"
     hx-swap="outerHTML">
  <!-- bracket fragment rendered server-side from current state -->
</div>
```

The client treats every WS message as "re-fetch the bracket fragment"; the server decides what the fragment looks like based on current state. This is the idiomatic htmx-on-WebSocket pattern and keeps the client trivial.

### Architecture seams (W12)

- `core/ports/realtime.go` defines the `Broadcaster` port: `Broadcast(ctx context.Context, evt Event) error`. Events are a tagged union (one Go type per closed-state event above); the WS adapter marshals them.
- `adapters/inbound/ws/hub.go` implements `Broadcaster` as a per-tournament-room hub: a `map[TournamentID]map[*Client]struct{}` plus a publish loop. Each connected client is identified by a random token so the same user with two tabs gets two entries (correct, for connection counting when we ever need it).
- `adapters/inbound/http/handlers/` call `broadcaster.Broadcast(ctx, MatchStatusChanged{...})` after each state-changing write. The handler does NOT wait for or depend on the broadcast — it's a fire-and-forget `go` goroutine with a panic-recover boundary so a WS hiccup never breaks the write.

### Per-tournament rooms, not global

Each WS connection is scoped to one tournament: `ws://host/ws/tournaments/{id}`. Events are routed only to clients in that tournament's room. A separate "lobby" channel (tournament lifecycle changes across the whole system) is not in v1 scope; if it becomes useful in v2 it adds one more room, not a refactor.

### Mobile-app replaceability

The state-only shape is mobile-friendly in two ways:

- **WebSocket events are already JSON and tagged**, so a mobile client can subscribe and parse them directly. No binary protocol, no hidden state.
- **The HTTP read endpoints** that htmx currently hits return HTML fragments. A mobile client wants JSON. Adding JSON read handlers in v2 is a new inbound adapter; the domain services and outbound adapters are unchanged. Hexagonal architecture (W12) makes this an additive change, not a rewrite.

Two decisions in this resolution specifically support that future:

1. **Closed enum of event types** (no stringly-typed fields). Mobile can write `switch (evt.type) { ... }` against a stable, versioned set.
2. **`schema_version` on every event.** Mobile can branch on it; breaking changes don't require a coordinated deploy.

A v2 mobile app would also add a `Bearer <session-token>` middleware that reads the same `scs/v2` session store as the browser — one new middleware, no change to sessions.

### Considered options

- **State + presence** (viewer count, per-match claim): rejected for v1. Per-match edit collisions are solvable at the write layer (audit log + downstream-lock rule from W7); no WebSocket presence needed for it. Both features are additively addable in v2 by introducing new `type` strings — clients written against v1 still work.
- **Server-Sent Events instead of WebSockets**: rejected. SSE is one-way (server → client) which fits state-only, but htmx's `hx-ext="ws"` pattern is well-trodden and matches `coder/websocket` (W2) which is already locked. SSE would force a different client-side wiring for no win.
- **Full state dump on every event** (instead of event-typed messages): rejected. Sends more bytes per event; client doesn't know which fragment to swap in. Typed events + per-fragment re-fetch is more bandwidth-efficient and lets the server evolve the message shape without coordinating with the client.
- **Long-polling instead of WebSockets**: rejected. WS is locked (W2); long-polling would burn more connections and not buy anything for this workload.
- **One global WS (events for every tournament, client filters)**: rejected. A per-tournament room lets the server skip fan-out for tournaments nobody is viewing and keeps the connection's permission scope tight (the WS upgrade handler enforces the same visibility/auth as the bracket HTML).

### Effects on adjacent tickets

- `core/ports/realtime.go` (from W12) gets the closed enum of event types and the `schema_version` field.
- `adapters/inbound/ws/hub.go` is new — implements the Broadcaster port. Skeleton stubbed during W12 implementation.
- No ADRs (this is a scope decision, not a hard-to-reverse architectural pick; the closed event enum + schema_version keep it cheap to revise).
- Nothing else on the wayfinder map blocks implementation. With W11, W19, and W20 closed, **the wayfinder map is empty of frontier work** and the destination is in reach.