## ADDED Requirements

### Requirement: Controller shows an inline start lights widget
The controller page SHALL render a compact five-bulb start lights widget (bulbs, status line, Abort button) inside the Grid card, visible on page load without opening any other tab.

#### Scenario: Widget renders on load
- **WHEN** an operator opens the controller page
- **THEN** the Grid card shows five start light bulbs
- **AND** a status line reads "START LIGHTS • READY"

#### Scenario: Widget shows sequence state
- **WHEN** a start lights sequence is running
- **THEN** the widget's bulbs reflect the current countdown state
- **AND** the status line shows the current phase

### Requirement: Triggering start lights runs the countdown in the widget
Pressing "Start Lights" in Quick Actions SHALL start the shared countdown engine, so the inline widget lights the bulbs red one by one at 1s intervals, holds with all five lit, then turns them green with the GO horn before returning to idle.

#### Scenario: Trigger runs the full sequence inline
- **WHEN** an operator presses "Start Lights"
- **THEN** bulb 1 lights red after the first interval
- **AND** bulbs 2–5 light at subsequent 1s intervals
- **AND** after the hold all bulbs turn green and the status shows the race-start state
- **AND** the widget returns to READY after the green phase

### Requirement: Widget state is driven by the flag broadcast
The controller's WebSocket connection SHALL handle `type:'flag'` / `flag:'startlights'` messages (states `sequence`, `abort`, `reset`) and drive the same engine the standalone page uses.

#### Scenario: Sequence command starts the widget
- **WHEN** a `startlights` sequence command arrives over the controller's WebSocket
- **THEN** the inline widget starts the countdown

#### Scenario: Abort command resets the widget
- **WHEN** a `startlights` abort command arrives over the WebSocket during a countdown
- **THEN** all bulbs turn off
- **AND** the status line shows the aborted state

### Requirement: Operator can abort from the widget
The widget SHALL provide an Abort button that broadcasts `state:'abort'` via `POST /api/flags` and resets the inline widget immediately.

#### Scenario: Abort button stops the countdown
- **WHEN** an operator presses Abort while the countdown is running
- **THEN** the running sequence stops
- **AND** all bulbs turn off and the status shows aborted

### Requirement: Standalone and controller share one engine
The countdown engine (state machine and timing constants) SHALL live in a single shared module used by both `startlights.html` and the controller page, so their behavior cannot diverge.

#### Scenario: Timing constants are defined once
- **WHEN** the shared engine module is inspected
- **THEN** `LIGHT_INTERVAL`, `HOLD_ON_LIGHTS`, and `GREEN_DURATION` are defined in the shared module only

### Requirement: Standalone display page keeps working
The full-screen start lights page SHALL remain functional as a standalone display, including its own WebSocket handling and the `window.triggerStartLights` / `abortStartLights` / `resetStartLights` hooks.

#### Scenario: Standalone page still runs sequences
- **WHEN** an operator opens `/static/startlights.html` and triggers start lights
- **THEN** the full-screen sequence runs exactly as before
- **AND** the controller's inline widget (when open) runs the same sequence in sync
