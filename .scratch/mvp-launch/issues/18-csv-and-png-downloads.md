# 18: CSV and PNG bracket downloads

**What to build:** Two download endpoints per tournament: `GET /tournaments/{id}/download.csv` returns a CSV of the bracket (one row per match, with columns: round, position, status, slot1_name, slot1_position, slot2_name, slot2_position, … up to 4 slots; the CSV is RFC 4180 compliant with proper quoting) and `GET /tournaments/{id}/download.png` returns a PNG render of the bracket (server-side: the same SVG the HTML uses, rendered via an SVG-to-PNG library; the library choice is implementation-time but must be cgo-free for cross-platform builds). Both endpoints respect the same visibility / role / spectator-token matrix as the show page (so an anonymous user can download a `public` tournament's bracket; a token holder can download a `private` tournament's; a non-manager player can download too). The `Content-Disposition: attachment; filename="<tournament-name>-bracket.{csv,png}"` header is set so browsers download rather than display.

**Blocked by:** 13 (Match page, record and mark in-progress)

**Status:** ready-for-agent

- [ ] `internal/core/services/export_bracket_csv.go` with `Handle(ctx, ExportBracketCSVCmd) (csv string, error)`: queries the bracket (matches + participants + player names), formats as RFC 4180 CSV
- [ ] `internal/core/services/export_bracket_png.go` with `Handle(ctx, ExportBracketPNGCmd) (png bytes, error)`: builds the SVG (the same template the HTML uses, just standalone), renders to PNG
- [ ] `internal/core/ports/repository.go` adds `MatchRepository.GetByTournamentWithParticipants(tournamentID)` returning a single denormalised result
- [ ] `internal/adapters/outbound/sqlite/match_repo.go` implements the above
- [ ] `internal/adapters/inbound/http/handlers/download.go` has `DownloadCSV`, `DownloadPNG` — both set `Content-Type` and `Content-Disposition` correctly
- [ ] The tournament show page has download links in the manager / spectator toolbar
- [ ] `internal/adapters/inbound/http/server.go` registers the routes with the same auth chain as the show page
- [ ] Tests at the service layer: `export_bracket_csv_test.go` covers the CSV formatting (commas, quotes, newlines in player names are escaped correctly, empty slots render as empty cells); `export_bracket_png_test.go` covers the PNG being a valid PNG (check the magic bytes) and being non-empty
- [ ] Tests at the HTTP handler layer: `GET /tournaments/{id}/download.csv` returns 200 with `Content-Type: text/csv` and the expected body for a known bracket; `download.png` returns 200 with `Content-Type: image/png`; an unauthorized request returns 401/403 per the visibility matrix
- [ ] Manual smoke test: with a tournament in progress, click "Download CSV", open in a spreadsheet; click "Download PNG", open in an image viewer; verify the bracket matches the show page
