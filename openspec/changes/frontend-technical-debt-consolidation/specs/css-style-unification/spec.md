## ADDED Requirements

### Requirement: Single canonical stylesheet

All visual styles for the application SHALL be defined in `static/style.css`. No page SHALL define visual styles in an inline `<style>` block, except for page-specific dynamic values (e.g., a color set by JavaScript).

#### Scenario: Inline styles moved to shared stylesheet

- **WHEN** inspecting any HTML page in `static/`
- **THEN** the page SHALL NOT contain a `<style>` block with CSS rules (except `prefers-color-scheme` dark-mode inline script blocks)
- **THEN** all CSS rules previously in inline blocks SHALL exist in `static/style.css` under a section comment identifying the source page

### Requirement: Unified CSS custom properties

All pages SHALL use the canonical CSS custom properties defined in `:root` in `style.css` (`--primary`, `--racing-green`, `--background`, `--text-main`, `--text-muted`, `--surface`, `--border-color`, `--border-width`, `--card-shadow`, `--vintage-opacity`). No page SHALL define its own color variables.

#### Scenario: TV page uses canonical variables

- **WHEN** inspecting the TV page styles
- **THEN** the TV page SHALL use `--primary` instead of raw `#e74c3c` or custom `--heat-red`
- **THEN** the TV page SHALL use `--background` and `--text-main` instead of hardcoded `#000` and `#fff`
- **THEN** the TV page SHALL respect the `data-theme` attribute (dark/light mode)

### Requirement: No duplicate CSS rules

If two or more pages define visually identical rules (e.g., card styling, button shapes), those rules SHALL exist once in `static/style.css` under a shared class name.

#### Scenario: Shared card styles consolidated

- **WHEN** comparing `.controller-card`, `.spec-card`, and similar card classes across pages
- **THEN** shared card patterns SHALL be consolidated into a single set of CSS classes in `style.css`
