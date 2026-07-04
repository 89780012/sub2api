# Backend Database Guidelines

Sub2API uses PostgreSQL, Ent schemas, a custom embedded migration runner, and raw SQL in repositories for operations Ent cannot express cleanly.

## Ent Schemas

- Define persistent entities in `backend/ent/schema/`. Use `entsql.Annotation{Table: "..."}`
  to keep explicit table names, as seen in `user.go` and `group.go`.
- Use shared mixins for common fields. `User` and `Group` include `mixins.TimeMixin{}` and `mixins.SoftDeleteMixin{}`.
- Decimal-like values are represented with `field.Float(...).SchemaType(map[string]string{dialect.Postgres: "decimal(...)"})`; follow existing precision choices in `user.go` and `group.go`.
- JSONB config fields use `field.JSON(...).SchemaType(map[string]string{dialect.Postgres: "jsonb"})`, as in `Group.model_routing`, `messages_dispatch_model_config`, and `models_list_config`.
- After schema changes, regenerate with `cd backend && make generate` or the explicit commands from `README.md`: `go generate ./ent` and `go generate ./cmd/server`.

## Repository Patterns

- Repositories should accept `context.Context` and return service-layer structs or repository interface results. Do not return Gin objects from repositories.
- Use `clientFromContext(ctx, r.client)` when a repository method should participate in an Ent transaction. This helper is defined in `backend/internal/repository/error_translate.go`.
- For active records, preserve soft-delete filters. Existing repositories use `DeletedAtIsNil()` and partial unique indexes for soft-deleted records.
- Use Ent for ordinary CRUD and relationship loading. Use raw SQL for atomic updates, complex counters, batch operations, or PostgreSQL-specific behavior, as in `IncrementQuotaUsedAndGetState`, `IncrementRateLimitUsage`, and dashboard aggregation repositories.
- Keep entity conversion helpers close to repositories. `apiKeyEntityToService`, `userEntityToService`, and `groupEntityToService` are the local model for mapping generated Ent objects to service structs.
- Translate persistence errors at the repository boundary with `translatePersistenceError` so service and handler layers do not depend on `sql.ErrNoRows`, `pq.Error`, or Ent internals.

## Migrations

- Add SQL migrations under `backend/migrations/` using `NNN_description.sql`.
- Migrations are forward-only and are executed as complete SQL files by `backend/internal/repository/migrations_runner.go`.
- Do not edit, rename, delete, or reorder migrations after they may have been applied. The runner stores SHA256 checksums in `schema_migrations` and rejects unexpected changes.
- Use idempotent SQL where possible, such as `ADD COLUMN IF NOT EXISTS` and `CREATE INDEX IF NOT EXISTS`.
- Any migration using `CREATE INDEX CONCURRENTLY` or `DROP INDEX CONCURRENTLY` must end in `_notx.sql`. The runner validates that these files contain only concurrent index statements, include `IF NOT EXISTS` or `IF EXISTS`, and do not contain transaction control.
- Migrations are embedded through `backend/migrations/migrations.go` and run automatically on startup through `ApplyMigrations`.

## Transactions And Concurrency

- Prefer Ent transactions for multi-row business operations; pass the transaction through context so repository helpers pick it up with `clientFromContext`.
- Use PostgreSQL advisory locks for global migration serialization only. The migration runner already does this through `pg_try_advisory_lock`.
- Use single-statement SQL for atomic counters, quotas, and rate windows when race conditions matter. `api_key_repo.go` is the reference.
- For connection pool behavior, use `applyDBPoolSettings` and the clamping rules in `backend/internal/repository/db_pool.go`; do not configure pool lifetime ad hoc.

## Common Mistakes

- Do not place executable Down SQL in migration files; the runner does not parse Up/Down sections.
- Do not mix ordinary DDL/DML into `_notx.sql` migrations.
- Do not bypass soft-delete filters in user-facing reads unless the method name and test make that explicit.
- Do not leak database errors through handlers. Translate to service/application errors first.
- Do not update only Ent schemas or only SQL migrations when adding persistent fields; both sides and generated code must agree.
