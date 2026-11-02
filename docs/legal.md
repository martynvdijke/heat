# Legal and privacy implementation

This document describes how HEAT implements its GDPR / Google Play legal
obligations. It is an operator and developer reference, not the published
notice. The published pages live in `static/templates/` and are rendered at:

| Page | Route | Auth |
| --- | --- | --- |
| Privacy notice (GDPR Art. 13) | `GET /privacy` | none |
| Terms of use | `GET /terms` | none |
| Account deletion instructions | `GET /account/delete` | none |

All three are plain public GETs, so a store reviewer and a locked-out user can
read them. They use the same `base.html` shell, stylesheet and footer as the
rest of the site.

## Configuration

The operator identity is read from the environment on startup
(`app/legal.go`) and exposed to the templates through `handlers/legal.go`:

| Variable | Meaning | Default |
| --- | --- | --- |
| `LEGAL_CONTROLLER_NAME` | Controller's legal or trading name | "the operator of this HEAT instance" |
| `LEGAL_CONTACT_EMAIL` | Privacy contact address | empty; the page then does not invent one |
| `LEGAL_CONTACT_ADDRESS` | Postal address | omitted |
| `LEGAL_HOSTING_REGION` | Where the server and DB are hosted | "the European Union" |
| `PUBLIC_BASE_URL` | Canonical public URL (`LEGAL_BASE_URL` accepted as a fallback) | `https://heat.vandijke.xyz` |

`.env.example` and `docker-compose.yml` contain the same variables. A deployment
must set the controller name and contact before submitting the app to a store.

Processor flags are not configuration: they are read from the live settings
tables on every request (`email_settings`, `telegram_settings`,
`notification_settings`, `umami_settings`, `otel_settings`, `wled_settings`,
`ai_settings`, plus `OIDC_ENABLED`). The policy therefore only names an
integration that is actually enabled. The CDN / Google Fonts disclosures are
unconditional because `base.html` always loads them.

## Account models

HEAT has two unrelated identity models.

### Admin accounts

- `admin_users`: `username`, bcrypt `password` hash, optional `email`,
  `oidc_sub`, `auth_method`.
- `web_sessions`: random session id, expiry, client IP, and (since this change)
  `admin_user_id`. The in-memory `SessionStore` mirrors it.
- Admin login: `POST /api/login` (password) or OIDC.
- Self-service deletion: `DELETE /api/account` with a signed-in session and
  `{"confirm":"DELETE"}`. It is refused while the account is the last admin.
  On success it deletes the admin row and its password-reset tokens, then calls
  `Server.DeleteAllSessions()` so no admin session survives its user. Race,
  racer and result data are untouched.
- Export: `GET /api/account/export` returns the profile only. Password hashes
  and session identifiers are never included.

A session created before `admin_user_id` was tracked cannot prove which admin
owns it, so `/api/account` and `/api/account/export` return 401 and ask the user
to sign in again. Re-login records the id.

### Racer identities

- `racers` plus `racer_emails`, `racer_stats`, `player_upgrades`,
  `racer_legend_abilities`, `racer_achievements`, `driver_shares`.
- Identity links: `telegram_links`, `telegram_link_tokens`, `player_sessions`,
  `telegram_notify_prefs`, `telegram_rsvp`.
- Gameplay records: `heat_cards`, `gear_shifts`, `lap_records`, `racer_sectors`,
  `turbo_logs`, `race_events`, `race_radio`.
- Login is passwordless: `POST /api/me/request-link` emails a single-use link
  (15 minutes). Verifying it (`POST /api/telegram/verify`) creates a
  `racer_session` cookie (30 days) and optionally links a Telegram chat.
- Mobile uses a `player_sessions` token sent as `X-Player-Token`.
- `GET /api/me`, `/api/me/export`, `/api/me/upgrades` and `DELETE /api/me`
  accept either the `racer_session` cookie or `X-Player-Token`
  (`middleware.RacerOrPlayerAuthMiddleware`). Both resolve to the same racer id.
- Self-service deletion: `DELETE /api/me` with `{"confirm":"DELETE"}`.

## Deletion cascade

`handlers/account.go:deleteRacer` runs in one SQLite transaction:

1. Find the racer's Telegram chat ids.
2. Delete `telegram_notify_prefs` and `telegram_rsvp` for those chats; delete
   `telegram_links`, `telegram_link_tokens`, and any remaining `telegram_rsvp`
   rows for the racer id.
3. Delete `racer_emails`, `player_sessions`, `player_upgrades`,
   `racer_legend_abilities`, `racer_stats`, `racer_achievements`,
   `driver_shares`, `heat_cards`, `gear_shifts`, `lap_records`, `turbo_logs`,
   `racer_sectors`, `race_radio`, and `race_events` that reference the racer.
4. Anonymise historical records instead of deleting them:
   `race_results` and `round_snapshot_scores` get `racer_id = 0` and
   `racer_name = 'Deleted racer'`. The competitive fact (who finished where)
   remains; the link to a person does not.
5. Delete the `racers` row.

After commit the handler drops every in-memory racer session for that id and
clears the caller's cookie. The same cascade is used by the admin racer-delete
endpoints (`DELETE /api/racers?id=` and the HTMX delete), so an administrator
cannot leave orphaned email or Telegram links behind.

An optional-table-not-found error in a cascade statement is ignored so an
upgrade from an older database still succeeds. Any other SQL error aborts and
rolls back the transaction.

## Retention

| Data | Retention |
| --- | --- |
| Admin account, racer identity | Until self-service deletion or operator action |
| Admin web sessions | 24 hours |
| Racer sessions | 30 days |
| Racer email / Telegram link tokens | 15 minutes, single use |
| Admin password-reset tokens | 30 minutes, single use |
| Race results / round scores after racer deletion | Indefinite, anonymised and detached |
| Server logs | Per instance log settings |
| Backups | `backup_settings.retention_count` (default 7), pruned oldest first |

**Backups and erasure.** Deletion removes live rows immediately, but an existing
database backup can still contain the old data until it is pruned. This is
disclosed on `/account/delete` and `/privacy`. The operator must not restore a
backup solely to resurrect deleted identities, and should shorten retention if a
shorter erasure window is promised.

## Processor inventory

Only list a processor in an operator's own DPA paperwork when the matching
integration is enabled:

- SMTP provider (`email_settings`) — sign-in links, resets, race email.
- Telegram Bot API (`telegram_settings`) — chat id, username, first name and
  message content.
- Gotify (`notification_settings.gotify_url`) — race notifications, usually
  operator-hosted.
- Umami (`umami_settings`) — page views and request metadata.
- OIDC / Authelia (`OIDC_ENABLED`) — subject, email, groups.
- OpenTelemetry collector (`otel_settings`) — traces, metrics, logs.
- WLED (`wled_settings`) — race-state commands on the LAN, no profile data.
- AI endpoints (`ai_settings`, `AI_TRACK_EXTRACT_URL` / `AI_TEXT_GEN_URL`) —
  submitted text/images.
- jsDelivr, cdnjs and Google Fonts — always contacted by the browser shell.

## Testing

Tests live in `handlers/legal_test.go` / the root package and cover:

- public legal pages render without a session and include the configured
  controller;
- racer deletion removes identity links and anonymises results;
- the last admin cannot delete their account, and a successful admin deletion
  revokes every session;
- both export endpoints omit secrets.

Run `go test ./...`.
