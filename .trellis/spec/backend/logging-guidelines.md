# Backend Logging Guidelines

Backend logging is centralized through `internal/pkg/logger`, which bridges zap, slog, and legacy standard-library logs.

## Logger Setup

- `backend/cmd/server/main.go` calls `logger.InitBootstrap()` early, then reinitializes from config in normal server mode with `logger.Init(logger.OptionsFromConfig(cfg.Log))`.
- Use `logger.L()` for structured zap logs, `logger.S()` for sugared logs, and `logger.FromContext(ctx)` when request-scoped fields may exist.
- `logger.Sync()` is deferred in `main`; do not add scattered sync calls in normal request paths.
- Standard `log.Print` and `slog` output are bridged into the configured zap logger after initialization.

## Request Logs

- HTTP access logs are emitted in `backend/internal/server/middleware/logger.go`.
- Access logs include component `http.access`, status code, latency, client IP, protocol, method, path, and optional account/platform/model context.
- High-frequency probe paths `/health` and `/setup/status` are skipped.
- Middleware records Gin errors as a warning after the request completes.

## Sensitive Data

- Use `backend/internal/util/logredact` before logging error text that may contain credentials or tokens. `response.ErrorFrom` already redacts 500-level error text before logging.
- Account credentials should be redacted with service helpers such as `backend/internal/service/account_credentials_redact.go`.
- Never log raw API keys, access tokens, refresh tokens, payment secrets, TOTP secrets, full credentials JSON, or unredacted upstream authorization headers.

## Log Levels

- Debug: noisy diagnostic details useful only during local investigation.
- Info: lifecycle events, access logs, successful background transitions, and effective configuration summaries.
- Warn: recoverable degraded behavior, fallback paths, invalid configuration that was clamped, skipped optional providers, and retryable operational issues.
- Error: failed operations that require operator attention or represent unexpected internal failure.
- Fatal: startup failures where the process cannot continue; `main.go` uses `log.Fatalf` during setup/config/application initialization failures.

## Operational Log Sink

- `logger.SetSink` and `logger.WriteSinkEvent` support writing operational log events to an additional sink independent of normal logger level gating.
- Use `WriteSinkEvent` only when observability ingestion must be decoupled from business log output level, as documented in `internal/pkg/logger/logger.go`.

## Examples To Follow

- `backend/internal/repository/db_pool.go` logs clamped pool durations with `slog.Warn` and final effective settings with `slog.Info`.
- `backend/internal/server/http.go` logs trusted proxy and H2C configuration warnings without failing startup unless configuration construction itself fails.
- `backend/internal/server/middleware/logger.go` shows how to attach structured request fields and use the context logger.

## Common Mistakes

- Do not use `fmt.Println` for server diagnostics.
- Do not create per-feature global loggers; use the central logger and fields.
- Do not log complete request/response bodies for gateway traffic unless explicitly redacted and bounded.
- Do not log client secrets or auth headers inside error messages.
