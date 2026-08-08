## Context

Heat is a self-hosted "Heat: Pedal to the Metal" companion app (Go/gin + SQLite + HTMX). It exposes a rich, mostly public read-only JSON API under `/api/*` (race history, racer stats, points progression, etc.). The sister project Sandwitches ships an official TRMNL e-ink plugin (recipe 247547): a `trmnl/` directory with four Liquid templates and a `settings.yml` manifest that polls public JSON endpoints via TRMNL's polling strategy. Heat has no TRMNL presence; this change adds one that surfaces the championship state (latest race + season standings) on a TRMNL e-ink display.

Data model facts established during exploration:

- `seasons` — `(id, name, start_date, end_date, status)` where `status` is `'active'` or `'archived'`.
- `round_snapshots` — `(id, season_id, race_name, race_date, round, created_at, status)` where `status` is `'draft'` or `'final'`.
- `round_snapshot_scores` — `(snapshot_id, racer_id, racer_name, points, position, dnf, dns, spins, overheated)`. Stores `racer_name` denormalized, ordered by `position`.
- `race_history` / `race_results` — the manually-saved race results archive (used by `/api/race-history`).
- Standings today: `racing.RacerStatsBySeason(db, seasonID)` aggregates `round_snapshot_scores` joined to finalized `round_snapshots`, ordered by points DESC. It returns `models.RacerStats` keyed by `racer_id` (no name).
- Existing read-only GET endpoints (`/api/race-history`, `/api/stats/*`, `/api/seasons`, ...) are **public** — only admin/mutation routes carry `middleware.AuthMiddleware`.

## Goals / Non-Goals

**Goals:**

- Add a single public JSON endpoint `GET /api/trmnl/summary` that returns a compact, TRMNL-shaped payload: latest race (finishing order with points) + current season standings.
- Add a `trmnl/` plugin directory (four Liquid templates + `settings.yml`) mirroring the Sandwitches integration, so the same change can be published as an official TRMNL recipe.
- Keep the endpoint unauthenticated, read-only, and consistent with the existing public stats API.

**Non-Goals:**

- NOT re-introducing the removed web e-ink theme (that was a browser CSS theme; this is a TRMNL device plugin).
- No authentication/OAuth for the endpoint (matches existing public GET endpoints; data is already public).
- No new database tables or schema changes.
- No changes to existing endpoints or payloads (fully additive).

## Decisions

### D1: One dedicated endpoint instead of reusing existing endpoints

TRMNL plugins using the polling strategy can poll multiple URLs (Sandwitches polls `recipe-of-the-day` + `users`). We choose a single dedicated `GET /api/trmnl/summary` instead.

- **Why**: The existing endpoints return full datasets (e.g. `GetRaceHistory` embeds pipe-joined results; stats endpoints require `racer_id`/`season_id` query params). A TRMNL poll needs one compact, self-contained document — one URL in `settings.yml`, one small payload per refresh, trivial Liquid templates.
- **Alternatives considered**: Multi-URL polling of `/api/race-history` + `/api/rounds` (rejected: heavy payloads, needs query params, template logic must correlate two responses); a TRMNL "static data" plugin with manually pasted JSON (rejected: not live stats).

### D2: Latest race = most recent finalized round snapshot

`latest_race` is derived from `round_snapshots` where `status = 'final'`, ordered by `race_date DESC` (then `round DESC`), joined to `round_snapshot_scores` ordered by `position`.

- **Why**: Season standings come from finalized round snapshots (`racing.RacerStatsBySeason`). Using the same source for the "latest race" keeps the display internally consistent — the race shown is exactly the one whose points are in the standings. `round_snapshot_scores` already stores `racer_name` and `position`, so no extra joins are needed.
- **Alternatives considered**: `race_history`/`race_results` (rejected: it is the manually-curated archive and can diverge from championship points; also lacks round linkage to seasons).
- **Fallback**: if no finalized rounds exist, `latest_race` is `null` and the templates render an empty state.

