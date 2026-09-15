## 1. Admin Season tab template

- [x] 1.1 Add Spins and Overheated `<th>` columns to the Driver Statistics table in `static/templates/tab-season.html` (after DNS)
- [x] 1.2 Add spins and overheated number inputs to the statsModal form in `static/templates/tab-season.html` (min=0, matching existing fastest_laps/dnf/dns inputs)

## 2. ts/admin.ts rendering and save paths

- [x] 2.1 Render spins/overheated cells in `renderStatsList()` (after the dnf/dns badges)
- [x] 2.2 Populate spins/overheated inputs in the stats modal load path (add + edit)
- [x] 2.3 Include spins/overheated in the stats modal save payload (stats modal save path)
- [x] 2.4 Include spins/overheated in the inline stats save payload (second save path)

## 3. Tests

- [x] 3.1 Add Go handler test file `14_test_admin_season_stats_test.go`: POST `/api/racer-stats` with spins/overheated, assert `GET /api/racer-stats` returns them and the `racer_stats` row persists them
- [x] 3.2 Add/update Playwright e2e in `tests/rounds-season-stats.spec.ts` (or new spec): Season tab Stats table shows Spins/Overheated columns with values; editing a driver via the modal persists and re-renders them
- [x] 3.3 Run `task pre-push` (gofmt, go test, vet+govulncheck, tsc, build) and fix failures
