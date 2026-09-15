## 1. Workflow changes

- [x] 1.1 `.github/workflows/release.yaml`: added a `trml` job between `release` and `otel` — `runs-on: ubuntu-latest`, `needs: [release]`, permissions `contents: read`; steps: `actions/checkout@v7` → `ruby/setup-ruby@v1` (`ruby-version: "4.0"`) → `gem install trmnl_preview` → `trmnlp lint` → `trmnlp push --force` with `env: TRMNL_API_KEY: ${{ secrets.TRML_TOKEN }}`
- [x] 1.2 `.github/workflows/release.yaml`: changed `otel` to `needs: [ci, release, notify, trml]` (kept `if: always()`)
- [x] 1.3 Verified `ruby-version: "4.0"` is available (Ruby 4.0.0 released 2025-12-25; latest 4.0.6) — no fallback needed
- [x] 1.4 Lint and push steps run with `working-directory: trmnl` (trmnlp project root)

## 2. Plugin project layout (trmnlp requirements)

- [x] 2.1 Restructured `trmnl/` into a trmnlp project: templates + `settings.yml` moved into `trmnl/src/`
- [x] 2.2 Added `trmnl/.trmnlp.yml` (watch `.trmnlp.yml` + `src`, `custom_fields: {url: ""}`, `variables.trmnl`)
- [x] 2.3 Shortened `settings.yml` description to `"Latest race results and standings"` (≤ 35 chars required by lint)

## 3. Secrets

- [x] 3.1 `TRML_TOKEN` Actions secret exists (verified via `gh secret list`, added 2026-08-15)

## 4. Verification

- [x] 4.1 Confirmed plugin source is in place: `trmnl/src/full.liquid`, `half_horizontal.liquid`, `half_vertical.liquid`, `quadrant.liquid`, `settings.yml` + `trmnl/.trmnlp.yml`
- [x] 4.2 `trmnlp lint -d trmnl` (0.11.0) passes: "✓ All checks passed!"
- [x] 4.3 `trmnlp push -d trmnl --force` with dummy `TRMNL_API_KEY` passes auth gate and reaches the API (401 "Invalid API key") — env-var auth path confirmed
- [x] 4.4 `task pre-push` passes (gofmt, go test, vet+govulncheck, tsc, frontend build, sw-precache, go build)
- [ ] 4.5 Push to `main`; in the Actions UI confirm the `trml` job runs after `release` and before `otel`, and that the plugin appears updated in the TRMNL dashboard — **user action**
- [ ] 4.6 Confirm a forced failure (e.g. temporary secret removal) still lets `otel` run and report — **user action**
