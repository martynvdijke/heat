## ADDED Requirements

### Requirement: Board game track list setting
The system SHALL maintain a curated board game track list in a `board_game_tracks` table (`track_id` primary key). The list SHALL be seeded on first startup with the tracks from the base game. `GET /api/board-game/tracks` SHALL return the current `track_ids`; `PUT /api/board-game/tracks` SHALL replace the entire list with the submitted `track_ids`.

#### Scenario: First startup seeds board game tracks
- **WHEN** the app starts and the `board_game_tracks` table is empty
- **THEN** the seeded base game tracks are inserted as board game tracks

#### Scenario: Restart does not duplicate the seed
- **WHEN** the app restarts with existing `board_game_tracks` rows
- **THEN** no additional board game tracks are inserted

#### Scenario: Admin reads the board game track list
- **WHEN** an admin GETs `/api/board-game/tracks`
- **THEN** the response contains the list of board game track ids

#### Scenario: Admin replaces the board game track list
- **WHEN** an admin PUTs `/api/board-game/tracks` with a new set of track ids
- **THEN** the stored board game track list equals the submitted set

#### Scenario: Newly created tracks are not board game tracks
- **WHEN** an admin creates a new custom track
- **THEN** it is not part of the board game track list until the admin adds it

### Requirement: Track list flags board game membership
`GET /api/tracks` SHALL include an `is_board_game` flag per track indicating membership in the board game track list.

#### Scenario: Track list flags board game tracks
- **WHEN** an admin GETs `/api/tracks`
- **THEN** each track includes `is_board_game` set according to the board game track list

### Requirement: Race track selector presents board game and custom tracks
The race track selector (admin race form and controller) SHALL list board game tracks first in a "Board Game" group, followed by custom tracks (tracks not in the board game list) in a "Custom" group, so the admin can pick from the board game's tracks alongside custom ones.

#### Scenario: Selector groups board game and custom tracks
- **WHEN** the race track selector is loaded with both board game and custom tracks present
- **THEN** board game tracks appear in a "Board Game" group and all other tracks appear in a "Custom" group

#### Scenario: Selector shows only custom tracks
- **WHEN** the race track selector is loaded and no tracks are in the board game list
- **THEN** all tracks appear in the "Custom" group

### Requirement: Tracks table shows board game badges
The admin Tracks table SHALL display a "Board Game" badge for tracks in the board game list and a "Custom" badge for the rest.

#### Scenario: Tracks table badges
- **WHEN** the admin Tracks table is rendered
- **THEN** each row shows a badge reflecting whether the track is a board game track or a custom track

### Requirement: Board game tracks editor UI
The Tracks pane SHALL provide a Board Game Tracks editor: a checkbox list of all tracks letting the admin toggle membership, and a save action that persists the selection via `PUT /api/board-game/tracks`.

#### Scenario: Admin edits the board game track list from the UI
- **WHEN** an admin toggles track checkboxes in the Board Game Tracks editor and saves
- **THEN** the board game track list is updated and the tracks table badges and track selector grouping refresh
