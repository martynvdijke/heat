## Why

The admin dashboard and race controller UI have accumulated several UX issues that make them harder to use than they should be:

- **controller.html** has duplicate Quick Action buttons (Safety Car, Red Flag, Chequered Flag each appear twice) and duplicate HTML IDs (`safety-btn`, `redflag-btn`), which violates HTML spec and causes undefined behavior
- The controller page's 12+ sections are stacked vertically with no logical grouping or visual hierarchy — weather, turbo logs, gear shifts, sound effects, race events, and settings all blended together
- **admin.html** has 16+ flat tabs in a single row, making navigation overwhelming — settings tabs (Backup, Logs, Email, Telemetry, Analytics) are mixed with primary race-management tabs (Racers, Tracks, Stats)
- Widespread use of `alert()` for feedback is interruptive and disruptive during live race control
- Inconsistent spacing, font sizing, and card styling between the two pages
- Some sections in the controller (e.g., Quick Actions) waste space with poor grid layout
- Missing error handling and loading states in TypeScript code

This change cleans up both UIs to be more organized, visually consistent, and pleasant to use.

## What Changes

### Admin Tab Reorganization
- Group 16+ tabs into categorized sections: **Race** (Race Setup, Qualification, Racers, Tracks), **Results** (Stats, Rounds, Seasons), **Content** (Quotes, Teams), **Settings** (Notifications, Email, Telemetry, Analytics, AI, Backup), **System** (Logs)
- Use a secondary tab bar, dropdown groups, or accordion for settings to reduce visual noise
- Make the active tab more visually prominent
- Ensure tab bar wraps gracefully on smaller screens

### Controller Card Reorganization
- Group related sections: **Race Controls** (start/pause/stop + stats), **Live Data** (standings, weather), **Tracking** (turbo, gear, events, lap recording), **Actions** (quick actions, driver flags), **Settings** (race settings, season)
- Add visual separators and section headers
- Remove duplicate Quick Action buttons
- Fix duplicate HTML IDs

### Visual Polish & Consistency
- Move common styles from inline `<style>` blocks into `style.css`
- Standardize card spacing, button sizing, and typography between admin and controller pages
- Improve responsive layout for both pages
- Add subtle hover states and transitions for interactive elements

### UX Improvements
- Replace `alert()` calls with a toast notification system (for both pages)
- Add loading spinners for async operations
- Fix the `moveUp`/`moveDown` position-swap bug in `controller.ts`
- Add basic error handling for API calls (show toast on failure)
- Add confirmation dialogs for destructive actions (discard race, delete racer)

## Capabilities

### New Capabilities
- *(none — no new features, only UX polish)*

### Modified Capabilities
- `admin-ui`: Reorganized tab structure, categorized navigation, consistent styling
- `controller-ui`: Cleaned up card layout, removed duplicates, consistent styling
- `ux-toast-notifications`: Replaces `alert()` across both pages with non-blocking toast messages

## Impact

- **Frontend (HTML)**: `static/admin.html` — restructure tab bar into groups, consistent card markup. `static/controller.html` — remove duplicates, regroup sections, clean up inline styles.
- **Frontend (CSS)**: `static/style.css` — add shared styles extracted from inline blocks, toast styles, consistent card/button theming.
- **Frontend (TypeScript)**: `ts/admin.ts`, `ts/controller.ts` — replace `alert()` with toast calls, add error handling, fix position-swap bug, add loading states.
- **New file**: `ts/toast.ts` — shared toast notification utility.
- **Backend**: No changes.
- **Database**: No schema changes.
