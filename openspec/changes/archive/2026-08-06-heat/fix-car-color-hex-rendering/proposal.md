## Why

The admin UI accepts any hex car color (`<input type="color">` in `tab-drivers.html`), but the main leaderboard (`ts/index.ts`) only renders 9 hardcoded named colors. For any hex value set in admin, the leaderboard's color dots are invisible (`.color-indicator.#800080` is invalid CSS — `#` is illegal in a class selector) and map markers fall back to white. Other frontend pages (spectator, pitboard, tv, player) already render hex correctly via inline styles; the leaderboard is the outlier. A racer's car color and name set in admin should be viewable in the frontend normal UI with the specific color, whatever hex it is.

## What Changes

- **Add a shared `normalizeHex` helper** (`ts/color.ts`) that maps the 9 legacy named colors to their hex values (`red`→`#ff4444`, etc.), prepends `#` when missing, validates the 6-digit hex form, and falls back to a neutral default for invalid/empty input. Reused across `index.ts` and (optionally) `admin.ts` to DRY up the 4× `startsWith('#')` repetition.
- **Render car color via inline styles in `index.ts`**, replacing the broken CSS-class approach: `<span class="color-indicator me-2" style="background:${normalizeHex(r.car_color)}">` at the leaderboard row (line 240), stats modal (line 271), and qualification grid (line 303). Map markers (line 178) use `fillColor: normalizeHex(r.car_color)` instead of the `colorMap` lookup. Matches the pattern already used by `spectator.ts`, `pitboard.ts`, `tv.ts`, `player.ts`, and `admin.ts`.
- **Remove the dead `.color-indicator.<name>` CSS rules** (`static/style.css:261-268`) since rendering is now inline-style-driven. The base `.color-indicator` rule (size/border, lines 253-260) stays.
- **Add a Playwright test** that sets a hex color in admin and verifies it renders on the `/` leaderboard.

## Capabilities

### New Capabilities
- `car-color-rendering`: Any hex car color set in the admin UI renders with that exact color across all frontend views (leaderboard table, map markers, stats modal, qualification grid, spectator, pitboard, tv, player). Legacy named color values continue to render in their existing hex. Invalid or empty values fall back to a neutral default and never break rendering.

### Modified Capabilities
_None — `openspec/specs/` is empty; no existing requirements change._

## Impact

- **Code**:
  - NEW `ts/color.ts` — `normalizeHex(color: string): string` + the named-color map (moved from `index.ts:169-173`).
  - `ts/index.ts` — 4 sites: map markers (line 178), leaderboard rows (line 240), stats modal (line 271), qualification grid (line 303). Replace `colorMap[r.car_color] || '#ffffff'` and `class="color-indicator ${r.car_color}"` with `normalizeHex` + inline `style="background:..."`.
  - `static/style.css` — remove lines 261-268 (`.color-indicator.red`, `.blue`, …, `.orange`). Keep lines 253-260 (base `.color-indicator`).
  - (Optional) `ts/admin.ts` — replace the 4× `r.car_color.startsWith('#')?r.car_color:'#'+r.car_color` sites (lines 156, 608, 642, 712) and the edit-form normalization (line 751) with `normalizeHex` imports.
  - `tests/` — new Playwright spec: set `#800080` in admin, assert the leaderboard dot's `background` style contains `#800080` (or `rgb(128, 0, 128)`).
- **APIs**: None. `car_color` is already a string field on `Racer` (`models/models.go:14`); no schema or API change.
- **Dependencies**: None. Pure TS refactor.
- **Systems**: None.
- **Breaking**: None. Legacy named-color seed data (`db/seed.go`) continues to render identically via the named→hex map. No DB migration.
