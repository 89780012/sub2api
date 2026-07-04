# Backend Error Handling

Backend HTTP responses use a standard envelope and application errors for controllable failures.

## Response Envelope

Use `backend/internal/pkg/response` from handlers:

- Success: `response.Success(c, data)` returns `{ "code": 0, "message": "success", "data": ... }`.
- Creation: `response.Created(c, data)` returns HTTP 201 with the same success envelope.
- Async acceptance: `response.Accepted(c, data)` returns HTTP 202.
- Pagination: `response.Paginated(...)` or `response.PaginatedWithResult(...)` returns `data.items`, `total`, `page`, `page_size`, and `pages`.
- Errors: `response.ErrorFrom(c, err)` converts application errors through `internal/pkg/errors.ToHTTP`.

Handlers should return immediately after writing an error response.

## Application Errors

- Use `backend/internal/pkg/errors` for service-level failures that should control HTTP status, reason, message, and metadata.
- Constructors include `BadRequest`, `Unauthorized`, `Forbidden`, `NotFound`, `Conflict`, `TooManyRequests`, `ServiceUnavailable`, `GatewayTimeout`, and `ClientClosed`.
- Preserve causes with `.WithCause(err)` and attach structured fields with `.WithMetadata(map[string]string{...})`.
- `ApplicationError.Is` supports `errors.Is`-style matching by code and reason; prefer this over string matching.
- Unknown errors become HTTP 500 with message `internal error`.

## Handler Pattern

Follow `backend/internal/handler/user_handler.go` and `payment_handler.go`:

- Get authenticated identity with middleware helpers such as `middleware2.GetAuthSubjectFromContext` or a local `requireAuth`.
- Bind JSON using Gin tags (`binding:"required"`, `binding:"required,email"`, etc.).
- Return `response.BadRequest(c, "Invalid request: "+err.Error())` for binding errors when the handler is not using a richer application error.
- Call service methods with `c.Request.Context()`.
- Return service failures with `response.ErrorFrom(c, err)`.

## Repository Error Translation

- Repositories should hide persistence details from services. Use `translatePersistenceError` in `backend/internal/repository/error_translate.go`.
- Map Ent not-found and `sql.ErrNoRows` to service not-found errors.
- Map PostgreSQL unique constraint errors (`pq.Error` code `23505`) to conflict errors.
- For not-found cases on atomic updates, return the existing service error such as `service.ErrAPIKeyNotFound`.

## Streaming And Gateway Errors

- Once an SSE response has started, HTTP status cannot be changed. For OpenAI Responses streams, use `writeResponsesFailedSSE` from `backend/internal/handler/stream_error_event.go` so strict clients receive a `response.failed` terminal event.
- Keep protocol-specific error mapping in focused helpers such as `mapResponsesErrorCode`; do not scatter protocol status strings across handlers.
- Preserve request IDs where possible. `synthesizeResponseID` reuses the request ID from context when creating a synthetic failed response.

## Recovery

- Gin recovery is centralized in `backend/internal/server/middleware/recovery.go`.
- Broken pipes and connection resets are aborted without writing a response.
- Panics before headers are written become the standard 500 response envelope.

## Common Mistakes

- Do not return raw `error.Error()` from internal failures to clients unless the error is already an intended application error.
- Do not write JSON directly in new handlers unless the endpoint is intentionally outside the standard envelope, such as `/health`.
- Do not continue handler execution after writing `response.ErrorFrom`, `response.BadRequest`, or `response.Unauthorized`.
- Do not rely on frontend-only auth checks; backend handlers and middleware must enforce authorization.
