## Context

Current state: `GET /ws` (`main.go:331`) upgrades without authentication. Inbound `ws.go:57-79` accepts `flag`, `self_service`, `lap_update`, `weather_update` from any client and re-broadcasts to all viewers. `BroadcastSelfService` (`ws/ws.go:189-200`) sends every player's telemetry to every connected client, with filtering only on the client (`player.ts`). There is no server-side identity, role, or authorization.

Dependency: `websocket-hardening` introduces the outbound-queue writer wrapper and connection lifecycle hardening. This change builds on that wrapper and replaces the `map[*websocket.Conn]bool` registry with a metadata-bearing struct. `websocket-resilience` (snapshots) and `live-race-state-broadcast` (subscriptions) will consume the topic/identity plumbing added here.

Existing auth primitives to reuse: `middleware` session validation for controller/admin (cookie-based), player-token validation for racer binding. All TS clients (`controller.ts`, `tv.ts`, `spectator.ts`, `pitboard.ts`, `player.ts`, `index.ts`) currently open WS without credentials or topic negotiation.

Stakeholders: race director (controller), drivers (player devices), pitboards, TV/spectator public displays.

## Goals / Non-Goals

**Goals:**
- Classify every WS connection on upgrade (controller / player / spectator) and attach identity metadata used for delivery and authorization.
- Introduce topic subscriptions with filtered delivery so sensitive data is server-filtered and public displays keep working without auth.
- Enforce inbound authorization so only authorized roles can mutate flagged state or spoof another racer.
- Deliver per-racer targeted notifications without leaking to other clients.
- Track and broadcast presence by role/racer for the controller.

**Non-Goals:**
- Sequence numbers, snapshots, and catch-up replay — owned by `websocket-resilience`.
- Reconnection/backoff strategy — owned by `websocket-resilience`.
- Full race-state topic decomposition — owned by `live-race-state-broadcast`.
- Template/i18n or audio concerns.
- Persisting presence history.

## Decisions

### D1: Auth sources — session cookie + Sec-WebSocket-Protocol subprotocol
Browser `new WebSocket()` cannot set custom headers but does send cookies automatically on the upgrade request, so controller identity is derived from the existing session cookie. Player devices authenticate via a player token carried in the `Sec-WebSocket-Protocol` header (the WebSocket subprotocol mechanism) — the server echoes the selected subprotocol on success. Rejected: query-string token (`wss://...?token=...`) — leaks in access logs, referrers, and browser history.

Alternatives considered: custom header (impossible from browser WS), first-message auth (leaves a window where the unauthenticated socket can receive/send).

### D2: Absence of credentials yields read-only spectator
No cookie and no player token → `spectator` role with read-only public topics. Only an *invalid or expired* credential rejects the upgrade (401 / close with `4401`). This preserves today's public TV/spectator displays which open WS anonymously. Breaking those displays on rollout is the primary adoption risk, so the default is permissive-read.

### D3: Topic registry + per-connection subscription set
Server maintains a topic registry (known topics → default visibility/role map) and a per-connection subscription set. Every broadcast is tagged with a `topic` field in the envelope. `broadcastToClients` iterates the registry and delivers only to subscribers. Role→default topics map:
- All roles (including spectator): `flags`, `racers`, `commentary`, `weather`, `race_state` — preserves current public behavior.
- Restricted: `telemetry` (player self + controller), `presence` (subscribers only), `race_radio` (subscribers). Sensitive topics are never in spectator defaults.

Client protocol: `{type:'subscribe', topics:[...]}` replaces the per-connection set (idempotent). Unsubscribe is a subscribe with a reduced list.

### D4: Registry value becomes a connection-metadata struct
Current registry is `map[*websocket.Conn]bool`. Replace with `map[*websocket.Conn]*ConnMeta` where `ConnMeta` holds `{id, role, racerID *int, topics map[string]bool, lastSeen, writer}`. The `writer` field is the outbound-queue wrapper introduced by `websocket-hardening` — coordinate the struct change so hardening lands first or in the same patch. `ponytail: single mutex guarding registry; per-topic shards if fan-out becomes hot path.`

### D5: Targeted delivery filters server-side by racer_id
`{type:'notify', racer_id, ...}` delivery iterates the registry and writes only to connections where `meta.racerID == racer_id`. This fixes the `BroadcastSelfService` leak at `ws/ws.go:189-200` which currently broadcasts all player telemetry to all clients. Client-side filtering is removed as a security boundary; TS `player.ts` keeps a local guard only as defense-in-depth.

### D6: Backwards-compatible envelope
Existing `type` values (`flag`, `self_service`, `lap_update`, `weather_update`, `notify`, etc.) are unchanged. A new `topic` field is added to every outbound envelope so old clients that ignore it still receive public topics (until they adopt subscriptions). New `subscribe` and `notify_ack` message types are additive.

## Risks / Trade-offs

- [Breaking public TV/spectator displays if default topics misconfigured] → All roles default-subscribe to the public topic set; spectator role tested explicitly; rollout check: open TV page without credentials still receives `flags`/`racers`.
- [Token via subprotocol is unfamiliar and may surprise proxies] → `Sec-WebSocket-Protocol` is standard; fallback is documented; query-string fallback explicitly not offered to avoid log leakage (trade-off: slightly more TS wiring).
- [Registry scan on every broadcast scales as O(connections × topics)] → Acceptable at track scale (<200 connections); `ponytail: linear scan, per-topic index if load testing shows contention`.
- [Presence chatter on flaky links] → Debounce disconnect broadcast by the hardening layer's grace period; presence updates are `presence` topic only (opt-in).
- [Coordination with websocket-hardening] → If hardening's writer wrapper is not yet merged, this change vendors a minimal interface for the writer field and adapts when hardening lands.

## Migration Plan

1. Land `websocket-hardening` first (writer wrapper + tests).
2. Deploy this change behind the existing `/ws` handler: add classification, metadata struct, subscribe handler, filtered broadcast, inbound auth, targeted notify, presence.
3. Update all TS clients to supply credentials (cookie automatically; player token via subprotocol) and send `{type:'subscribe', topics:[...]}` on open. Public pages subscribe to public topics explicitly.
4. Verify: unauthenticated TV/spectator still receive public topics; invalid token gets 401; controller flag write succeeds; player spoof rejected.
5. Rollback: revert to `map[*websocket.Conn]bool` registry and unfiltered broadcast; clients ignore unknown `topic` field so rollback is safe.

## Open Questions

- Exact player-token issuance/rotation flow (out of scope for WS; assumed existing validation helper is reusable — confirm with auth owner).
- Whether `tv` and `pitboard` deserve distinct roles or map to `spectator` with different default subscriptions (proposal lists them; design maps them to `spectator` + topic profile unless a distinct permission is identified during implementation).
