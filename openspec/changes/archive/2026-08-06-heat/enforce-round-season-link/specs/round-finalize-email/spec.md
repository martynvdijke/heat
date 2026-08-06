## ADDED Requirements

### Requirement: Email sent on round finalization
When a round snapshot is finalized via `/api/rounds/finalize`, the system SHALL send race result emails to all racers who have an email address configured in `racer_emails`.

#### Scenario: Successful email send on finalize
- **WHEN** an admin finalizes a round via `PATCH /api/rounds/finalize?id=<id>`
- **AND** email settings are configured (SMTP host, from address, enabled)
- **AND** at least one racer has an email address configured
- **THEN** the round is marked as final
- **AND** an email is sent to each racer with round results (race name, date, positions, points)

#### Scenario: No email when SMTP not configured
- **WHEN** an admin finalizes a round
- **AND** email settings are not configured or disabled
- **THEN** the round is finalized successfully
- **AND** no email is sent (no error)

#### Scenario: No email when no racer emails configured
- **WHEN** an admin finalizes a round
- **AND** no racers have email addresses configured
- **THEN** the round is finalized successfully
- **AND** no email is sent (no error)

#### Scenario: Email content includes round results
- **WHEN** a round finalization triggers an email
- **THEN** the email body SHALL include: round name, race date, driver positions, points, DNF/DNS status

### Requirement: Async email sending
Email sending during round finalization SHALL happen asynchronously (non-blocking) so the finalize API response is not delayed.

#### Scenario: API response before email send completes
- **WHEN** an admin finalizes a round
- **THEN** the API returns HTTP 200 immediately after the round is marked final
- **AND** emails are sent in the background
