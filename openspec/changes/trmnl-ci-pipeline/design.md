## Context

- `.github/workflows/release.yaml` runs on `push` to `main` + `workflow_dispatch`. Job chain today: `ci` (reusable) → `release` (semantic-release; publishes version + Docker images) → `notify` (Gotify) → `otel` (`if: always()`, `needs: [ci, release, notify]`, exports the run trace via `corentinmusard/otel-cicd-action@v4`).
- `trmnl/` contains the TRMNL plugin. Before this change it held the flat export layout (templates + `settings.yml` directly in `trmnl/`, `framework_version: 2.3.7`, strategy `polling`, `polling_url: "{{url}}/api/trmnl/summary"`).
- The `trmnl_preview` Ruby gem (latest 0.11.0) provides the `trmnlp` CLI: `trmnlp lint` validates Liquid/JSON, `trmnlp push --force` uploads the plugin to the TRMNL cloud. **Project layout is fixed and not configurable**: `.trmnlp.yml` must exist at the project root (`Paths#valid? = trmnlp_config.exist?`) and templates/settings are read from `<root>/src/` (`src_dir = root_dir.join('src')`, `plugin_config = src_dir.join('settings.yml')`). A flat `trmnl/` layout fails with "not a plugin directory". Auth resolves `TRMNL_API_KEY` env var first, then local login config.
- Repo conventions: `actions/checkout@v7` is used in `ci.yaml`/`release.yaml`. The pasted workflow used `actions/checkout@v6`; both work, repo convention is v7.

## Goals / Non-Goals

**Goals:**
- Lint the TRMNL plugin on every release-pipeline run (push to main / workflow_dispatch).
- Deploy (force-push) the plugin to the TRMNL account on those runs.
- Guarantee the deploy completes before the `otel` trace export, per "called on release before the otel export".
- Zero changes to the plugin source in `trmnl/`.

**Non-Goals:**
- PR-time linting: `ci.yaml` is not touched; lint happens as part of the release pipeline only (a standalone PR lint workflow can be added later).
- Gating the push on `new_release_published`: the pasted workflow pushes on every main push; documented as an optional follow-up, not implemented.
- Plugin template changes, multi-plugin support, or any non-`trmnl/` deployment target.

## Decisions

### D1: `trml` as a job inside `release.yaml` (user-confirmed)
Chosen over a standalone `.github/workflows/trml.yaml` because a standalone workflow offers no ordering guarantee relative to the `otel` job (separate workflows run concurrently). Placing the job between `release` and `otel` and adding it to `otel`'s `needs` guarantees "before the otel export".

### D2: Lint then push in a single job
`trmnlp lint` runs first; any failure fails the job and the `trmnlp push --force` step never runs. Mirrors the pasted workflow's `push` job gating on the `lint` job.

### D3: Secret mapping
`TRMNL_API_KEY: ${{ secrets.TRML_TOKEN }}` exactly as pasted — GitHub secret names are case-insensitive, but the env var name passed to `trmnlp` must be `TRMNL_API_KEY`. If the real secret is named `TRMNL_TOKEN`, only the reference needs updating; the env var stays.

### D4: OTel export still runs even if TRMNL fails
`otel` keeps `if: always()` and gains `trml` in `needs: [ci, release, notify, trml]`. A TRMNL failure never blocks the trace export.

### D5: Push scope
Push runs on every release-pipeline run (every push to main + manual dispatch), matching the pasted workflow's `push: branches: [main]`. Optional refinement (not implemented): wrap the push step in `if: steps.semantic.outputs.new_release_published == 'true'`.

### D6: Ruby version
`ruby/setup-ruby@v1` with `ruby-version: "4.0"` as pasted. **Verified available**: Ruby 4.0.0 shipped 2025-12-25 (latest 4.0.6, 2026-07-14), so `"4.0"` resolves on ubuntu-latest; no fallback needed.

### D7: trmnlp project layout under `trmnl/`
Because trmnlp hardcodes `.trmnlp.yml` + `src/`, the plugin was restructured *inside* the `trmnl/` folder: `trmnl/.trmnlp.yml` (watch `.trmnlp.yml` + `src`, `custom_fields: {url: ""}`, `variables.trmnl`) and `trmnl/src/` holding the four liquid templates + `settings.yml`. The lint and push steps run with `working-directory: trmnl` (equivalent to `trmnlp -d trmnl`). This keeps all plugin code within `trmnl/` as required, and `--force` overwrites the remote plugin.

### D8: Description length
`trmnlp lint` enforces a ≤ 35-character description. The `settings.yml` description (78 chars) was shortened to `"Latest race results and standings"` (34 chars) so lint passes. `trmnlp lint -d trmnl` now reports "✓ All checks passed!".

## Risks / Trade-offs

- [Ruby 4.0 availability] → **Resolved**: 4.0.0 released 2025-12-25; verified `"4.0"` is valid for `ruby/setup-ruby`.
- [Missing `TRML_TOKEN` secret] → push step fails; `otel` still exports because of `if: always()`. Documented setup step: add the secret before merging.
- [`push --force` overwrites the remote plugin] → intended; the TRMNL plugin is a single mutable remote resource, no versioning needed.
- [Gem install latency] → `gem install trmnl_preview` runs every run; acceptable (~10–25s). Optional: `gem install trmnl_preview --no-document` to shave time.
- [Plugin layout mismatch] → **Resolved**: the flat `trmnl/` export layout is not a valid trmnlp project; D7 restructures it (`.trmnlp.yml` + `src/`) and the workflow pins `working-directory: trmnl`, so detection is explicit.
- [Lint gate on description length] → **Resolved** in D8; lint passes locally (`✓ All checks passed!`). Future edits to `settings.yml` must keep the description ≤ 35 chars or the release pipeline fails.