### D3: Standings = active season via a name-enriched aggregation

Standings come from the `'active'` season; if none is active, fall back to the most recently created season (`ORDER BY id DESC`). The aggregation reuses the `round_snapshot_scores` + finalized-`round_snapshots` query from `RacerStatsBySeason`, extended to also select the racer name (e.g. `MAX(rss.racer_name)`), and limited to the top N (e.g. top 8) for the display.

- **Why**: Reuses a proven query shape and the "finalized rounds only" semantics; avoids touching `racing` package logic broadly. A tiny helper (e.g. `racing.SeasonStandings(db, seasonID, limit)`) keeps `handlers/trmnl.go` thin.
- **Alternatives considered**: join the `racers` table for names (rejected: `round_snapshot_scores.racer_name` already exists and is what the rounds UI displays); expose full standings unbounded (rejected: payload size; TRMNL renders at most ~10 rows).

### D4: TRMNL plugin structure mirrors Sandwitches

New `trmnl/` directory at repo root containing `settings.yml` and four Liquid templates: `full.liquid`, `half_horizontal.liquid`, `half_vertical.liquid`, `quadrant.liquid`.

- **Why**: TRMNL's plugin framework (v2.x, matching the 2.3.7 `framework_version` Sandwitches uses) requires these four templates for the four device layouts, plus a `settings.yml` manifest. Reusing the exact structure makes publishing as an official recipe mechanical.
- `settings.yml` uses `strategy: polling`, `polling_url` set to `<instance_url>/api/trmnl/summary`, `refresh_interval: 60` (minutes), and a `custom_fields.url` field ("Address of heat instance", mirroring Sandwitches' instance-URL field) so the plugin works against any self-hosted instance.
- **Template content**: `full.liquid` shows race name/track/date, finishing order (top 3 with points, rest compact) and standings table (position, name, wins, points, top ~8). Half/quadrant layouts show progressively trimmed views (e.g. podium + top-3 standings). All use TRMNL layout classes (`layout`, `grid`, `item`, `value`, `label`, `hr`, `title_bar`) and Liquid guards (`{% if latest_race %}`) for the empty state.

### D5: No CORS/auth handling

TRMNL polling is server-side; the device never makes browser requests to the instance. The endpoint uses the same middleware stack as other public GETs (gzip, request ID, security headers, otel) and no `AuthMiddleware`.

- **Why**: zero-friction install for self-hosters; consistent with existing `/api/stats/*` endpoints.
- **Risk**: a public endpoint adds an unauthenticated read path. Mitigation: it returns only data already public via existing endpoints (racer names, positions, points); no settings, no personal data beyond usernames.

## Risks / Trade-offs

- [No finalized rounds or no seasons exist] → Endpoint returns `latest_race: null` and empty `standings` with a valid `season: null`; Liquid templates guard with `{% if %}` and show a "No race data yet" state. Covered by spec scenarios + template-level tests.
- [Standings (from rounds) and `race_history` archive diverge] → Documented in README: the TRMNL view reflects the championship (finalized rounds), not the manual archive. No code change needed.
- [Payload size on wide seasons] → Standings limited to top 8; latest race results limited to ~10 rows. Refresh interval defaults to 60 min (configurable), keeping poll load negligible (two aggregate queries).
- [Name denormalization drift in `round_snapshot_scores`] → Same names already shown in the rounds UI; acceptable, and the aggregation picks the stored name per snapshot.

## Migration Plan

No schema migration. Deploy is additive:

1. Merge backend handler + route; `GET /api/trmnl/summary` live.
2. Merge `trmnl/` templates + `settings.yml`.
3. Update README (API table row + TRMNL section) and CHANGELOG.
4. Optionally publish the plugin recipe on TRMNL (outside this repo's scope; the `trmnl/` dir is the artifact).

Rollback: revert the single commit; remove route + `trmnl/` dir. No data migration involved.
