## ADDED Requirements

### Requirement: Weather is visible on the Race Control card
The Race Control card SHALL show a compact weather chip (condition icon + name) that is visible without opening the Conditions card, populated from the latest weather entry for the current race.

#### Scenario: Chip renders on page load
- **WHEN** an operator opens the controller page and weather entries exist for the current race
- **THEN** the Race Control card header shows the latest condition (e.g. "☀️ Dry" or "🌧️ Wet")

#### Scenario: Chip updates after setting weather
- **WHEN** an operator sets weather in the Conditions card
- **THEN** the chip updates to the newly set condition immediately

#### Scenario: No weather recorded
- **WHEN** no weather entries exist for the current race
- **THEN** the chip shows the default dry state (or the element is hidden)

### Requirement: Standings show gap to leader from recorded laps
Each driver row in the standings list SHALL show its lap-based gap to the leader, computed from `lap_records`: the leader shows "LEAD", drivers at least one lap behind show "+N", and drivers on the same lap show no gap.

#### Scenario: Gap renders after laps are recorded
- **WHEN** lap positions have been recorded and the leader has completed more laps than a trailing driver
- **THEN** the leader row shows "LEAD"
- **AND** the trailing driver row shows "+N" where N is the completed-laps difference

#### Scenario: Same-lap drivers show no gap
- **WHEN** a driver has completed the same number of laps as the leader
- **THEN** the driver row shows no gap value

#### Scenario: Gap column hidden without lap data
- **WHEN** no lap records exist for the current race
- **THEN** no gap values are rendered

#### Scenario: Gap updates after recording a lap
- **WHEN** an operator records lap positions
- **THEN** the gaps re-render from the updated lap data

### Requirement: Next race countdown in Configuration
The Configuration card SHALL show the next race date and days remaining from `GET /api/race-info` (`next_race_date`), and SHALL hide the line when the date is unset.

#### Scenario: Countdown renders when set
- **WHEN** `next_race_date` is set and an operator opens the controller page
- **THEN** the Configuration card shows "Next race: YYYY-MM-DD · in N days" (or "today"/"tomorrow" as appropriate)

#### Scenario: Countdown hidden when unset
- **WHEN** `next_race_date` is empty
- **THEN** no next-race line is shown

#### Scenario: Countdown refreshes after saving race settings
- **WHEN** an operator saves race settings after the next race date was changed
- **THEN** the displayed next-race line reflects the current value
