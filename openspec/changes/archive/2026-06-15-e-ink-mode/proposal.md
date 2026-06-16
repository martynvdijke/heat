## Why

HEAT is used during race sessions where a tablet or e-ink display on the table provides quick reference to standings, leaderboards, and race control. E-ink displays are readable in direct sunlight (great for outdoor race days) and use no power when static — perfect for a race-day companion display.

## What Changes

- Add `?eink=1` URL parameter and/or cookie-activated e-ink mode
- Apply high-contrast black-on-white palette to all pages (standings, leaderboard, race control, driver view)
- Remove all animations, transitions, gradients, shadows, backdrop filters
- Enforce 48px minimum touch targets — especially for race control actions
- Replace color-coded status indicators (flag colors, position changes) with icon + text labels
- Simplify the standings table — remove alternating row colors, use solid borders
- Increase font sizes for readability at a distance on a table
- Remove hover-dependent tooltips and interactions
- Add a grayscale-friendly flag indicator system (icons + text for flag status)

## Capabilities

### New
- `eink-mode-stylesheet`: Alternative high-contrast, flat CSS for all pages
- `eink-mode-toggle`: URL param and settings toggle mechanism
- `eink-race-display`: Simplified race control and standings for tabletop e-ink

### Modified
- *(none)*

## Impact

- **Frontend**: New eink.css stylesheet; JS toggle handler; updated HTML templates for e-ink class
- **Backend**: (minimal) Persist e-ink preference in admin settings
- **Database**: Optional
- **Dependencies**: None
