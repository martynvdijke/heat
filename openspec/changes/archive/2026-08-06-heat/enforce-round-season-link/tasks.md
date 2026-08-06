## 1. Database Migration — Round-Season Constraints

- [x] 1.1 Add SQL migration: backfill NULL/0 season_id values in round_snapshots to the active season (or season 1)
- [x] 1.2 Add SQL migration: deduplicate any existing (season_id, round) pairs that would violate the upcoming UNIQUE constraint
- [/] 1.3 SQLite limitation: ALTER TABLE ADD CONSTRAINT not supported. Enforced via Go-level validation (non-zero season_id + valid season check) + UNIQUE(season_id, round) index. FK semantics handled by application cascade on season delete.
- [x] 1.4 Add SQL migration: CREATE UNIQUE INDEX on round_snapshots(season_id, round)
- [x] 1.5 Ensure `PRAGMA foreign_keys = ON` is set in the database connection initialization in db/init.go
- [x] 1.6 Update ent schema `RoundSnapshot` to remove `.Optional().Default(1)` from `season_id` field and add proper constraints

## 2. Backend — Round Creation Validation

- [x] 2.1 Update `TakeRoundSnapshot` handler: require `season_id` in request body (reject with 400 if missing/zero)
- [x] 2.2 Keep existing validations: reject if season is archived (404) or not found (404)
- [x] 2.3 Remove the `if input.SeasonID == 0 { input.SeasonID = 1 }` fallback in `TakeRoundSnapshot`
- [x] 2.4 Ensure round auto-numbering (`SELECT COALESCE(MAX(round), 0) + 1`) is scoped to `WHERE season_id = ?` (already done)
- [x] 2.5 Handle SQLite UNIQUE constraint violation errors gracefully in `TakeRoundSnapshot` (return 409)

## 3. Backend — Round Finalization Email

- [x] 3.1 Create `SendRoundEmail` function in handlers/email.go that:
      - Fetches round snapshot data (race_name, race_date) and scores from round_snapshot_scores
      - Builds email content similar to `buildRaceEmailContent` but with round data
      - Sends via existing SMTP infrastructure
- [x] 3.2 Create `buildRoundEmailContent` function that formats round results into an HTML email (similar to `buildRaceEmailContent`)
- [x] 3.3 Call `SendRoundEmail` from `FinalizeRound` handler after the transaction commits
- [x] 3.4 Ensure email sending is async (goroutine) to not block the API response

## 4. Admin UI — Season Context for Snapshots

- [x] 4.1 Remove the hardcoded season_id=1 fallback in `takeAdminRoundSnapshot` JS function (already using getActiveSeasonId())
- [x] 4.2 Ensure the "Take Snapshot" button always uses the result of `getActiveSeasonId()` (already done)
- [x] 4.3 Verify the rounds list in the admin UI shows rounds filtered by active season (already done — loadRoundsList passes season_id)

## 5. Verification

- [x] 5.1 Run existing tests to confirm no regressions (`go test ./...`) — 0.459s, all pass
- [ ] 5.2 Test creating a round without season_id → 400 (manual: start app, POST /api/rounds without season_id)
- [ ] 5.3 Test creating a round with invalid season_id → 404 (manual: POST with season_id=999)
- [ ] 5.4 Test creating a round in archived season → 409 (manual: archive season, then POST)
- [ ] 5.5 Test duplicate round number in same season → 409 (manual: create two rounds with same round number)
- [ ] 5.6 Test finalizing a round with email configured → email sent, round finalized (manual)
- [ ] 5.7 Test finalizing a round without email configured → round finalized, no error (manual)
- [x] 5.8 Run pre-push checks — `go build`, `go test`, `go vet`, `gofmt` all pass
