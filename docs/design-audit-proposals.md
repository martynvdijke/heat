# HEAT Board Game Companion — Web Design Audit & Feature Proposals

> Generated from review against [Vercel Web Interface Guidelines](https://github.com/vercel-labs/web-interface-guidelines)
> Date: 2026-05-24

---

## Audit: Critical Issues

### `controller.html`

```
controller.html:21  - user-scalable=no + maximum-scale=1.0 — disables pinch-zoom (WCAG 1.4.4)
controller.html:367 - Sound FX "engine" button icon-only — missing aria-label
controller.html:368 - Sound FX "horn" button icon-only — missing aria-label
controller.html:369 - Sound FX "finish" button icon-only — missing aria-label
controller.html:370 - Sound FX "crash" button icon-only — missing aria-label
controller.html:246 - Blue flag button icon-only (title attr only) — use aria-label
controller.html:249 - Black/white flag button icon-only (title attr only) — use aria-label
controller.html:163 - <select id="race-type"> not associated with a <label>
```

### `player.html`

```
player.html:5   - user-scalable=no + maximum-scale=1.0 — disables pinch-zoom (WCAG 1.4.4)
player.html:51  - <select id="player-select"> no associated <label>
player.html:54  - <input id="device-name"> no associated <label>
player.html:74  - Logout button icon-only (fa-right-from-bracket) — missing aria-label
player.html:143 - Turbo button icon-only (fa-bolt) — missing aria-label
```

### `spectator.html`

```
spectator.html:39 - Home link icon-only (fa-house in <a>) — missing aria-label
```

---

## Audit: Accessibility

### Skip-to-Content

No page has a skip-to-content link for keyboard users. Add:

```html
<a href="#main-content" class="skip-link">Skip to main content</a>
```

### `aria-live` Regions

| File | Missing `aria-live="polite"` on |
|---|---|
| `controller.html` | Standings list, gear log, race events, connected players |
| `player.html` | Dashboard (heat counts, lap info, position updates) |
| `spectator.html` | Events container, heat chart area |
| `pitboard.html` | Pit board driver cards grid (live auto-refresh) |
| `tv.html` | Leaderboard, events ticker |

### Icon-Only Buttons (all pages)

Bulk fix: any `<button>` containing only an `<i>` element (no visible text) needs `aria-label`. Found in:

- `controller.html:367-370` — Sound FX row
- `controller.html:246,249` — Driver flag buttons
- `player.html:74` — Logout (dashboard)
- `player.html:143` — Turbo button
- `spectator.html:39` — Home link
- `admin.html:887` — Upload image button
- `admin.html:1007` — Map upload button
- Various table action cells in `admin.html`

### Missing `<label>` Elements

| File | Element | Fix |
|---|---|---|
| `player.html:51` | `<select id="player-select">` | Add `<label for="player-select">` |
| `player.html:54` | `<input id="device-name">` | Add `<label for="device-name">` |
| `controller.html:163` | `<select id="race-type">` | Add `<label for="race-type" class="visually-hidden">` |
| `trophies.html:62` | `<select id="driver-select">` | Add `<label for="driver-select">` |

### Missing `autocomplete`

| File | Input | Expected |
|---|---|---|
| `player.html:54` | Device Name | `autocomplete="device-name"` |
| `controller.html:309` | Lap Number | `autocomplete="off"` |
| `controller.html:381` | Race Name | `autocomplete="off"` |
| `controller.html:382` | Number of Laps | `autocomplete="off"` |
| `admin.html:640` | AI API Key | `autocomplete="off"` (security!) |

---

## Audit: Animation & `prefers-reduced-motion`

**No `prefers-reduced-motion` media query exists anywhere in the app.** Add:

```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

Affected animations:

| Location | Animation |
|---|---|
| `style.css:587-601` | `flagPulse`, `screenShake` (red flag) |
| `tv.html:50-53` | `slideIn`, `flash` |
| `admin.html:104-118` | `shuffleBounce`, `lockFlash` |
| `controller.html:91-97` | `pulse` (status dot) |

---

## Audit: Typography

### Missing `font-variant-numeric: tabular-nums`

Every numeric display that updates dynamically shifts layout as digits change size. Add to:

- **All stat values** across `style.css` (`.stat-value`, `.value` in stat cards)
- **Lap counters** (`#current-lap`, `#tv-lap`, `#spec-lap`, `#pit-lap`, `#my-lap`)
- **Race timers** (`#race-timer`, `#tv-clock`, `#spec-timer`, `#pit-clock`)
- **Positions/ranks** (`.pos`, `.rank`, `.tv-position`, `.pit-pos`)
- **Table cells** with numbers (points, gaps, lap times)

### Missing `text-wrap: balance` / `text-pretty`

Add to all `<h1>`–`<h6>` and heading-like elements to prevent widows:

```css
h1, h2, h3, h4, h5, h6, .country, .stat-label {
  text-wrap: balance;
}
```

---

## Audit: By-Page Quick Summary

| File | Verdict |
|---|---|
| `index.html` | Minor — skip link, reduced-motion, tabular-nums |
| `controller.html` | **2 critical** (zoom), 4+ accessibility, reduced-motion |
| `player.html` | **2 critical** (zoom), 4+ accessibility, reduced-motion |
| `spectator.html` | 1 accessibility (home link), reduced-motion |
| `tv.html` | Reduced-motion, missing aria-live |
| `pitboard.html` | Missing aria-live |
| `trophies.html` | Missing label on driver select, tabular-nums |
| `stats.html` | Tabular-nums, reduced-motion |
| `admin.html` | 3+ icon-only buttons, missing autocomplete on API key |
| `race-report.html` | ✅ pass (print page) |
| `login.html` | ✅ pass (well-structured form) |
| `setup.html` | ✅ pass (well-structured form) |
| `style.css` | Reduced-motion, tabular-nums |

---

## Proposed Feature: Race Start Light System

A digital F1-style start sequence that brings dramatic tension to physical game race starts.

### What It Does

- Five red lights illuminate one by one at 1s intervals on a full-screen dark page
- After a random 0.5–3s hold, **all lights extinguish** → race begins
- Audio: beep per light, horn blast on GO, engine roar background
- Options: manual mode (race director clicks each light), auto mode (random delay)
- Broadcasts start signal over WebSocket to all connected player dashboards
- Integrates with controller view as a "Start Lights" button

### Implementation Sketch

```
ts/startlights.ts
  - StartLightsEngine class: countdown, random delay, light states
  - AudioManager: loads media/beep.mp3, media/horn.mp3, media/engine.mp3
  - WebSocket integration: broadcast GO signal

static/startlights.html
  - Full-screen dark viewport, no chrome
  - 5 large red circles (CSS: radial-gradient for glow)
  - Transition: each light fades in
  - All-off reveals green "GO!" text

static/controller.html update
  - "Start Lights" button added to Quick Actions
  - Opens startlights in new window / fullscreen

static/style.css additions
  - .start-light dimensions, glow, transition
  - .start-light.on state
  - .go-text animation
  - prefers-reduced-motion fallback
```

### UX Flow

```
Admin taps "Start Lights" on controller
  → startlights.html opens (fullscreen/new tab)
  → Lights illuminate 1-by-1 (1s intervals)
  → Random hold (0.5-3s)
  → ALL OFF → GO!
  → WebSocket: { type: "race_start", timestamp: ... }
  → Player dashboards show green flag / "RACE ON!"
  → Race timer begins on controller
```

### Why It Fits

| Aspect | Fit |
|---|---|
| Theme | Directly from F1 — pure racing drama |
| Physical game | Race starts are a key board game moment; this adds ceremony |
| Existing infra | WebSocket already in place for real-time sync |
| Spectator value | TV view and Pit Board can display the start sequence |
| Complexity | Low — CSS animations + setTimeout + basic audio |
| Risk | None — purely additive, no existing code changes needed |

---

## Proposed Feature: Race Director Commentary Feed

An automatic narrative ticker that brings the race to life.

### What It Does

- Generates contextual commentary from race events (overtakes, crashes, weather shifts, flags)
- Displays as a scrolling ticker on TV view, spectator view, and pit board
- Pulls from the existing Commentator Quotes database with driver name insertion
- Supports manual entries from the race director (custom announcements)
- Timestamps each entry for race chronology

### Implementation Sketch

```
Backend:
  POST /api/commentary          — add commentary entry
  GET  /api/commentary?since=N  — poll new entries
  Quote template engine: "{{driver}} makes a bold move!"
  Auto-generation: map race events → commentary templates

Frontend (ts/commentary.ts):
  CommentaryTicker class
    - Polls /api/commentary every 3s
    - Renders scrolling ticker on tv.html, spectator.html
    - Auto-fade for old entries (>30s)
    - Pause on hover

Static updates:
  tv.html:       #tv-commentary ticker below weather panel
  spectator.html:#spec-commentary section above events
  controller.html: manual commentary input
```

### Quote Template Examples

| Event | Template |
|---|---|
| Overtake | "{{driver}} sweeps past {{target}} into turn 3! |
| Crash | "Contact! {{driver}} spins — yellow flags waved!" |
| Weather change | "Rain at turn 9 — {{driver}} first to react!" |
| Race start | "And they're off! {{driver}} gets a fantastic start!" |
| Fastest lap | "{{driver}} sets the fastest lap — 1:23.456!" |
| Safety car | "Safety car deployed — field bunches up!" |

### Why It Fits

| Aspect | Fit |
|---|---|
| Existing data | Reuses Commentator Quotes DB table and author system |
| Low effort | Template expansion + poll endpoint — no new models |
| High value | Transforms a static race display into a narrative experience |
| Scales | From casual home games to streaming/spectating |
| Complementary | Works alongside all existing views without conflict |

---

## Recommended Implementation Order

1. **Audit fixes** (1–2 sessions)
   - Remove `user-scalable=no` from controller.html + player.html
   - Add `aria-label` to all icon-only buttons
   - Add `aria-live="polite"` to dynamic sections
   - Add `prefers-reduced-motion` media query to style.css
   - Add `font-variant-numeric: tabular-nums` to numeric displays
   - Fix missing `<label>` elements and `autocomplete` attributes

2. **Race Start Light System** (1 session)
   - `ts/startlights.ts`
   - `static/startlights.html`
   - CSS additions to `style.css`
   - Controller integration

3. **Race Director Commentary** (1–2 sessions)
   - Backend commentary endpoint
   - Frontend ticker component
   - Quote template engine
   - TV + spectator integration

---

*This document describes design improvements only. No code changes have been made.*
