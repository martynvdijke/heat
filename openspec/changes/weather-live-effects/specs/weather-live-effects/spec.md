## ADDED Requirements

### Requirement: Pace heatmap includes weather context
`GET /api/stats/pace-heatmap` SHALL annotate every pace point with the weather condition and grip modifier active on that lap (from `weather_conditions` matching the race and lap range), defaulting to `dry`/`1.0` when no entry matches.

#### Scenario: Laps inside a wet entry are annotated
- **WHEN** a wet entry covers laps 10–20 of a race with recorded lap data
- **THEN** pace points for laps 10–20 include `condition: "wet"` and the entry's `grip_modifier`
- **AND** points for laps outside the range include `condition: "dry"` and `grip_modifier: 1.0`

#### Scenario: No weather data defaults to dry
- **WHEN** a race has no weather entries
- **THEN** all pace points report `dry` / `1.0`

### Requirement: Stats page pace heatmap shows weather
The stats page pace heatmap SHALL render a weather badge per cell (icon per condition) with a legend, and SHALL show condition + grip % on hover, without changing the layout for dry races.

#### Scenario: Wet laps are visually marked
- **WHEN** a pace heatmap contains cells with wet condition
- **THEN** those cells show a wet badge and the legend explains the icon

### Requirement: TV page shows a live weather banner
The TV page SHALL display a prominent, color-coded banner with the current condition (icon + name) and grip percentage, updated live from WS `weather_update` messages and populated on load from `GET /api/weather?race_id=0`.

#### Scenario: Banner renders the active condition
- **WHEN** weather is set to Wet with grip 0.7
- **THEN** the TV banner shows the wet icon, "Wet", and "70% grip"

#### Scenario: Banner updates live
- **WHEN** weather changes while the TV page is open
- **THEN** the banner updates without a page reload

#### Scenario: Dry default
- **WHEN** no weather entries exist
- **THEN** the banner shows the dry state

### Requirement: Forecast previews for scheduled weather
When a weather entry with `lap_start` beyond the current lap exists, TV, spectator, and pitboard SHALL show a preview of the upcoming change (e.g. "Rain from lap 10").

#### Scenario: Future entry shows a forecast line
- **WHEN** a wet entry is scheduled from lap 10 and the current lap is 5
- **THEN** TV, spectator, and pitboard show a forecast line announcing the change at lap 10

### Requirement: Pitboard shows weather banner and forecast
The pitboard SHALL replace its icon+name row with a banner showing condition, grip %, and any forecast, kept fresh via its existing weather fetch mechanism.

#### Scenario: Pitboard banner renders
- **WHEN** the pitboard loads with weather data
- **THEN** it shows the condition, grip %, and forecast in banner form

### Requirement: Controller can schedule weather changes
The controller Conditions card SHALL provide an "Until Lap" (`lap_end`) input (default 999) sent with `setWeather`, and SHALL list the active entry and upcoming scheduled entries after saving.

#### Scenario: Lap end is persisted
- **WHEN** an operator sets weather from lap 1 until lap 20
- **THEN** the stored entry has `lap_end: 20`

#### Scenario: Active and upcoming entries are listed
- **WHEN** the operator saves weather entries
- **THEN** the Conditions card lists the active entry and any upcoming (scheduled) entries

### Requirement: Weather broadcasts are identifiable
Weather WS broadcasts SHALL include `type: 'weather_update'` so clients can route them.

#### Scenario: Typed weather message
- **WHEN** weather is set
- **THEN** clients receive a WS message with `type: 'weather_update'` and the weather fields
