# Backend Development Guidelines

Source-backed conventions for the Go backend in `backend/`.

## Guidelines Index

| Guide | Purpose | Status |
|-------|---------|--------|
| [Directory Structure](./directory-structure.md) | Where backend code belongs and how layers interact | Complete |
| [Database Guidelines](./database-guidelines.md) | Ent schemas, repositories, SQL, and migrations | Complete |
| [Error Handling](./error-handling.md) | Application errors and JSON response envelopes | Complete |
| [Logging Guidelines](./logging-guidelines.md) | Zap/slog logging, access logs, redaction, and sink usage | Complete |
| [Quality Guidelines](./quality-guidelines.md) | Tests, generation, review checks, and forbidden patterns | Complete |

## Pre-Development Checklist

Read these before backend changes:

- Always read [Directory Structure](./directory-structure.md).
- Read [Database Guidelines](./database-guidelines.md) for Ent schemas, repositories, SQL, migrations, billing, quotas, auth, groups, accounts, or settings.
- Read [Error Handling](./error-handling.md) for HTTP handlers, service errors, repository error translation, gateway streams, or API contract changes.
- Read [Logging Guidelines](./logging-guidelines.md) for request logging, operational logs, background jobs, gateways, or security-sensitive data.
- Read [Quality Guidelines](./quality-guidelines.md) before adding tests or running verification.

## Main Verification Commands

- `cd backend && go test ./...`
- `cd backend && go test -tags=unit ./...`
- `cd backend && go test -tags=integration ./...` when repository or migration behavior changes.
- `cd backend && make generate` after editing `backend/ent/schema` or Wire providers.
