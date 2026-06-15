## ADDED Requirements

### Requirement: TypeScript bundled per page

Each HTML page SHALL load a single bundled JavaScript file that includes all its dependencies (page logic plus shared modules), rather than loading multiple separate script files.

#### Scenario: Shared modules are bundled

- **WHEN** a page uses the toast notification system from `ts/toast.ts`
- **THEN** the toast module SHALL be bundled into the page's single output JS file
- **THEN** the page SHALL NOT require a separate `<script src="/static/js/toast.js">` tag

### Requirement: Production builds are minified

Frontend builds in production mode SHALL produce minified JavaScript output.

#### Scenario: Minified output

- **WHEN** running the production build command
- **THEN** the output JS files SHALL be minified (whitespace removed, variable names shortened where safe)
- **THEN** sourcemaps SHALL be generated for debugging

### Requirement: Development builds include sourcemaps

Development builds SHALL generate sourcemaps so that browser DevTools show original TypeScript sources, not compiled output.

#### Scenario: Sourcemaps available in dev

- **WHEN** running the development build command
- **THEN** the output JS files SHALL have associated `.map` files
- **THEN** browser DevTools SHALL display original `.ts` file names and line numbers

### Requirement: Build commands in Taskfile

The project SHALL have Taskfile commands for frontend build, replacing or supplementing the current `task build:ts` (`tsc`).

#### Scenario: Build commands work

- **WHEN** running `task build:frontend`
- **THEN** esbuild SHALL bundle and output all page JS files to `static/js/`
- **WHEN** running `task build:frontend:dev`
- **THEN** esbuild SHALL output with sourcemaps enabled, without minification
- **WHEN** running `task typecheck`
- **THEN** `tsc --noEmit` SHALL run for type checking only (no output files)
