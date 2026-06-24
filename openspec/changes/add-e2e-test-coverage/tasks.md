## 1. Setup & Shared Helpers

- [ ] 1.1 Extract shared `API_HEADERS` constant into a reusable helper file for admin auth tests
- [ ] 1.2 Verify the `loginAdminViaAPI` helper works reliably across all new spec files

## 2. Racer CRUD Tests

- [ ] 2.1 Create `tests/racers.spec.ts` with describe block for racer CRUD via API
- [ ] 2.2 Test: Create racer with all fields via POST /api/racers and verify in GET /api/racers response
- [ ] 2.3 Test: Edit existing racer name via POST /api/racers and verify the update persists
- [ ] 2.4 Test: Unauthenticated POST /api/racers returns 401
- [ ] 2.5 Test: Newly created racer name appears on the index page leaderboard

## 3. Race Flow Tests

- [ ] 3.1 Create `tests/race-flow.spec.ts` with describe blocks for the full race lifecycle
- [ ] 3.2 Test: Update race info via POST /api/race-info and verify via GET /api/race-info
- [ ] 3.3 Test: Save completed race with multiple racers via POST /api/race-history and verify id returned
- [ ] 3.4 Test: GET /api/race-history returns saved races
- [ ] 3.5 Test: Delete race history and verify it's removed
- [ ] 3.6 Test: Create and query one-off races (race_type "oneoff")
- [ ] 3.7 Test: Delete one-off race via DELETE /api/oneoff-races
- [ ] 3.8 Test: One-off race does NOT update racer_stats (races count unchanged)

## 4. Round Snapshot Tests

- [ ] 4.1 Create `tests/rounds.spec.ts` for round snapshot lifecycle
- [ ] 4.2 Test: Take round snapshot with auto-incremented round number
- [ ] 4.3 Test: Take round snapshot with explicit round number
- [ ] 4.4 Test: Get all round snapshots
- [ ] 4.5 Test: Get snapshot by ID and verify scores array
- [ ] 4.6 Test: Delete snapshot and verify it returns 404 on subsequent GET

## 5. Stats Verification Tests

- [ ] 5.1 Create `tests/stats-verification.spec.ts` for post-race stats validation
- [ ] 5.2 Test: Save season race with 5 racers and verify gold/silver/bronze/wins counts per racer
- [ ] 5.3 Test: Save season race with DNF and DNS racers and verify dnf/dns counts
- [ ] 5.4 Test: GET /api/racer-stats returns stats object with all expected fields
- [ ] 5.5 Test: Head-to-head endpoint returns racer names
- [ ] 5.6 Test: Points progression endpoint returns array
- [ ] 5.7 Test: Streaks endpoint returns array
- [ ] 5.8 Test: ELO ratings endpoint returns array
- [ ] 5.9 Test: Stats CSV export returns text/csv with Name, Gold, Silver, Bronze headers
- [ ] 5.10 Test: Track performance endpoint works (all racers and specific racer)

## 6. Player Session Tests

- [ ] 6.1 Create `tests/player-session.spec.ts` (note: existing tests in `new-features.spec.ts` cover login, this file adds full lifecycle)
- [ ] 6.2 Test: Player login with valid racer_id returns token, racer_id, racer_name
- [ ] 6.3 Test: Player login with invalid racer_id returns 404
- [ ] 6.4 Test: Validate valid player token returns racer_id and racer_name
- [ ] 6.5 Test: Validate invalid token returns 401
- [ ] 6.6 Test: Player reports gear shift
- [ ] 6.7 Test: Player reports heat card usage
- [ ] 6.8 Test: Player reports turbo usage
- [ ] 6.9 Test: Player status endpoint returns racer and heat_cards fields
- [ ] 6.10 Test: Player logout invalidates token (subsequent validate returns 401)

## 7. Flag & Lights Tests

- [ ] 7.1 Expand `tests/start-lights.spec.ts` with additional flag scenarios
- [ ] 7.2 Test: POST /api/flags with invalid type returns 400
- [ ] 7.3 Test: Trigger sound via POST /api/sound (engine, finish)
- [ ] 7.4 Verify start lights page UI elements (logo, status bar, message area, script, close button)
- [ ] 7.5 Verify start lights page accessibility (prefers-reduced-motion, no user-scalable=no)

## 8. Weather Conditions Tests

- [ ] 8.1 Create `tests/weather.spec.ts` for weather condition CRUD
- [ ] 8.2 Test: Set weather condition via POST /api/weather
- [ ] 8.3 Test: Get weather conditions by race_id
- [ ] 8.4 Test: Delete weather condition

## 9. Game Mechanics Tests

- [ ] 9.1 Create `tests/game-mechanics.spec.ts` for heat cards, gear shifts, upgrades, lap records
- [ ] 9.2 Test: Initialize heat decks via POST /api/heat-cards/init-decks and verify card count
- [ ] 9.3 Test: Get heat cards filtered by racer_id
- [ ] 9.4 Test: Add individual heat card
- [ ] 9.5 Test: Move heat card between locations
- [ ] 9.6 Test: Delete individual heat card
- [ ] 9.7 Test: Clear all heat cards
- [ ] 9.8 Test: Get gear shifts by racer_id
- [ ] 9.9 Test: Get upgrade cards (at least 8)
- [ ] 9.10 Test: Get legend abilities (at least 5)
- [ ] 9.11 Test: Record and query lap records
- [ ] 9.12 Test: Submit batch lap records
- [ ] 9.13 Test: Get sectors for Monza (at least 5)
- [ ] 9.14 Test: Add and list race events (overtake)
- [ ] 9.15 Test: Get AI difficulty defaults
- [ ] 9.16 Test: Set AI difficulty

## 10. Deck Builder Tests

- [ ] 10.1 Create `tests/deck-builder.spec.ts` for upgrade purchasing and management
- [ ] 10.2 Test: Get available upgrades for a racer
- [ ] 10.3 Test: Buy upgrade for racer
- [ ] 10.4 Test: Get player upgrades list
- [ ] 10.5 Test: Toggle upgrade equipped status
- [ ] 10.6 Test: Assign legend ability to racer
- [ ] 10.7 Test: Get racer legend abilities
