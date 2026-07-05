# Admin balance and rate snapshots

## Goal

Add admin-visible external balance and upstream rate multiplier snapshots to account management rows and the group quality panel.

The user value is that operators can see each locally configured account's upstream key balance and upstream billing multiplier without leaving the admin panel, while still preserving manual refresh and automated refresh every 3 hours.

## Confirmed Facts

- `shell/balance` already provides a unified CLI-style fetcher with one JSON contract for NewAPI and Sub2API sites. It emits `sites[].keys[].masked_key`, `key_balance`, `subscription_balance`, and `rate_multiplier`; it also emits `sites[].groups[].rate_multiplier`. See `shell/balance/model.go`, `shell/balance/main.go`, `shell/balance/newapi_adapter.go`, and `shell/balance/sub2api_adapter.go`.
- The prior Trellis task `07-04-unified-balance-fetcher` explicitly scopes the unified fetcher to a standalone CLI and excludes backend/frontend integration.
- Admin accounts already have an internal `rate_multiplier` field, but it is documented as the local account billing multiplier and participates in existing backend/frontend account contracts. It should not be reused for upstream key multiplier display.
- Admin account list responses are built in `backend/internal/handler/admin/account_handler.go` and returned as `AccountWithConcurrency`, which wraps `dto.Account`.
- The group quality panel is backed by `GET /api/v1/admin/groups/:id/account-quality`; its response items are assembled in `backend/internal/handler/admin/group_handler.go` as `groupAccountQualityItem`.
- Frontend account rows are rendered by `frontend/src/views/admin/AccountsView.vue`; the group quality panel is rendered by `frontend/src/components/admin/group/GroupQualityPanel.vue`.
- Frontend types for `Account` and `GroupAccountQualityItem` live in `frontend/src/types/index.ts`.
- Account credentials returned to the frontend are already sanitized, so upstream balance site usernames/passwords must be stored and exposed through a separate safe admin contract rather than leaking secrets in account list payloads.

## Requirements

- R1. Add an admin-configurable balance site source for NewAPI and Sub2API platforms, including site name, base URL, username or email, password, enabled state, refresh interval, and last refresh status.
- R1a. The first implementation must support multiple balance site configurations with admin CRUD, not only one global site.
- R2. Store upstream site credentials securely enough for existing project conventions; frontend responses must not include plaintext passwords and should only expose whether a password is configured.
- R3. Refresh enabled balance sites automatically every 3 hours by default.
- R4. Provide a manual refresh action for admins so a configured site can be refreshed on demand without waiting for the scheduled job.
- R5. Persist latest fetched upstream key snapshots in the database, including masked key, key last four characters, upstream key name/status/group, key balance, subscription balance when available, upstream rate multiplier, fetched timestamp, and match status.
- R6. Match upstream key snapshots to local accounts by the local account key's last four characters when possible.
- R7. Support manual resolution when last-four matching is missing or ambiguous.
- R8. Add external balance and upstream multiplier fields to admin account list rows.
- R8a. Add the balance site configuration UI under the account management page's "more tools" menu as "balance sync configuration".
- R8b. The account table balance cell must display account/site balance, subscription/package balance, and key balance as separate independent values when each source is available; missing fields must not be synthesized from another balance field.
- R9. Add external balance and upstream multiplier fields to group quality panel account rows.
- R10. Keep the existing local account `rate_multiplier` behavior unchanged.
- R11. Preserve stale-but-known snapshot values when a refresh fails, while surfacing refresh errors and stale timestamps to admins.
- R12. Reuse the existing `shell/balance` normalization contract or extract equivalent shared code instead of reimplementing NewAPI/Sub2API parsing from scratch.

## Acceptance Criteria

- [ ] Admins can create, edit, enable/disable, and delete balance site configurations for NewAPI and Sub2API sources.
- [ ] The balance sync configuration panel is reachable from the account management page's more tools menu.
- [ ] Passwords entered in the balance site configuration are not returned by any API response.
- [ ] Enabled balance sites refresh automatically on a 3-hour cadence by default.
- [ ] Admins can manually refresh a balance site and see success/failure status.
- [ ] A successful refresh stores latest upstream key snapshots with balance, subscription balance when available, rate multiplier, last-four key matcher, fetched timestamp, and source site metadata.
- [ ] A local account with a uniquely matching key last-four displays external balance and upstream multiplier in the account management table.
- [ ] Account table balance cells show account/site balance, subscription/package daily/weekly/monthly windows, and key balance as separate values without fallback or mixing.
- [ ] A missing or ambiguous last-four match is visible to the admin and can be resolved manually.
- [ ] The group quality panel includes the same external balance and upstream multiplier for each account row where a snapshot is matched.
- [ ] Existing local billing multiplier values and account scheduling/charging behavior do not change.
- [ ] Refresh failures do not erase the previous successful snapshot.

## Out Of Scope

- Changing how local account billing multipliers are calculated or applied.
- Changing user-facing API key quota semantics.
- Replacing the existing `shell/balance` CLI as a user-facing utility.
- Supporting upstream platforms beyond NewAPI and Sub2API in the first implementation.
