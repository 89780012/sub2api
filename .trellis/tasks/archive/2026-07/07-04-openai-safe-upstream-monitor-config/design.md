# OpenAI-safe upstream monitor config design

## Scope

Add optional monitor-only configuration to an account. The monitor collector reads this configuration and never falls back to mutating or replacing the account's inference credentials.

## Data Model

Reuse `accounts.credentials` for account-scoped configuration to avoid adding a separate monitor channel table for the MVP.

Proposed nested shape:

```json
{
  "base_url": "https://api.openai.com/v1",
  "api_key": "sk-...",
  "upstream_monitor": {
    "enabled": true,
    "provider": "newapi",
    "site_url": "https://newapi.example.com",
    "credential_mode": "token",
    "newapi_user_id": "123"
  }
}
```

Sensitive fields are accepted in write payloads but redacted from responses:

```json
{
  "upstream_monitor": {
    "newapi_cookie": "session=...",
    "sub2api_access_token": "..."
  }
}
```

`credentials_status` should expose existence flags such as `has_upstream_monitor_newapi_cookie` and `has_upstream_monitor_sub2api_access_token` so the edit form can preserve existing monitor secrets.

## Service Behavior

Replace `resolveAccountMonitorCredentials` with monitor-config parsing:

- If no `credentials.upstream_monitor` exists: return a snapshot with `provider=unknown`, `status=unsupported`, `last_error=upstream monitor not configured`.
- If `enabled=false`: return `provider=unknown`, `status=unsupported`, `last_error=upstream monitor disabled`.
- Validate `provider`, `site_url`, and token-mode required fields.
- Validate monitor site URL through the existing URL allowlist validator.
- Do not read inference `base_url`, `api_key`, `token`, or `access_token` for monitor collection.

Provider-specific requests:

- Sub2API uses `Authorization: Bearer <sub2api_access_token>`.
- NewAPI uses `Cookie: <newapi_cookie>` and `New-Api-User: <newapi_user_id>`.
- Existing body limits, timeouts, proxy behavior, TLS profile resolution, sanitized errors, and snapshot/rate persistence remain.

Scheduled refresh should query only accounts that have monitor config enabled. If the repository keeps the existing due-query broad, the service must skip no-config accounts without saving endpoint-not-supported failures.

## API And UI

Account create/update can continue using existing account payloads; the new nested monitor config rides inside `credentials`.

Frontend changes:

- Add an "Upstream monitor" section in `EditAccountModal` for API key/upstream-compatible accounts.
- Show provider selector, monitor site URL, enabled toggle, and token-mode fields.
- For NewAPI: Cookie textarea and User ID input.
- For Sub2API: Access Token textarea.
- On edit, leave secret inputs blank by default and preserve saved secrets if existence flags are present.
- Account list labels no-config/disabled states clearly.

## Security

- Never include monitor cookies or access tokens in monitor snapshot responses.
- Redact monitor secret keys wherever account credentials are converted to response objects.
- Do not log request headers or raw upstream responses.
- Tests must assert sensitive monitor values are not returned.

## Rollback

Because monitor config is nested inside credentials and only read by the monitor service, rollback is low risk: disabling the monitor section or ignoring the nested key restores current inference behavior. Existing snapshots remain display-only.
