## Why

Round snapshots are weakly linked to seasons via an optional `season_id` field with a hardcoded default of 1, making it possible to create rounds without a season or assign them ambiguously. This undermines the multi-season structure. Additionally, finalizing a round in the admin backend updates driver scores but does not trigger email notifications to racers, breaking the expected notification flow.

## What Changes

1. **Round-Season Link Enforcement**: Make `season_id` a required field in `round_snapshots` with a foreign key constraint. Add a unique constraint on `(season_id, round)` to prevent duplicate round numbers within a season. Update all round creation paths to require a valid season_id.
2. **Email Notifications on Finalize**: After a round is finalized via the admin backend (`/api/rounds/finalize`), automatically send race result emails to all racers who have email addresses configured.
3. **Admin UI Improvements**: The "Take Snapshot" button should always target the currently selected active season. The rounds list should always require a season context.

## Capabilities

### New Capabilities
- `round-season-constraint`: Database-level enforcement that each round belongs to exactly one season, with unique round numbers per season
- `round-finalize-email`: Automated email notifications sent when a round is finalized in the admin backend

### Modified Capabilities
*(none — no existing specs are changing)*

## Impact

- **Database schema**: `round_snapshots.season_id` becomes NOT NULL with a foreign key to `seasons.id`. New UNIQUE constraint on `(season_id, round)`. Existing data migration needed.
- **Go handlers**: `TakeRoundSnapshot` and `FinalizeRound` — validate season linkage. `FinalizeRound` triggers email send.
- **Admin JS**: `takeAdminRoundSnapshot` passes required season context. Finalize flow waits for email dispatch.
- **Email system**: Reuse existing `SendRaceEmail` infrastructure; may need to adapt it for round-based (non-race) data.
