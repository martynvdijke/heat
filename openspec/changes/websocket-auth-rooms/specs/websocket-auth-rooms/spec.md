## ADDED Requirements

### Requirement: Handshake classification
The system SHALL classify each WebSocket upgrade by deriving identity from credentials: a valid session cookie yields role `controller`; a valid player token (via `Sec-WebSocket-Protocol`) yields role `player` bound to its `racer_id`; absent credentials yields read-only role `spectator`; invalid or expired credentials SHALL cause the upgrade to be rejected with 401.

#### Scenario: Valid session becomes controller
- **WHEN** a client upgrades `GET /ws` with a valid session cookie
- **THEN** the connection is established with role `controller`

#### Scenario: Valid player token becomes player bound to racer
- **WHEN** a client upgrades `GET /ws` with a valid player token for racer 7 via `Sec-WebSocket-Protocol`
- **THEN** the connection is established with role `player` and `racer_id` 7

#### Scenario: No credentials becomes spectator
- **WHEN** a client upgrades `GET /ws` with no session cookie and no player token
- **THEN** the connection is established with role `spectator` (read-only)

#### Scenario: Invalid token is rejected
- **WHEN** a client upgrades `GET /ws` with an invalid or expired player token or session
- **THEN** the upgrade is rejected with 401 and no WebSocket connection is established

### Requirement: Connection identity
Each WebSocket connection SHALL record its role (`controller` | `player` | `spectator`), optional `racer_id` (set for `player`), and a unique connection id, and the server SHALL use this identity for delivery and authorization decisions.

#### Scenario: Identity is used for authorization
- **WHEN** a connection with role `player` and `racer_id` 7 sends a message that requires controller privilege
- **THEN** the server rejects it based on the stored role

### Requirement: Topic subscription and filtered delivery
The system SHALL support topic subscriptions via `{type:'subscribe', topics:[...]}` and SHALL tag every broadcast with a `topic`; the server SHALL deliver a broadcast only to connections subscribed to that topic. Default public read topics (`flags`, `racers`, `commentary`, `weather`, `race_state`) SHALL be subscribed for all roles by default to preserve current public behavior; sensitive topics (`telemetry`, `presence`, `race_radio`) SHALL be restricted by role/subscription.

#### Scenario: Subscribed topic is received
- **WHEN** a client subscribes to `flags` and a `flags` broadcast is emitted
- **THEN** the client receives the message with `topic:'flags'`

#### Scenario: Unsubscribed topic is not received
- **WHEN** a client is subscribed only to `flags` and a `telemetry` broadcast is emitted
- **THEN** the client does not receive the telemetry message

#### Scenario: Defaults preserve TV behavior
- **WHEN** a spectator (no credentials) connects and subscribes to default public topics
- **THEN** it receives `flags`, `racers`, `commentary`, `weather`, and `race_state` broadcasts without additional auth

### Requirement: Inbound authorization
The system SHALL enforce inbound authorization: only a connection with role `controller` (authenticated) SHALL be allowed to emit `flag`, `lap_update`, or `weather_update`; a connection with role `player` SHALL be allowed to emit `self_service` only when the message `racer_id` matches its bound `racer_id`; all other inbound messages that violate these rules SHALL be rejected and SHALL NOT be re-broadcast.

#### Scenario: Spectator cannot emit flag
- **WHEN** a `spectator` connection sends `{type:'flag', ...}`
- **THEN** the server rejects the message and does not broadcast it

#### Scenario: Player cannot spoof another racer
- **WHEN** a `player` bound to racer 7 sends `{type:'self_service', racer_id: 9, ...}`
- **THEN** the server rejects the message and does not broadcast it

#### Scenario: Controller flag is accepted
- **WHEN** a `controller` connection sends `{type:'flag', ...}`
- **THEN** the server accepts and broadcasts the flag update with its topic

#### Scenario: Player self_service for own racer is accepted
- **WHEN** a `player` bound to racer 7 sends `{type:'self_service', racer_id: 7, ...}`
- **THEN** the server accepts and broadcasts it (to authorized subscribers)

### Requirement: Targeted driver notifications
The system SHALL deliver `{type:'notify', racer_id, ...}` messages only to connections whose bound `racer_id` matches the target `racer_id`; other connections SHALL NOT receive the notification. A client MAY send `{type:'notify_ack', id}` to acknowledge a notification, and the server SHALL relay or record the ack as applicable.

#### Scenario: Only target racer receives notification
- **WHEN** the server sends `{type:'notify', racer_id: 7, id:'n1', ...}` and there are connected players for racers 7 and 9
- **THEN** only connections bound to racer 7 receive the notification

#### Scenario: Ack round-trip
- **WHEN** a player that received notification `n1` sends `{type:'notify_ack', id:'n1'}`
- **THEN** the server receives the ack for `n1`

### Requirement: Presence tracking and broadcast
The server SHALL track connected clients by role, optional `racer_id`, and `last_seen`, and SHALL broadcast presence changes (connect/disconnect) to subscribers of the `presence` topic; a controller subscribed to `presence` SHALL be able to render connected roles. Presence updates SHALL NOT be delivered to connections not subscribed to `presence`.

#### Scenario: Connect emits presence update to subscribers
- **WHEN** a new controller connects and a controller subscribed to `presence` is already connected
- **THEN** the subscribed controller receives a presence update reflecting the new connection

#### Scenario: Disconnect emits presence update
- **WHEN** a player disconnects and a controller is subscribed to `presence`
- **THEN** the controller receives a presence update reflecting the removal

#### Scenario: Presence not delivered to non-subscribers
- **WHEN** a presence change occurs and a client is not subscribed to `presence`
- **THEN** that client does not receive the presence message
