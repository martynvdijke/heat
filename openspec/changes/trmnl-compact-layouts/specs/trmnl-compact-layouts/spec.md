## ADDED Requirements

### Requirement: Templates render compactly on 800x480
All four layouts (`full`, `half_horizontal`, `half_vertical`, `quadrant`) SHALL use the compact TRMNL design tokens (`gap--xsmall`, `label--small`, `value--xsmall` / `value--xxsmall`, `divider--h`, `grid--cols-2`) so the plugin fits the 800x480 e-ink display without overflow.

#### Scenario: Compact rendering
- **WHEN** any layout renders on the TRMNL device
- **THEN** the layout uses the compact design tokens and fits the 800x480 screen

### Requirement: Top results are emphasized on small layouts
The half and quadrant layouts SHALL render only the top 3 race results and top 3 standings entries; the full layout SHALL render all results in a two-column grid.

#### Scenario: Half layout shows podium
- **WHEN** `half_horizontal`, `half_vertical`, or `quadrant` renders with more than 3 race results
- **THEN** only entries with `forloop.index <= 3` are displayed

#### Scenario: Full layout shows every result
- **WHEN** the `full` layout renders
- **THEN** every race result and standings entry is displayed in a `grid--cols-2` layout

### Requirement: Fallback states for missing data
Each layout SHALL render a readable fallback (`No race data yet` / `No standings yet`) when the payload contains no race or no standings data.

#### Scenario: No race data
- **WHEN** `latest_race` is `nil`
- **THEN** the layout renders `No race data yet` instead of an empty screen

#### Scenario: No standings data
- **WHEN** the standings array is empty
- **THEN** the layout renders `No standings yet` instead of an empty section

### Requirement: Payload compatibility
The templates SHALL render only fields already present in the `/api/trmnl/summary` payload (`latest_race.*`, `standings[].{racer_name, wins, points}`, `season.name`); no backend or payload change SHALL be required.

#### Scenario: Standings wins field
- **WHEN** `half_horizontal` renders a standings entry
- **THEN** it reads `standing.wins` and `standing.points` from the existing `SeasonStanding` payload (`json:"wins"`, `json:"points"`)
