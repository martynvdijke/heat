## 1. Remove ignore and reconcile tracking

- [ ] 1.1 Remove `openspec/` (line 21) from `.gitignore`; add negation only if a sub-path (e.g. `openspec/scratch/`) must stay ignored.
- [ ] 1.2 Run `git add openspec` and verify `git status` now shows artifacts (previously reported 0 untracked files despite 63 open tasks on disk).
- [ ] 1.3 Verify the 6 historically tracked `tasks.md` files (`controller-polish`, `controller-start-lights`, `race-commentary-feed`, `sound-customization`, `stats-season-selection-comparison`, `weather-live-effects`) remain tracked consistently.
- [ ] 1.4 Ensure other ignored outputs remain ignored (`heat`, `heat-server`, `heat.db*`, `static/js/`, `node_modules/*`, `test-results/*`, `playwright-report/*`, `playwright-report.zip`, `media/backups/`, `backups/`, `tmp/*`, `tools/`, `docs/*.md`) via `git check-ignore`.

## 2. Commit currently-active websocket changes

- [ ] 2.1 Stage and commit `openspec/changes/websocket-auth-rooms` (20 open tasks, with `proposal.md`, `design.md`, `tasks.md`, `specs/`).
- [ ] 2.2 Stage and commit `openspec/changes/live-race-state-broadcast` (19 open tasks, with `proposal.md`, `design.md`, `tasks.md`, `specs/`).
- [ ] 2.3 Stage and commit `openspec/changes/websocket-resilience` (12 open tasks, with `proposal.md`, `design.md`, `tasks.md`, `specs/`).
- [ ] 2.4 Stage and commit `openspec/changes/websocket-hardening` (12 open tasks, with `proposal.md`, `design.md`, `tasks.md`, `specs/`).
- [ ] 2.5 Verify `git ls-files openspec/changes/ | wc -l` reflects all active change artifacts and no active task is single-disk-only.

## 3. Add CI validation gate

- [ ] 3.1 Add CI step running `openspec validate --all --strict` to the workflow in `.github/workflows/`.
- [ ] 3.2 Verify the gate fails on intentionally malformed spec and passes on `version-openspec-artifacts` (`openspec validate version-openspec-artifacts --strict`).
- [ ] 3.3 Document CI placement as open question if workflow path/name is undecided.

## 4. Establish archival convention

- [ ] 4.1 Document archival convention using `openspec archive` (or move to `openspec/changes/archive/`) for completed changes.
- [ ] 4.2 Decide open questions: whether archived changes move under `openspec/changes/archive/` and whether `openspec/specs/` main specs are promoted at archive time.
- [ ] 4.3 Record cross-cutting recommendation (do NOT modify those changes): explicitly sequence/dependency-order the four websocket changes as one epic (auth+rooms and resilience/hardening overlap in WS envelope; state-broadcast builds on rooms).

## 5. Guardrails and verification

- [ ] 5.1 Add review policy: no secrets/PII in specs (scan for credentials before commit).
- [ ] 5.2 Note PR-noise trade-off (editing tasks mid-PR causes diff churn) and recommend focused changes.
- [ ] 5.3 Verify rollback: re-adding `openspec/` to `.gitignore` restores ignore without deleting content on disk.
- [ ] 5.4 Run `openspec validate --all --strict` from repo root and ensure it passes.
