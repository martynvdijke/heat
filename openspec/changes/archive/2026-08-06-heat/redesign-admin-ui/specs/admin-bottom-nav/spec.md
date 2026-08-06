## ADDED Requirements

### Requirement: Bottom tab bar is the primary admin navigation
The admin UI SHALL expose its top-level navigation as a bottom-positioned bar on viewports <768px wide and as a left side rail on viewports ≥768px, using a single shared DOM node that media-queries reflow between the two layouts. The bar SHALL contain exactly four tabs in this fixed order: Race Day, Season, Drivers, Config.

#### Scenario: Mobile bottom bar renders below content
- **WHEN** a signed-in admin opens `/admin.html` on a viewport <768px wide
- **THEN** the nav bar renders fixed at the bottom of the viewport
- **AND** the bar's bottom edge sits at `env(safe-area-inset-bottom)` so it is not obscured by an iOS home indicator
- **AND** no horizontal scrollbar is produced by the bar itself

#### Scenario: Desktop side rail renders left of content
- **WHEN** a signed-in admin opens `/admin.html` on a viewport ≥768px wide
- **THEN** the same nav node renders as a sticky left rail 200–240px wide
- **AND** the admin content area begins to the right of the rail
- **AND** no fixed bottom bar is visible

#### Scenario: Four tabs only and in fixed order
- **WHEN** the admin nav DOM is inspected
- **THEN** it contains exactly four tab buttons labelled "Race Day", "Season", "Drivers", "Config", in that order, regardless of viewport

### Requirement: Touch targets meet mobile guidance
Each tab button in the nav SHALL have a minimum touch target of 44×44 CSS pixels on viewports <768px.

#### Scenario: Minimum tap target on mobile
- **WHEN** the rendered tab button's box is measured on a viewport <768px
- **THEN** its effective tappable area is at least 44px wide and 44px tall (accounting for padding)

### Requirement: Keyboard and screen-reader accessibility
The nav SHALL be an ARIA `tablist`, each button an ARIA `tab` with `aria-controls` pointing to its pane container, and tab activation updates `aria-selected` on the active tab and `aria-hidden` on the inactive panes.

#### Scenario: Keyboard activation
- **WHEN** a user focusses a tab button and presses Enter or Space
- **THEN** that tab becomes active and its pane container content is fetched/mounted
- **AND** focus remains on the activated tab button

#### Scenario: Screen reader announces tab switch
- **WHEN** tab activation completes
- **THEN** the active tab button exposes `aria-selected="true"` and inactive tabs expose `aria-selected="false"`

### Requirement: Active tab persists across reloads via URL hash
The active tab SHALL be reflected in the URL fragment (`#race-day`, `#season`, `#drivers`, `#config`) so a reload restores the same active tab, and the default on cold load with no hash SHALL be Race Day.

#### Scenario: Reload preserves the active tab
- **WHEN** a user activates the Drivers tab and reloads the page
- **THEN** the page loads with the Drivers tab active and its panes fetched via `hx-get="/api/html/admin/drivers"`

#### Scenario: Cold load defaults to Race Day
- **WHEN** a user opens `/admin.html` with no URL hash
- **THEN** the Race Day tab is active and its panes are rendered eagerly as part of the initial response

### Requirement: prefers-reduced-motion short-circuit
When the user agent reports `prefers-reduced-motion: reduce`, the nav bar SHALL disable any entry animation and tab-switch fade.

#### Scenario: Reduced-motion user sees no animations
- **WHEN** the admin shell loads with `prefers-reduced-motion: reduce`
- **THEN** tab switches occur with no opacity transition and no bottom-bar slide animation