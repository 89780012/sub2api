# Unify NewAPI and Sub2API balance fetcher

## Goal

Create a unified Go CLI for fetching account balance, owned API keys, key rate multipliers, and subscription/package balances from both NewAPI-style sites and Sub2API-style sites.

The user value is one consistent JSON shape across platforms, so downstream scripts do not need to understand the different `newapi.go` and `sub2api.go` output contracts.

## Confirmed Facts

- `shell/newapi.go` is a standalone `package main` script that logs in to NewAPI, fetches `/api/user/self`, `/api/user/self/groups`, and `/api/token/?p=1&size=20`, then emits user quota, keys, and `group_map` data. See `shell/newapi.go:44`, `shell/newapi.go:57`, `shell/newapi.go:82`, `shell/newapi.go:173`, and `shell/newapi.go:533`.
- NewAPI key output already includes each key's group and group ratio through `outputKey.GroupRatio`. See `shell/newapi.go:116`.
- `shell/sub2api.go` is a standalone `package main` script that logs in to Sub2API, fetches account data, keys, groups, and active subscriptions. See `shell/sub2api.go:51`, `shell/sub2api.go:74`, `shell/sub2api.go:89`, `shell/sub2api.go:222`, and `shell/sub2api.go:669`.
- Sub2API key output already includes each key's `group_id`, `group_name`, `rate_multiplier`, and quota fields. See `shell/sub2api.go:145`.
- Sub2API subscriptions include group ID, active period, usage windows, and group limits, which can be joined to keys by `group_id`. See `shell/sub2api.go:89` and `shell/sub2api.go:289`.
- The existing scripts duplicate helper concepts such as `.env` loading, URL normalization, site indexing, JSON encoding, and key masking.
- `shell/` is not currently its own Go module. Existing tests pass when invoked file-by-file, but `go test ./shell` from the repository root fails because the root has no `go.mod`; `backend/go.mod` is a separate module.
- The checked-in `shell/newapi.json` and `shell/sub2api.json` samples contain mojibake and invalid JSON string quoting, so they should not be used as golden fixtures unless regenerated or repaired.

## Requirements

- R1. Add a unified Go CLI under `shell/` that can fetch from both platform types in one run.
- R2. Support NewAPI sites using the currently working NewAPI login, account, group, and token endpoints.
- R3. Support Sub2API sites using the currently working auth, account, keys, groups, and active subscription endpoints.
- R4. Emit one consistent top-level JSON shape with `sites`, each site containing platform, name, base URL, optional error, account balance, keys, groups or plans when useful, and fetch timestamp.
- R5. Each key entry must include ID, name, masked key, status, group identity/name, rate multiplier, key-level quota/balance when available, and subscription/package balance when the platform exposes it.
- R6. For Sub2API, attach active subscription/package balance to keys by matching the key group ID to the subscription group ID.
- R7. For NewAPI, expose account balance as `user.quota / 500000`, expose key quota fields separately, and leave subscription/package balance absent or null unless a real NewAPI package source is discovered during implementation.
- R8. Add the unified CLI alongside the current standalone `shell/newapi.go` and `shell/sub2api.go`; do not replace or deprecate the old scripts in this task.
- R9. Reuse shared helpers in the new implementation so the unified CLI does not copy the same `.env`, URL, masking, and JSON-output logic into every adapter.
- R10. Provide tests for config loading, output normalization, NewAPI normalization, Sub2API subscription-to-key matching, and helper behavior.
- R11. Update `shell/.env.example` with the unified configuration format while keeping notes for legacy environment variables.

## Acceptance Criteria

- [ ] Running the unified CLI with a mixed NewAPI and Sub2API environment configuration returns valid indented JSON with a `sites` array.
- [ ] A NewAPI result includes account balance derived from `user.quota / 500000`, does not present `user.quota` as a limit, and each key includes a `rate_multiplier` derived from its group map.
- [ ] A Sub2API result includes account balance and each key includes a `rate_multiplier` derived from group data.
- [ ] A Sub2API key whose `group_id` matches an active subscription includes package balance windows; a key without a matching active subscription has a null or omitted package balance.
- [ ] Site-level fetch errors are captured per site in multi-site mode without preventing other configured sites from being fetched.
- [ ] Existing script tests still pass using their current file-by-file invocation.
- [ ] New unified CLI tests pass without real network credentials by using local normalization/unit tests and HTTP test servers where endpoint behavior is needed.
- [ ] `shell/.env.example` documents enough variables for one NewAPI site and one Sub2API site.

## Out Of Scope

- Changing backend application behavior under `backend/`.
- Repairing or regenerating the checked-in sample JSON files, unless tests need fresh valid fixtures.
- Adding a frontend view for the unified result.
- Discovering undocumented NewAPI package endpoints beyond the endpoints already used by the existing script.
