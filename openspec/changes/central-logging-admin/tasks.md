## 1. Database Layer

- [x] 1.1 Create `app_logs` table migration in `db/init.go` (columns: id INTEGER PK, timestamp TEXT, level TEXT, module TEXT, message TEXT, data TEXT nullable, trace_id TEXT nullable)
- [x] 1.2 Create `log_settings` table migration (columns: id INTEGER PK, module TEXT UNIQUE, level TEXT default 'WARN')
- [x] 1.3 Add `LogEntry` model struct to `models/models.go`
- [x] 1.4 Add `LogSetting` model struct to `models/models.go`

## 2. Logger Package

- [x] 2.1 Create `pkg/logger/logger.go` with leveled structured logger (Debugf, Infof, Warnf, Errorf methods taking module, message, optional data)
- [x] 2.2 Implement level filtering — check per-module setting from in-memory cache (default WARN); only emit if entry level >= configured level
- [x] 2.3 Implement SQLite writer that INSERTs into `app_logs` table with auto-prune to 10k rows
- [x] 2.4 Implement OTel bridge writer that creates OTel log records via the SDK Logger
- [x] 2.5 Wire both writers into the logger (async: buffered channel + background goroutine)

## 3. OTel Integration

- [x] 3.1 Initialize OTel SDK in `main.go` before Gin — set up stdouttrace exporter and logger provider
- [x] 3.2 Create `telemetry.go` OTel init function with graceful fallback (non-fatal on error)
- [x] 3.3 Connect logger to OTel log record pipeline — each log call produces an OTel log record with severity, body, and module attribute

## 4. API Endpoints

- [x] 4.1 Create `handlers/logs.go` — implement `GET /api/admin/logs` with pagination (page, pageSize params) and filtering (level, module params)
- [x] 4.2 Create `handlers/logs.go` — implement `GET /api/admin/logs/modules` returning distinct module names
- [x] 4.3 Create `handlers/log_settings.go` — implement `GET /api/admin/log-settings`
- [x] 4.4 Implement `POST /api/admin/log-settings` with validation of level values
- [x] 4.5 Register all new routes in `main.go` under the admin group (behind AuthMiddleware)

## 5. Migrate log.Printf Calls to Structured Logger

- [x] 5.1 Replace `log.Printf` in `handlers/email.go` (SendRaceEmail, sendSMTP failures) with `logger.Warnf` / `logger.Errorf`
- [x] 5.2 Replace `log.Printf` in `main.go` (backup loop, server start) with structured logger calls
- [x] 5.3 Replace `log.Printf` in backup.go, ws/manager.go, and other service files

## 6. Admin UI — "Logs" Tab

- [x] 6.1 Add "Logs" nav-tab button in `static/admin.html` with fa-list icon, after the Backup tab
- [x] 6.2 Add log viewer pane HTML (table, filter controls, pagination, refresh button)
- [x] 6.3 Add log settings sub-section within the Logs pane (verbosity per module table with dropdowns)
- [x] 6.4 Implement log viewer JS: fetch logs, auto-refresh polling (5s), filter by level/module, pagination
- [x] 6.5 Implement log settings JS: load current settings, change module level, save via POST
- [x] 6.6 Add `logs-tab` event listener in `admin.js` to trigger load on tab activation / stop polling on deactivation

## 7. Onboard Settings Endpoints to Central Logging

- [x] 7.1 Add structured log calls to Email settings handlers (GetEmailSettings, SaveEmailSettings)
- [x] 7.2 Add structured log calls to AI settings handlers (GetAISettings, SaveAISettings)
- [x] 7.3 Add structured log calls to Notification settings handlers (GetNotificationSettings, SaveNotificationSettings, TestNotification)
- [x] 7.4 Add structured log calls to Backup settings handlers (GetBackupSettings, SaveBackupSettings, TriggerManualBackup)
- [x] 7.5 Add structured log calls to Umami analytics settings handlers (GetUmamiSettings, SaveUmamiSettings)
