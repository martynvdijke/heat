## ADDED Requirements

### Requirement: TRMNL plugin is linted in the release pipeline
The release pipeline SHALL run a `trml` job that installs the `trmnl_preview` gem and lints the plugin in the `trmnl/` directory with `trmnlp lint`.

#### Scenario: Release pipeline lints the plugin
- **WHEN** the release pipeline runs (push to `main` or manual dispatch)
- **THEN** a `trml` job installs `trmnl_preview` and runs `trmnlp lint`
- **AND** invalid Liquid templates or settings fail the job

#### Scenario: Plugin source lives in the trmnl directory
- **WHEN** the repository is checked out
- **THEN** the plugin source exists under `trmnl/` as a valid trmnlp project: `trmnl/.trmnlp.yml`, `trmnl/src/full.liquid`, `trmnl/src/half_horizontal.liquid`, `trmnl/src/half_vertical.liquid`, `trmnl/src/quadrant.liquid`, `trmnl/src/settings.yml`
- **AND** the lint and push steps run with `working-directory: trmnl`

### Requirement: Plugin is a valid trmnlp project
The plugin under `trmnl/` SHALL be a valid trmnlp project (`.trmnlp.yml` at root, templates and `settings.yml` in `src/`) and SHALL pass `trmnlp lint`.

#### Scenario: Lint passes on the restructured plugin
- **WHEN** `trmnlp lint` runs against `trmnl/`
- **THEN** all checks pass, including the ≤ 35-character description rule

### Requirement: TRMNL plugin is deployed on main
The `trml` job SHALL deploy the plugin with `trmnlp push --force`, authenticated via the `TRMNL_API_KEY` environment variable populated from the `TRML_TOKEN` secret.

#### Scenario: Push to main deploys the plugin
- **WHEN** the release pipeline runs after a push to `main`
- **THEN** the `trml` job runs `trmnlp push --force` with `TRMNL_API_KEY` set from the `TRML_TOKEN` secret
- **AND** the plugin is force-pushed to the TRMNL account

### Requirement: Lint failure blocks deployment
Deployment SHALL NOT run when linting fails; a lint failure SHALL fail the `trml` job.

#### Scenario: Invalid template aborts the push
- **WHEN** `trmnlp lint` reports an error
- **THEN** the `trmnlp push --force` step is skipped
- **AND** the `trml` job fails

### Requirement: Deployment completes before the OpenTelemetry export
The `otel` export job SHALL wait for the `trml` job, and SHALL still run when the `trml` job fails.

#### Scenario: TRMNL deploy precedes the trace export
- **WHEN** the release pipeline runs
- **THEN** the `otel` job's `needs` includes `trml`
- **AND** the `otel` job starts only after the `trml` job finishes

#### Scenario: Trace export survives TRMNL failure
- **WHEN** the `trml` job fails
- **THEN** the `otel` job still runs (`if: always()`) and exports the run trace
