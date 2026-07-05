# Admin balance and rate snapshots implementation plan

## Steps

1. Load backend/frontend coding specs before editing.
   - Read backend directory, database, error handling, logging, quality, and tooling guidelines.
   - Read frontend directory, component, state, type safety, and quality guidelines.

2. Add backend data model and migrations.
   - Add Ent schemas for `balance_sites`, `balance_key_snapshots`, and `account_balance_bindings`.
   - Add idempotent SQL migration with indexes for site refresh lookup, snapshot last-four matching, account snapshot joins, and manual binding lookup.
   - Regenerate Ent code.

3. Extract or share balance fetcher logic.
   - Refactor `shell/balance` normalization/fetch contracts into reusable code or mirror the contract in a backend package with tests.
   - Preserve the existing CLI behavior.

4. Add service and repository layer.
   - Implement balance site CRUD with encrypted password storage and sanitized response models.
   - Implement refresh orchestration for one site and all enabled sites.
   - Implement latest snapshot upsert and account matching.
   - Implement manual account binding.
   - Add a started background refresh service with default 3-hour cadence and bounded HTTP timeouts.

5. Add admin APIs and route wiring.
   - Add `/api/v1/admin/balance-sites` CRUD and refresh routes.
   - Add snapshot/binding routes needed by the admin panel.
   - Extend account list/detail responses with external balance fields.
   - Extend group quality response items with external balance fields.

6. Add frontend API/types.
   - Add TypeScript types for balance site config, snapshots, bindings, and external balance display.
   - Add API wrapper methods for balance site CRUD, refresh, and binding updates.
   - Extend `Account` and `GroupAccountQualityItem` types with external balance fields.

7. Add frontend UI.
   - Add a "balance sync configuration" entry in the account management more tools menu.
   - Build the multi-site CRUD panel with password mask semantics and manual refresh status.
   - Add account table columns for external balance and upstream multiplier. Balance displays key balance first and subscription/package windows second.
   - Add unmatched/ambiguous/manual binding affordance.
   - Add balance/multiplier display to group quality panel rows with the same display priority.

8. Verify and iterate.
   - Backend unit/integration tests for repository, encryption masking, matching, refresh failure preservation, API contracts, and group/account response decoration.
   - Frontend typecheck and focused component/API tests for account rows, config panel, and group quality panel.

## Validation Commands

- `cd backend && make generate`
- `cd backend && go test ./...`
- `cd backend && go test -tags=integration ./...` if migration/repository integration tests are touched
- `cd frontend && pnpm run typecheck`
- `cd frontend && pnpm run lint:check`
- `cd frontend && pnpm run test:run`
- `cd shell/balance && go test ./...`

## Risk Points

- Ent generation touches many files; review generated changes separately from hand-written logic.
- Password updates need clear patch semantics: omitted or empty password should preserve existing encrypted password unless the UI explicitly clears it.
- Last-four matching can collide; ambiguous results must not silently bind an account.
- Account list ETag generation currently hashes returned rows, so newly added external balance fields will naturally affect ETags.
- `shell/balance` is currently `package main`; extracting shared code can temporarily break the CLI if imports are not handled carefully.

## Rollback Points

- Data model and migration are additive, so existing runtime behavior should continue if services/routes are disabled.
- If shared fetcher extraction becomes risky, keep the CLI intact and implement backend fetcher code behind equivalent tests for this task.
- If frontend binding UI needs more iteration, account/group display can still ship with unmatched/ambiguous status and manual binding can remain API-only until the UI is ready.
