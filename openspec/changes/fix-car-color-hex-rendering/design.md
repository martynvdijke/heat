## Context

The admin UI (`static/templates/tab-drivers.html:133-134`) accepts any hex car color via `<input type="color">` + a text input; `ts/admin.ts` normalizes the `#` prefix. The `Racer.CarColor` field (`models/models.go:14`) is a free-form string. Seed data (`db/seed.go`) uses 9 named colors (`red`, `blue`, `green`, `yellow`, `grey`, `silver`, `black`, `purple`, `orange`).

The main leaderboard (`ts/index.ts`) only renders those 9 named colors:
- Map markers (line 178): `fillColor: colorMap[r.car_color] || '#ffffff'` — any hex falls back to white.
- Leaderboard rows (line 240): `<span class="color-indicator ${r.car_color} me-2">` — `.color-indicator.#800080` is **invalid CSS** (the `#` is illegal in a class selector), so the dot has no background for hex colors.
- Stats modal (line 271): same broken class + displays raw `r.car_color` text.
- Qualification grid (line 303): same broken class.

The `.color-indicator.<name>` CSS rules live at `static/style.css:261-268`. Other frontend pages already render hex correctly via **inline styles**: `spectator.ts:41,46` (`style="border-left-color:${r.car_color}"`), `pitboard.ts:45-46`, `tv.ts:43` (`style="background:${r.car_color}"`), `player.ts:57` (`style.background = me.car_color`). `admin.ts` uses the same inline-style pattern (lines 156, 608, 642, 712) with a `startsWith('#')` normalization repeated 4×. `index.ts` is the outlier.

## Goals / Non-Goals

**Goals:**
- Any hex car color set in admin renders with that exact color in the leaderboard (table dot, map marker, stats modal, qualification grid).
- Legacy named-color seed values continue to render in their existing hex (no visual regression).
- Invalid or empty `car_color` falls back to a neutral default and never breaks rendering or throws.
- DRY up the `#`-prefix normalization repeated across `admin.ts` and `index.ts` via one shared helper.

**Non-Goals:**
- Changing the `CarColor` field type, DB schema, or API. It stays a free-form string.
- Migrating seed data from named colors to hex. Named colors are human-readable in the DB; normalization happens at render time.
- Adding a color-picker to the frontend (admin is the only edit surface).
- Re-skinning the leaderboard layout — only the color-rendering mechanism changes.

## Decisions

### D1 — Shared `normalizeHex` helper in `ts/color.ts`
**Rationale**: The named-color → hex map currently lives inline in `index.ts:169-173`; the `#`-prefix normalization is repeated 4× in `admin.ts` and once in `index.ts`'s edit form. One helper centralizes both: (a) map the 9 named colors to hex; (b) prepend `#` if missing; (c) validate `/^#?[0-9a-fA-F]{6}$/`; (d) fall back to `#cccccc` for invalid/empty. Reused by `index.ts` (required) and `admin.ts` (optional DRY). Pure function, trivially unit-testable.
**Alternatives considered**:
- Inline the fix in `index.ts` only. Rejected: leaves the 4× `startsWith('#')` repetition in `admin.ts` and the named-map duplication risk.
- Move the named map to a JSON file. Rejected: 9 entries — a TS const is simpler and type-safe.

### D2 — Inline `style="background:..."` replaces CSS-class color-indicator in `index.ts`
**Rationale**: `.color-indicator.${car_color}` is fundamentally broken for hex (illegal CSS class selector). Inline styles work for any valid CSS color string and match the pattern already proven in `spectator.ts`, `pitboard.ts`, `tv.ts`, `player.ts`, and `admin.ts`. The base `.color-indicator` CSS (size/border) stays; only the per-color subclasses go away.
**Alternatives considered**:
- Generate per-racer CSS classes with escaped hex (`.color-indicator\.\\#800080`). Rejected: fragile, ugly, and still requires a stylesheet injection per render — inline styles are simpler and equivalent.
- Use CSS custom properties (`style="--car-color:#800080"` + `.color-indicator { background: var(--car-color) }`). Rejected: indirection without benefit here; inline `background` is direct.

### D3 — Keep DB seed as named colors; normalize at render
**Rationale**: Named colors (`red`, `blue`, …) are readable in the DB and in API responses. `normalizeHex` maps them to hex at render time, so the UI is correct without a migration. New racers get hex from the color picker; legacy seed data keeps working unchanged.
**Alternatives considered**: Migrate seed data to hex in `db/seed.go`. Rejected: unnecessary DB churn; the helper handles both forms transparently.

### D4 — Remove dead `.color-indicator.<name>` CSS rules
**Rationale**: After D2, no template emits `class="color-indicator red"` etc. — all use `class="color-indicator" style="background:..."`. The 8 named-subclass rules (`static/style.css:261-268`) become dead code. Removing them avoids confusion ("why is there a `.color-indicator.purple` rule but no element uses it?") and keeps the stylesheet honest. The base `.color-indicator` rule (lines 253-260) stays.
**Alternatives considered**: Keep them as a fallback. Rejected: dead CSS is a maintenance trap; if a future template reintroduces the class pattern, the inline-style approach is the documented standard.

### D5 — (Optional) Refactor `admin.ts` to import `normalizeHex`
**Rationale**: `admin.ts` repeats `r.car_color.startsWith('#')?r.car_color:'#'+r.car_color` at lines 156, 608, 642, 712 and the edit-form normalization at line 751. Importing `normalizeHex` DRYs those up and ensures the same validation/fallback behavior. Marked optional because `admin.ts` already works for hex — this is a consistency/DRY win, not a bug fix.
**Alternatives considered**: Leave `admin.ts` as-is. Acceptable; the helper is still required for `index.ts`.

## Risks / Trade-offs

- **[Inline styles bypass CSP `style-src` if `'unsafe-inline'` is ever removed]** → Mitigation: the current CSP (`middleware/security.go:14`) allows `'unsafe-inline'` for `style-src`, and every other car-color consumer already uses inline styles. Tightening CSP is a separate change that would need a coordinated migration (hash-based styles or custom properties) across all pages, not just this one.
- **[Named-color map in `normalizeHex` must stay in sync with seed data]** → Mitigation: the map is the same 9 entries already in `index.ts`; moving it to `color.ts` doesn't change the sync surface. If seed data adds a new named color, the map needs an entry — same as today. A unit test covers all 9 named values.
- **[Removing `.color-indicator.<name>` CSS breaks a page I missed]** → Mitigation: grep confirms only `index.ts` (and its build output `static/js/index.js`) emits `class="color-indicator ${car_color}"`. Other pages use inline styles or `color-dot`. The Playwright test added in this change covers the leaderboard; a grep assertion in tasks confirms no other emitter exists before the CSS is removed.
- **[Map marker color differs slightly from the dot color for named colors]** → Mitigation: both call `normalizeHex(r.car_color)`, so they're identical by construction. Today the map uses `colorMap` (e.g. `red`→`#ff4444`) and the dot uses `.color-indicator.red { background-color: #ff4444 }` — already the same hex. After the change, both use `#ff4444` from the one helper.
