## ADDED Requirements

### Requirement: Public stats page displays only live statistics sections
The public stats page SHALL render only the statistics sections that are actually populated: top stat cards (Total Races, Championships, Fastest Laps, Active Drivers), Points Progression chart, Win Distribution, Driver Performance (incl. Spins/Overheated), Track Statistics, Championship Battle, Points Leaderboard, ELO Ratings, and the deeper-stats cards Qualifying vs Race, Consistency, Race Incidents and Pace Heatmap. The Most Golds, Most Silvers, Fastest Laps, Most Poles, Track Performance, Points Progression and Streaks cards, the Avg Lap Time column, and the Lap Time Trends chart SHALL NOT be rendered.

#### Scenario: Removed deeper-stats cards are absent
- **WHEN** a visitor loads the public stats page
- **THEN** no Most Golds, Most Silvers, Fastest Laps, Most Poles, Track Performance, Points Progression or Streaks section exists in the page

#### Scenario: Track Statistics table has no Avg Lap Time column
- **WHEN** a visitor loads the public stats page and the Track Statistics table renders
- **THEN** the table has only the Track, Races and Winner columns

#### Scenario: Lap Time Trends chart is absent
- **WHEN** a visitor loads the public stats page
- **THEN** no Lap Time Trends chart or canvas exists in the page

#### Scenario: Live sections still render
- **WHEN** a visitor loads the public stats page with season data present
- **THEN** the Driver Performance table includes the Spins and Overheated columns
- **AND** the Qualifying vs Race, Consistency, Race Incidents and Pace Heatmap bodies are populated from their APIs

### Requirement: Stats page placeholder rows match remaining columns
The empty-state placeholder row of the Track Statistics table SHALL span exactly the number of columns in the table.

#### Scenario: Track Statistics placeholder spans three columns
- **WHEN** the Track Statistics table has no data and renders its placeholder row
- **THEN** the placeholder cell spans 3 columns
