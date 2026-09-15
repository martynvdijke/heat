## ADDED Requirements

### Requirement: Shared layout markup has a single source
The system SHALL maintain shared TRMNL Liquid layout markup (common chrome, responsive scaffolding, row/image patterns) in a single source, with each plugin's `src/*.liquid` derived from that source via either Liquid `{% include %}`/`{% render %}` partials (if verified supported by `trmnlp lint`/`build`) or a deterministic shared-source + compose/build step that generates per-plugin liquid.

#### Scenario: Shared markup change propagates to both plugins
- **WHEN** shared layout markup (e.g. row scaffolding, grid, responsive classes) is updated in the single source
- **THEN** both `trmnl/src/*.liquid` and `trmnl-next-race/src/*.liquid` reflect the change after compose/build (or via shared include) without requiring duplicate manual edits

#### Scenario: Per-plugin data bindings remain distinct
- **WHEN** the two plugins render different data contracts (summary vs next-race)
- **THEN** each plugin's thin wrapper/override binds its own endpoint data while sharing common chrome from the single source

### Requirement: Shared settings and defaults have a single source
The system SHALL maintain TRMNL `settings.yml` schema and shared defaults in a single canonical source, with per-plugin overlays for fields that legitimately differ (e.g. plugin name, description, polling URL, data-contract-specific fields); both generated `src/settings.yml` files SHALL remain valid YAML and pass `trmnlp lint`.

#### Scenario: Common setting default updated once
- **WHEN** a shared settings default (e.g. Heat instance URL help text or default) is changed in the canonical source
- **THEN** both `trmnl/src/settings.yml` and `trmnl-next-race/src/settings.yml` reflect the updated default after generation without manual duplication

#### Scenario: Per-plugin settings overrides preserved
- **WHEN** per-plugin overlays define distinct values (e.g. `description`, polling URL)
- **THEN** each generated `settings.yml` retains its overlay values while inheriting shared defaults

### Requirement: Plugin built outputs are generated and reproducible
The system SHALL treat each plugin's `_build/*.html` (quadrant, half_horizontal, full, half_vertical) as generated output, reproducible via `trmnlp build -d <plugin>` (or compose then build), and SHALL keep `_build/` gitignored consistently in both plugins; hand-editing `_build/` SHALL NOT be the source of truth.

#### Scenario: Reproducible build from src
- **WHEN** `trmnlp build -d trmnl` and `trmnlp build -d trmnl-next-race` are run from the shared/derived `src/`
- **THEN** the generated `_build/*.html` files are byte-reproducible (modulo timestamps) and match what CI produces, with no manual `_build/` edits required

#### Scenario: Gitignore prevents committed _build drift
- **WHEN** inspecting `trmnl/.gitignore` and `trmnl-next-race/.gitignore`
- **THEN** `_build/` is ignored in both plugins so stale generated HTML cannot be committed as source

### Requirement: Both plugins pass TRMNL publishing checks
The system SHALL ensure both `trmnl/` and `trmnl-next-race/` pass TRMNL publishing checks as defined in `.opencode/skills/trmnl-publishing/SKILL.md` and enforced by `trmnlp lint`/`build`: no inline `style` attributes (use framework classes from `https://trmnl.com/css/2.3.7/plugins.css`), every `<img>` includes `image-dither`, responsive `lg:`/`portrait:`/`lg:portrait:` classes per layout type, no plain-text URLs in `settings.yml` field text (anchors only), and `description` ≤ 35 chars.

#### Scenario: Lint passes for both plugins from shared source
- **WHEN** `trmnlp lint -d trmnl` and `trmnlp lint -d trmnl-next-race` are run after any shared-source change
- **THEN** both lints pass with no inline-style, image-dither, responsive-class, URL-in-settings, description-length, or other publishing violations

#### Scenario: Shared convention prevents inline-style and image-dither regressions
- **WHEN** the shared layout convention is applied (e.g. `flex flex--row flex--left gap--xsmall w--full`, `shrink-0`, `w--min-0 grow`, `image-dither` on every `<img>`)
- **THEN** `rg -n 'style=' trmnl/src trmnl-next-race/src` returns no output and `rg -n '<img' trmnl/src trmnl-next-race/src | rg -v 'image-dither'` returns no output for both plugins

### Requirement: Plugin data contracts remain correct
The system SHALL preserve the two distinct backend data contracts — `GET /api/trmnl/summary` (via `handlers/trmnl.go`, `GetTRMNLSummary`) and `GET /api/trmnl/next-race` (via `handlers/trmnl_next_race.go`, `GetTRMNLNextRace`) — and each plugin's markup SHALL correctly bind its own contract without cross-contamination.

#### Scenario: Each plugin polls its own endpoint
- **WHEN** `trmnl/` and `trmnl-next-race/` are configured with their respective polling URLs
- **THEN** the summary plugin renders data from `/api/trmnl/summary` and the next-race plugin renders data from `/api/trmnl/next-race`, with no plugin displaying the other's data shape

#### Scenario: Backend handlers unchanged in contract
- **WHEN** the consolidation is applied
- **THEN** `handlers/trmnl.go` and `handlers/trmnl_next_race.go` continue to serve their existing response shapes and `12_test_trmnl_test.go` / `12_test_trmnl_next_race_test.go` remain green (or are updated only to reflect intentional contract changes)
