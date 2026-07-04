# Backend Directory Structure

The backend is a Go module in `backend/` using Gin, Ent, Wire, PostgreSQL, Redis, and embedded Vue assets.

## Layer Layout

- `backend/cmd/server/` is the application entry point and Wire generation target. `main.go` handles flags, setup mode, logger bootstrap, config bootstrap, graceful shutdown, and calls the generated application initializer.
- `backend/internal/server/` owns HTTP server construction. `http.go` provides the Gin engine and `http.Server`; `router.go` wires middleware, embedded frontend serving, and route registration.
- `backend/internal/server/routes/` maps paths to handlers. Keep route registration here rather than inside service packages.
- `backend/internal/server/middleware/` contains Gin middleware for auth, request logging, recovery, CORS, body limits, security headers, and request context.
- `backend/internal/handler/` owns HTTP request/response DTO handling. Handlers bind Gin requests, fetch auth subjects from middleware, call services, and return responses through `internal/pkg/response`.
- `backend/internal/service/` owns business logic and service-layer domain structs. Examples include account scheduling, auth flows, billing, payment, subscriptions, settings, and gateway decisions.
- `backend/internal/repository/` owns persistence adapters for Ent, raw SQL, Redis/cache helpers, migrations, and entity-to-service conversion.
- `backend/internal/domain/` contains shared constants and domain-level config structs that are used by multiple layers.
- `backend/internal/pkg/` contains reusable backend packages such as `response`, `errors`, `logger`, `pagination`, protocol adapters, OAuth helpers, and provider-specific clients.
- `backend/internal/util/` contains smaller utilities such as URL validation, response-header filtering, and log redaction.
- `backend/internal/payment/` owns payment abstractions and provider implementations under `payment/provider/`.
- `backend/internal/setup/` owns first-run setup CLI and HTTP setup wizard support.
- `backend/internal/web/` serves the built frontend from `frontend/dist` when compiled with embed support.
- `backend/internal/testutil/` contains test fixtures and HTTP helpers for backend tests.
- `backend/ent/schema/` contains Ent schema definitions; generated Ent code lives under `backend/ent/`.
- `backend/migrations/` contains embedded forward SQL migrations and the migration README.

## Boundary Rules

- Keep HTTP details in `handler` and `server/routes`; do not pass `*gin.Context` into services or repositories. Pass `context.Context`, IDs, typed request structs, or service-domain structs instead.
- Keep DB details in `repository`. Services should depend on repository interfaces and service-domain structs rather than Ent entities where possible.
- Convert Ent entities to service structs in repositories. Existing examples are `userEntityToService`, `groupEntityToService`, and `apiKeyEntityToService` in `backend/internal/repository/api_key_repo.go`.
- Put protocol translation and upstream API compatibility code under `internal/pkg` or focused service files, not in route registration.
- Add shared constants to `internal/domain` when both backend layers or frontend contract docs need the same concept; avoid duplicating string literals across services.

## Naming Patterns

- Go files use snake_case when the concept is multiword: `account_credentials_redact.go`, `request_body_limit.go`, `error_passthrough_repo.go`.
- Tests sit beside the code and use `_test.go`; integration-specific tests use build tags such as `//go:build integration`.
- Repository files are usually named `<entity>_repo.go`; service files are named by feature (`account.go`, `auth_service.go`, `billing_service.go`).
- Route registration files live under `internal/server/routes` and use feature names such as `auth.go`, `admin.go`, `gateway.go`, and `payment.go`.

## Examples To Follow

- `backend/internal/server/http.go` for server construction, trusted-proxy setup, H2C configuration, and request-size wrapping.
- `backend/internal/server/router.go` for middleware ordering and centralized route registration.
- `backend/internal/handler/user_handler.go` for handler shape: authenticate, bind request, call service, return `response.Success` or `response.ErrorFrom`.
- `backend/internal/repository/api_key_repo.go` for repository conversions, pagination, sorting, and focused raw SQL where Ent is not enough.
- `backend/internal/service/account.go` for service-domain methods that normalize and interpret account configuration without HTTP or database dependencies.
