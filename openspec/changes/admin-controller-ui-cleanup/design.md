## Context

The HEAT project ships two main operational UIs:

1. **Admin dashboard** (`admin.html` + `admin.ts`) — a full-screen desktop-oriented panel with 16+ tabs for managing racers, tracks, seasons, settings, and logs
2. **Race Controller** (`controller.html` + `controller.ts`) — a mobile-friendly live race control page with start/pause/stop, live standings, flags, weather, and various tracking tools

Both pages have evolved organically, resulting in duplicated UI elements, inconsistent styling, flat navigation, and interruptive `alert()` dialogs. This design addresses these issues while keeping the same functional surface area.

## Goals / Non-Goals

**Goals:**
- Reorganize admin tabs into logical groups with clear visual hierarchy
- Remove duplicate Quick Action buttons and fix duplicate HTML IDs in controller page
- Group controller sections into meaningful categories
- Replace all `alert()` calls with a shared toast notification system
- Extract shared styles from inline `<style>` blocks into `style.css`
- Standardize card, button, and spacing across both pages
- Fix the `moveUp`/`moveDown` position-swap bug in `controller.ts`
- Add basic loading states and error handling for API calls
- Improve mobile responsiveness for both pages

**Non-Goals:**
- Adding new features or functionality
- Changing the backend API surface
- Database schema changes
- Rewriting the entire UI framework
- Changing the vintage/racing aesthetic
- Modifying the live board (`index.html`) or other pages beyond admin and controller

## Decisions

### Decision 1: Admin tab grouping with sub-navigation
- **Chosen**: Group 16+ tabs into 5 categories with a two-level navigation: a primary tab bar for categories (Race, Results, Content, Settings, System) and secondary tab-like pills within each pane for sub-pages.
- **Alternatives considered**:
  - **Dropdown menu**: Condenses tabs but adds an extra click for every navigation
  - **Accordion sidebar**: More complex layout change, harder to implement cleanly
  - **Single scrollable tab bar** (current): Overwhelming with 16+ items
- **Rationale**: Two-level hierarchy keeps all navigation visible without overwhelming. Categories are intuitive: Race (things you do during a race), Results (post-race data), Content (quotes/teams), Settings (admin configuration), System (technical).

### Decision 2: Toast notification system
- **Chosen**: A lightweight, dependency-free toast utility in `ts/toast.ts` with CSS animations. Supports `success`, `error`, `info`, `warning` types. Auto-dismiss after 3s (configurable). Appends to a fixed container in the DOM.
- **Alternatives considered**: Bootstrap Toasts (requires Bootstrap JS dependency), third-party libs (unnecessary dependency).
- **Rationale**: ~30 lines of vanilla TypeScript + ~30 lines of CSS. Shared single instance between admin and controller pages.

### Decision 3: Controller section grouping
- **Chosen**: 6 logical groups with collapsible card headers: **Race Control** (start/pause/stop + stats), **Grid** (standings, shuffle), **Tracking** (turbo, gear, events, lap recording), **Conditions** (weather, flags), **Players & Audio** (connected players, sound FX), **Config** (race settings, season actions).
- **Alternatives considered**: Keeping flat list (current), accordion sections, wizard layout.
- **Rationale**: Bootstrap collapse on card headers allows collapsing sections that aren't currently needed, especially useful on mobile. Groups follow the mental model of "what you do before/during/after a race."

### Decision 4: Button layout for Quick Actions
- **Chosen**: 2-column grid with clearly differentiated colors: Shuffle (primary), Yellow Flag (dark-blue), Safety Car (warning/orange toggle), Red Flag (danger toggle), Chequered Flag (light), Snapshot (success), Start Lights (gradient orange).
- **Alternatives considered**: 3-column grid (too cramped on mobile), single column (too much scrolling).
- **Rationale**: 2-column fits well on both mobile and desktop. Toggle buttons show active state with color fill + icon change. No duplicate entries.

### Decision 5: Shared CSS in `style.css`
- **Chosen**: Extract common styles (cards, buttons, status indicators, spacing utilities) into `static/style.css`. Keep page-specific styles (admin tab layout, controller-specific elements) in each page's `<style>` block.
- **Alternatives considered**: Separate `admin.css` and `controller.css` files (more HTTP requests), single inline block per page (current — causes duplication).
- **Rationale**: Single shared CSS file reduces duplication. Page-specific overrides remain inline for clarity.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Tab reorganization may confuse existing users | Maintain same tab order within groups. Add subtle visual hint (section label) above category groups. |
| Collapsible sections on controller may hide important info during race | Key race state (lap, time, status) is always visible in the header. Standings and race control are non-collapsible. |
| Toast notifications overlapping on rapid successive actions | Queue toasts vertically with max 3 visible. Older toasts dismiss faster (2s) when queue is full. |
| CSS changes may affect other pages | Only extract styles that are currently duplicated. Verify other pages (index.html, stats.html, trophies.html, driver.html) aren't broken. |
| TypeScript changes may introduce bugs | Each change targets specific functions. The `moveUp`/`moveDown` fix is a one-line change. Toast replacement is a search-and-replace pattern. |
