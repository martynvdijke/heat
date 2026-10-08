# Telegram Racer Identity & Personal Area — Design

- **Date:** 2026-10-08
- **Status:** Approved (design sign-off given)
- **Related OpenSpec change (local workflow artifact):** `openspec/changes/telegram-racer-identity/`

## Problem & Goal
Racers can see aggregate results on public pages, but there is no way for an individual racer
to prove who they are and get a personal view of their own results and upgrades. This feature
lets a racer authenticate from a Telegram private chat using an email verification link, then
access their personal stats and track their upgrades from Telegram and/or the website.

## Decisions
1. **Identity is matched by the email on file.** A successful verification maps the Telegram
   chat to an existing racer via `racer_emails` (admin-managed). No free-form accounts.
2. **One website verification URL completes both the Telegram link and the web session.** The
   bot stores the pending `chat_id` on the token; confirming the link links the chat and signs
   the browser in.
3. **Scanner-safe verification:** `GET …/verify/validate` has no side effects; only
   `POST …/verify` consumes the token. Mirrors the existing password-reset pattern.
4. **Racer sessions are isolated from admin sessions:** a dedicated `RacerSessions` store,
   `racer_session` cookie, and `RacerAuthMiddleware`. A racer session can never satisfy admin
   routes. Racer sessions are not IP-bound (mobile IPs rotate); admin sessions keep IP binding.
5. **Reads stay public; auth gates linking and writes only.** All personal *read* data is
   already public API, so the bot fetches it over its existing path. Only upgrade *writes*
   require a racer session, and the racer id always comes from the session, never the body.
6. **Tokens are 32 random bytes hex, 15-minute TTL, single-use.** Generic anti-enumeration
   responses on request endpoints.
7. **The bot starts linking via an internal, bot-token-gated endpoint.** This prevents an
   attacker from pre-seeding a victim's email with the attacker's `chat_id`.
8. **`PUBLIC_BASE_URL` config** supplies publicly reachable links for emails (the bot calls the
   server over localhost). Falls back to request-derived absolute URLs; required for the
   Telegram-initiated flow.
9. **Upgrade management mirrors existing admin rules** (`card_type='upgrade'`, extension owned,
   not already owned); bought rows are created equipped, matching current admin behavior.

## Data Model (`db/init.go`)
```sql
telegram_link_tokens(id, token TEXT UNIQUE, racer_id, chat_id TEXT DEFAULT '',
                     created_at, expires_at, used DEFAULT 0)   + INDEX(racer_id)
telegram_links(chat_id TEXT PK, racer_id, linked_at)            + INDEX(racer_id)
```
- One chat maps to one racer; a racer may have several chats.
- `chat_id` is empty for website-initiated ("email me a link") tokens.

## Auth & Sessions
- `app.RacerSession{RacerID int; Expiry int64; IP string}` plus `Server.RacerSessions` and
  `RacerSessionsMu`, initialised in `NewServer` and pruned by the existing 15-minute janitor.
- Cookie `racer_session`: HttpOnly, `SameSite=Strict`, `Path=/`, TTL 30 days, `Secure` when
  `SecureCookies`.
- `middleware.RacerAuthMiddleware(s)` validates the cookie/expiry and sets `racer_id` in the
  Gin context.

## Flows
**Telegram link flow**
1. Racer sends `/login`; bot starts a guided prompt (or accepts `/login <email>`).
2. Bot calls `POST /api/telegram/link/start` with `X-Bot-Token`; server matches the email,
   creates a token with the chat id, and emails `/verify.html?token=…`.
3. `/verify.html` calls the side-effect-free validate endpoint and shows a **Confirm sign-in**
   button.
4. Confirm POSTs `/api/telegram/verify`; the server consumes the token, upserts
   `telegram_links`, sets the `racer_session` cookie, enqueues an `identity_linked` Telegram
   event, and redirects to `/me.html`.
5. The bot confirms sign-in in the chat.

**Website re-login flow:** `POST /api/me/request-link` sends the same link with an empty
`chat_id`; confirming sets the web session and links no chat.

**Personal reads:** bot reads public endpoints (`/api/racer-stats`, `/api/racer-recent-results`,
`/api/telegram/summary`, `/api/player-upgrades`). **Personal writes:** web only, via
session-scoped `/api/me/upgrades/*`.

## HTTP Endpoints
Bot-only (`X-Bot-Token`, constant-time match against `telegram_settings.bot_token`):
- `POST /api/telegram/link/start` `{email, chat_id}`.

Public (rate-limited, anti-enumeration):
- `POST /api/me/request-link` `{email}`
- `GET  /api/telegram/verify/validate?token=` → `{valid, racer_name}` (no side effects)
- `POST /api/telegram/verify` `{token}`
- `GET  /api/racer-recent-results?racer_id=&limit=`

Session-protected (`RacerAuthMiddleware`, CSRF on writes):
- `GET  /api/me`
- `GET  /api/me/upgrades`
- `POST /api/me/upgrades/buy` `{upgrade_id, season_id, round}`
- `PUT  /api/me/upgrades/toggle` `{id, equipped}`
- `POST /api/me/logout`

## Telegram Commands
Registered in `commandRegistry` (category `catBot`, so `/help` and the native menu include
them): `/login`, `/logout`, `/mystats`, `/myupgrades`. `/mystats` and `/myupgrades` prompt
`/login` when the chat is unlinked. `handleEvent` gains an `identity_linked` kind.

## Website Pages
- `static/verify.html` + `ts/verify.ts` — validate, confirm, POST, redirect to `/me.html`.
- `static/me.html` + `ts/me.ts` — session check; email-link form when signed out; dashboard
  with career stats, season standing, recent results, points progression, head-to-head, and
  upgrade buy/equip when signed in.
- Both served raw like `/driver.html`; the existing `/driver.html?token=` share page is
  untouched.

## Config
- Optional `PUBLIC_BASE_URL` env; documented in `README.md` and `docker-compose.yml`
  (e.g. `https://heat.vandijke.xyz`). Required for the Telegram-initiated flow.

## Security & Edge Cases
Single-use 15-minute tokens; case-insensitive email match; generic responses; POST-only
confirm; bot-token-gated link start; no body-supplied racer ids; no logging of emails/tokens.
Changing the email on file invalidates old links. Unlinked/departed chats degrade gracefully.

## Testing
- Go: token issue/verify/consume, chat linking, racer-session middleware and admin isolation,
  upgrade buy/toggle ownership, migration.
- Telegram: guided `/login`, `/mystats` & `/myupgrades` linked vs unlinked, `identity_linked`.
- Playwright: `verify.html` confirm → `/me.html` sign-in and upgrade buy/equip.
- Run `task pre-push` (gofmt, tests, vet + govulncheck, TS compile, build).

## Out of Scope (YAGNI)
Passwords, OTP-code or `t.me` deep-link linking, upgrading from Telegram, refunds/deletion,
admin authorization changes, multiple racers per chat.
