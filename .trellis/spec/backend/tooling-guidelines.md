# Backend Tooling Guidelines

Standalone Go utilities under `shell/` are operational tooling for this repository. They are not part of the `backend/` module, but they often share backend concerns such as credentials, upstream HTTP calls, quotas, and billing data.

## Scenario: Standalone Go Utility CLI

### 1. Scope / Trigger

Use this guide when adding or changing a Go CLI under `shell/`, especially when it introduces command behavior, environment variables, upstream API calls, or JSON output consumed by scripts.

### 2. Signatures

- Directory: `shell/<tool>/`.
- Module: add a local `go.mod` when the tool cannot compile as part of the existing `shell/` scripts.
- Entrypoint: `go run .` from the tool directory.
- Tests: `go test ./...` and `go vet ./...` from the tool directory.

### 3. Contracts

- Document environment keys in `shell/.env.example`.
- Prefer indexed multi-site config such as `<TOOL>_SITE_1_*` when a tool can query more than one upstream.
- Keep JSON output stable and typed. Include units for balances or quotas when platforms use different accounting units.
- Mask API keys and tokens in output by default.
- For `shell/balance`, NewAPI account `user.quota` is a current balance source, not a limit. Normalize it as `user.quota / 500000` and do not derive account remain from `quota - used_quota`; key-level `remain_quota` / `used_quota` stays separate.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| Missing required env key | Return a clear config error before any network call |
| Invalid URL | Reject it during config loading |
| Unsupported platform or mode | Return a clear unsupported-value error |
| Single-site fetch failure | Exit non-zero with a concise error |
| Multi-site fetch failure | Put the error on that site and continue other sites |
| Upstream decode failure | Return the decode error without echoing raw response bodies |

### 5. Good/Base/Bad Cases

- Good: local unit tests cover config parsing, normalization, key masking, and HTTP adapter flows with `httptest`.
- Base: existing standalone scripts still pass their file-by-file tests after adding a new utility.
- Bad: adding another root-level `package main` file under `shell/` that collides with existing script function names.

### 6. Tests Required

- Config tests for indexed ordering, missing fields, legacy fallback, and URL normalization.
- Normalization tests for platform-specific payloads into the stable output contract.
- HTTP adapter tests with local `httptest.Server` for login and authenticated fetch flows when endpoints are wrapped.
- Legacy script tests when the task promises compatibility with existing tools.

### 7. Wrong vs Correct

#### Wrong

```text
shell/new-tool.go
```

This can collide with existing `package main` symbols in `shell/newapi.go` and `shell/sub2api.go`, and it makes `go test` harder to scope.

#### Correct

```text
shell/new-tool/
  go.mod
  main.go
  config.go
  *_test.go
```

The tool can be tested independently with `cd shell/new-tool && go test ./...` while preserving old standalone scripts.
