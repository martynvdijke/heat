# Tasks

## 1. Template restyle

- [x] 1.1 `trmnl/src/full.liquid`: compact tokens (`gap--xsmall`, `label--small`, `value--xsmall/xxsmall`, `divider--h`, `grid--cols-2`); all race results and standings in two-column rows; fallback states for missing race/standings; text-only title bar
- [x] 1.2 `trmnl/src/half_horizontal.liquid`: compact tokens; top-3 race results and standings (`forloop.index <= 3`); standings rows show `wins` and `points`; fallback states; text-only title bar
- [x] 1.3 `trmnl/src/half_vertical.liquid`: compact tokens; top-3 race results and standings; fallback states; text-only title bar
- [x] 1.4 `trmnl/src/quadrant.liquid`: compact tokens; top-3 race results; fallback state; text-only title bar

## 2. Project hygiene

- [x] 2.1 Added `trmnl/.gitignore` ignoring `_build/` (trmnlp build output)

## 3. Verification

- [x] 3.1 Payload compatibility verified: templates only use fields from `/api/trmnl/summary` (`latest_race.*`, `standings[].{racer_name, wins, points}`, `season.name`) — `models.SeasonStanding.Wins` carries `json:"wins"`, so `standing.wins` resolves
- [x] 3.2 `openspec validate trmnl-compact-layouts`
- [ ] 3.3 Confirm on-device rendering after the next release-pipeline run (plugin deploy via `trml` job) — **user action**
