# trmnl-e-ink-display Specification

## Purpose
TRMNL e-ink device plugin assets — four Liquid templates and a `settings.yml` manifest — that poll `GET /api/trmnl/summary` and render the latest race results and season championship standings on a TRMNL display.

## ADDED Requirements

### Requirement: Plugin manifest
The repo SHALL ship a `trmnl/settings.yml` manifest configuring the TRMNL plugin for the polling strategy against the summary endpoint.

#### Scenario: Polling configuration
- **WHEN** a TRMNL device loads the plugin
- **THEN** `settings.yml` SHALL declare `strategy: polling`
- **AND** `polling_url` SHALL point to the `/api/trmnl/summary` endpoint of the configured Heat instance
- **AND** `polling_verb` SHALL be `get`

#### Scenario: Configurable instance URL
- **WHEN** a user installs the plugin on their own Heat instance
- **THEN** the instance URL SHALL be configurable via a `custom_fields` entry (e.g. `url`, "Address of heat instance", placeholder including `http://` or `https://`)
- **AND** the configured URL SHALL be used as the base for `polling_url`

#### Scenario: Refresh interval
- **WHEN** `settings.yml` is parsed
- **THEN** `refresh_interval` SHALL be set to 60 minutes by default

### Requirement: Liquid templates for all layouts
The repo SHALL ship four Liquid templates — `full.liquid`, `half_horizontal.liquid`, `half_vertical.liquid`, `quadrant.liquid` — one for each TRMNL device layout.

#### Scenario: Template set completeness
- **WHEN** the plugin is packaged for TRMNL
- **THEN** all four template files SHALL exist under `trmnl/` with the TRMNL-conventional names

#### Scenario: Full layout renders race and standings
- **WHEN** `full.liquid` renders a non-empty summary payload
- **THEN** it SHALL display the latest race name, track/country, and date
- **AND** it SHALL display the finishing order with points (top 3 prominent, remainder compact)
- **AND** it SHALL display the standings table (position, racer name, wins, points)

#### Scenario: Reduced layouts render trimmed content
- **WHEN** `half_horizontal.liquid`, `half_vertical.liquid`, or `quadrant.liquid` renders a non-empty summary payload
- **THEN** the template SHALL render without errors and SHALL display at minimum the podium (top 3) of the latest race and the top 3 of the standings

### Requirement: Empty-state handling
All templates SHALL render a graceful empty state when the payload contains no race or standings data.

#### Scenario: No data yet
- **WHEN** a template renders a payload where `latest_race` is null and `standings` is empty
- **THEN** the template SHALL display a readable "No race data yet" style message and SHALL NOT error

#### Scenario: Liquid null guards
- **WHEN** any template accesses `latest_race` fields
- **THEN** the access SHALL be guarded with a Liquid conditional so a null `latest_race` does not render blank values

### Requirement: Display conventions
Templates SHALL use the standard TRMNL layout classes and a black-on-white, motion-free presentation appropriate for e-ink.

#### Scenario: Layout classes
- **WHEN** templates render
- **THEN** they SHALL use TRMNL structural classes (`layout`, `grid`, `item`, `value`, `label`, `hr`, `title_bar`) consistent with the TRMNL framework version declared in `settings.yml`

#### Scenario: Static presentation
- **WHEN** templates render
- **THEN** they SHALL contain no animations, transitions, or color gradients (e-ink suitable, high-contrast text only)
