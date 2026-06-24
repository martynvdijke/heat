## Why

The existing Playwright E2E tests cover page existence and basic API smoke tests, but miss the core game flow entirely — racer lifecycle CRUD, full race orchestration (start lights → qualifying → race → finish), round/snapshot management, and the interaction between player actions and game state. Without these, regressions in the main gameplay loop go undetected.

## What Changes

Introduce comprehensive Playwright E2E tests that exercise the full game lifecycle:

- **Racer CRUD tests**: Create, edit, delete racers and verify persistence in the racer list and leaderboard
- **Race lifecycle tests**: Full race flow from start lights → qualifying → race rounds → lap recording → race finish → history save
- **Round/snapshot tests**: Take round snapshots, verify standings, verify historical data
- **Stats verification tests**: After races, verify gold/silver/bronze, points, DNF/DNS, streaks, ELO ratings
- **Player interaction tests**: Full player login → gear reporting → heat card usage → turbo → status queries
- **Flag/start lights tests**: Verify the start lights sequence triggers, abort, reset, and that the UI reflects the correct state
- **Weather & dynamic conditions tests**: Set weather conditions, verify they're persisted and returned by the API
- **Spectator state tests**: Verify spectator API returns correct state after race events
- **Game mechanics tests**: Heat card decks, upgrades, gear shifts, legend abilities
- **Deck builder flow**: Buy upgrades, equip/unequip, verify they appear in player upgrades

## Capabilities

### New Capabilities
- `racer-crud`: Full racer lifecycle — create, read, update, delete racers via admin UI
- `race-flow`: End-to-end race lifecycle — start lights, qualifying, rounds, lap recording, race completion, history save
- `round-snapshot`: Round snapshot creation, retrieval, and deletion with standings verification
- `stats-verification`: Post-race stats validation — gold/silver/bronze, wins, DNF/DNS, streaks, ELO, consistency
- `player-session`: Full player session lifecycle — login, validate, report gear/heat/turbo, status, logout
- `flag-lights`: Start lights sequence — trigger, abort, reset, and UI state reflection
- `weather-conditions`: Weather condition CRUD and retrieval
- `game-mechanics`: Heat card deck initialization, gear shifts, upgrades, legend abilities
- `deck-builder`: Upgrade purchasing, equip/unequip toggling, available upgrades querying

### Modified Capabilities
*(No existing spec requirements are changing — this is purely additive test coverage)*

## Impact

- `tests/` — new Playwright spec files covering the above capabilities
- `playwright.config.ts` — potentially update if test organization needs tweaks
- No changes to application code, handlers, models, or database schema
