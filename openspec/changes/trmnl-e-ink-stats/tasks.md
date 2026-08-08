## 1. Backend: summary endpoint

- [x] 1.1 Add `racing.SeasonStandings(db *sql.DB, seasonID int, limit int)` helper in `racing/stats.go` reusing the `round_snapshot_scores` + finalized `round_snapshots` aggregation, extended to select the racer name (e.g. `MAX(rss.racer_name)`) and ordered by points DESC
- [x] 1.2 Create `handlers/trmnl.go` with `GetTRMNLSummary(c *gin.Context)` returning JSON: `latest_race` (most recent finalized round snapshot by `race_date` DESC then `round` DESC, with top-10 results ordered by position), `standings` (top-8 via `SeasonStandings` for the active season, falling back to most recent season), and `season` metadata
- [x] 1.3 Handle the empty state: return HTTP 200 with `latest_race: null`, `standings: []`, `season: null` when no finalized rounds / no seasons exist
- [x] 1.4 Register `r.GET("/api/trmnl/summary", h.GetTRMNLSummary)` in `main.go` in the public (non-admin) route group

## 2. Backend: tests

- [x] 2.1 Add handler tests covering: populated response shape (latest race + standings ordering), draft-round exclusion, empty state, and the no-active-season fallback
- [x] 2.2 Verify `go test ./...` passes for the new tests

## 3. TRMNL plugin assets

- [x] 3.1 Create `trmnl/settings.yml` manifest (strategy `polling`, `polling_verb` `get`, `polling_url` with `/api/trmnl/summary`, `refresh_interval: 60`, `custom_fields.url` instance-URL field, `framework_version` matching the TRMNL convention used by the Sandwitches plugin)
- [x] 3.2 Create `trmnl/full.liquid` rendering latest race (name, track/country, date, top-3 prominent + compact remaining finishers with points) and standings table (position, name, wins, points)
- [x] 3.3 Create `trmnl/half_horizontal.liquid`, `trmnl/half_vertical.liquid`, and `trmnl/quadrant.liquid` rendering trimmed views (podium + top-3 standings) using TRMNL layout classes
- [x] 3.4 Ensure every template guards null `latest_race` / empty `standings` with Liquid conditionals rendering a "No race data yet" state
- [x] 3.5 Validate templates against a sample summary payload (render each layout with non-empty and empty data)

## 4. Docs

- [x] 4.1 Add `GET /api/trmnl/summary` row to the README API table
- [x] 4.2 Add a TRMNL section to the README (plugin directory, polling URL, install/publish notes, note that the display reflects finalized championship rounds)
- [x] 4.3 Update CHANGELOG with the new feature entry

## 5. Verification

- [x] 5.1 Run `task pre-push` (gofmt, go test, vet + govulncheck, TS compile, build) and confirm all green
