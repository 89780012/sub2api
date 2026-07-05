# Admin balance and rate snapshots design

## Architecture

This feature extends the existing standalone `shell/balance` fetcher into a backend-managed admin feature. The backend owns three responsibilities:

- Balance site configuration CRUD for NewAPI/Sub2API sources.
- Refresh orchestration that fetches enabled sites automatically and manually.
- Snapshot lookup that decorates admin account rows and group quality rows with external balance data.

The implementation should keep the local account billing multiplier separate from upstream multiplier snapshots. Existing `accounts.rate_multiplier` remains the local billing field. New response fields should use explicit names such as `external_balance`, `external_rate_multiplier`, `external_balance_fetched_at`, and `external_balance_match_status`.

Sensitive upstream site passwords should use the existing `SecretEncryptor` convention. This matches existing backend patterns such as channel monitor API key storage, where the service layer encrypts before repository writes and decrypts only for execution paths.

## Data Model

Add a `balance_sites` table:

- `id`
- `platform`: `newapi` or `sub2api`
- `name`
- `base_url`
- `username`
- `email`
- `password_encrypted`
- `enabled`
- `refresh_interval_minutes`, default `180`
- `last_refresh_at`
- `last_refresh_status`: empty, `success`, or `error`
- `last_refresh_error`
- standard timestamps

Add a `balance_key_snapshots` table for the latest known upstream key state:

- `id`
- `site_id`
- `external_key_id`
- `masked_key`
- `key_last4`
- `account_id`, nullable
- `match_status`: `unmatched`, `matched`, `ambiguous`, or `manual`
- `key_name`
- `status`
- `group_id`
- `group_name`
- `balance` JSONB
- `account_balance` JSONB
- `subscription_balance` JSONB
- `rate_multiplier`
- `fetched_at`
- standard timestamps

Add `account_balance_bindings` for manual resolution:

- `account_id`, unique
- `site_id`
- `external_key_id`, nullable
- `key_last4`
- `match_mode`: `auto` or `manual`
- timestamps

The snapshot table stores current state, not full history, to keep account list joins cheap. A later task can add refresh history if operators need audit trails.

## Refresh Flow

`BalanceRefreshService` runs on backend startup and refreshes enabled sites when their configured interval elapses. Manual refresh endpoints invoke the same service path for one site or all enabled sites.

The fetcher should reuse the `shell/balance` unified models and normalization behavior. If direct import is awkward because `shell/balance` is currently `package main`, implementation should extract shared code to a reusable backend-safe package while preserving the CLI contract.

Refresh behavior:

- Load enabled site config.
- Decrypt password server-side.
- Fetch the upstream site.
- Normalize keys, groups, balances, and rate multipliers.
- Compute each key's `key_last4` from upstream `masked_key` or full key when available.
- Match snapshots to local accounts using manual binding first, then unique last-four auto matching.
- Upsert latest snapshots.
- Record site refresh status.

Refresh failure must update site error metadata without deleting previous successful snapshots.

## API Contracts

Add admin endpoints under `/api/v1/admin/balance-sites`:

- `GET /balance-sites`
- `POST /balance-sites`
- `PUT /balance-sites/:id`
- `DELETE /balance-sites/:id`
- `POST /balance-sites/:id/refresh`
- `POST /balance-sites/refresh`
- `GET /balance-sites/snapshots` for configuration panels and manual binding support
- `PUT /accounts/:id/balance-binding` or equivalent endpoint for manual account-to-upstream-key binding

Responses must omit plaintext passwords and expose `password_configured`.

Extend admin account list and account detail responses with the latest matched external snapshot fields. Extend `GET /admin/groups/:id/account-quality` item rows with the same display fields.

## Frontend

Account management adds:

- Two toggleable columns: external balance and upstream multiplier.
- The external balance cell displays account/site balance, key balance, and subscription/package balance as separate independent values. The UI must not fall back from one balance source to another; if a value is missing, that field is simply absent.
- A balance sync configuration panel for multi-site CRUD, manual refresh, and refresh status, opened from the account management page's "more tools" menu.
- A row-level manual binding affordance for unmatched or ambiguous accounts.

Group quality panel adds balance/multiplier display for each account row using the existing group quality response payload. It should follow the same split-field display as the account table: account/site balance, key balance, and subscription/package windows remain independent.

## Compatibility And Rollback

The feature is additive. Existing account billing, scheduling, and group quality calculations must not read the new external snapshot fields. Rollback can remove the new routes/UI and ignore the new tables without changing existing account behavior.

## Risks

- Last-four matching can collide. Manual bindings must take precedence and ambiguous auto matches must not silently pick one upstream key.
- Refreshing external sites can fail or become slow. The service should keep timeouts bounded and preserve stale snapshots.
- Password handling must follow existing project redaction and secret-response conventions.
