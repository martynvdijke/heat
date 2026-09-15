## Why

The project's roadmap, decisions and acceptance criteria live in OpenSpec change artifacts under `openspec/changes/`, but `.gitignore` line 21 is `openspec/` — the entire directory is ignored. As a result collaborators, CI and agents cannot see current work, work can be lost if the single disk is lost, and tracking is inconsistent: 6 files are already tracked because they were committed before the ignore (`openspec/changes/controller-polish/tasks.md`, `controller-start-lights/tasks.md`, `race-commentary-feed/tasks.md`, `sound-customization/tasks.md`, `stats-season-selection-comparison/tasks.md`, `weather-live-effects/tasks.md`), while all other artifacts are invisible to git. `openspec/changes/` currently has 16 changes; four are ACTIVE and entirely untracked (invisible to `git status` because they are ignored): `websocket-auth-rooms` (20 open tasks), `live-race-state-broadcast` (19 open tasks), `websocket-resilience` (12 open tasks), `websocket-hardening` (12 open tasks) — 63 open tasks total, each with `proposal.md`, `design.md`, `tasks.md`, `specs/`. `git status` reports 0 untracked files, hiding that active work exists only on one disk. No `openspec/specs/` main-spec directory exists yet; specs live only under each change.

## What Changes

- Stop ignoring `openspec/` so all change artifacts (and `openspec/specs/` main specs when promoted) are tracked in version control. If a sub-path must stay ignored (e.g. local scratch), use `.gitignore` negation patterns for that sub-path only; otherwise remove the `openspec/` rule.
- Commit currently-active changes (`websocket-auth-rooms`, `live-race-state-broadcast`, `websocket-resilience`, `websocket-hardening`) so their 63 open tasks and associated `proposal.md`/`design.md`/`tasks.md`/`specs/` are versioned.
- Reconcile the 6 already-tracked `tasks.md` files with the new policy (they remain tracked; no special-casing needed beyond the ignore removal).
- Add a CI validation gate (`openspec validate --all --strict`) so invalid specs/changes fail the build.
- Establish archival convention for completed changes (`openspec archive` / move to archive location) so history is preserved without lingering active changes.

## Capabilities

### New Capabilities

- `spec-artifact-versioning`: OpenSpec change artifacts and main specs are tracked in version control, validated in CI (strict), and archived via a defined convention; no secrets or credentials appear in spec artifacts.

### Modified Capabilities

- None.

## Impact

Git tracking and `.gitignore`; CI workflow (new validation step); OpenSpec workflow (creation/archival of changes becomes visible in PRs). No runtime code, API or data-model change. Low risk; rollback is re-adding the ignore rule (content stays on disk).
