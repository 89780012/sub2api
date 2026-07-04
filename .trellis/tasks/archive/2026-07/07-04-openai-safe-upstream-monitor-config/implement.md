# OpenAI-safe upstream monitor config implementation plan

## Checklist

1. Backend monitor config model
   - Add service structs/helpers for `credentials.upstream_monitor`.
   - Parse token-mode NewAPI and Sub2API monitor credentials.
   - Add sanitized no-config and disabled reasons.

2. Backend collector
   - Stop using inference `base_url` / `api_key` in `resolveAccountMonitorCredentials`.
   - Add request helper that supports provider-specific auth headers.
   - Update NewAPI collection to send `Cookie` and `New-Api-User`.
   - Keep Sub2API bearer-token behavior.

3. Backend repository/scheduler
   - Update due-account selection to include accounts with enabled monitor config where practical.
   - Ensure scheduler skips missing/disabled config without recording endpoint-not-supported failures.

4. Backend response redaction
   - Extend account credential redaction/status helpers for nested monitor secrets.
   - Ensure create/update preserves existing monitor secrets when edit payload leaves them blank.

5. Frontend types/API
   - Add `AccountUpstreamMonitorConfig` and credential status flags.
   - Keep API wrappers unchanged unless account payload typing needs expansion.

6. Frontend edit UI
   - Add a compact upstream monitor config section to `EditAccountModal`.
   - Distinguish it from OpenAI inference endpoint config.
   - Submit nested `credentials.upstream_monitor`.
   - Preserve blank secret fields on edit when status flags indicate saved secrets.

7. Account list display
   - Show not-configured/disabled labels for the new sentinel states.
   - Avoid showing endpoint-not-supported for pure OpenAI inference endpoints with no monitor config.

8. Tests
   - Extend service tests for no-config, disabled, NewAPI token headers, Sub2API token headers, and no inference credential usage.
   - Extend redaction/update tests for nested monitor secrets.
   - Extend frontend edit tests for preserving existing monitor secrets and keeping inference base URL unchanged.

## Validation Commands

```powershell
cd backend
go generate ./ent
go generate ./cmd/server
go test ./internal/service ./internal/handler/admin ./internal/repository ./cmd/server
go test ./...
```

```powershell
pnpm --dir frontend run typecheck
pnpm --dir frontend run lint:check
pnpm --dir frontend exec vitest run src/components/account/__tests__/EditAccountModal.spec.ts src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts src/views/admin/__tests__/AccountsView.usageWindowsHint.spec.ts
```

## Risk Points

- Account update merge behavior must not accidentally drop existing redacted inference or monitor secrets.
- NewAPI requires `New-Api-User`; bearer auth is not enough.
- UI copy must prevent admins from pasting OpenAI inference API keys into the monitor token field by mistake.
- Existing full frontend test suite has unrelated failures; targeted tests and type/lint remain the useful gate for this task unless those are fixed separately.

## Verification Results

- `cd backend && go test ./internal/service ./internal/handler/dto ./internal/repository` passed.
- `cd backend && go test ./...` passed.
- `pnpm --dir frontend exec vitest run src/components/account/__tests__/EditAccountModal.spec.ts` passed.
- `pnpm --dir frontend run typecheck` passed.
- `pnpm --dir frontend run lint:check` passed.
