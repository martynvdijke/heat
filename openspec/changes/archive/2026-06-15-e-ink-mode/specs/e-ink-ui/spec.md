# e-ink-ui Specification

## Purpose
The e-ink mode adapts HEAT for e-ink displays used as race-day companion screens — providing high-contrast standings, readable-from-a-distance tables, and simplified race control suitable for tabletop or outdoor use.

## Requirements

### Requirement: E-ink mode toggle
The application SHALL provide a mechanism to enable e-ink mode via URL parameter (`?eink=1`) and optionally persist via cookie or admin setting.

#### Scenario: Enable via URL parameter
- **WHEN** a user appends `?eink=1` to any page URL
- **THEN** the page SHALL render in e-ink mode for that session
- **AND** a cookie SHALL be set to persist the preference

#### Scenario: Enable via admin settings
- **WHEN** an admin toggles "E-ink mode" in settings and saves
- **THEN** e-ink mode SHALL be enabled for all users site-wide

### Requirement: High contrast palette
All e-ink mode pages SHALL use strict black-on-white with no gradients, shadows, or semi-transparency.

#### Scenario: Colors
- **WHEN** e-ink mode is active
- **THEN** all text SHALL be `#000000` on `#ffffff` background
- **THEN** secondary/muted text SHALL use `#333333` on `#ffffff`
- **THEN** table borders and dividers SHALL use `#cccccc` solid lines
- **THEN** alternating row colors SHALL be removed in favor of solid white with border separators

### Requirement: No motion or CSS effects
- **WHEN** e-ink mode is active
- **THEN** all CSS transitions, animations, keyframes SHALL be disabled
- **THEN** no `box-shadow`, `text-shadow`, `backdrop-filter`, `gradient`, or `filter` SHALL render

### Requirement: Large touch targets
- **WHEN** e-ink mode is active
- **THEN** all buttons, links, and interactive elements SHALL be minimum 48×48px
- **THEN** race control action buttons SHALL be minimum 56×56px with visible borders
- **THEN** touch targets SHALL have 8px minimum spacing

### Requirement: Race control display
Race control elements SHALL be simplified for high-contrast readability.

#### Scenario: Flag indicators
- **WHEN** viewing race control in e-ink mode
- **THEN** flag status SHALL use text labels ("GREEN", "YELLOW", "RED", "CHECKERED") instead of colored circles
- **THEN** flag labels SHALL use solid black border and white fill with bold text
- **THEN** active flag SHALL use black fill with white text for maximum contrast

#### Scenario: Race timer
- **WHEN** viewing race control in e-ink mode
- **THEN** the race timer SHALL display as monospaced bold black text on white
- **THEN** no blinking or pulsing effects SHALL be used on the timer

### Requirement: Standings table
Standings and leaderboard tables SHALL be optimized for e-ink.

#### Scenario: Table layout
- **WHEN** viewing standings in e-ink mode
- **THEN** table header text SHALL be minimum 16px bold
- **THEN** table cell text SHALL be minimum 15px
- **THEN** position changes (up/down) SHALL use text arrows (↑ / ↓) with position numbers
- **THEN** row selection/hover highlighting SHALL use solid black border, not background color change

#### Scenario: Points display
- **WHEN** viewing the leaderboard in e-ink mode
- **THEN** championship points SHALL be displayed as bold numbers
- **THEN** podium positions SHALL use (1st/2nd/3rd) text labels, not medal icons

### Requirement: Driver view
The driver detail view SHALL be simplified.

- **WHEN** viewing a driver page in e-ink mode
- **THEN** stats SHALL be displayed in a clean grid layout with solid borders
- **THEN** trend indicators SHALL use arrows + text, not color
- **THEN** charts SHALL use solid black lines on white with no fills or gradients
