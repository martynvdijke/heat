## Context
Heat ships two TRMNL plugins in `trmnl/` and `trmnl-next-race/`, each with `src/full.liquid`, `src/half_vertical.liquid`, `src/half_horizontal.liquid`, `src/quadrant.liquid`, `src/settings.yml`, `.trmnlp.yml`, `.gitignore`, and generated `_build/*.html`. All five source files differ between the directories but are near-duplicate layouts. Backends are separate: `handlers/trmnl.go` (`GET /api/trmnl/summary`) and `handlers/trmnl_next_race.go` (`GET /api/trmnl/next-race`), with tests `12_test_trmnl_test.go` and `12_test_trmnl_next_race_test.go`. The repo skill `.opencode/skills/trmnl-publishing/SKILL.md` defines the definitive publishing checks via `trmnlp lint`/`trmnlp build` (executable `/root/.local/share/gem/ruby/3.4.0/bin/trmnlp`). Only `src/` is edited; `_build/` is generated and gitignored.

Publishing constraints from the skill (cited verbatim where applicable):
- No inline `style` attributes — use framework utility classes from `https://trmnl.com/css/2.3.7/plugins.css` (e.g. `flex flex--row flex--left w--full`, `gap--xsmall`, `shrink-0`, `w--min-0` + `grow`).
- Every `<img>` MUST include `image-dither` (or `image--dither`); checked by `ImageGenerationService::markupContainsDitherImage`.
- Responsive classes for TRMNL X and portrait: `lg:` (screen--lg), `portrait:`, `lg:portrait:` prefixes (e.g. `lg:grid--cols-4`, `lg:portrait:grid--cols-2`, `portrait:flex--col`, `lg:value--small`, `lg:label--base`, `portrait:w--full`, etc., all verified in `plugins.css`).
- No plain-text URLs in `settings.yml` field text (`help_text`/`description`); URLs must be `<a href="...">` anchors (single-quoted YAML). Structured fields (`placeholder`, `github_url`, `learn_more_url`, `email_address`) may hold raw URLs.
- Plugin `description` ≤ 35 chars (`description_length` lint).

Current pain: layout fixes are applied twice, publishing checks run per-plugin with no shared guarantee, and settings can drift.

## Goals / Non-Goals
**Goals:**
- Single source of truth for shared layout markup and settings defaults across both plugins.
- Keep two distinct plugin manifests/endpoints where data contracts differ (summary vs next-race) — no forced merge if devices/endpoints need separate plugins.
- Both plugins pass `trmnlp lint` and `trmnlp build` from the shared source.
- `_build/` treated as generated, reproducible output with consistent gitignore.
- CI enforces publishing checks for both plugins on every change.

**Non-Goals:**
- Merging into a single TRMNL plugin with a mode toggle unless proven that one plugin can serve both data contracts/devices without regression.
- Changing backend data contracts (`/api/trmnl/summary` vs `/api/trmnl/next-race`) beyond preserving them.
- Hand-editing `_build/*.html` or committing generated output as source.
- Adding new TRMNL features or layouts beyond consolidation.

## Decisions
### D1: Keep two plugin manifests but share layout partials via Liquid includes — otherwise share via a compose/build step
Rationale: The least disruptive path is to keep `trmnl/` and `trmnl-next-race/` as separate TRMNL plugins (separate `.trmnlp.yml` manifests, separate polling URLs, separate device installs) while deduplicating their near-identical Liquid. If the TRMNL Liquid pipeline supports `{% include %}`/`{% render %}` partials, shared partials are the native mechanism (no custom tooling). The skill and current plugin dirs do not assert include support, and the definitive check is `trmnlp lint`/`build` — so the choice must be verified, not assumed.
Alternatives: (a) Merge into one plugin with a `mode` setting toggle — rejected unless verified that TRMNL can switch data sources/endpoints per setting without separate installs; risks breaking existing device installs and conflating divergent data contracts. (b) Always use a compose step regardless of include support — heavier than needed if includes work, but is the correct fallback.
Decision: Attempt shared Liquid partials first; if `trmnlp lint`/`build` proves includes are unsupported by the TRMNL rendering pipeline, adopt a shared source directory plus a deterministic compose/build step that generates each plugin's `src/*.liquid` from shared partials + per-plugin overrides.

### D2: Share `settings.yml` schema/defaults with a single source and per-plugin overrides
Rationale: Both `src/settings.yml` files define overlapping fields (notably the Heat instance URL) with divergent help text and defaults; drift is already observable. A single canonical schema/defaults file with per-plugin overlay (e.g. plugin name, description, polling URL, any next-race-specific fields) eliminates drift while allowing legitimate differences.
Alternatives: Keep two independent `settings.yml` files and lint for drift — preserves duplication. Generate both from one file with no overrides — too rigid for per-plugin descriptions/polling URLs.
Decision: Single canonical settings source; per-plugin YAML overlay for name/description/polling URL and any data-contract-specific fields. Both generated `settings.yml` files must remain valid YAML and pass `trmnlp lint` field checks.

