## ADDED Requirements

### Requirement: Per-category volume control
Every sound category (`engine`, `horn`, `finish`, `crash`, `flag`) SHALL have a volume setting from 0 (muted) to 1 (full) that is applied to all playback of that category on the device.

#### Scenario: Volume applies to playback
- **WHEN** a device plays a sound with its category volume set to 0.5
- **THEN** the sound plays at half volume

#### Scenario: Muted category is silent
- **WHEN** a category volume is set to 0 and a sound of that category is triggered
- **THEN** no audio is played

#### Scenario: Default volume is full
- **WHEN** a device has no stored settings
- **THEN** all categories play at their current (full) synthesized volume

### Requirement: Settings persist per device
Volume and override settings SHALL be stored in `localStorage` under a versioned key and survive page reloads; corrupt or missing stored data SHALL fall back to defaults without errors.

#### Scenario: Settings survive reload
- **WHEN** a user adjusts volumes and reloads the TV page
- **THEN** the stored values are applied and shown in the modal

#### Scenario: Corrupt stored data falls back
- **WHEN** stored settings are invalid JSON or have an unexpected shape
- **THEN** defaults are used and the UI shows defaults

### Requirement: Custom sound overrides
A user SHALL be able to upload an audio file (MIME `audio/mpeg`, `audio/ogg`, or `audio/wav`, ≤ 2 MB) for any category; the device SHALL play the uploaded file instead of the synthesized tone, and SHALL allow removing the override or resetting all settings.

#### Scenario: Upload replaces the tone
- **WHEN** a user uploads a valid audio file for a category
- **THEN** triggering that category plays the uploaded file at the category volume

#### Scenario: Invalid upload is rejected
- **WHEN** a user uploads a file with an unsupported type or over 2 MB
- **THEN** the upload is rejected with feedback
- **AND** no override is stored

#### Scenario: Override can be removed
- **WHEN** a user removes the override for a category
- **THEN** the category plays its default synthesized tone again

#### Scenario: Reset restores defaults
- **WHEN** a user presses reset-to-defaults
- **THEN** all volumes return to full and all overrides are removed

### Requirement: Accessible settings modal on the TV page
The TV page SHALL provide a settings modal with a labeled trigger button; every control SHALL have an associated label, Esc SHALL close the modal, and focus SHALL return to the trigger.

#### Scenario: Modal opens and closes
- **WHEN** a user activates the sound settings button
- **THEN** the modal opens with volume sliders and upload controls for each category
- **AND** pressing Esc closes it and focus returns to the trigger button

### Requirement: Default behavior unchanged
When no settings have been changed, triggering sound FX SHALL behave exactly as before this change.

#### Scenario: Untouched device plays defaults
- **WHEN** a device with no custom settings receives a sound command
- **THEN** the original synthesized tone plays
