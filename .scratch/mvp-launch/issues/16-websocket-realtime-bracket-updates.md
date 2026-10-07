# 16: WebSocket hub, per-tournament rooms, Broadcaster port, htmx-on-ws bracket updates

**What to build:** A WebSocket endpoint at `GET /ws/tournaments/{id}` upgrades the connection (after visibility/role/spectator-token checks — same auth as the show page) and joins the client to the per-tournament room. The `Broadcaster` port (declared in T01) gets four event types: `match_status_changed`, `result_recorded`, `bracket_rebuilt`, `tournament_lifecycle_changed`, all carrying `schema_version: 1`. The hub (`internal/adapters/inbound/ws/hub.go`) is a `map[TournamentID]map[*Client]struct{}` plus a publish loop. Every state-changing service call from T12–T15 (and T13 in particular) is followed by a `go broadcaster.Broadcast(ctx, event)` with a panic-recover boundary (a WS hiccup must never break the write). The show page's bracket container has `hx-ext="ws" ws-connect="/ws/tournaments/{id}" hx-trigger="ws:message from:body" hx-get="/fragments/tournaments/{id}/bracket" hx-swap="outerHTML"` so every WS message triggers a bracket fragment re-fetch. A status pill container has the same pattern against `/fragments/tournaments/{id}/status`. The integration test (`internal/adapters/inbound/ws/hub_test.go`) opens two real clients against the real hub, broadcasts an event, and asserts both receive it.

**Blocked by:** 04 (Dashboard, RequireUser middleware, base layout), 13 (Match page, record and mark in-progress)

**Status:** ready-for-agent

- [ ] `internal/core/ports/realtime.go` defines the `Broadcaster` interface (`Broadcast(ctx, Event) error`) and the `Event` interface plus the four concrete event types: `MatchStatusChanged`, `ResultRecorded`, `BracketRebuilt`, `TournamentLifecycleChanged`. All carry `SchemaVersion int` and `TournamentID`.
- [ ] `internal/adapters/inbound/ws/hub.go` implements `Broadcaster`: a `Hub` struct with `map[TournamentID]map[*Client]struct{}`, `Register`, `Unregister`, `Broadcast`. Uses a buffered channel per client to avoid blocking the publish loop.
- [ ] `internal/adapters/inbound/ws/client.go` is the per-connection read/write loop: reads heartbeats (a ping every 30s, close on missed pong); writes JSON-encoded events.
- [ ] `internal/adapters/inbound/http/handlers/ws.go` (or a dedicated file in the ws package) handles the upgrade: extracts the tournament ID from the URL, runs the same auth/visibility/spectator-token check as the show page, then registers the client with the hub
- [ ] `internal/adapters/inbound/http/server.go` registers `GET /ws/tournaments/{id}`
- [ ] Every state-changing service in T12 (start_tournament), T13 (record_match_result, mark_match_in_progress), T14 (correct_match_result), T15 (force_close, cancel) gets a follow-up `go broadcaster.Broadcast(...)` in the **handler** (not the service) — the service returns; the handler fires the broadcast. The handler wraps the broadcast in a panic-recover so a WS bug can't 500 the response.
- [ ] The tournament show page's bracket container has the `hx-ext="ws" ws-connect=…` attributes; the status pill container has the same against `/fragments/tournaments/{id}/status`
- [ ] `internal/adapters/inbound/http/handlers/bracket.go` (added in T13 but extended here) returns the bracket fragment from `/fragments/tournaments/{id}/bracket` and the status pill from `/fragments/tournaments/{id}/status`
- [ ] Tests at the integration level: `hub_test.go` opens two real `httptest.Server` clients, registers both with the hub, broadcasts an event, asserts both clients receive the JSON message within a timeout
- [ ] Tests at the service layer: each broadcast call is verified via a fake `Broadcaster` that records the events
- [ ] Tests at the HTTP handler layer: `GET /ws/tournaments/{id}` upgrades for an authorized request, returns 401/403 for an unauthorized request (same auth matrix as the show page)
- [ ] Manual smoke test: open two browser tabs on the same tournament show page; in tab 1, record a match result; tab 2 updates without a refresh