### D3: One shared CSS/class convention so both plugins pass publishing checks
Rationale: Publishing checks are identical for both plugins (skill §1–5). A shared convention — no inline `style`, `image-dither` on every `<img>`, required `lg:`/`portrait:` responsive classes per layout type, anchor-wrapped URLs in settings, description ≤ 35 chars — ensures a fix in one place fixes both. Reference patterns from the skill (e.g. `flex flex--row flex--left gap--xsmall w--full` + `shrink-0` + `w--min-0 grow` for rows; `grid grid--cols-2 lg:grid--cols-4 lg:portrait:grid--cols-2` for full; `columns portrait:flex--col` for half_horizontal) become the shared baseline.
Alternatives: Per-plugin conventions with separate lint configs — perpetuates divergence.
Decision: One shared class/convention document (or linted shared partials) that satisfies all five skill checks; verified by `rg -n 'style='`, `rg -n '<img' | rg -v 'image-dither'`, `rg -n 'lg:'`, `rg -n 'https?://'`, and `trmnlp lint` on both plugins.

### D4: Ensure `_build/` outputs are generated (not hand-edited) and gitignored consistently
Rationale: The skill states only `src/` is edited; `_build/` is generated by `trmnlp build` and gitignored. Currently both plugins have `_build/*.html` on disk but `.gitignore` coverage must be consistent to prevent committed drift.
Alternatives: Commit `_build/` as source — bloats diffs and invites hand-edits. Ignore per-plugin inconsistently — risks accidental commits.
Decision: Both plugins keep `_build/` gitignored; `_build/*.html` is always regenerated via `trmnlp build -d <plugin>` and never hand-edited. CI may optionally generate `_build/` as an artifact but not require it committed.

### D5: CI runs the publishing checks for BOTH plugins
Rationale: The skill's verification checklist requires `trmnlp lint -d trmnl`, `trmnlp lint -d trmnl-next-race`, `trmnlp build -d trmnl`, `trmnlp build -d trmnl-next-race` plus `rg` checks. Running only one plugin's checks would let the other regress silently — the whole point of consolidation is to keep both green from one source.
Alternatives: Check only the changed plugin — insufficient. Check only `trmnl/` as representative — misses per-plugin overrides.
Decision: CI step runs lint+build for both plugin directories on every change touching `trmnl*/**`, shared sources, or settings. Failure in either plugin fails the workflow.

## Risks / Trade-offs
- TRMNL Liquid `{% include %}`/`{% render %}` not supported in the hosted rendering pipeline → Mitigation: verify with `trmnlp lint`/`build` on a trial partial; if unsupported, fall back to shared-source + compose/build step that generates each `src/*.liquid` (D1).
- Publishing-check regression in one plugin but not the other (e.g. missing `image-dither`, stray `style=`, plain URL in overrides) → Mitigation: shared convention + `rg` checks + `trmnlp lint` for both plugins in CI (D3, D5).
- Divergent data contracts (summary vs next-race) force different markup that resists sharing → Mitigation: share only common chrome (headers, tables, responsive scaffolding); keep per-plugin data bindings in thin per-plugin wrappers/overrides; do not force identical markup where data differs.
- Generated `_build/` drift (committed stale HTML, inconsistent gitignore) → Mitigation: gitignore `_build/` in both plugins, treat `_build/` as derived, regenerate in CI and optionally publish as artifact (D4).
- Compose step adds build complexity if includes are unsupported → Mitigation: keep compose minimal (file concatenation/generation, no new runtime dep), document it, and gate on verification that includes truly fail.

## Migration Plan
1. Inventory shared vs distinct markup across `trmnl/src/*.liquid` and `trmnl-next-race/src/*.liquid` and shared vs per-plugin `settings.yml` fields.
2. Decide sharing mechanism: prototype a shared Liquid partial with `{% include %}`/`{% render %}` and run `trmnlp lint -d trmnl` / `trmnlp build -d trmnl` and the same for `trmnl-next-race` to verify include support; if it fails, scaffold a minimal shared source + compose step that generates each plugin's `src/*.liquid` and `src/settings.yml` from shared + overlay.
3. Extract shared partials/compose source (common layout chrome, responsive scaffolding, row/image patterns, shared settings defaults) and wire per-plugin thin wrappers/overlays for summary vs next-race bindings.
4. Regenerate both plugins (`trmnlp build -d trmnl`, `trmnlp build -d trmnl-next-race` or compose then build) and run the full skill verification checklist: `rg -n 'style='`, `rg -n '<img' | rg -v 'image-dither'`, `rg -n 'lg:'`, `rg -n 'https?://'`, YAML validity, and `trmnlp lint` for both.
5. Add/verify CI jobs that run `trmnlp lint` and `trmnlp build` for both plugins and the `rg` checks on every relevant change.
6. Verify device rendering in all four profiles (OG landscape, OG portrait, half, X landscape; X portrait where applicable) for all eight liquid files.

Rollback: Keep the current `trmnl/` and `trmnl-next-race/` directories intact (on main/a backup branch) until the shared-source build passes `trmnlp lint`/`build` for both plugins and manual preview is approved. Revert is `git revert` or restore of the two plugin dirs; no backend/data-contract change to roll back.

## Open Questions
- Are two distinct TRMNL plugins genuinely needed (different polling endpoints `/api/trmnl/summary` vs `/api/trmnl/next-race`, different device installs), or could a single plugin with a mode/setting serve both without breaking existing installs?
- Where should the shared source live (e.g. `trmnl-shared/` at repo root vs `trmnl/shared/`), and how does it interact with `trmnlp`'s `-d` directory layout?
- Should `_build/` remain gitignored and generated in CI only, or should generated `_build/*.html` be committed as a preview artifact? What does `trmnlp` expect for publishing?
- Does the TRMNL hosted pipeline support Liquid `{% include %}`/`{% render %}` partials, or must sharing be done via a compose/build step? (Must be verified with `trmnlp lint`/`build`, not assumed.)
