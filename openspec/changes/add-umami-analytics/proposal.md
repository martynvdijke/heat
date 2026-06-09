## Why

Umami is a privacy-focused, self-hosted analytics alternative to Google Analytics. The project has a basic Umami integration scaffold (settings model, admin tab, middleware for script injection) but lacks onboarding guidance, connection verification, and custom event tracking. This change enhances the existing integration to provide a complete analytics onboarding experience and ensures the application actually sends meaningful analytics data.

## What Changes

- Add a "Test Connection" button in the admin Analytics panel to verify the Umami instance is reachable and the website ID is valid
- Add a backend endpoint `POST /api/umami-settings/test` that probes the configured Umami URL
- Add input validation on the Umami URL (must be a valid URL) and Website ID (must be a valid UUID) both client- and server-side
- Replace the basic `alert()` save feedback with a proper toast notification
- Add client-side custom event tracking for key HEAT actions (race events, flag changes, page views on dynamic pages) using the Umami tracking API
- Add inline onboarding hints in the admin panel explaining how to find/set up Umami
- Add server-side validation that rejects malformed settings

## Capabilities

### New Capabilities
- `umami-connection-test`: Backend endpoint + frontend button to verify Umami connectivity and website ID validity
- `umami-custom-events`: Client-side custom event tracking for key HEAT application actions
- `umami-onboarding`: Guided setup hints and validation in the admin panel

### Modified Capabilities
- *(none — no existing specs are being modified)*

## Impact

- **Backend**: New handler endpoint for connection testing; validation added to existing save handler; new middleware or utility for server-side tracking events
- **Frontend**: Enhanced admin panel with test button, toast notifications, inline help text; new TypeScript module for custom event dispatch
- **Database**: No schema changes — existing `umami_settings` table is sufficient
- **Dependencies**: No new Go or JS dependencies required; uses standard `net/http` for connection test and the existing Umami tracking script API
