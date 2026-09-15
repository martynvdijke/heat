## 1. Config & deps

- [x] 1.1 Add `github.com/coreos/go-oidc/v3` + `golang.org/x/oauth2` to go.mod (skip if already present) and env config: OIDC_ENABLED, OIDC_ISSUER_URL, OIDC_CLIENT_ID, OIDC_CLIENT_SECRET_FILE, OIDC_REDIRECT_URL, OIDC_SCOPES.
- [x] 1.2 Wire env into docker-compose.yml (secrets via file/env, never in git); add example to .env.example/docs.
- [x] 1.3 Add DB migration: `oidc_sub` (text, unique, nullable), `auth_method` columns; index on email.

## 2. Backend OIDC flow

- [x] 2.1 Implement `/api/auth/oidc/login` (PKCE S256, state+nonce cookies, redirect to Authelia authorize).
- [x] 2.2 Implement `/api/auth/oidc/callback` (verify state/nonce, exchange code, verify ID token via JWKS/RS256, require email_verified, extract email/groups).
- [x] 2.3 Extend AuthMiddleware to accept OIDC sessions (same `session` cookie, check auth_method + oidc_sub). NOTE: no middleware change needed — OIDC sessions reuse the identical cookie/store, so existing AuthMiddleware accepts them as-is.
- [x] 2.4 Implement `/api/auth/oidc/logout` (clear local session, redirect to Authelia logout).

## 3. User linking/migration

- [x] 3.1 Link by verified email: match existing admin_users on email, link oidc_sub; first OIDC login auto-provisions user.
- [x] 3.2 Sync admin flag from `groups` claim containing `admins` on each login (and LEDit: map groups->roles per add-user-roles; traces: respect family-logins email key). NOTE: admin_users has no admin column (every row is admin), so implemented as login gate: non-`admins` members get 403, no provisioning.
- [x] 3.3 Keep password/session login enabled as fallback; ensure both methods set same session cookie shape.

## 4. Frontend + compose

- [x] 4.1 Add "Login with Authelia" button to login page; hide only when OIDC_ENABLED=false.
- [x] 4.2 Update docker-compose.yml and docs for OIDC env vars.

## 5. Homelab onboarding + verification

- [ ] 5.1 Register Authelia client: client_id heat, redirect https://heat.vandijke.xyz/api/auth/oidc/callback, scopes openid email profile groups, PKCE S256, client_secret_post/basic, consent explicit, policy one_factor.
- [ ] 5.2 Add NPM bypass for https://heat.vandijke.xyz (forward-auth would break OIDC callback); verify bypass.
- [ ] 5.3 Verify web login via Authelia, verify API tokens/session still work, verify logout clears session and redirects.
- [ ] 5.4 Keep password login ON until verified end-to-end; document rollback (set OIDC_ENABLED=false).
- [ ] 5.5 Verify public URLs return 200 anonymous; verify private mutations redirect to OIDC login (web 302) or 401 (API) — public: /, /api/race-info, /api/trmnl/summary; private: POST /api/* mutations

<!-- Public reads stay anonymous — OIDC is for SSO login only, not to gate public reads -->
