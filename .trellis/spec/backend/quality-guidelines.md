# Backend Quality Guidelines

Backend quality is enforced through Go tests, generated-code discipline, migration safety, and contract-preserving handlers.

## Verification Commands

- Full backend check: `cd backend && go test ./...`
- Unit-tagged tests: `cd backend && go test -tags=unit ./...`
- Integration-tagged tests: `cd backend && go test -tags=integration ./...`
- E2E local tests: `cd backend && go test -tags=e2e -v -timeout=300s ./internal/integration/...`
- Generation after Ent/Wire changes: `cd backend && make generate`
- The Makefile target `make test` also runs `golangci-lint run ./...` after `go test ./...`; run it when the lint tool is installed.

## Test Patterns

- Put tests next to code. Existing examples include `internal/pkg/response/response_test.go`, `internal/server/middleware/recovery_test.go`, and many `internal/service/*_test.go` files.
- Use build tags to separate expensive tests. Repository integration tests use `//go:build integration`; some package tests use `//go:build unit`.
- Use `testify/require` for assertions in backend tests.
- Use `gin.SetMode(gin.TestMode)`, `httptest.NewRecorder`, and `gin.CreateTestContext` for handler and response helper tests.
- Use stubs for service-layer tests when the behavior does not require a real database. Large examples are in service tests with compile-time interface checks.
- Use testcontainers-backed integration harnesses for repository behavior that depends on PostgreSQL or Redis.

## Required Checks Before Backend Changes Land

- New or changed handlers preserve the standard response envelope unless the endpoint is intentionally raw.
- Auth, admin, setup, and backend-mode behavior are enforced server-side, not only in the frontend router.
- Service logic is covered by focused unit tests when business rules change.
- Repository and migration changes have integration coverage or a clear manual verification path.
- Ent schema changes include regenerated code and compatible SQL migration.
- Changes to gateway streaming, billing, quotas, auth, payments, migrations, or scheduling get tests for edge cases and failure paths.

## Forbidden Patterns

- Do not edit generated Ent or Wire output by hand; update schemas/providers and regenerate.
- Do not modify applied migrations. Create a new migration for every schema/data change.
- Do not pass `*gin.Context` into services or repositories.
- Do not add unbounded goroutines or tickers without cancellation and tests.
- Do not bypass context cancellation on DB, Redis, or upstream HTTP operations.
- Do not introduce raw SQL string concatenation with user-controlled values; use parameters.
- Do not weaken request-size, URL allowlist, auth, CORS, CSP, or response-header filtering without a source-backed reason and tests.

## Review Checklist

- Layering: route -> handler -> service -> repository direction is preserved.
- Contracts: frontend API wrappers and TypeScript types still match backend JSON fields.
- Errors: expected failures use `internal/pkg/errors` and `response.ErrorFrom`.
- Logs: sensitive fields are redacted and logs are structured.
- Database: migrations are idempotent where practical and obey `_notx.sql` rules.
- Tests: added or changed tests exercise the behavior, not only implementation details.
