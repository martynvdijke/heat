## ADDED Requirements

### Requirement: Extension catalog enumerates full track, upgrade and legend content
The system SHALL return, for each extension, its tracks, upgrade cards and legend abilities using the same full content shapes used elsewhere on the site (`models.Track` with geojson, length, module_id and is_board_game; `models.UpgradeCard` with description, effects, card_type and cost; `models.LegendAbility` with description, ability_type and racer_name), so that extension content can drive the rest of the site's selection UIs.

#### Scenario: Extension detail includes full track information
- **WHEN** a client requests `GET /api/extensions/detail` for an extension that has tracks
- **THEN** each returned track includes id, name, country, geojson, length, module_id and is_board_game
- **AND** module-owned tracks are included with the same full shape

#### Scenario: Extension detail includes full upgrade card information
- **WHEN** a client requests `GET /api/extensions/detail` for an extension that has upgrade cards
- **THEN** each returned upgrade includes id, name, description, card_type, cost and effects

#### Scenario: Extension detail includes full legend ability information
- **WHEN** a client requests `GET /api/extensions/detail` for an extension that has legend abilities
- **THEN** each returned legend includes id, name, description, ability_type and racer_name

#### Scenario: Extension without content returns empty lists
- **WHEN** a client requests `GET /api/extensions/detail` for an extension that has no tracks, upgrades or legends
- **THEN** the response contains empty arrays for tracks, upgrades and legends
