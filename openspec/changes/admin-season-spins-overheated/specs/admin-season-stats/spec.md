## ADDED Requirements

### Requirement: Admin Season tab shows spins and overheated per driver
The Driver Statistics table in the admin Season tab's Stats pane SHALL display Spins and Overheated columns populated from the driver's season stats.

#### Scenario: Stats table renders spins and overheated columns
- **WHEN** an admin opens the Season tab's Stats pane and the driver statistics are loaded
- **THEN** the table shows Spins and Overheated columns
- **AND** each driver row shows that driver's spins and overheated totals

#### Scenario: Zero values render as 0
- **WHEN** a driver has no recorded spins or overheated incidents
- **THEN** the row shows 0 in both columns

### Requirement: Admin can edit spins and overheated from the stats modal
The add/edit driver-stats modal SHALL provide spins and overheated number inputs, SHALL prefill them with the driver's current values when editing, and SHALL include them when saving.

#### Scenario: Modal prefills spins and overheated when editing
- **WHEN** an admin opens the stats modal to edit a driver with recorded spins and overheated
- **THEN** the spins and overheated inputs contain the driver's current values

#### Scenario: Saving updated spins and overheated persists them
- **WHEN** an admin changes the spins or overheated values in the modal and saves
- **THEN** the saved values are sent to the stats save endpoint
- **AND** the Stats table re-renders showing the updated values

#### Scenario: Modal save does not zero spins and overheated
- **WHEN** an admin edits a driver's other fields via the modal without changing spins or overheated
- **THEN** the driver's spins and overheated values remain unchanged after saving

### Requirement: Inline stats save round-trips spins and overheated
The inline stats save path (outside the modal) SHALL also include spins and overheated in its payload so saving never resets them.

#### Scenario: Inline save preserves spins and overheated
- **WHEN** an admin uses the inline stats save for a driver with recorded spins and overheated
- **THEN** the payload includes the spins and overheated values
- **AND** the stored values remain unchanged
