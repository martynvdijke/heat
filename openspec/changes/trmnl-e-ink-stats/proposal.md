## Why

Heat tracks everything about a "Heat: Pedal to the Metal" championship — race results, season standings, streaks. The sister project Sandwitches already ships an official [TRMNL](https://trmnl.com/) e-ink plugin (recipe 247547) that renders its daily content on a kitchen e-ink display. Heat has the same opportunity: a TRMNL e-ink mode that puts the latest race results and season championship standings on an always-on desk display, so players can glance at the championship state without opening the app.

## What Changes

- Add a new **public** JSON endpoint `GET /api/trmnl/summary` returning a compact, TRMNL-shaped payload: the latest race (track, date, finishing order with points) and the current season standings (position, racer, points, wins).
- Add a `trmnl/` plugin directory with Liquid templates (`full.liquid`, `half_horizontal.liquid`, `half_vertical.liquid`, `quadrant.liquid`) and a `settings.yml` manifest (polling strategy, `polling_url` pointing at the summary endpoint, refresh interval).
- Publish the plugin as an official TRMNL recipe, mirroring the Sandwitches integration.

## Capabilities

### New Capabilities

- `trmnl-summary-api`: Public, unauthenticated `GET /api/trmnl/summary` endpoint returning compact JSON (latest race results + current season standings) suitable for TRMNL polling.
- `trmnl-e-ink-display`: The TRMNL Liquid templates and `settings.yml` manifest that render the summary payload on the e-ink device.

### Modified Capabilities

<!-- No existing specs change -->

## Impact

- **Backend**: new handler (e.g. `handlers/trmnl.go`) + route registration in `main.go`; standings derived from existing `race_results` data (reusing the standings logic in `racing/`). No new tables, no schema migration.
- **Plugin assets**: new `trmnl/` directory at repo root with four Liquid templates and `settings.yml`.
- **Docs**: TRMNL section + `/api/trmnl/summary` row in the README API table.
- **No breaking changes**: existing endpoints untouched; the new endpoint is additive and unauthenticated (consistent with the other public read-only `/api/stats/*` GETs).
