## 1. Sound module

- [x] 1.1 Create `ts/sound.ts`: `getVolume(category)`, `playCategory(category)` (override → `HTMLAudioElement` at volume; else synthesized tone at volume; 0 → silent), `loadSettings`/`saveSettings`/`resetSettings` with defaults merging, validation helpers (MIME, size ≤ 2 MB, clamp 0..1), quota-error handling
- [x] 1.2 Refactor `ts/tv.ts` `tvPlaySound()` to call `playCategory()`; keep flag/sound trigger behavior identical

## 2. Settings modal

- [x] 2.1 Add modal markup + labeled trigger button to `static/tv.html` (per-category volume slider with % readout, upload button, remove override, reset-all)
- [x] 2.2 Modal wiring (new `ts/sound-settings.ts` or inline in `ts/tv.ts`): open/close, Esc, focus return, file validation feedback, persistence via `ts/sound.ts`

## 3. Tests

- [x] 3.1 TS unit test for settings helpers: clamping, defaults merge, override round-trip, invalid upload rejection, quota failure
- [x] 3.2 Playwright e2e: set volume → persisted after reload; upload tiny valid audio → override active; reset → defaults
- [x] 3.3 Run `task pre-push` (gofmt, go test, vet+govulncheck, tsc, build) and fix failures
