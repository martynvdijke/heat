## Why

The start light system is fully built (`ts/startlights.ts` + `static/startlights.html`) but the operator has no inline feedback: the controller page only offers "Trigger Start Lights" and a link that opens the full-screen display in a separate tab. The operator must switch windows to see the countdown, and `controller.ts` never handles `flag:startlights` messages — `startlights.ts` even carries a comment noting "The controller.ts will need to handle startlights flag messages". The WS infrastructure (`FlagBroadcast` → `BroadcastFlags`) already delivers `flag:startlights` commands to every client, so the controller can mirror the sequence locally with no backend work.

## What Changes

- **Extract a shared countdown engine** into a new `ts/startlights-core.ts` (state machine: idle/counting/green/done, the existing timings `LIGHT_INTERVAL=1000`, `HOLD_ON_LIGHTS=1000`, `GREEN_DURATION=3000`, plus the WebAudio beep/horn helpers). `ts/startlights.ts` is refactored to use it; the `window.triggerStartLights` / `window.abortStartLights` / `window.resetStartLights` hooks are preserved.
- **Embed an inline start lights widget** in `static/controller.html` (Grid card, next to "Open Lights Display"): five compact bulbs, a status line, and an Abort button.
- **`ts/controller.ts` opens its own WebSocket connection** (its first) with a handler for `type === 'flag' && flag === 'startlights'` that drives the shared engine — the same pattern `startlights.html` uses today. Triggering the countdown from the controller posts `/api/flags` as before; the broadcast round-trip runs the sequence in the inline widget.
- **No backend changes** — `POST /api/flags` with `flag:startlights` and the existing flag broadcast are unchanged.
- **Tests**: Playwright e2e covering trigger → bulbs light sequentially → green state, and abort → reset.

## Capabilities

### New Capabilities
- `controller-start-lights`: The inline start lights widget on the controller page — shared countdown engine, WS-driven state, trigger/abort controls.

### Modified Capabilities
<!-- No archived specs exist yet (openspec/specs/ is empty), so no modified capabilities. -->

## Impact

- `ts/startlights-core.ts` (new): shared engine + audio.
- `ts/startlights.ts`: refactored to import the engine; window hooks kept.
- `static/controller.html`: widget markup in the Grid card.
- `ts/controller.ts`: WS connection + `flag:startlights` handling, Abort action.
- `static/style.css`: compact inline variant of `.start-light` styles.
- Tests: Playwright (new or extended controller spec). No Go changes.
