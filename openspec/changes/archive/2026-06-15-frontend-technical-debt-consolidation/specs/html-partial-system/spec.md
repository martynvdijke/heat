## ADDED Requirements

### Requirement: Shared HTML base template

The system SHALL provide a single base HTML template that contains the shared page structure: `<head>` with all meta tags, CSS/JS imports, header with logo and navigation, and footer with version display.

#### Scenario: Base template renders on all pages

- **WHEN** any page is served via `servePage()` or equivalent
- **THEN** the response SHALL include the shared `<head>` with Bootstrap 5.3, FontAwesome 7, Google Fonts, Leaflet, style.css, and favicon
- **THEN** the response SHALL include the header with HEAT logo, navigation links, theme toggle, and language selector
- **THEN** the response SHALL include the footer with federation copyright and version display

### Requirement: Page content defined as template blocks

Each page SHALL define only its unique content as Go `html/template` blocks, extending the shared base template.

#### Scenario: Home page uses base template

- **WHEN** the home page (`/`) is served
- **THEN** the HTML SHALL be produced by rendering a template that extends the base layout
- **THEN** the rendered output SHALL contain the same visual structure as the current static `index.html`
- **THEN** the rendered output SHALL NOT duplicate the `<head>`, header, or footer markup

### Requirement: Zero duplication across pages

All 14+ pages SHALL share a single source of truth for header, nav, footer, and document head.

#### Scenario: Navigation link change propagates automatically

- **WHEN** a navigation link is added, removed, or changed in the base template
- **THEN** the change SHALL automatically apply to all pages that extend the base template
- **THEN** no individual page file SHALL need to be edited for the change to take effect
