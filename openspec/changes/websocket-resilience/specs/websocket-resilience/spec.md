## ADDED Requirements

### Requirement: Reconnect with backoff
Every WebSocket page SHALL reconnect automatically after a disconnect with exponential backoff, full jitter, a capped maximum delay, and `onerror` handling; the backoff counter SHALL reset on successful `onopen`. This replaces the current partial coverage (only `controller.ts:80`, `index.ts:478`, `startlights.ts:100` with fixed 5s, no jitter; `tv.ts`/`spectator.ts`/`pitboard.ts`/`player.ts` do not reconnect).

#### Scenario: Disconnect schedules a backed-off retry
- **WHEN** the WebSocket closes or errors
- **THEN** the client schedules a reconnect after a delay computed as `min(cap, base * 2^attempt)` with full jitter `random(0, delay)` and increments the attempt counter

#### Scenario: Success resets backoff
- **WHEN** a reconnected WebSocket fires `onopen` successfully
- **THEN** the client resets its backoff attempt counter to zero so the next disconnect starts again from the base delay

#### Scenario: Simultaneous clients jitter apart
- **WHEN** multiple clients disconnect at the same time (e.g., server restart)
- **THEN** their reconnect attempts are spread over the jitter window rather than retrying in lockstep

### Requirement: Monotonic sequence numbers
Every server broadcast envelope SHALL carry a global monotonically increasing `seq` assigned from a single atomic counter, so clients can order and detect loss.

#### Scenario: Successive broadcasts have increasing seq
- **WHEN** the server sends two consecutive broadcasts
- **THEN** the second envelope has a `seq` strictly greater than the first

#### Scenario: Client tracks last seq
- **WHEN** a client receives an envelope with `seq`
- **THEN** it records that `seq` as its last seen value for gap detection

### Requirement: Gap detection
A client that observes a non-consecutive `seq` (jump greater than 1) or that has just reconnected SHALL recognize that messages were missed and initiate recovery.

#### Scenario: Skipped seq triggers resync
- **WHEN** a client whose last seen `seq` is 10 receives an envelope with `seq` 13
- **THEN** the client treats the gap as missed messages and sends a resync request

### Requirement: Hello snapshot on connect
Immediately after a successful WebSocket connect (including reconnects), the server SHALL send a `hello` snapshot containing the current state scoped to the connection's topics/identity, so the client is correct without an additional REST call. Depends on `websocket-auth-rooms` topics/identity.

#### Scenario: New client receives current state without a REST call
- **WHEN** a client establishes a WebSocket connection subscribed to flags, weather, and standings topics
- **THEN** it receives a `hello` message carrying the current flags, weather, racers, and standings (and `race_state` when `live-race-state-broadcast` is present) for those topics

### Requirement: Resync
A client MAY send `{type:'resync', topics:[...]}` and the server SHALL reply with a fresh snapshot scoped to those topics, reflecting current server state.

#### Scenario: Resync returns current state and client replaces stale state
- **WHEN** a client sends `{type:'resync', topics:['racers','flags']}` after detecting a gap
- **THEN** the server replies with a snapshot for racers and flags and the client replaces its stale collections with the snapshot contents

### Requirement: Idempotent snapshot application
Applying a snapshot SHALL be idempotent: snapshot-backed collections (racers, standings, flags, weather, race_state) are replaced wholesale, and append-only streams (commentary, race radio) are de-duplicated by stable `id` so no entry is duplicated.

#### Scenario: Applying the same snapshot twice does not duplicate entries
- **WHEN** a client applies the same snapshot twice in succession
- **THEN** keyed collections remain identical after the second apply and append streams contain no duplicate ids

### Requirement: Envelope protocol (BREAKING, internal)
The raw `[]Racer` broadcast SHALL be replaced by an envelope `{type:'racers', seq, payload: []Racer}` so it can carry `seq`; all in-repo WebSocket clients SHALL be updated to handle the envelope. Other typed broadcasts SHALL use `{type, topic, seq, payload}` with `seq` from the global counter.

#### Scenario: Racers arrive as an envelope with seq and payload
- **WHEN** the server broadcasts racer positions
- **THEN** each connected client receives a message with `type:'racers'`, a monotonic `seq`, and `payload` containing the `[]Racer` array
