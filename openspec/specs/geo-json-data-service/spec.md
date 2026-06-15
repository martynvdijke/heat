# geo-json-data-service Specification

## Purpose
Serve track circuit GeoJSON data via a REST API endpoint with server-side caching, removing hardcoded GeoJSON from client-side TypeScript.

## Requirements

### Requirement: Track GeoJSON served from API

The system SHALL serve track circuit GeoJSON data via a REST API endpoint rather than embedding it in client-side TypeScript code.

#### Scenario: GeoJSON endpoint returns track data

- **WHEN** `GET /api/tracks/geojson` is called
- **THEN** the response SHALL be a JSON object mapping track IDs to their GeoJSON Feature objects
- **THEN** the response SHALL include all tracks that have GeoJSON data in the database

#### Scenario: Frontend fetches GeoJSON from API

- **WHEN** the home page loads the circuit map
- **THEN** the frontend SHALL fetch GeoJSON data from `/api/tracks/geojson` instead of reading from the hardcoded `trackGeoJSON` object in `ts/index.ts`

### Requirement: Embedded GeoJSON removed from TypeScript

The hardcoded `trackGeoJSON` object in `ts/index.ts` SHALL be removed after the API endpoint is operational.

#### Scenario: No hardcoded GeoJSON in compiled JS

- **WHEN** searching `ts/index.ts` for track coordinate arrays
- **THEN** the file SHALL NOT contain hardcoded coordinate arrays for circuits
- **THEN** all coordinate data SHALL be served from the API or a static data file

### Requirement: GeoJSON data caching

The GeoJSON endpoint SHALL implement server-side caching to avoid excessive database queries, since track data changes infrequently.

#### Scenario: GeoJSON response is cached

- **WHEN** `GET /api/tracks/geojson` is called twice within 5 minutes
- **THEN** the second call SHALL NOT execute a new database query
- **THEN** both responses SHALL return identical data
