## 1. Shared helper

- [ ] 1.1 Create `ts/color.ts` exporting `normalizeHex(color: string): string`:
  - Named-color map: `red`→`#ff4444`, `blue`→`#4444ff`, `green`→`#44ff44`, `yellow`→`#ffff44`, `grey`→`#aaaaaa`, `silver`→`#aaaaaa`, `black`→`#333333`, `purple`→`#9b59b6`, `orange`→`#e67e22`.
  - If input is a named color, return the mapped hex.
  - Else strip whitespace, prepend `#` if missing, validate `/^#[0-9a-fA-F]{6}$/`; return the normalized hex if valid.
  - Else (invalid/empty) return `#cccccc`.
- [ ] 1.2 Add a unit test for `normalizeHex` covering: all 9 named colors, `#800080`, `800080`, empty string, `not-a-color`, `#fff` (3-digit — should fall back, since the spec requires 6-digit), `#800080ff` (8-digit — fall back).

## 2. Leaderboard rendering

- [ ] 2.1 `ts/index.ts` line 178 (map markers): replace `fillColor: colorMap[r.car_color] || '#ffffff'` with `fillColor: normalizeHex(r.car_color)`. Remove the inline `colorMap` declaration (lines 169-173) — it moves to `ts/color.ts`.
- [ ] 2.2 `ts/index.ts` line 240 (leaderboard rows): replace `<span class="color-indicator ${r.car_color} me-2">` with `<span class="color-indicator me-2" style="background:${normalizeHex(r.car_color)}">`.
- [ ] 2.3 `ts/index.ts` line 271 (stats modal): replace `<span class="color-indicator ${r.car_color}"></span> ${r.car_color}` with `<span class="color-indicator" style="background:${normalizeHex(r.car_color)}"></span> ${escapeHtml(r.car_color)}`. (Keep the text display; escape it for safety.)
- [ ] 2.4 `ts/index.ts` line 303 (qualification grid): replace `<span class="color-indicator ${r.car_color}">` with `<span class="color-indicator" style="background:${normalizeHex(r.car_color)}">`.
- [ ] 2.5 Add `import { normalizeHex } from './color';` at the top of `ts/index.ts`.
- [ ] 2.6 Grep `ts/` and `static/templates/` for any remaining `color-indicator ${` emitters; confirm none exist outside the 4 sites fixed above.

## 3. CSS cleanup

- [ ] 3.1 Remove lines 261-268 from `static/style.css` (`.color-indicator.red`, `.blue`, `.green`, `.yellow`, `.grey, .silver`, `.black`, `.purple`, `.orange`).
- [ ] 3.2 Keep lines 253-260 (base `.color-indicator` — size, border-radius, border).
- [ ] 3.3 Grep `static/style.css` for any other `.color-indicator.` subclass rules; remove if found.

## 4. Optional: DRY up admin.ts

- [ ] 4.1 `ts/admin.ts` lines 156, 608, 642, 712: replace `r.car_color.startsWith('#')?r.car_color:'#'+r.car_color` with `normalizeHex(r.car_color)`. Add `import { normalizeHex } from './color';`.
- [ ] 4.2 `ts/admin.ts` line 751 (edit form): replace the inline normalization with `normalizeHex(r.car_color)` for the color-picker value (note: the color input needs a valid `#rrggbb` — `normalizeHex` guarantees that).

## 5. Tests

- [ ] 5.1 Add a Playwright test in `tests/` that: logs in as admin, navigates to the Drivers tab, edits a racer's `car_color` to `#800080`, saves, navigates to `/`, and asserts the racer's leaderboard dot has `background` containing `#800080` or `rgb(128, 0, 128)`.
- [ ] 5.2 Add a Playwright assertion that a seeded `red` racer still renders with `background` containing `#ff4444` or `rgb(255, 68, 68)` (no regression).
- [ ] 5.3 `npm run typecheck` green.
- [ ] 5.4 `task build` green (esbuild bundles `ts/color.ts`).
- [ ] 5.5 `task test` and `task test:e2e` green.

## 6. Cleanup

- [ ] 6.1 `task pre-push` green.
- [ ] 6.2 Commit with `fix: render any hex car color in the leaderboard via shared normalizeHex helper`.
