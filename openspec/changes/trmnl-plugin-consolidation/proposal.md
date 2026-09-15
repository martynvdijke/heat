## Why
Heat ships two near-identical TRMNL e-ink plugins — `trmnl/` (Heat Championship Stats, `GET /api/trmnl/summary` via `handlers/trmnl.go`) and `trmnl-next-race/` (Heat Next Race, `GET /api/trmnl/next-race` via `handlers/trmnl_next_race.go`). Each plugin directory contains the same five source files (`src/full.liquid`, `src/half_vertical.liquid`, `src/half_horizontal.liquid`, `src/quadrant.liquid`, `src/settings.yml`) plus `.trmnlp.yml`, `.gitignore`, and generated `_build/*.html`. All five source files differ between the two plugins (`diff -rq` confirms divergence), yet the layouts are structurally near-duplicates. A layout fix, publishing-check fix, or settings default change must currently be applied twice, publishing checks (`trmnlp lint`/`build`) must pass independently in both, and settings schemas can silently drift.

## What Changes
- Consolidate shared layout markup and settings defaults into a single source of truth while keeping two distinct TRMNL plugin manifests/endpoints where their data contracts differ (summary vs next-race).
- Introduce a sharing mechanism appropriate to TRMNL: prefer Liquid `{% include %}`/`{% render %}` partials if the TRMNL rendering pipeline supports includes; otherwise introduce a shared source directory plus a build/compose step that generates each plugin's `src/*.liquid` from shared partials. The decision is gated on verification against the `trmnlp` toolchain and the repo skill `.opencode/skills/trmnl-publishing/SKILL.md` (no TRMNL feature is asserted without verification).
- Unify `settings.yml` schema and defaults to a single source with per-plugin overrides, so common fields (e.g. Heat instance URL) are defined once.
- Establish one shared CSS/class convention so both plugins satisfy the same publishing checks (no inline `style`, `image-dither` on every `<img>`, responsive `lg:`/`portrait:` classes, no plain-text URLs in settings, description ≤ 35 chars).
- Ensure `_build/` outputs are generated artifacts (not hand-edited) and `.gitignore`/`_build` handling is consistent across both plugins.
- Run TRMNL publishing checks (`trmnlp lint` + `trmnlp build`) for both plugins in CI.

## Capabilities
### New Capabilities
- `trmnl-plugins`: Single-source shared layout partials and settings defaults for the two TRMNL plugins with per-plugin data contracts preserved, generated `_build` outputs, and publishing checks enforced for both plugins.
### Modified Capabilities
None.

## Impact
TRMNL plugin sources (`trmnl/src/*`, `trmnl-next-race/src/*`), `settings.yml`, `_build/` generation, `.gitignore`, and CI publishing-check steps. Backend handlers (`handlers/trmnl.go`, `handlers/trmnl_next_race.go`) and their data contracts (`/api/trmnl/summary` vs `/api/trmnl/next-race`) remain distinct; no breaking change to plugin consumers. Risk is limited to build/compose wiring and layout regressions, mitigated by keeping current plugin dirs until checks pass on both.
