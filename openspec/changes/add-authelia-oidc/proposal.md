## Why
Heat currently uses local password/session auth (see handlers/auth.go + middleware/auth.go), requiring per-app passwords alongside homelab forward-auth. This creates double-login, breaks SSO, and duplicates identity management. Homelab IdP is Authelia at https://authelia.vandijke.xyz (see /root/homelab/openspec/changes/authelia-oidc/proposal.md) — Heat should become a native OIDC relying party.

## What Changes
- Add OIDC login via Authelia discovery (issuer https://authelia.vandijke.xyz, scopes `openid email profile groups`, PKCE S256, `client_secret_post` or `client_secret_basic` per Authelia template).
- Auto-provision/link users by verified email (match existing admin_users on email; first OIDC login creates user, admin flag follows `groups` claim containing `admins`).
- Keep existing password/session login enabled until OIDC verified end-to-end (fallback + rollback path).
- Add env-based config: OIDC_ENABLED, OIDC_ISSUER_URL, OIDC_CLIENT_ID, OIDC_CLIENT_SECRET_FILE, OIDC_REDIRECT_URL, OIDC_SCOPES; secrets via file/env, never in git. Wire into docker-compose.yml.
- Add `/api/auth/oidc/login`, `/api/auth/oidc/callback`, `/api/auth/oidc/logout` handlers reusing existing session cookie; extend AuthMiddleware to accept OIDC sessions.
- Document Authelia client registration (client_id heat, redirect https://heat.vandijke.xyz/api/auth/oidc/callback, consent explicit, policy one_factor->one_factor) + NPM bypass step for the host.

- Public GETs stay anonymous (landing, public feeds, shared links); OIDC only replaces/augments login for private mutations + user-scoped pages. No new login wall on public content (e.g. `/`, `/api/race-info`, `/api/trmnl/summary`).

## Capabilities
### New Capabilities
- `oidc-authentication`
### Modified Capabilities
- `password-authentication` (fallback only)

## Impact
Backend auth handlers, middleware, user persistence, docker-compose env, docs. Frontend login page gains "Login with Authelia" button. No breaking change while fallback stays on.

<!-- Public reads stay anonymous — OIDC is for SSO login only, not to gate public reads -->
