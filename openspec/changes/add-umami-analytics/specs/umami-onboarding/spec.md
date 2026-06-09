## ADDED Requirements

### Requirement: Inline onboarding hints in the Analytics settings pane

The system SHALL display contextual onboarding hints in the Analytics settings pane to guide admins through setting up Umami for the first time.

#### Scenario: Onboarding hints shown when Umami is not yet configured
- **WHEN** an admin opens the Analytics tab
- **AND** the Umami URL and Website ID fields are both empty
- **THEN** the pane shows a small info banner: "Welcome! To get started, you need a self-hosted Umami instance. Enter your Umami server URL and Website ID below."
- **AND** each input field shows an inline hint below it explaining how to find the value

#### Scenario: Onboarding hints hidden after configuration
- **WHEN** an admin opens the Analytics tab
- **AND** Umami settings have been saved (URL and Website ID are populated)
- **THEN** the welcome banner is hidden
- **AND** normal settings display shows instead

#### Scenario: Save feedback via toast notification
- **WHEN** admin saves Umami settings successfully
- **THEN** the system shows a green success toast: "Analytics settings saved!"
- **AND** the toast auto-dismisses after 3 seconds

#### Scenario: Save failure feedback
- **WHEN** admin saves Umami settings and the server returns an error
- **THEN** the system shows a red error toast with the error message
- **AND** the toast auto-dismisses after 5 seconds

### Requirement: Toast notification component

The system SHALL provide a reusable toast notification component in TypeScript that can be called from any admin panel page.

#### Scenario: Toast displayed with correct styling
- **WHEN** `showToast('message', 'success')` is called
- **THEN** a notification appears in the top-right corner of the viewport
- **AND** it has a green background for success, red for error
- **AND** it contains the provided message text
- **AND** it auto-dismisses after the configured duration
