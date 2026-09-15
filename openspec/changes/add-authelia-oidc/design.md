## Context

Heat is a Go + session-cookie app with local password auth (handlers/auth.go, middleware/auth.go). Homelab forward-auth via Authelia at https://authelia.vandijke.xyz causes double-login and breaks non-browser clients. This change makes Heat a native OIDC relying party against Authelia, aligning with /root/homelab/openspec/changes/authelia-oidc. Port: 3000, host: https://heat.vandijke.xyz.
## Goals / Non-Goals

### Goals

- Single sign-on via Authelia for browser login; eliminate double-login.
- Email-verified identity; auto-provision and link by email.
- Groups -> admin mapping (`admins` group grants admin flag).
- Keep password login until OIDC verified (safe cutover/rollback).
- Secrets never in git (file/env via OIDC_CLIENT_SECRET_FILE).

- Preserve anonymous public reads

### Non-Goals

- Removing password auth in this change (follow-up after verification).
- Replacing Google OAuth where it exists (datey/dnd) — that stays for sync.
- Changing public/read endpoints or CSRF handling.

- Forcing login for public content

## Decisions

- Library: `github.com/coreos/go-oidc/v3` + `golang.org/x/oauth2` (already a dependency for dnd/datey/youtube; add to go.mod elsewhere). RS256 verification via Authelia JWKS discovered from `OIDC_ISSUER_URL/.well-known/openid-configuration`.
- Flow: Authorization Code + PKCE S256, `state` + `nonce` cookies (httpOnly, secure, sameSite=Lax), `client_secret_post` (fallback `client_secret_basic` per Authelia template).
- Session reuse: same `session` cookie as password auth; add columns `auth_method` (`oidc`|`password`) and `oidc_sub` (issuer+sub). AuthMiddleware accepts either; no parallel session store.
- Email linking: require `email_verified=true`; lookup `admin_users` (or equivalent users table) by email. If found, link `oidc_sub` to existing row; if not, create new user. On every login, sync `admin` flag from `groups` claim containing `admins`.
- Groups->roles: generic `groups` claim passthrough; `admins` maps to admin. LEDit additionally maps groups to roles per add-user-roles.
- Logout: clear local session cookie, then redirect to Authelia `https://authelia.vandijke.xyz/logout` (or ` /api/oidc/logout` if configured) with post-logout redirect to app.

- AuthMiddleware composition — OptionalAuth/public routes unchanged, AuthRequired/StaffRequired routes accept OIDC session as alternative to password session

## Risks / Trade-offs

- Authelia JWKS/issuer misconfig breaks login — mitigated by OIDC_ENABLED toggle and password fallback.
- Email mismatch creates duplicate users — mitigated by verified-email requirement and admin doc to align Authelia email with existing user email.
- NPM forward-auth double-auth breaks OIDC callback — requires NPM bypass for the host before enabling OIDC.
- Secrets sprawl — standardize on `OIDC_CLIENT_SECRET_FILE` pointing at Docker secret/env file, never committed.

## Migration Plan

1. Add deps, config, DB columns (`oidc_sub`, `auth_method`), and OIDC handlers behind `OIDC_ENABLED=false` default.
2. Register Authelia client (`client_id` = app, redirect `https://<host>/api/auth/oidc/callback`, consent explicit, policy `one_factor`).
3. Add NPM bypass for the host, enable OIDC in compose, verify login/logout end-to-end.
4. Keep password fallback on; remove only after successful verification period.

## Open Questions

- Should `groups` beyond `admins` map to finer-grained roles now or in a follow-up?
- Token refresh / offline_access scope needed? Default no — session lifetime covers it; add only if long-lived API tokens are required.

<!-- Public reads stay anonymous — OIDC is for SSO login only, not to gate public reads -->
