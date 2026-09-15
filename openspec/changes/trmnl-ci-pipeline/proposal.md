## Why

The TRMNL plugin source (Liquid templates + `settings.yml`) lives in `trmnl/` but is never validated or deployed from CI. Layout errors ship untested to the device, and pushing updates to the TRMNL cloud is a manual step that gets skipped. The `trmnl_preview` Ruby gem ships a `trmnlp` CLI that lints the templates and force-pushes them to the TRMNL account via the API. The release pipeline already has an `otel` job that exports the run trace — the TRMNL deploy should slot in before that export, on release.

## What Changes

- **New `trml` job in `.github/workflows/release.yaml`**, placed between the `release` job and the `otel` export job, named `trml` (matches the pasted workflow intent, "called on release before the otel export").
- **Job steps**: checkout → `ruby/setup-ruby@v1` with Ruby `4.0` → `gem install trmnl_preview` → `trmnlp lint` → `trmnlp push --force` with `TRMNL_API_KEY: ${{ secrets.TRML_TOKEN }}` (env name exact, per the TRMNL CLI).
- **Ordering guarantee**: the `otel` job's `needs` gains `trml` (`[ci, release, notify, trml]`) while keeping `if: always()`, so the TRMNL deploy always finishes before the OpenTelemetry trace export runs.
- **Plugin source reorganized into a trmnlp project under `trmnl/`**: `trmnlp` requires `.trmnlp.yml` at its project root and reads templates from a `src/` directory — so the templates and `settings.yml` moved from `trmnl/` into `trmnl/src/`, with `trmnl/.trmnlp.yml` added. The lint and push steps run with `working-directory: trmnl`. The `settings.yml` description was shortened (78 → 34 chars) because `trmnlp lint` enforces ≤ 35 characters.

## Capabilities

### New Capabilities
- `trmnl-ci`: Lint and deploy the TRMNL plugin during the release pipeline, before the OTel export.

### Modified Capabilities
<!-- No archived specs exist yet (openspec/specs/ is empty), so no modified capabilities. -->

## Impact

- `.github/workflows/release.yaml`: adds the `trml` job and updates the `otel` job's `needs`.
- `trmnl/`: restructured into a trmnlp project — templates + `settings.yml` moved into `trmnl/src/`, new `trmnl/.trmnlp.yml`, `settings.yml` description shortened to pass lint.
- Repo settings: new Actions secret `TRML_TOKEN` (TRMNL account API key).
- No changes to Go, TypeScript, or Go tests. `task pre-push` is unaffected.
