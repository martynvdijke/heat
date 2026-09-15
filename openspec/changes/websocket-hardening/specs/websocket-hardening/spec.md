## ADDED Requirements

### Requirement: Heartbeat liveness detection
The system SHALL ping every WebSocket connection on a fixed interval, SHALL set a read deadline refreshed by the pong handler (`SetReadDeadline` + `SetPongHandler`), and SHALL close and remove connections that fail to return a pong within the deadline.

#### Scenario: Dead client is reaped
- **WHEN** a connected client stops responding to pings and misses its pong within the read deadline
- **THEN** the server closes the connection and removes it from the active client set

#### Scenario: Live client stays connected
- **WHEN** a connected client responds to pings with pongs before the read deadline
- **THEN** the server keeps the connection open and refreshes its read deadline

### Requirement: Message size limit
The system SHALL set a read limit of 64 KiB on each WebSocket connection via `SetReadLimit` and SHALL close the connection when an inbound frame exceeds the limit.

#### Scenario: Oversized message rejected
- **WHEN** a client sends a frame larger than 64 KiB
- **THEN** the server closes the connection and the oversized payload is not processed

### Requirement: Per-connection write isolation
The system SHALL provide each WebSocket client with an independent buffered outbound queue and a dedicated writer goroutine; broadcast enqueues to each queue SHALL NOT block on any other client's socket, and each write SHALL be bounded by a write deadline.

#### Scenario: Slow client does not block others
- **WHEN** one client's socket is stalled and another client is healthy
- **THEN** broadcasts to the healthy client are delivered without delay caused by the stalled client

#### Scenario: Write deadline exceeded disconnects the client
- **WHEN** a write to a client exceeds its write deadline
- **THEN** the server closes and removes that client's connection

### Requirement: Lagging client eviction
The system SHALL evict a client whose outbound queue is full at broadcast time by disconnecting and cleaning up that client, and SHALL never block the broadcast producer waiting for queue space.

#### Scenario: Full outbound queue evicts only that client
- **WHEN** a client's outbound queue is full when a broadcast is enqueued
- **THEN** that client is disconnected and removed while other clients remain connected and the broadcast producer does not block

### Requirement: Observable broadcast backpressure
The system SHALL buffer server broadcast channels and SHALL drop the newest message non-blocking when a channel is full, incrementing the `heat_ws_broadcast_dropped_total` counter and emitting a log line for each drop.

#### Scenario: Dropped broadcast increments counter and logs
- **WHEN** a broadcast channel is full and a new message is sent
- **THEN** the newest message is dropped, `heat_ws_broadcast_dropped_total` increments by one, and a log line is emitted
