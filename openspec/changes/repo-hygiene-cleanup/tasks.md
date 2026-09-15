## 1. Verify and delete fully-merged stale branches

- [ ] 1.1 Run `git branch --merged main` and `git branch -a` to confirm `feat/heat-upgrades` (2026-04-18), `feat/multi-device-playwright` (2026-05-09), `feat/new-features-stats-trophies-controller-chat-oneoff` (2026-04-30) have no unique commits
- [ ] 1.2 Delete the three fully-merged branches locally and remotely (`git branch -d` / `git push origin --delete`)
- [ ] 1.3 Inventory stale renovate branches (`renovate/all-minor-patch` 19 ahead / 3 behind, `renovate/github.com-swaggo-files-2.x`, `renovate/pin-dependencies`) and confirm ownership via `renovate.json`; prune only if safe / recreatable by Renovate

## 2. Remove dead source file

- [ ] 2.1 Grep `handlers/stats.go` to verify it contains only `package handlers` and no defined symbols; confirm real handlers live in `handlers/stats_basic.go`, `stats_advanced.go`, `stats_incidents.go`, `stats_performance.go`, `stats_scope.go`
- [ ] 2.2 Remove `handlers/stats.go`

## 3. Align .gitignore with reality

- [ ] 3.1 Audit current `.gitignore` entries (`heat`, `heat.db`, `heat.db-shm`, `heat.db-wal`, `heat-server`, `static/js/`, `vendor/`, `*.exe`, `*.test`, `*.out`, `node_modules/*`, `playwright.local.config.ts`, `test-results/*`, `playwright-report/*`, `playwright-report.zip`, `tmp/*`, `tools/`, `media/backups/`, `backups/`, `docs/*.md`, `openspec/`, `.opencode/`) against actual build/runtime outputs vs source-of-truth
- [ ] 3.2 Update `.gitignore` to ignore only generated/runtime artifacts; do NOT duplicate the `openspec/` line owned by `version-openspec-artifacts`; decide `docs/*.md` handling (open question)
- [ ] 3.3 Verify `docs/docs.go`, `docs/swagger.json`, `static/swagger.json` remain tracked as source-of-truth under the chosen Swagger policy

## 4. Decide and enforce generated Swagger policy

- [ ] 4.1 Confirm `task generate-swagger` regenerates `docs/docs.go` (3594 lines), `docs/swagger.json`, `static/swagger.json` and document the regeneration path
- [ ] 4.2 Add CI staleness check that runs `task generate-swagger` and fails if `git diff` on `docs/docs.go`, `docs/swagger.json`, `static/swagger.json` is non-empty (install `swag` in CI); document alternative "gitignore + generate at build" and rationale for "commit + CI check"
- [ ] 4.3 Verify `.github/workflows/stale-branches.yml` is referenced for ongoing stale-branch automation; no duplicate workflow

## 5. Validate

- [ ] 5.1 Run `task pre-push` (formatting, tests, vet/vuln, TS compile, Go build, Swagger regeneration) and verify clean with no drift
- [ ] 5.2 Run `openspec validate repo-hygiene-cleanup --strict` and fix until valid
