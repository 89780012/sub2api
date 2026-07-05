# Implementation Plan

## Steps

1. Create the unified CLI structure under `shell/balance/`.
   - Output: `main.go`, shared models, config, helpers, and platform adapter files exist.
   - Test: `cd shell/balance && go test ./...` compiles.

2. Implement unified config loading.
   - Output: `BALANCE_SITE_<n>_PLATFORM/NAME/URL/USERNAME|EMAIL/PASSWORD` is supported, with legacy NewAPI and Sub2API environment fallbacks.
   - Test: unit tests cover ordering, missing fields, URL normalization, and legacy fallback.

3. Implement unified models and helper functions.
   - Output: account, key, group, subscription, and balance structs represent both platforms.
   - Test: unit tests cover key masking, balance-window remain math, and JSON shape-sensitive fields.

4. Implement the NewAPI adapter.
   - Output: existing NewAPI endpoint flow normalizes account quota, key quota, and group ratios into the unified output.
   - Test: HTTP test server or pure normalization tests cover login, account, groups, token mapping, and per-site error behavior.

5. Implement the Sub2API adapter.
   - Output: existing Sub2API endpoint flow normalizes account balance, keys, groups, and subscriptions into the unified output.
   - Test: tests cover group-derived rate multipliers and subscription attachment by key `group_id`.

6. Wire orchestration and update docs/config example.
   - Output: unified CLI fetches all configured sites, emits indented JSON, and `shell/.env.example` documents mixed platform config.
   - Test: run unified tests plus existing file-by-file tests for `newapi.go` and `sub2api.go`.

## Validation Commands

- `cd shell/balance && go test ./...`
- `go test .\\shell\\newapi.go .\\shell\\newapi_test.go`
- `go test .\\shell\\sub2api.go .\\shell\\sub2api_test.go`
- Optional real credential smoke test: run the unified CLI with `shell/.env` and inspect that JSON is valid and keys are masked.

## Risk Points

- `shell/` currently has no module at the repository root, so verification commands may need to run from `shell/balance` or use explicit file paths.
- Existing sample JSON files are not reliable golden fixtures because they contain invalid JSON.
- Sub2API balance units are USD-like decimals while NewAPI quotas are integer quota units; the output must preserve units rather than merging them as one numeric currency.
- NewAPI subscription/package balance is not present in existing code evidence; implementation should not invent an endpoint.

## Rollback

The preferred implementation is additive. If the unified CLI causes trouble, remove `shell/balance/` and the `.env.example` additions without touching the existing scripts.
