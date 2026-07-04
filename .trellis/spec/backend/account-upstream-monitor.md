# Account Upstream Monitor

## Scenario: Monitor-Only Account Upstream Configuration

### 1. Scope / Trigger

- Trigger: account upstream monitoring reads provider balance, cost, and group rate data across backend service, repository filtering, account credential redaction, and frontend edit forms.
- Scope: monitor credentials are account-scoped metadata stored under `accounts.credentials.upstream_monitor`.
- Boundary: monitor configuration must never replace or infer from inference credentials such as `base_url`, `api_key`, `token`, or `access_token`.

### 2. Signatures

- Storage path: `accounts.credentials->'upstream_monitor'`.
- Service parser: `resolveAccountMonitorCredentials(account *Account) (*accountUpstreamMonitorCredentials, string, error)`.
- Snapshot status uses existing `AccountUpstreamMonitorSnapshot` fields:
  - `provider`: `newapi`, `sub2api`, `unknown`, or `unsupported`.
  - `status`: `success`, `failed`, `unsupported`, or `unknown`.
  - `last_error`: sanitized human-readable reason.

### 3. Contracts

Monitor config shape:

```json
{
  "upstream_monitor": {
    "enabled": true,
    "provider": "newapi",
    "site_url": "https://newapi.example.com",
    "credential_mode": "token",
    "newapi_user_id": "123",
    "newapi_cookie": "session=..."
  }
}
```

Provider-specific fields:

- `newapi`: requires `newapi_cookie` and `newapi_user_id`; requests send `Cookie` and `New-Api-User`.
- `sub2api`: requires `sub2api_access_token`; requests send `Authorization: Bearer <token>`.
- `credential_mode`: currently only `token` is supported. Empty means `token`.

Response redaction:

- `newapi_cookie` and `sub2api_access_token` are sensitive credential keys.
- Account responses must omit those values and expose existence through `credentials_status.has_upstream_monitor_newapi_cookie` or `credentials_status.has_upstream_monitor_sub2api_access_token`.
- Blank secret fields in edit payloads preserve saved secrets through recursive sensitive-credential merge.

### 4. Validation & Error Matrix

- Missing `upstream_monitor` -> `provider=unknown`, `status=unsupported`, `last_error="upstream monitor not configured"`.
- `enabled=false` -> `provider=unknown`, `status=unsupported`, `last_error="upstream monitor disabled"`.
- Unsupported `credential_mode` -> `status=unsupported`, `last_error="unsupported upstream monitor credential mode"`.
- Unsupported `provider` -> `status=unsupported`, `last_error="unsupported upstream monitor provider"`.
- Missing `site_url` -> `status=unsupported`, `last_error="missing upstream monitor site URL"`.
- Missing NewAPI cookie or user ID -> `status=unsupported`, `last_error="missing NewAPI monitor cookie or user ID"`.
- Missing Sub2API token -> `status=unsupported`, `last_error="missing Sub2API monitor access token"`.
- Invalid or disallowed `site_url` -> service error sanitized through the existing monitor error path.

### 5. Good/Base/Bad Cases

- Good: OpenAI-compatible inference account keeps `base_url=https://api.openai.com` and adds a separate NewAPI monitor `site_url`.
- Base: account has no monitor config; manual refresh returns the not-configured sentinel without probing inference endpoints.
- Bad: monitor code reads `api_key`, `token`, or `access_token` as a fallback for balance collection.

### 6. Tests Required

- Service tests assert no-config and disabled states do not call inference URLs.
- Service tests assert NewAPI sends `Cookie` and `New-Api-User`.
- Service tests assert Sub2API sends bearer auth.
- Repository tests or focused SQL review assert scheduled refresh only selects enabled/configured monitor accounts.
- Redaction tests assert nested monitor secrets are omitted and status flags are present.
- Merge tests assert blank nested monitor secrets preserve saved values.
- Frontend edit tests assert blank monitor secret fields do not re-submit secrets and inference `base_url/api_key` are not changed by monitor-only edits.

### 7. Wrong vs Correct

#### Wrong

```go
siteURL := normalizeUpstreamMonitorSiteURL(account.GetCredential("base_url"))
token := account.GetCredential("api_key")
```

This breaks pure OpenAI-compatible inference accounts because `/v1` endpoints are not NewAPI/Sub2API monitor APIs.

#### Correct

```go
rawMonitor := account.Credentials["upstream_monitor"]
// Parse provider, site_url, and provider-specific token-mode credentials only.
```

The inference account remains schedulable for OpenAI-compatible traffic while monitor refresh uses the separate monitor site and credential.
