# 实现计划

## Checklist

1. Backend types and repository
   - Add account pool rate analysis response types under `internal/pkg/usagestats` or `internal/service`, following existing usage stats patterns.
   - Extend the `UsageLogRepository` interface with a group-level rate analysis method.
   - Implement the SQL aggregation in `backend/internal/repository/usage_log_repo.go`.
   - Add unit or integration tests for the formula, uncovered rows, zero theoretical cost, and sorting.

2. Backend service and handler
   - Add `AccountUsageService.GetPoolRateAnalysis`.
   - Add `AccountHandler.GetPoolRateAnalysis` with date parsing and sort validation.
   - Register `GET /api/v1/admin/accounts/pool-rate-analysis` before `/:id` routes to avoid route capture.
   - Add handler/contract tests for request parsing and response shape.

3. Frontend API and types
   - Add TypeScript response types and API function in `frontend/src/api/admin/accounts.ts`.
   - Export through existing admin API barrel as part of `accountsAPI`.

4. Frontend page and navigation
   - Add `frontend/src/views/admin/AccountPoolRateAnalysisView.vue`.
   - Add router entry `/admin/accounts/rate-analysis`.
   - Add admin sidebar child or nearby navigation entry under Account Management.
   - Use existing `DateRangePicker`, table styling, loading and error patterns.

5. I18n and tests
   - Add Chinese and English labels under existing `admin.accounts` or a new nested key.
   - Add focused frontend tests for default 7-day range, API loading, no-valid-sample rows, and sort action if local pattern supports it.

## Validation Commands

- `cd backend && go test ./internal/repository ./internal/service ./internal/handler/admin ./internal/server`
- `cd frontend && pnpm run typecheck`
- `cd frontend && pnpm run test:run -- AccountPoolRateAnalysis`
- `cd frontend && pnpm run lint:check`

## Risk Points

- Route order: `/admin/accounts/pool-rate-analysis` must be registered before `/admin/accounts/:id`.
- Date range semantics must match existing admin usage endpoints: inclusive start date and exclusive end date after parsing.
- SQL must not mix `total_cost` fallback into the valid multiplier formula.
- Large `usage_logs` scans should remain bounded by date range and indexed `group_id, created_at` path.

## Rollback

- Frontend route and sidebar entry can be removed independently.
- Backend endpoint is additive; removing handler route and repository method returns behavior to current state.
- No migration rollback needed because MVP uses existing columns.
