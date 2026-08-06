## ADDED Requirements

### Requirement: Initial admin shell renders only the active tab's panes
The initial `/admin.html` response SHALL contain the nav, the active tab's panes, and the active tab's modals only. Panes and modals belonging to inactive tabs SHALL NOT be present in the initial DOM.

#### Scenario: Cold-load DOM contains only Race Day content
- **WHEN** a signed-in admin opens `/admin.html` with no hash (default to Race Day)
- **THEN** the response HTML contains the Race Day panes and the modals used by Race Day only
- **AND** no element with `data-tab-pane="season"`, `data-tab-pane="drivers"`, or `data-tab-pane="config"` is present in the DOM

### Requirement: Non-active tabs are fetched on first activation via htmx
Activating an inactive tab SHALL issue an htmx `GET /api/html/admin/{tab}` request whose response replaces the contents of a single shared `#admin-tab-container` element. The fetched panes and modals SHALL then be live in the DOM for that tab.

#### Scenario: First activation of a tab triggers a fetch
- **WHEN** a user on the freshly loaded shell clicks the Drivers tab for the first time
- **THEN** a `GET /api/html/admin/drivers` request is issued
- **AND** the response HTML is inserted into `#admin-tab-container` via htmx swap
- **AND** the Drivers panes are now interactive (e.g., its "Add Racer" button opens the racer modal)

#### Scenario: Active-tab content is removed before the new tab's content mounts
- **WHEN** a user activates the Season tab while the Drivers tab is active
- **THEN** the previous tab's panes are removed from `#admin-tab-container` before the Season panes are mounted

### Requirement: Already-mounted tabs are not re-fetched
Re-activating a tab that has already been mounted SHALL NOT issue another network request. The client SHALL short-circuit the activation using a `data-tab-mounted` attribute or equivalent state.

#### Scenario: Warm cache re-activation is instant
- **WHEN** a user activates Drivers, switches to Season, then switches back to Drivers
- **THEN** no `GET /api/html/admin/drivers` request is issued for the second Drivers activation
- **AND** the Drivers panes are visible within ≤200ms of the click

### Requirement: Tab-switch latency SLO
A warm tab switch (no network) SHALL complete within 200ms. A cold tab switch (first activation, over the wire) SHALL complete within 1s on a typical LAN connection.

#### Scenario: Warm switch meets 200ms SLO
- **WHEN** a tab that has already been mounted is re-activated
- **THEN** the new tab's panes are visible (paint complete) within 200ms of the click event

#### Scenario: Cold switch meets 1s SLO on LAN
- **WHEN** a tab is activated for the first time over a LAN connection
- **THEN** the new tab's panes are visible within 1s of the click event

### Requirement: Loading indicator shows during cold fetch
While a cold tab fetch is in flight, a visible loading indicator SHALL be shown in `#admin-tab-container` so the user perceives progress.

#### Scenario: Indicator appears then disappears
- **WHEN** a user clicks an inactive tab
- **THEN** a skeleton/spinner is shown inside the container
- **AND** the indicator is removed when the `htmx:afterOnLoad` event fires for that swap

### Requirement: Initial-load eager-render of the active tab remains no-JS friendly
The active tab (Race Day by default) SHALL be server-rendered into the initial response, so that a no-JS client sees functioning Race Day panes and modals even though it cannot switch to other tabs.

#### Scenario: No-JS user can operate the default tab
- **WHEN** a browser with JavaScript disabled opens `/admin.html`
- **THEN** the Race Day panes and modals are present and interactive (forms submit via standard POST, modals are reachable by anchor)
- **AND** activating other tabs is unavailable (documented limitation)