## Context

The HEAT project already ships a minimal Umami analytics integration: a `UmamiSettings` struct (URL, WebsiteID, Enabled), an admin Settings tab with inputs, a middleware that injects the tracking `<script>` tag into HTML pages, and database persistence. However, the integration has gaps:

- No connection verification — users enter a URL and WebsiteID blindly
- No client-side feedback beyond a browser `alert()` on save
- No custom event tracking for in-app actions (race starts, flag changes, etc.)
- No input validation before saving

This design covers the enhancements needed to make the integration production-ready.

## Goals / Non-Goals

**Goals:**
- Allow admins to test the Umami connection from the admin panel
- Validate Umami URL (must be a valid, reachable HTTP(S) URL) and Website ID (must be a valid UUID) before saving
- Replace `alert()` with a toast notification system for save feedback
- Add custom event tracking for key HEAT application actions using Umami's tracking API
- Provide inline onboarding hints to guide new users through setup

**Non-Goals:**
- Replacing or removing the existing Umami settings model or database schema
- Adding a full Umami dashboard or analytics reporting within HEAT
- Tracking server-side events (all event tracking is client-side via the Umami snippet)
- Supporting Google Analytics or other analytics providers

## Decisions

### Decision 1: Connection test via HTTP HEAD probe to Umami script URL
- **Chosen**: New `POST /api/umami-settings/test` endpoint that sends an HTTP HEAD/GET request to `<umami_url>/script.js` and reports whether the server responds (2xx) and whether the response contains expected script content.
- **Alternatives considered**:
  - **Umami API /api/verify endpoint**: Umami does not have a standard public verify endpoint. Using `/script.js` is the most reliable generic check.
  - **Client-side only test**: Could be done via JS `fetch` from the browser — but this would expose the Umami URL to CORS issues. Server-side proxy is more reliable.
- **Rationale**: A simple connectivity check gives high confidence without requiring auth credentials. The `/script.js` path is guaranteed to exist on any Umami instance.

### Decision 2: Toast notifications instead of native alert()
- **Chosen**: A lightweight CSS-based toast notification system implemented directly in TypeScript, avoiding any new dependency (like toastr or notyf).
- **Alternatives considered**: Bootstrap toasts (requires Bootstrap JS), third-party libs.
- **Rationale**: Minimizes dependencies. A 20-line toast component is sufficient. The app already uses Bootstrap CSS which provides styling primitives.

### Decision 3: Custom events dispatched via the Umami `umami.track()` API
- **Chosen**: After the Umami tracking script loads, it exposes `window.umami.track(eventName, data)`. We dispatch events for HEAT actions when they occur in the browser (race start, flag change, lap record, etc.).
- **Alternatives considered**: Send events to a custom backend endpoint.
- **Rationale**: No additional backend infrastructure needed. The Umami script already handles batching and network transport. Events appear directly in the Umami dashboard.

### Decision 4: Server-side validation on save
- **Chosen**: The `SaveUmamiSettings` handler validates URL format and Website ID UUID format before persisting. Returns 400 with a descriptive error message.
- **Alternatives considered**: Client-side only validation.
- **Rationale**: Defense-in-depth. Client-side validation provides UX feedback; server-side prevents bad data from reaching the database regardless of client.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Connection test endpoint could be used to probe internal networks (SSRF) | The endpoint only makes outbound requests to the user-configured URL. Rate-limiting can be added if needed. No user-supplied paths are accepted — the URL is used as-is with a fixed `/script.js` suffix. |
| Custom events could cause performance issues if too many fire rapidly | Events are dispatched synchronously but the Umami script queues them asynchronously. We throttle to reasonable events only (not every UI interaction). |
| Toast notifications could conflict with existing UI | Use z-index overlay and auto-dismiss after 3 seconds. Position fixed in top-right corner to avoid layout shifts. |
| Umami URL validation might reject valid but non-standard URLs | Use Go's `url.ParseRequestURI` which is permissive enough for standard Umami deployments. Allow HTTPS only for security. |
