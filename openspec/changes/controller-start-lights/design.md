## Context

`ts/startlights.ts` implements the full F1-style sequence: five red bulbs light at 1s intervals, a 1s hold on all lights, then all-green with a horn and a 3s green phase, then back to idle. Commands arrive over WS as `type:'flag'`, `flag:'startlights'`, `state: sequence|abort|reset` (broadcast by `BroadcastFlags` in `ws/ws.go` from `POST /api/flags`). The standalone page connects its own WS and runs the sequence locally with deterministic timings, so all listeners stay in sync.

`static/controller.html` already loads `/static/js/startlights.js` (line 327) and exposes the trigger button, but the controller page has no `#start-lights` element and `ts/controller.ts` has **no WebSocket connection at all** — it is purely polling + `POST /api/flags`. The sequence state therefore lives only on the standalone display tab.

## Goals / Non-Goals

**Goals:**
- Inline five-bulb widget on the controller page that mirrors the countdown in real time.
- Single shared engine module so the standalone page and controller cannot drift.
- Trigger and Abort from the controller, with the widget reflecting the broadcast state.
- Keep the full-screen "Open Lights Display" page fully working.

**Non-Goals:**
- Backend/API changes (flags flow is unchanged).
- TV/spectator/pitboard start-light displays.
- Audio asset files (synthesized WebAudio beeps stay).
- Server-authoritative timing — clients run the same deterministic timers, as today.

## Decisions

### D1: Extract the engine into `ts/startlights-core.ts`
Move the state machine (`lightsState`, `runSequence`, `abortSequence`, `resetAllLights`, timers, `playBeep`/`playHorn`/`getAudioContext`) into a new module exporting a `StartLightsEngine` (start/abort/reset/handleCommand + render hooks). `ts/startlights.ts` becomes a thin wrapper wiring the engine to the standalone DOM. Window hooks (`triggerStartLights` etc.) stay exported for the controller's existing `data-action="triggerStartLights"` handler.

### D2: Controller gets its own WebSocket connection
`ts/controller.ts` opens `/ws` and handles `type === 'flag' && flag === 'startlights'` exactly like `startlights.html` does today. Triggering still POSTs `/api/flags {state:'sequence'}`; the broadcast round-trip drives the inline engine. Abort broadcasts `state:'abort'` and resets the widget immediately.

### D3: Widget placement and styling
Compact widget in the Grid card, after the Quick Actions row and before/next to "Open Lights Display": five bulbs reusing `.start-light`-family styles with a smaller inline variant (e.g. `.start-light--inline`), a status line (mirroring `#start-status-bar`), and an Abort button (visible/active while a sequence is running). The existing "Open Lights Display" link remains for the big screen.

### D4: Audio on the controller
The inline widget plays the same beeps/horn (operator feedback). No mute toggle in this change.

### D5: Tests
Playwright: trigger from the controller → assert bulb 1 is lit ~1s later and the widget reaches the green phase; abort during the sequence → all bulbs off and status shows aborted. Because timings are fixed constants, the e2e can assert intermediate states with generous waits.

## Risks / Trade-offs

- [Two WS consumers reacting to the same broadcast] → Both run the identical engine; this is the existing standalone-page behavior, now extended to the controller.
- [Timer drift between devices] → Inherent to the current design (identical local timers); acceptable for a physical board game.
- [startlights.js double-init on controller] → The current `DOMContentLoaded` guard (`#start-lights` exists?) already prevents standalone init on the controller page; the refactor keeps that guard in `startlights.ts` while `controller.ts` drives the engine itself.
