## ADDED Requirements

### Requirement: Commentary entries are stored and retrievable
The system SHALL store commentary entries (id, race_id, lap, racer_id nullable, message, template_key nullable, created_at) and SHALL expose them via `GET /api/commentary` with `race_id`, `since` (only entries with id > since), and `limit` filters, ordered newest-first.

#### Scenario: Manual entry is stored
- **WHEN** a race director posts a manual entry via `POST /api/commentary`
- **THEN** the entry is stored with its message, optional racer and lap, and a timestamp

#### Scenario: Retrieval with filters
- **WHEN** `GET /api/commentary?race_id=0&since=42&limit=20` is requested
- **THEN** only entries for race 0 with id > 42 are returned, up to 20, newest first

### Requirement: Auto-generation from race events and weather
Adding a race event (`POST /api/race-events`) or changing weather (`POST /api/weather`) SHALL generate a commentary entry from the template engine, substituting the driver name (and overtake target when present) and lap, and store + broadcast it.

#### Scenario: Overtake generates commentary
- **WHEN** an overtake event is recorded for a driver (with a target driver) on lap 5
- **THEN** a commentary entry is created using an overtake template with the driver and target names and lap 5

#### Scenario: Weather change generates commentary
- **WHEN** weather is set to Wet from lap 10
- **THEN** a commentary entry is created announcing the condition change at lap 10

#### Scenario: Manual entry passes through verbatim
- **WHEN** a race director posts a manual commentary message
- **THEN** the message is stored exactly as written, with no template substitution

### Requirement: Commentary broadcasts over WebSocket
Every stored commentary entry SHALL be broadcast to all connected clients as a WS message with `type: 'commentary'` and the entry fields.

#### Scenario: Entry appears on connected clients
- **WHEN** a commentary entry is created (manual or auto-generated)
- **THEN** every connected client receives a `type:'commentary'` message containing it

### Requirement: TV page shows a commentary ticker
The TV page SHALL render a commentary ticker below the leaderboard that appends entries from WS messages, catches up via polling (`GET /api/commentary?since=<lastId>` every 5s), fades entries older than 30s, pauses on hover, and is announced via `aria-live="polite"`.

#### Scenario: Manual entry appears on the TV ticker
- **WHEN** a race director posts a manual entry from the controller
- **THEN** the entry appears in the TV ticker without a page reload

#### Scenario: Old entries fade out
- **WHEN** a ticker entry is older than 30 seconds
- **THEN** it fades out and is removed

#### Scenario: Poll catches up after reconnect
- **WHEN** a TV page reconnects after a disconnect
- **THEN** it fetches entries newer than its last seen id

### Requirement: Controller can post manual commentary
The controller page SHALL provide a manual commentary input (message, optional driver) in the Tracking card that POSTs to `/api/commentary`, and SHALL show a compact feed of recent entries.

#### Scenario: Race director announces
- **WHEN** a race director types a message and submits it on the controller
- **THEN** the message is posted and appears in the controller's commentary feed
