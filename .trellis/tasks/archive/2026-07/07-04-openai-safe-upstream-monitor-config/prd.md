# OpenAI-safe upstream monitor config

## Goal

Fix account upstream monitoring so it never requires changing the existing OpenAI-compatible inference `base_url` / `api_key`. Admins should be able to keep an account's normal OpenAI request routing intact while optionally configuring a separate NewAPI/Sub2API monitoring site credential for balance, cost, and upstream group/rate display.

## Background

- The first monitoring implementation reused the account's existing inference credentials. Pure OpenAI-compatible accounts often expose `/v1/*` only, so they show `unsupported` even though the inference account is healthy.
- The user confirmed their account is an OpenAI-compatible account and changing its `base_url` would break normal requests.
- Evidence from `upstream-hub-src` shows the correct boundary:
  - `backend/storage/model.go` defines a separate monitor `Channel` with `SiteURL`, `CredentialMode`, encrypted credentials, and monitor status fields.
  - `frontend/components/monitor/channel-form-dialog.tsx` asks for a monitor `site_url` plus independent token-mode credentials.
  - `backend/channel/service.go` validates token-mode credentials and builds monitor sessions without touching inference accounts.
  - `backend/connector/newapi/newapi.go` monitors NewAPI using `Cookie` plus `New-Api-User`.
  - `backend/connector/sub2api/sub2api.go` monitors Sub2API using `Authorization: Bearer <access_token>`.
- This project should adapt that model inside account management, not migrate upstream-hub's full channel system.

## Requirements

- R1: Existing inference fields (`base_url`, `api_key`, `token`, `access_token`), scheduling, local quota, billing multiplier, and group bindings must not be changed by monitor configuration or refresh.
- R2: Add optional account-level monitor configuration separate from inference credentials.
- R3: MVP supports token-mode monitor credentials only:
  - provider: `newapi` or `sub2api`
  - monitor site URL
  - enabled flag
  - NewAPI: cookie plus user ID
  - Sub2API: access token
- R4: If monitor configuration is absent or disabled, the account list displays a clear not-configured state instead of `upstream monitor endpoint not supported`.
- R5: Manual refresh and scheduled refresh use the separate monitor configuration when present.
- R6: Scheduled refresh skips accounts with monitoring disabled or missing monitor config without marking them as protocol failures.
- R7: Sensitive monitor credentials must not appear in account list responses, monitor snapshot responses, logs, or post-save UI state.
- R8: Edit forms may leave monitor credential fields blank to preserve saved secrets.
- R9: Existing snapshot/rate storage can be reused; this task changes how the collector obtains monitor credentials and how the UI configures them.
- R10: UI copy must clearly distinguish inference configuration from upstream monitor configuration.

## Acceptance Criteria

- [ ] An OpenAI-compatible account can keep its inference `base_url` and `api_key` unchanged while adding a separate monitor site URL and token credential.
- [ ] A pure OpenAI-compatible account with no monitor config displays a not-configured monitor state, not `upstream monitor endpoint not supported`.
- [ ] Manual refresh uses monitor site URL and monitor credential, not inference `base_url`.
- [ ] Scheduled refresh skips monitor-disabled or missing-config accounts without recording protocol failures.
- [ ] NewAPI monitor config supports Cookie plus User ID and sends the required `Cookie` and `New-Api-User` headers.
- [ ] Sub2API monitor config supports access token and sends `Authorization: Bearer`.
- [ ] API responses and frontend state do not expose cookie, access token, API key, or authorization headers.
- [ ] Tests cover no-config state, OpenAI inference preservation, NewAPI credential parsing, Sub2API credential parsing, and credential redaction.

## Out of Scope

- No upstream-hub-style independent channel management UI.
- No recharge, announcement sync, API key management, captcha login, password login, or notifications.
- No OpenAI official account balance monitoring unless a reliable supported API is later identified.
- No gateway request routing or billing behavior changes.

## Decisions

- D1: MVP is token-mode only. Username/password login, captcha, Turnstile, 2FA handling, and automatic session renewal are out of scope.
