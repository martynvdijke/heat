## 1. Shared countdown engine

- [x] 1.1 Create `ts/startlights-core.ts`: export `StartLightsEngine` (state machine idle/counting/green/done, `LIGHT_INTERVAL`/`HOLD_ON_LIGHTS`/`GREEN_DURATION` constants, `runSequence`/`abortSequence`/`resetAllLights`, `handleCommand`) plus shared `playBeep`/`playHorn`/`getAudioContext` audio helpers
- [x] 1.2 Refactor `ts/startlights.ts` to import and drive the engine; keep the `#start-lights` existence guard and the `window.triggerStartLights`/`abortStartLights`/`resetStartLights` hooks

## 2. Controller widget

- [x] 2.1 Add widget markup to `static/controller.html` Grid card: five bulbs, status line, Abort button (data-action wiring), placed next to "Open Lights Display"
- [x] 2.2 Add `.start-light--inline` (or equivalent) compact styling in `static/style.css`, reusing the existing bulb/glow styles
- [x] 2.3 `ts/controller.ts`: open a WebSocket to `/ws`; handle `type==='flag' && flag==='startlights'` and drive the engine; wire the Abort button to broadcast `state:'abort'`

## 3. Tests

- [x] 3.1 Playwright e2e: trigger start lights from the controller → inline bulbs light sequentially and reach green; abort mid-sequence → bulbs off, status aborted
- [x] 3.2 Run `task pre-push` (gofmt, go test, vet+govulncheck, tsc, build) and fix failures
