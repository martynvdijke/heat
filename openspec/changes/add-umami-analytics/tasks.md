## 1. Backend — Connection Test Endpoint & Validation

- [ ] 1.1 Add `POST /api/umami-settings/test` route in `main.go` under the admin group
- [ ] 1.2 Implement `HandleTestUmamiConnection` handler in `handlers/settings.go` that sends HTTP HEAD/GET to `<umami_url>/script.js` and reports connectivity
- [ ] 1.3 Add server-side validation in `SaveUmamiSettings` to reject invalid URL format (parse with `url.ParseRequestURI`, require HTTPS)
- [ ] 1.4 Add server-side validation in `SaveUmamiSettings` to reject Website ID that is not a valid UUID format
- [ ] 1.5 Add helper function to validate UUID format (or use existing library)

## 2. Frontend — Toast Notification System

- [ ] 2.1 Create toast notification TypeScript helper in `ts/` (e.g., `ts/toast.ts`) with `showToast(message, type, duration)` function
- [ ] 2.2 Add toast CSS styles in the admin panel or a shared stylesheet (position fixed, top-right, color-coded success/error, auto-dismiss animation)
- [ ] 2.3 Wire toast into existing settings forms to replace `alert()` calls (start with Umami, then generalize)

## 3. Frontend — Connection Test Button & Form Validation

- [ ] 3.1 Add "Test Connection" button to the Analytics tab in `static/admin.html` next to the Save button
- [ ] 3.2 Implement test connection click handler in `ts/admin.ts` that calls `POST /api/umami-settings/test` and displays result via toast
- [ ] 3.3 Add client-side URL format validation in the Umami form (disable submit if invalid)
- [ ] 3.4 Add client-side UUID validation for Website ID field
- [ ] 3.5 Add inline error styles for invalid fields (red border, error message below input)
- [ ] 3.6 Replace the `alert()` in the Umami form's submit handler with toast notification

## 4. Frontend — Custom Event Tracking

- [ ] 4.1 Create `ts/analytics.ts` module with `trackUmamiEvent(eventName, data)` helper that safely wraps `window.umami.track()`
- [ ] 4.2 Add flag change tracking: dispatch `umami.track('flag_change', ...)` in the flag handler (find flag UI code)
- [ ] 4.3 Add race start/stop tracking in the race control UI
- [ ] 4.4 Add lap record tracking in the lap recording UI
- [ ] 4.5 Ensure all `trackUmamiEvent` calls are no-ops when Umami is disabled or script is not loaded

## 5. Frontend — Onboarding Hints

- [ ] 5.1 Add welcome banner/info box above the Analytics form in `static/admin.html` (visible when URL and Website ID are both empty)
- [ ] 5.2 Add inline hint text below each input field explaining where to find the value
- [ ] 5.3 Toggle welcome banner visibility based on whether settings are populated (via `loadUmamiSettings`)
- [ ] 5.4 Update the "About Umami" card with more detailed setup instructions linking to umami.is

## 6. Tests

- [ ] 6.1 Add Go test for `POST /api/umami-settings/test` endpoint in test file
- [ ] 6.2 Add Go test for server-side URL/Website ID validation in save handler
- [ ] 6.3 Extend Playwright test in `tests/admin.spec.ts` to verify connection test button exists and toast notifications render
- [ ] 6.4 Extend Playwright test to verify client-side validation rejects invalid inputs
