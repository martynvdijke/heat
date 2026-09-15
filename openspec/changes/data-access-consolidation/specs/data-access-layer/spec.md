## ADDED Requirements

### Requirement: Single data-access convention with Ent as system of record

The system SHALL use Ent as the system of record for all application reads and writes, and SHALL confine raw `database/sql` (`*sql.DB`) to the `db/` migration/seed/pragma layer. Application code outside `db/` MUST NOT assume raw SQL is the source of truth.

#### Scenario: Application read goes through Ent

- **WHEN** application code outside `db/` performs a read for domain data (e.g. racers, rounds)
- **THEN** the read is served via an Ent-backed repository, not via `Server.DB` raw SQL

#### Scenario: Raw SQL confined to db layer

- **WHEN** inspecting raw `database/sql` usage
- **THEN** all `*sql.DB` query/exec calls reside in `db/` (migrations in `db/init.go`, seeding in `db/seed*.go`, pragmas/backup/healthcheck) and no new raw SQL exists in `handlers/` or `racing/` after migration

### Requirement: No raw SQL in handlers — per-domain data-access boundaries

The system SHALL provide per-domain repository/data-access functions so that `handlers/` SHALL NOT issue raw SQL. Handlers MUST depend on domain repository interfaces that return domain models (from `models/models.go` or equivalent) rather than on `Server.DB` or `*sql.DB`.

#### Scenario: Handler uses repository interface

- **WHEN** a handler needs domain data (e.g. listing racers or rounds)
- **THEN** it calls a domain repository interface method and receives domain models, without importing `database/sql` or issuing `DB.Query` / `DB.Exec`

#### Scenario: New handler cannot add raw SQL

- **WHEN** a new handler is added that needs persistence
- **THEN** it gains data access by depending on an existing or new domain repository interface, not by adding a raw SQL query in `handlers/`

### Requirement: Consistent transaction handling across database/sql and ent.Tx

The system SHALL define and use a single transaction pattern that bridges `database/sql` (`sql.Tx`) and Ent (`ent.Tx`) so callers handle commit/rollback uniformly. Repositories that participate in a transaction MUST expose a `WithTx` or equivalent `Tx`-scoped API with unified commit/rollback semantics.

#### Scenario: Multi-repo write commits atomically

- **WHEN** a caller performs writes via multiple repositories inside a single `WithTx` scope and all operations succeed
- **THEN** the transaction commits once and all writes become visible

#### Scenario: Failed write rolls back

- **WHEN** any operation inside a `WithTx` scope returns an error
- **THEN** the transaction rolls back and no partial writes are persisted, regardless of whether the underlying implementation uses `sql.Tx` or `ent.Tx`

### Requirement: Ent as system of record for application reads/writes

The system SHALL treat Ent schemas (`ent/schema/`) as the canonical domain model for application reads/writes, and SHALL NOT introduce new application tables or columns via raw DDL outside the Ent-driven or `db/init.go`-coordinated migration path. Application writes outside `db/` MUST go through Ent.

#### Scenario: New application column uses Ent schema

- **WHEN** a new domain field is added for application use
- **THEN** it is added via an Ent schema change (and generated code under `ent/`) rather than via ad-hoc raw `CREATE TABLE` / `ALTER TABLE` in handlers

#### Scenario: Seeding still uses raw SQL via db layer

- **WHEN** `db/seed*.go` populates initial data
- **THEN** it MAY use raw `*sql.DB` within `db/` as the designated exception, while application handlers still write through Ent

### Requirement: Data-access boundaries are testable without raw SQL

Per-domain data-access boundaries SHALL be testable in isolation, without requiring handlers to open a raw `*sql.DB` connection. Repository interfaces MUST be mockable or substitutable so that handler and `racing/` pure-stats tests can run with in-memory or Ent-test implementations.

#### Scenario: Handler test with fake repository

- **WHEN** a handler is tested with a fake/mock implementation of its repository interface
- **THEN** the test succeeds without opening a real `*sql.DB` and without issuing raw SQL

#### Scenario: Racing stats logic tested via interface

- **WHEN** `racing/` pure-stats logic (e.g. elo, consistency, head-to-head) needs data for a test
- **THEN** it receives domain models via a repository interface or in-memory fixture, not via direct `Server.DB` queries

### Requirement: Tracing and metrics consistent at repository boundary

The system SHALL instrument data access at the repository boundary so that OpenTelemetry tracing and metrics are consistent and not duplicated per handler. Each repository operation MUST emit a span/metric with a domain-qualified operation name, reusing the existing `telemetry.go` / `middleware/tracing.go` infrastructure.

#### Scenario: Repository call emits a span

- **WHEN** a repository method is called (whether backed by Ent or the interim raw-SQL implementation)
- **THEN** a trace span is emitted at the repository boundary with a domain-qualified name (e.g. `racers.list` or `rounds.get`), without requiring the handler to create a duplicate span

#### Scenario: No duplicate instrumentation per handler

- **WHEN** a handler calls multiple repositories
- **THEN** each repository call produces its own boundary span/metric, and the handler does not emit additional per-query data-access spans
