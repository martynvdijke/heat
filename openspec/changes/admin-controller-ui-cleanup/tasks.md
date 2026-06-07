## 1. Toast Notification System

- [ ] 1.1 Create `ts/toast.ts` with `showToast(message, type, duration?)` function using a fixed-position container, supporting types: success, error, info, warning
- [ ] 1.2 Add toast CSS styles to `static/style.css` (fixed top-right, slide-in animation, color-coded, auto-dismiss, max 3 visible stacked)
- [ ] 1.3 Import/bundle `toast.ts` so it's available on both admin and controller pages

## 2. Controller Page — Fix Duplicates & Clean Up HTML

- [x] 2.1 Remove duplicate Quick Action buttons (Safety Car, Red Flag, Chequered Flag rows at lines 225-248 in `controller.html`)
- [x] 2.2 Fix duplicate HTML IDs: rename second `safety-btn` and `redflag-btn` or remove them with step 2.1
- [x] 2.3 Move inline `<style>` rules for `.controller-card`, `.driver-row`, `.position-btn`, `.action-btn`, `.race-control-btn`, `.status-indicator`, `.race-stat`, `.quick-action-btn` into `static/style.css` under a `/* === Controller Page === */` section
- [x] 2.4 Group controller sections into logical card groups with headers: Race Control, Grid, Tracking, Conditions, Players & Audio, Config (add `data-bs-toggle="collapse"` on card headers for collapsible sections)
- [x] 2.5 Make key sections non-collapsible: Race Control (start/pause/stop + lap/time/status) and Live Standings
- [x] 2.6 Fix consistent spacing and font sizing across all controller cards

## 3. Admin Page — Tab Reorganization

- [x] 3.1 Design tab category structure: **Race** (Race Setup, Qualification, Racers, Tracks), **Results** (Stats, Rounds, Seasons), **Content** (Quotes, Teams), **Settings** (Notifications, Email, Telemetry, Analytics, AI, Backup), **System** (Logs)
- [x] 3.2 Implement primary tab bar with 5 category tabs
- [x] 3.3 Create secondary sub-tab navigation (pills or dropdown) within each category pane using Bootstrap tab components
- [x] 3.4 Move existing tab panes into the new two-level structure, preserving all content
- [x] 3.5 Ensure tab state is preserved (active tab persists on page reload via URL hash or localStorage)
- [x] 3.6 Add responsive wrapping/scroll for the primary tab bar on smaller screens
- [x] 3.7 Extract shared admin card styles into `static/style.css`

## 4. Controller TypeScript — Bug Fixes & Toast Integration

- [x] 4.1 Fix `moveUp()` position swap bug: replace `controllerRacers.find(r => r.position === r.position)` with correct logic to find and swap with the driver above
- [x] 4.2 Fix `moveDown()` position swap bug: same pattern as moveUp but for the driver below
- [x] 4.3 Replace all `alert()` calls in `controller.ts` with `showToast()`:
  - `saveRaceResult()` alert → toast
  - `discardRace()` confirm → keep confirm but add toast feedback
  - `sendBlueFlag()` / `sendBlackWhiteFlag()` alert → toast
  - `takeRoundSnapshot()` alert/error → toast
  - `archiveCurrentSeason()` alert/confirm → toast

## 5. Admin TypeScript — Toast Integration

- [x] 5.1 Replace all `alert()` calls in `admin.ts` with `showToast()`:
  - Form save handlers (race, racer, quote, track, stats, notifications, email, backup, AI, OTel, Umami)
  - Load/save error handlers
  - Grid application confirmation
  - Season operations
- [x] 5.2 Keep `confirm()` dialogs for destructive actions (delete racer, delete track, delete season, discard race) — only replace informational `alert()` calls
- [ ] 5.3 Add loading button states (disable + spinner) for async operations that lack them: backup manual, stats save, email save, etc.
- [ ] 5.4 Add basic error handling on fetch calls that currently ignore failures (wrap in try/catch, show error toast)

## 6. Regression Verification

- [x] 6.1 Verify `controller.html` renders without duplicate IDs (check browser console for warnings)
- [x] 6.2 Verify all Quick Action buttons work: Shuffle, Yellow, Safety, Red Flag, Chequered, Snapshot, Start Lights
- [x] 6.3 Verify admin tabs all render content correctly after reorganization
- [x] 6.4 Verify `static/style.css` changes don't break other pages (index.html, stats.html, trophies.html, driver.html, login.html)
- [x] 6.5 Run Go tests: `go test ./...`
- [x] 6.6 Run TypeScript compilation: check for type errors
- [ ] 6.7 Run Playwright E2E tests: `npx playwright test`
- [x] 6.8 Verify toast notifications render on both pages with all types (success, error, info, warning)
