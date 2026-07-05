## ADDED Requirements

### Requirement: Any hex car color renders with that exact color in the leaderboard
The leaderboard (`/`, served by `ts/index.ts`) SHALL render a racer's `car_color` as that exact color in all four color-display sites: the leaderboard table dot, the Leaflet map marker fill, the driver-stats modal dot, and the qualification grid dot. The color SHALL be applied via an inline `style` attribute (e.g. `style="background:#800080"`), not via a per-value CSS class.

#### Scenario: Hex color set in admin appears on the leaderboard
- **WHEN** an admin sets a racer's `car_color` to `#800080` and the leaderboard is loaded
- **THEN** the racer's row color dot has `background` style equal to `#800080` (or the browser's normalized `rgb(128, 0, 128)`)
- **AND** the racer's map marker is filled with `#800080`
- **AND** the stats modal dot for that racer has `background` style equal to `#800080`
- **AND** the qualification grid dot for that racer has `background` style equal to `#800080`

#### Scenario: Hex color without leading hash renders
- **WHEN** a racer's `car_color` is `800080` (no `#` prefix) as stored in the DB
- **THEN** the leaderboard renders the dot with `background:#800080`
- **AND** no invalid CSS class `color-indicator.800080` is emitted

### Requirement: Legacy named color values continue to render in their existing hex
The 9 legacy named color values (`red`, `blue`, `green`, `yellow`, `grey`, `silver`, `black`, `purple`, `orange`) SHALL render in the leaderboard with the same hex values they use today (`red`→`#ff4444`, `blue`→`#4444ff`, `green`→`#44ff44`, `yellow`→`#ffff44`, `grey`→`#aaaaaa`, `silver`→`#aaaaaa`, `black`→`#333333`, `purple`→`#9b59b6`, `orange`→`#e67e22`). No visual regression for seeded racers.

#### Scenario: Seeded red racer renders as #ff4444
- **WHEN** the leaderboard is loaded with a racer whose `car_color` is `red` (seed data)
- **THEN** the racer's color dot has `background` style equal to `#ff4444`
- **AND** the map marker is filled with `#ff4444`

#### Scenario: Seeded silver racer renders as #aaaaaa
- **WHEN** the leaderboard is loaded with a racer whose `car_color` is `silver`
- **THEN** the racer's color dot has `background` style equal to `#aaaaaa`

### Requirement: Invalid or empty car color falls back to a neutral default
A `car_color` value that is empty, `null`, or not a valid 6-digit hex (after `#`-prefix normalization) and not one of the 9 named colors SHALL render as a neutral default (`#cccccc`) and SHALL NOT throw a JavaScript error or emit an invalid CSS value.

#### Scenario: Empty car color renders neutral default
- **WHEN** a racer's `car_color` is an empty string
- **THEN** the color dot has `background` style equal to `#cccccc`
- **AND** no JavaScript error is thrown during render

#### Scenario: Invalid car color renders neutral default
- **WHEN** a racer's `car_color` is `not-a-color`
- **THEN** the color dot has `background` style equal to `#cccccc`
- **AND** no invalid CSS value is emitted into a `style` attribute

### Requirement: Color normalization is shared via a single helper
A `normalizeHex(color: string): string` function SHALL exist in `ts/color.ts` and SHALL be the single source of truth for mapping named colors to hex, prepending the `#` prefix, validating the hex form, and applying the fallback. The leaderboard (`ts/index.ts`) SHALL use this helper at all four color-display sites.

#### Scenario: Helper maps named colors
- **WHEN** `normalizeHex('red')` is called
- **THEN** it returns `#ff4444`

#### Scenario: Helper normalizes missing hash
- **WHEN** `normalizeHex('800080')` is called
- **THEN** it returns `#800080`

#### Scenario: Helper validates hex
- **WHEN** `normalizeHex('#800080')` is called
- **THEN** it returns `#800080`

#### Scenario: Helper falls back for invalid input
- **WHEN** `normalizeHex('not-a-color')` is called
- **THEN** it returns `#cccccc`

#### Scenario: Helper falls back for empty input
- **WHEN** `normalizeHex('')` is called
- **THEN** it returns `#cccccc`

### Requirement: Dead named-color CSS subclasses are removed
The `.color-indicator.<name>` CSS rules in `static/style.css` (the per-color subclasses such as `.color-indicator.red`, `.color-indicator.blue`, etc.) SHALL be removed, since color rendering is now inline-style-driven. The base `.color-indicator` rule (size, border) SHALL remain.

#### Scenario: No per-color subclass rules remain
- **WHEN** `static/style.css` is inspected
- **THEN** no rule matching `.color-indicator.red`, `.color-indicator.blue`, `.color-indicator.green`, `.color-indicator.yellow`, `.color-indicator.grey`, `.color-indicator.silver`, `.color-indicator.black`, `.color-indicator.purple`, or `.color-indicator.orange` is present
- **AND** the base `.color-indicator` rule is present
