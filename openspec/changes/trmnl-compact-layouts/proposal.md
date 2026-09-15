# trmnl-compact-layouts

## Why

The TRMNL plugin templates (committed with the `trmnl-ci-pipeline` change) used the default TRMNL design tokens (`gap`, `value--large/medium/small`, `grid--cols-1`, `hr`) and rendered the race results split across two grids. On the physical 800x480 e-ink display the full layout overflowed and the half/quadrant layouts left unused space, making the plugin hard to read. The templates were restyled with the modern compact design tokens so every layout fits the 800x480 screen.

## What Changes

- **All four layouts (`full`, `half_horizontal`, `half_vertical`, `quadrant`)** restyled with compact TRMNL design tokens: `gap--xsmall`, `label--small`, `value--xsmall` / `value--xxsmall`, `divider--h`, and `grid--cols-2`.
- **Top-3 focus**: half and quadrant layouts show only the top 3 race results / standings entries; the full layout shows every result in a two-column grid.
- **Fallback states**: each layout renders `No race data yet` / `No standings yet` when the API returns no data, instead of a blank screen.
- **Simplified title bar**: text-only (`Heat` + season name), image removed.
- **`trmnl/.gitignore`** added so `trmnlp build` output (`trmnl/_build/`) stays untracked.
- No backend or API payload changes; the `/api/trmnl/summary` contract from `trmnl-ci-pipeline` is unchanged.

## Capabilities

### New Capabilities

- `trmnl-compact-layouts`: TRMNL plugin templates render compactly and readably on the 800x480 display, with fallback states when no race or standings data exists.

## Impact

- `trmnl/src/full.liquid`, `trmnl/src/half_horizontal.liquid`, `trmnl/src/half_vertical.liquid`, `trmnl/src/quadrant.liquid` — restyled (rendering only; payload contract unchanged).
- `trmnl/.gitignore` — new, ignores `_build/`.
- `handlers/trmnl.go`, `models/models.go` — untouched.
