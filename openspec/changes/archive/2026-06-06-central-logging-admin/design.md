## Context

The HEAT racing companion app uses ad-hoc `log.Printf` statements scattered across handlers and services. There is no centralized logging system. OTEL Go dependencies (`go.opentelemetry.io/otel`, `otel/sdk`, `otel/exporters/stdout/stdouttrace`) are already present in `go.mod` as direct dependencies but are completely unwired.

The admin panel has 13+ tabs managed via Bootstrap nav-tabs + static JS. New capabilities (logs, settings, OTel) need to integrate into this pattern without breaking existing functionality. The app uses SQLite via `database/sql` directly (not Ent for all operations) and Gin for HTTP routing.

## Goals / Non-Goals

**Goals:**
- Add a "Logs" tab to the admin panel showing application logs in real-time
- Capture all existing `log.Printf` calls into a structured logger that writes to both a SQLite `app_logs` table and OTel
- Create a log settings endpoint for per-module verbosity control (default: WARN+)
- Wire existing OTel SDK into the app so logs ship as OTel log records
- Add API endpoints for fetching logs (paginated, filterable by level/module) and managing settings
- Onboard the following settings endpoints into the central logging view (i.e., all their activity gets logged): Email, AI, Notification, Backup, Analytics (Umami)

**Non-Goals:**
- Replacing all logging calls in external dependencies (Ent, Gin, etc.) — only the app's own `log.Printf` calls will be captured
- Full OTLP exporter to an external collector — this design uses stdout exporter as a foundation; OTLP can be added later
- Real-time WebSocket streaming of logs — polling with auto-refresh is sufficient for v1
- Log rotation or retention policies (will store with a capped row count)
- Distributed tracing beyond what OTel provides by default

## Decisions

### 1. Structured Logger over intercepting `log.Printf`
**Decision:** Create a `pkg/logger` package with a leveled, structured logger that replaces direct `log.Printf` calls. The logger writes to both SQLite and OTel simultaneously.
**Rationale:** Wrapping `log.Printf` via `log.SetOutput` would capture output but lose structured fields (level, module, request ID). A dedicated logger gives structured, queryable logs.
**Alternatives:** Intercepting via `log.SetOutput` — simpler but loses structure. Using a full framework like zap/zerolog — adds dependency; the app already has OTel and only needs basic leveled logging.

### 2. Log persistence in SQLite `app_logs` table
**Decision:** Store logs in a new `app_logs` table with columns: id, timestamp, level, module, message, data (JSON), created_at.
**Rationale:** The app already uses SQLite with `database/sql`. No new database dependency. Simple bounded table (auto-prune to 10,000 rows).
**Alternatives:** Separate log file — can't query from admin UI. External log service — adds deployment complexity for a single-binary app.

### 3. OTel wiring: Bridge logs to OTel log record
**Decision:** In `telemetry.go`, initialize an OTel logger provider using the existing stdout exporter. Each app log call creates a log record routed through the OTel SDK.
**Rationale:** OTel deps are already present. The stdout exporter provides immediate visibility without external collector setup. The architecture supports swapping to OTLP exporter later.
**Alternatives:** No OTel — defeats the requirement to "end up in OTel." Direct OTLP — requires a collector endpoint, over-engineering for v1.

### 4. Verbosity levels stored per-module
**Decision:** Store log settings in a new `log_settings` table with columns: module (TEXT), level (TEXT: DEBUG/INFO/WARN/ERROR). Unconfigured modules default to WARN.
**Rationale:** Simple key-value model. Easy to query from settings API. Default WARN matches the stated requirement.
**Alternatives:** Single global level — less granular. Environment variables — can't change at runtime from admin UI.

### 5. Admin tab follows existing pattern
**Decision:** Add a "Logs" nav-tab in `admin.html` with a corresponding pane. Add API routes `/api/admin/logs` and `/api/admin/log-settings` under the existing admin group. Load log data via `fetch()` on tab activation.
**Rationale:** All existing admin features use this exact pattern (see admin.js tab event listeners). No new frontend framework needed.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Log writes slow down request handlers | Performance regression | Log writes are async (buffered channel + background goroutine flushes to DB/OTel) |
| `app_logs` table grows unbounded | Disk usage | Auto-prune to 10k rows on each write; configurable cap |
| Breaking existing `log.Printf` callers | Runtime errors | String replace all `log.Printf` → `logger.Infof(...)` etc. Keep `log` import only where appropriate |
| OTel init fails silently | No logs exported | Initialize at startup, log init failure via `log.Printf` before logger is active, but don't crash |
| Log verbosity changes don't take effect until restart | UX friction | Re-read settings on each log call (cheap, cached in map with mutex) |
