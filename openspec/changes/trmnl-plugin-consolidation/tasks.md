## 1. Inventory and sharing-mechanism decision
- [ ] 1.1 Inventory shared vs distinct markup across `trmnl/src/*.liquid` vs `trmnl-next-race/src/*.liquid` (all four layouts) and shared vs per-plugin `src/settings.yml` fields; record which chrome/patterns are common and which bindings are data-contract-specific (summary vs next-race).
- [ ] 1.2 Prototype Liquid `{% include %}`/`{% render %}` partial and verify with `trmnlp lint -d trmnl`, `trmnlp lint -d trmnl-next-race`, `trmnlp build -d trmnl`, `trmnlp build -d trmnl-next-race`; decide D1 — use native includes if supported, otherwise scaffold a minimal shared source + compose/build step that generates each plugin's `src/*.liquid`.
- [ ] 1.3 Decide shared-source location (e.g. `trmnl-shared/` or equivalent) and how it composes with `trmnlp -d` layout; document the choice in `design.md` if it diverges from the plan.

## 2. Shared layout and settings extraction
- [ ] 2.1 Extract shared Liquid partials/compose source for common chrome (headers, tables, row pattern `flex flex--row flex--left gap--xsmall w--full` + `shrink-0` + `w--min-0 grow`, image pattern with `image-dither`, grid/responsive scaffolding per layout type) with thin per-plugin wrappers for summary vs next-race data bindings.
- [ ] 2.2 Consolidate `settings.yml` to a single canonical source with per-plugin overlays (name, description ≤ 35 chars, polling URL, any next-race-specific fields); ensure both generated `src/settings.yml` remain valid YAML (`python3 -c "import yaml; ..."`) and use `<a href="...">` anchors for any URLs in field text.
- [ ] 2.3 Apply the shared CSS/class convention so both plugins satisfy publishing checks: no `style=` attributes, `image-dither` on every `<img>`, required `lg:`/`portrait:`/`lg:portrait:` classes per layout (full: `lg:grid--cols-*`, half_horizontal: `portrait:flex--col` etc., half_vertical/quadrant: `lg:value--*`/`lg:label--*`).

## 3. Build, gitignore, and verification
- [ ] 3.1 Ensure `_build/` is gitignored in both `trmnl/.gitignore` and `trmnl-next-race/.gitignore` and that `_build/*.html` is never hand-edited; wire `trmnlp build -d trmnl` / `trmnlp build -d trmnl-next-race` (or compose then build) as the sole generation path and verify reproducibility.
- [ ] 3.2 Run the full publishing verification checklist from `.opencode/skills/trmnl-publishing/SKILL.md` for both plugins: `rg -n 'style='`, `rg -n '<img' | rg -v 'image-dither'`, `rg -n 'lg:'`, `rg -n 'https?://'` on settings, YAML validity, `trmnlp lint -d trmnl`, `trmnlp lint -d trmnl-next-race`, `trmnlp build -d trmnl`, `trmnlp build -d trmnl-next-race`; fix until both plugins are green.
- [ ] 3.3 Verify backend data contracts remain correct: `GET /api/trmnl/summary` vs `GET /api/trmnl/next-race` still serve their expected shapes; run `12_test_trmnl_test.go` and `12_test_trmnl_next_race_test.go` and preview all eight liquid files in OG landscape, OG portrait, half, X landscape, and X portrait profiles.

## 4. CI and rollout
- [ ] 4.1 Add CI jobs that run `trmnlp lint` and `trmnlp build` for both `trmnl/` and `trmnl-next-race/` (plus the `rg` checks) on every change touching `trmnl*/**` or shared sources; failure in either plugin fails the workflow.
- [ ] 4.2 Keep current `trmnl/` and `trmnl-next-race/` dirs on main/a backup branch until shared-source checks pass; document rollback as revert/restore of the two plugin dirs (no backend change to roll back).
- [ ] 4.3 Resolve open questions: confirm whether two plugins are required vs one mode-toggled plugin, finalize shared-source location, and decide `_build/` artifact handling (generated in CI vs committed preview).
