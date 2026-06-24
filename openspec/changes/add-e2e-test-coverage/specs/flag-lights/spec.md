## ADDED Requirements

### Requirement: Start lights sequence can be triggered
The system SHALL accept a start lights trigger command via POST /api/flags with {type: "flag", flag: "startlights", state: "trigger"}.

#### Scenario: Trigger start lights
- **WHEN** a POST /api/flags request is sent with type "flag", flag "startlights", state "trigger"
- **THEN** the response status SHALL be 200 OK

### Requirement: Start lights sequence can be aborted
The system SHALL accept a start lights abort command via POST /api/flags with {type: "flag", flag: "startlights", state: "abort"}.

#### Scenario: Abort start lights
- **WHEN** a POST /api/flags request is sent with type "flag", flag "startlights", state "abort"
- **THEN** the response status SHALL be 200 OK

### Requirement: Start lights sequence can be reset
The system SHALL accept a start lights reset command via POST /api/flags with {type: "flag", flag: "startlights", state: "reset"}.

#### Scenario: Reset start lights
- **WHEN** a POST /api/flags request is sent with type "flag", flag "startlights", state "reset"
- **THEN** the response status SHALL be 200 OK

### Requirement: Invalid flag type is rejected
The system SHALL return 400 when a flag command is sent with an invalid type field.

#### Scenario: Reject invalid flag type
- **WHEN** a POST /api/flags request is sent with type set to something other than "flag"
- **THEN** the response status SHALL be 400

### Requirement: Start lights page loads correctly
The system SHALL serve the start lights display page at /static/startlights.html with the correct title and UI elements.

#### Scenario: Start lights page loads
- **WHEN** a browser navigates to /static/startlights.html
- **THEN** the page title SHALL contain "HEAT: Start Lights"
- **AND** the logo element SHALL contain "HEAT START LIGHTS"
- **AND** the status bar SHALL be visible and contain "READY"
- **AND** a close button with aria-label "Close" SHALL be present
- **AND** the compiled JavaScript at /static/js/startlights.js SHALL be loaded

### Requirement: Sound effects can be triggered via API
The system SHALL accept sound trigger commands via POST /api/sound with a "sound" field.

#### Scenario: Trigger engine sound
- **WHEN** a POST /api/sound request is sent with {sound: "engine"}
- **THEN** the response status SHALL be 200 OK

#### Scenario: Trigger finish sound
- **WHEN** a POST /api/sound request is sent with {sound: "finish"}
- **THEN** the response status SHALL be 200 OK
