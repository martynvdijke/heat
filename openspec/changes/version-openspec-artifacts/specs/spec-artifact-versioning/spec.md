## ADDED Requirements

### Requirement: OpenSpec artifacts are tracked in version control

All OpenSpec change artifacts (`proposal.md`, `design.md`, `tasks.md`, `specs/**/spec.md`) and main specs (`openspec/specs/**` when present) SHALL be tracked in version control and visible to collaborators, CI and agents.

#### Scenario: Active changes are visible in git

- **WHEN** a collaborator clones the repository and lists `openspec/changes/`
- **THEN** all active changes including `websocket-auth-rooms`, `live-race-state-broadcast`, `websocket-resilience`, and `websocket-hardening` are present and `git status` reflects their tracked state

#### Scenario: Previously inconsistent tracking is reconciled

- **WHEN** the repository contains the 6 historically tracked files (`controller-polish/tasks.md`, `controller-start-lights/tasks.md`, `race-commentary-feed/tasks.md`, `sound-customization/tasks.md`, `stats-season-selection-comparison/tasks.md`, `weather-live-effects/tasks.md`) alongside the four ACTIVE websocket changes (63 open tasks total)
- **THEN** all artifacts are tracked consistently after the `openspec/` ignore rule is removed — no file requires `git add -f`

### Requirement: Ignore rules exclude only build and runtime outputs

`.gitignore` SHALL exclude only build and runtime outputs (e.g. `heat`, `heat-server`, `heat.db*`, `static/js/`, `node_modules/*`, `test-results/*`, `playwright-report/*`, `tmp/*`, `media/backups/`, `backups/`, `tools/`) and SHALL NOT ignore planning artifacts under `openspec/`.

#### Scenario: Planning artifacts are not ignored

- **WHEN** `git check-ignore --no-index openspec/changes/version-openspec-artifacts/proposal.md` is run
- **THEN** the file is not ignored

#### Scenario: Build outputs remain ignored

- **WHEN** `git check-ignore --no-index heat` or `heat.db` or `static/js/bundle.js` is run
- **THEN** those paths are still ignored

#### Scenario: Local scratch stays ignored via negation

- **WHEN** a sub-path must stay ignored for local scratch (e.g. `openspec/scratch/`)
- **THEN** `.gitignore` uses a specific negation/sub-path rule for that location only, not a top-level `openspec/` ignore

### Requirement: CI validates OpenSpec changes in strict mode

Continuous integration SHALL run `openspec validate --all --strict` (or equivalent `openspec validate <change> --strict`) to ensure specs and changes are well-formed before merge.

#### Scenario: Invalid spec fails CI

- **WHEN** a change introduces a malformed spec delta or missing scenario
- **THEN** the CI validation step fails and blocks merge

#### Scenario: Valid changes pass strict validation

- **WHEN** all changes and main specs are well-formed
- **THEN** `openspec validate --all --strict` exits with success

### Requirement: Completed changes are archived

Completed OpenSpec changes SHALL be archived via `openspec archive` (or by moving the change to an archive location such as `openspec/changes/archive/`) so that active `openspec/changes/` contains only in-progress work and history is preserved.

#### Scenario: Completed change is archived

- **WHEN** a change's tasks and requirements are implemented and verified
- **THEN** the change is archived and no longer appears as an active change, but its files remain in version history

#### Scenario: Archive preserves audit trail

- **WHEN** an archived change is inspected in git history
- **THEN** its `proposal.md`, `design.md`, `tasks.md`, and `spec.md` deltas remain recoverable

### Requirement: No secrets in spec artifacts

Spec and change artifacts SHALL NOT contain credentials, tokens, secrets, or PII; references to secrets SHALL use env/file indirection.

#### Scenario: Secret scan passes

- **WHEN** a spec or proposal is reviewed or scanned
- **THEN** no credential value appears — only env var names or file-path references (e.g. `*_SECRET_FILE`) are present

### Requirement: WebSocket epic sequencing is documented

The four websocket changes SHALL be explicitly sequenced/dependency-ordered as a cross-cutting concern, without modifying those changes in this change.

#### Scenario: Sequencing recommendation is recorded

- **WHEN** the plan for websocket work is reviewed
- **THEN** the documentation notes that `websocket-auth-rooms` and `websocket-resilience`/`websocket-hardening` overlap in the WS envelope and `live-race-state-broadcast` builds on rooms, recommending explicit sequencing for the epic
