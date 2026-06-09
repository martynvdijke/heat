## ADDED Requirements

### Requirement: Custom events dispatched for key HEAT actions

The system SHALL dispatch custom Umami events for significant in-app actions using the `window.umami.track()` API, when the Umami tracking script is loaded and enabled.

#### Scenario: Event dispatched when a flag change occurs
- **WHEN** a flag is thrown or cleared during a race session
- **THEN** the system dispatches `umami.track('flag_change', { flag: '<flag_type>', action: 'thrown'|'cleared' })`

#### Scenario: Event dispatched on race start/stop
- **WHEN** a race starts (round snapshot taken, race timer begins)
- **THEN** the system dispatches `umami.track('race_start', { track: '<track_name>' })`
- **WHEN** a race ends or is stopped
- **THEN** the system dispatches `umami.track('race_stop', { track: '<track_name>' })`

#### Scenario: Event dispatched on lap record
- **WHEN** a lap record is set by a racer
- **THEN** the system dispatches `umami.track('lap_record', { racer: '<racer_name>', track: '<track_name>', time: '<lap_time>' })`

#### Scenario: Events silently skipped when Umami is disabled
- **WHEN** Umami tracking is disabled in settings or the tracking script has not loaded
- **THEN** the system does NOT throw errors when attempting to dispatch events
- **AND** the event dispatch call is a no-op

### Requirement: Type-safe event tracking helper

The system SHALL provide a TypeScript helper function for dispatching Umami events with proper type checking, to make event tracking consistent across the codebase.

#### Scenario: Helper safely dispatches when Umami is available
- **WHEN** `trackUmamiEvent('race_start', { track: 'Monza' })` is called
- **AND** `window.umami` is available
- **THEN** `window.umami.track('race_start', { track: 'Monza' })` is called

#### Scenario: Helper safely no-ops when Umami is unavailable
- **WHEN** `trackUmamiEvent('some_event', {})` is called
- **AND** `window.umami` is NOT available
- **THEN** no error is thrown
- **AND** the call is silently ignored
