## ADDED Requirements

### Requirement: No fully-merged long-lived branches remain

Branches fully merged into `main` with no unique commits SHALL be deleted; the repository SHALL NOT retain long-lived feature branches that are 4+ months stale and fully merged.

#### Scenario: Fully-merged stale branches are deleted

- **WHEN** `git branch --merged main` lists `feat/heat-upgrades` (last commit 2026-04-18), `feat/multi-device-playwright` (2026-05-09), and `feat/new-features-stats-trophies-controller-chat-oneoff` (2026-04-30) with no unique commits
- **THEN** those branches are deleted locally and remotely and no longer appear in `git branch --merged main`

#### Scenario: Non-destructive deletion only

- **WHEN** a branch is not fully merged into `main` (e.g. `renovate/all-minor-patch` 19 ahead / 3 behind)
- **THEN** it is NOT force-deleted; it is only pruned after confirming ownership via `renovate.json` and that Renovate will recreate it if needed

### Requirement: No dead or empty source files

The repository SHALL contain no dead or empty source files such as an empty package stub.

#### Scenario: Dead stats.go removed

- **WHEN** `handlers/stats.go` contains only `package handlers` (1 line) and real handlers live in `handlers/stats_basic.go`, `stats_advanced.go`, `stats_incidents.go`, `stats_performance.go`, `stats_scope.go`
- **THEN** `handlers/stats.go` is removed after verifying via grep that no symbols are defined there

#### Scenario: No empty package files remain

- **WHEN** scanning Go source files for files containing only a package declaration
- **THEN** no such dead files exist

### Requirement: .gitignore excludes only generated and runtime artifacts

`.gitignore` SHALL exclude build and runtime outputs and SHALL NOT ignore source-of-truth artifacts; it SHALL align with the actual repository layout.

#### Scenario: Build and runtime outputs are ignored

- **WHEN** inspecting `.gitignore`
- **THEN** it ignores build/runtime outputs such as `heat`, `heat.db`, `heat.db-shm`, `heat.db-wal`, `heat-server`, `static/js/`, `vendor/`, `*.exe`, `*.test`, `*.out`, `node_modules/*`, `playwright.local.config.ts`, `test-results/*`, `playwright-report/*`, `playwright-report.zip`, `tmp/*`, `tools/`, `media/backups/`, `backups/`

#### Scenario: Source-of-truth artifacts are not ignored

- **WHEN** inspecting `.gitignore` handling of `docs/` and `openspec/`
- **THEN** `docs/docs.go`, `docs/swagger.json`, and `static/swagger.json` are tracked (not ignored), and the `openspec/` line is owned by separate change `version-openspec-artifacts` and not duplicated here

### Requirement: Committed generated artifacts have a documented regeneration path and CI staleness check

Committed generated Swagger artifacts SHALL have a documented regeneration path and a CI check that fails when regeneration produces a diff.

#### Scenario: Swagger regeneration is documented and produces no drift

- **WHEN** `task generate-swagger` (run by `task pre-push`) regenerates `docs/docs.go` (3594 lines, generated), `docs/swagger.json`, and `static/swagger.json`
- **THEN** `git diff` on those paths is empty after regeneration, and CI fails if a diff is detected (staleness check)

#### Scenario: Alternative of gitignoring generated Swagger is documented

- **WHEN** evaluating the generated-artifact policy
- **THEN** the design documents the alternative of gitignoring `docs/docs.go`, `docs/swagger.json`, `static/swagger.json` and generating at build, with rationale for the chosen "commit + CI staleness check" approach that keeps `go install`/offline use working

### Requirement: Cleanup is non-destructive

Cleanup operations SHALL be non-destructive and reversible; no unmerged work SHALL be deleted via force.

#### Scenario: Only fully-merged branches are deleted

- **WHEN** performing branch cleanup
- **THEN** only branches returned by `git branch --merged main` with no unique commits are deleted; branches with unmerged commits are never force-deleted

#### Scenario: Deleted branches and files are recoverable

- **WHEN** a branch or file needs to be restored after cleanup
- **THEN** it is recoverable from git reflog, remote, or git history, and `task pre-push` passes after restoration

### Requirement: Stale-branch detection is automated

Stale-branch detection SHALL be automated via existing CI automation.

#### Scenario: Existing stale-branches workflow is referenced

- **WHEN** checking automation for stale branches
- **THEN** `.github/workflows/stale-branches.yml` is referenced and used rather than duplicating with a new workflow, and `Taskfile.yml` `clean` and `pre-push` remain the local hygiene entry points
