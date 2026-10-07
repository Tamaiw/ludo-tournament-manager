# 14: Manager correct (downstream-lock check)

**What to build:** A manager can correct a `completed` match via `POST /matches/{id}/correct`. The service: (1) validates the actor is a manager of the match's tournament; (2) checks the **downstream lock**: walks the `next_match_id` chain transitively; if ANY downstream match is `in_progress` or `completed`, the correction is rejected with `ErrDownstreamLocked`; (3) if the lock is clear, applies the new positions, recomputes advancers, walks back downstream to update the `match_participant` rows in the next match (replacing the previously-advanced players with the corrected advancers), and writes an audit log row (`match_result_corrected` with `before` and `after` snapshots). The form reuses the `match_record.html` template (or a thin variant). Players cannot correct a `completed` match (the matrix blocks this; the handler enforces it). The bracket fragment reflects the correction immediately (T16 will wire the WS push; for now, a page reload shows the corrected bracket).

**Blocked by:** 13 (Match page, record and mark in-progress)

**Status:** ready-for-agent

- [ ] `internal/core/services/correct_match_result.go` with `Handle(ctx, CorrectMatchResultCmd) (Match, error)`: orchestrates the downstream-lock check + correction + audit log. Reuses the validation logic from `record_match_result.go` (extract a shared `validateAdvancingPositions` helper if needed).
- [ ] `internal/core/ports/repository.go` adds `MatchRepository.GetDownstreamChain(matchID)` (recursive CTE: returns all matches transitively downstream)
- [ ] `internal/adapters/outbound/sqlite/match_repo.go` implements `GetDownstreamChain` via a recursive CTE
- [ ] `internal/adapters/inbound/http/handlers/match.go` adds `CorrectResultSubmit` (reuses the record form template, with a "correcting" flag in context)
- [ ] `internal/adapters/inbound/http/templates/pages/match_correct.html` (or reuse `match_record.html` with a different submit button label and a banner "Correcting a completed match")
- [ ] `internal/adapters/inbound/http/server.go` registers the route; the match_show page has a manager-only "Correct result" button (visible only when the match is `completed` and no downstream is `in_progress` or `completed`)
- [ ] Tests at the service layer: `correct_match_result_test.go` covers: happy path (round-1 match corrected, round-2 participants updated, audit log written); downstream lock — round-1 correction blocked when round-2 is `in_progress`; deep downstream lock — round-1 correction blocked when the final is `completed`; non-manager authorization (returns 403-equivalent error)
- [ ] Tests at the HTTP handler layer: `POST /matches/{id}/correct` as manager with clear downstream returns 302 to the match page; as manager with a locked downstream returns 409 with the downstream-match link; as a non-manager returns 403
- [ ] Manual smoke test: complete a round-1 match in an 8-player tournament; correct the result (swap 1st and 2nd); verify the round-2 participants are updated; complete round-2; attempt to correct round-1 again — blocked because round-3 (the final) is `ready`; mark round-3 in-progress; attempt the correction again — still blocked
