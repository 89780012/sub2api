# Unified Balance Fetcher Design

## Architecture

Add a new unified CLI under `shell/`, recommended as `shell/balance/`, with one `main` package and small internal files grouped by concern:

- `main.go` handles startup, `.env` loading, config loading, fetch orchestration, and JSON encoding.
- `config.go` loads unified `BALANCE_SITE_<n>_*` environment variables and also imports legacy `NEWAPI_*`, `GGNIAO_*`, and `AI_INPUT_*` configs when unified variables are absent.
- `model.go` defines the unified output contract.
- `newapi_adapter.go` fetches and normalizes NewAPI sites.
- `sub2api_adapter.go` fetches and normalizes Sub2API sites.
- `helpers.go` owns shared `.env`, URL, masking, balance-window, and HTTP helpers.

This avoids compiling the existing `shell/newapi.go` and `shell/sub2api.go` together, since both are standalone `package main` scripts with overlapping function names.

## Unified Output Contract

Top-level:

```json
{
  "sites": [],
  "fetched_at": "RFC3339 timestamp"
}
```

Each site:

```json
{
  "platform": "newapi|sub2api",
  "name": "main",
  "base_url": "https://example.com",
  "error": "",
  "account": {},
  "keys": [],
  "groups": [],
  "subscriptions": [],
  "fetched_at": "RFC3339 timestamp"
}
```

Key fields should be stable across platforms:

- `id`
- `name`
- `masked_key`
- `status`
- `group_id` when available
- `group_name`
- `rate_multiplier`
- `key_balance`
- `subscription_balance`
- `raw` optional platform-specific details only if needed for debuggability

Balances use a typed structure with unit metadata so NewAPI quota integers and Sub2API USD balances can share a shape without pretending they are the same currency:

```json
{
  "unit": "quota|usd",
  "limit": 100,
  "used": 25,
  "remain": 75,
  "unlimited": false
}
```

## Platform Mapping

NewAPI:

- Login through the existing `/api/user/login?turnstile=` flow.
- Fetch account quota through `/api/user/self`.
- Fetch group ratios through `/api/user/self/groups`.
- Fetch keys through `/api/token/?p=1&size=20`.
- Normalize `group_map[group].ratio` into key `rate_multiplier`.
- Normalize user `quota` into account balance by dividing by `500000`. Do not present `user.quota` as a limit and do not derive account remain from `quota - used_quota`.
- Normalize key `remain_quota`, `used_quota`, and `unlimited_quota` into key balance with unit `quota`.
- Leave subscription balance absent because no package endpoint is present in the current code evidence.

Sub2API:

- Login through `/api/v1/auth/login`.
- Fetch account through `/api/v1/auth/me`.
- Fetch keys through `/api/v1/keys`.
- Fetch groups through `/api/v1/groups/available`.
- Fetch subscriptions through `/api/v1/subscriptions/active`.
- Normalize account `balance` into account balance with unit `usd`.
- Normalize key quota into key balance when `quota` or `quota_used` is non-zero.
- Build a subscription map by `group_id`; when a key's group matches an active subscription, attach daily, weekly, and monthly remaining package windows.

## Compatibility

The task should not change current `shell/newapi.go` or `shell/sub2api.go` behavior. The unified CLI can reuse code concepts, but direct imports from those files are not practical while they remain `package main` scripts with colliding names.

The new CLI should be runnable without adding dependencies. If a `shell/balance` subdirectory is used, tests can be run from that directory with Go's command-line package mode.

## Error Handling

Single-site mode can exit non-zero on fetch/config errors. Multi-site mode should return per-site errors in JSON while continuing other sites, matching the pattern already used by both existing scripts.

Do not print secrets. Output masked keys by default. Full key output should stay out of the unified contract unless the user explicitly asks for it later.

## Trade-Offs

Adding a new CLI beside the old scripts creates a little duplication during the transition, but it avoids a risky refactor of two working scripts. This task intentionally does not deprecate or remove the old scripts; that can be handled later after the unified CLI is validated with real credentials.
