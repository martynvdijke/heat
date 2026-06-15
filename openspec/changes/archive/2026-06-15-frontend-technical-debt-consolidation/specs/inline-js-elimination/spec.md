## ADDED Requirements

### Requirement: No inline JavaScript in HTML pages

All JavaScript code SHALL be defined in TypeScript modules in `ts/` and compiled to JavaScript. HTML pages SHALL NOT contain inline `<script>` blocks with application logic (except for the dark-mode `prefers-color-scheme` blocking script that must run before page render).

#### Scenario: Login page has no inline script

- **WHEN** inspecting `static/login.html`
- **THEN** the page SHALL NOT contain a `<script>` block with application logic
- **THEN** the page SHALL load a compiled JS file from `/static/js/login.js` that provides the same functionality (form submit handler, setup check redirect)

#### Scenario: Driver page has no inline script

- **WHEN** inspecting `static/driver.html`
- **THEN** the page SHALL NOT contain a `<script>` block with application logic
- **THEN** the page SHALL load a compiled JS file from `/static/js/driver.js` that provides the same functionality (token validation, stats rendering)

### Requirement: Event handlers use addEventListener

HTML elements SHALL NOT use `onclick`, `onchange`, or other inline event handler attributes. All event binding SHALL be done via `addEventListener()` in TypeScript.

#### Scenario: Controller buttons use addEventListener

- **WHEN** inspecting `static/controller.html`
- **THEN** no element SHALL have an `onclick` attribute
- **THEN** all event handlers SHALL be registered via `addEventListener()` in `ts/controller.ts`

### Requirement: Consistent error handling in all TS modules

Every TypeScript module that makes HTTP requests SHALL implement try/catch error handling and display errors via the toast notification system.

#### Scenario: Failed API call shows error toast

- **WHEN** an API fetch call fails (network error or non-OK response)
- **THEN** a toast notification SHALL be displayed with the error message
- **THEN** the error SHALL be handled gracefully (no uncaught promise rejections)
