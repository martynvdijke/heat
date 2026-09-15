## Context

`openspec/changes/` is the project's source of truth for roadmap and acceptance criteria, but `.gitignore` line 21 (`openspec/`) excludes the entire directory from version control. Six `tasks.md` files were committed before the ignore and remain tracked, while all other artifacts are ignored. `git status` reports 0 untracked files even though four ACTIVE changes exist only on one disk: `websocket-auth-rooms` (20 open tasks), `live-race-state-broadcast` (19 open tasks), `websocket-resilience` (12 open tasks), `websocket-hardening` (12 open tasks) — 63 open tasks total. Additional ignored paths include `docs/*.md`, `node_modules/*`, `static/js/`, `heat`, `heat-server`, `heat.db*`, `test-results/*`, `playwright-report/*`, `playwright-report.zip`, `media/backups/`, `backups/`, `tmp/*`, `tools/`. No `openspec/specs/` main-spec directory exists yet. OpenSpec CLI supports `openspec validate --all --strict` and `openspec archive`.

## Goals / Non-Goals

**Goals:**

- Make all OpenSpec change artifacts (and `openspec/specs/` when it exists) visible to collaborators, CI and agents via version control.
- Commit the four currently-active websocket changes so 63 open tasks are no longer single-disk-only.
- Add a CI gate that validates specs/changes in strict mode.
- Define a clear archival convention for completed changes.

**Non-Goals:**

- Modifying the content of existing changes (including the four websocket changes) beyond committing them as-is.
- Promoting change specs into `openspec/specs/` main specs in this change (archival-time promotion is an open question for later).
- Changing runtime code, APIs, or DB schema.

## Decisions

### D1: Version the whole `openspec/` tree (not selectively)

Rationale: `openspec/` is project state (roadmap, decisions, acceptance criteria), not build output. Selective tracking re-introduces the inconsistency already observed (6 tracked files vs rest ignored) and requires ongoing allow-list maintenance. Versioning the whole tree makes the current state reproducible from git alone.

Alternatives: Keep ignoring `openspec/` and manually `git add -f` per change; or track only `openspec/changes/**/tasks.md`. Both were rejected — they preserve the visibility loss and the inconsistent tracking.

### D2: Remove the `openspec/` ignore rule; use negation only if a sub-path must stay ignored

Rationale: The simplest fix is deleting line 21 (`openspec/`) from `.gitignore`. If a sub-path genuinely must stay ignored (e.g. local scratch notes under `openspec/scratch/`), add a negated sub-path rule (e.g. `openspec/scratch/`) rather than keeping the top-level ignore.

Alternatives: Keep the top-level rule and add `!openspec/changes/**` negations. Rejected — negation ordering is brittle and easy to mis-scope; removing the rule is clearer.

### D3: Add CI step `openspec validate --all --strict`

Rationale: Prevents invalid or incomplete specs from merging. Strict mode catches missing scenarios, malformed deltas, and drift between `proposal.md`/`spec.md`/`tasks.md`.

Alternatives: No CI gate; or non-strict validation. Rejected — non-strict would allow broken invariants to accumulate.

### D4: Archival convention via `openspec archive` (move completed changes to archive location)

Rationale: Completed changes should not linger as active deltas. The CLI's `openspec archive` command (or equivalently moving the change under `openspec/changes/archive/`) preserves history while removing the change from the active set. This also defines when `openspec/specs/` main specs are updated.

Alternatives: Leave completed changes in place forever; or delete them. Both lose clarity — lingering actives create noise, deletion loses audit trail.

### D5: Reconcile the 6 already-tracked `tasks.md` files with the new policy

Rationale: The six files (`controller-polish/tasks.md`, `controller-start-lights/tasks.md`, `race-commentary-feed/tasks.md`, `sound-customization/tasks.md`, `stats-season-selection-comparison/tasks.md`, `weather-live-effects/tasks.md`) remain tracked unchanged. Removing the ignore rule simply makes their tracking consistent with all other artifacts — no migration of those files is needed.

Alternatives: Untrack and re-add them; or keep them as exceptions. Rejected — unnecessary churn.

## Risks / Trade-offs

- PR noise if changes are edited mid-PR (tasks ticked, specs tweaked cause diff churn) → Mitigation: treat `openspec/changes/` diffs as expected review surface; avoid ticking tasks unrelated to the PR; keep changes focused.
- Secrets/PII accidentally written into specs (credentials, tokens, personal data) → Mitigation: review policy — no credentials in OpenSpec; secrets via env/file references only; add to PR checklist.
- Merge conflicts on `tasks.md` if multiple people tick boxes concurrently → Mitigation: small, focused changes; rebase frequently; conflicts are trivial checkbox merges.

## Migration Plan

1. Remove the `openspec/` line (line 21) from `.gitignore` (add negation for a sub-path like `openspec/scratch/` only if such local scratch must stay ignored).
2. Run `git add openspec` to stage all change artifacts including the four ACTIVE websocket changes (`websocket-auth-rooms`, `live-race-state-broadcast`, `websocket-resilience`, `websocket-hardening`).
3. Verify `git status` now shows OpenSpec artifacts (previously reported 0 untracked files despite 63 open tasks on disk).
4. Add CI step running `openspec validate --all --strict` to the workflow.
5. Commit the tracking fix + active changes + CI gate as the `version-openspec-artifacts` change.
6. Rollback: re-add `openspec/` to `.gitignore` — content stays on disk, only versioning is reverted.

## Open Questions

- Where should the CI validation job live (`.github/workflows/` path and job name)?
- Should archived changes move under `openspec/changes/archive/` or rely solely on `openspec archive` tooling?
- Should main specs under `openspec/specs/` be promoted at archive time, and if so what is the promotion workflow?
- The four websocket changes (`websocket-auth-rooms`, `live-race-state-broadcast`, `websocket-resilience`, `websocket-hardening`) are arguably one epic — they should be explicitly sequenced/dependency-ordered (auth+rooms and resilience/hardening overlap in the WS envelope; state-broadcast builds on rooms). Do NOT modify those changes in this change; record the recommendation as a follow-up cross-cutting concern.
