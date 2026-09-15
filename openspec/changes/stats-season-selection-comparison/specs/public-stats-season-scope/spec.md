## ADDED Requirements

### Requirement: Season scope selection
The public stats page SHALL provide a single season-scope control that supports three explicit modes: **All Seasons** (default), **one season**, and **multiple seasons**. The control SHALL expose the "All Seasons" option by default and SHALL allow selecting any combination of existing seasons.

#### Scenario: Page loads with All Seasons selected
- **WHEN** a visitor opens `/stats.html` without a `seasons` query parameter
- **THEN** the scope control shows "All Seasons" selected and every section renders all-time data across all seasons

#### Scenario: Selecting a single season
- **WHEN** the visitor selects exactly one season in the scope control
- **THEN** every stats section on the page re-renders with data scoped to that season only

#### Scenario: Selecting multiple seasons
- **WHEN** the visitor selects two or more seasons in the scope control
- **THEN** the page enters comparison mode and the comparison view is shown for the selected seasons

#### Scenario: All Seasons is exclusive
- **WHEN** the visitor checks "All Seasons" while seasons are selected
- **THEN** all individual season selections are cleared and the page shows the all-time view

#### Scenario: All seasons unchecked
- **WHEN** the visitor unchecks every option including "All Seasons"
- **THEN** the page behaves as if "All Seasons" is selected

### Requirement: Scope applies to all stats sections
Every section of the public stats page — top stat cards, Points Progression chart, Win Distribution, Driver Performance table, Track Statistics, Championship Battle, Points Leaderboard, Qualifying vs Race Delta, Consistency Ratings, Incidents Report, and Pace Heatmap — SHALL respect the currently selected season scope.

#### Scenario: Deeper stats follow the selected season
- **WHEN** a single season is selected
- **THEN** the Qualifying vs Race Delta, Consistency Ratings, Incidents Report, and Pace Heatmap sections render data filtered to that season

#### Scenario: Deeper stats follow All Seasons
- **WHEN** "All Seasons" is selected
- **THEN** the deeper stats sections render data across all seasons

### Requirement: All-time aggregation across seasons
When the scope is "All Seasons", racer statistics SHALL be aggregated from all finalized round snapshots across every season, using the same aggregation rules as single-season statistics so the views are directly comparable.

#### Scenario: All Seasons totals combine seasons
- **WHEN** the scope is "All Seasons" and two seasons each contain finalized rounds
- **THEN** each racer's displayed races, wins, podiums, points, spins, and overheated counts equal the sum of the per-season values

### Requirement: Cross-season comparison view
When two or more seasons are selected, the page SHALL render a comparison view containing (a) a table with one row per driver and one column per selected season showing races, wins, podiums, points, spins, and overheated, and (b) a grouped chart comparing the selected seasons by points or wins.

#### Scenario: Comparison table renders per-season columns
- **WHEN** exactly two seasons are selected
- **THEN** the comparison table shows each driver's stats in a column for each of the two seasons

#### Scenario: Comparison chart renders
- **WHEN** two or more seasons are selected
- **THEN** a grouped chart is drawn with one series per selected season

#### Scenario: No shared drivers
- **WHEN** two or more seasons are selected and a driver has no data in any selected season
- **THEN** that driver is not shown in the comparison view

### Requirement: Empty season scoping is truthful
When a requested season scope contains no finalized rounds, the stats API SHALL return an empty result set, and the page SHALL show its existing empty-state placeholders rather than all-time data.

#### Scenario: Season with no finalized rounds
- **WHEN** a visitor selects a season that has no finalized rounds
- **THEN** the API returns an empty array for the scoped racer-stats request and the page displays the "no data" empty states

#### Scenario: No silent all-time fallback
- **WHEN** a request includes an explicit season scope that yields no rows
- **THEN** the API returns an empty result and never substitutes all-time statistics

### Requirement: Season scope query parameter on stats endpoints
Stats endpoints SHALL accept a `season_ids` query parameter containing a comma-separated list of season IDs. An absent or empty `season_ids` SHALL mean "all seasons". A single-season `season_id` parameter SHALL be accepted as an alias for `season_ids` with one value.

#### Scenario: Single season via season_ids
- **WHEN** a client requests `/api/racer-stats?season_ids=3`
- **THEN** the response contains racer statistics aggregated from season 3's finalized rounds only

#### Scenario: Multi-season via season_ids
- **WHEN** a client requests `/api/racer-stats?season_ids=1,2`
- **THEN** the response contains racer statistics aggregated from the finalized rounds of seasons 1 and 2 combined

#### Scenario: Absent scope means all seasons
- **WHEN** a client requests `/api/racer-stats` with no season parameters
- **THEN** the response contains racer statistics aggregated from finalized rounds across all seasons

#### Scenario: season_id alias
- **WHEN** a client requests `/api/racer-stats?season_id=3`
- **THEN** the response equals the response for `season_ids=3`

#### Scenario: Invalid season_ids rejected
- **WHEN** a client requests a stats endpoint with `season_ids=abc`
- **THEN** the API responds with HTTP 400 and an error message

### Requirement: Shareable season scope via URL
The stats page SHALL read a `seasons` query parameter (comma-separated season IDs, or `all`) on load and SHALL update the URL when the scope changes, so a given scope is shareable and bookmarked.

#### Scenario: Deep link to a season
- **WHEN** a visitor opens `/stats.html?seasons=3`
- **THEN** the scope control reflects season 3 selected and the page renders season 3 data

#### Scenario: Deep link to comparison
- **WHEN** a visitor opens `/stats.html?seasons=1,2`
- **THEN** the scope control reflects seasons 1 and 2 selected and the page renders the comparison view

#### Scenario: URL updates on scope change
- **WHEN** the visitor changes the season scope
- **THEN** the browser URL is updated to match the new selection without a page reload
