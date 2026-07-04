# 账号池倍率分析页面设计

## Architecture

新增管理员专用账号池倍率分析能力，沿用现有账号/使用统计层级：

- Frontend route: 新增 `/admin/accounts/rate-analysis`，放在账号管理相关导航下。
- Frontend API: 在 `frontend/src/api/admin/accounts.ts` 增加账号池倍率分析接口封装。
- Backend route: 新增 `GET /api/v1/admin/accounts/pool-rate-analysis`，挂在 `backend/internal/server/routes/admin.go` 的 `accounts` 路由组内。
- Handler: 在 `backend/internal/handler/admin/account_handler.go` 增加查询参数解析和响应。
- Service: 在 `backend/internal/service/account_usage_service.go` 增加账号池倍率分析方法。
- Repository: 在 `backend/internal/repository/usage_log_repo.go` 增加按分组聚合 usage_logs 的查询。

## Data Flow

1. 页面默认生成近 7 天日期范围，调用账号池倍率分析接口。
2. 后端按 `created_at >= start_time AND created_at < end_time` 过滤 `usage_logs`。
3. 聚合维度为 `usage_logs.group_id`，左连 `groups` 获取账号池名称、平台、当前分组倍率等展示字段。
4. 有效样本只纳入 `account_stats_cost IS NOT NULL AND account_stats_cost > 0` 的记录。
5. 有效理论成本为 `SUM(account_stats_cost)`。
6. 有效账号侧成本为 `SUM(account_stats_cost * COALESCE(account_rate_multiplier, 1))`。
7. 推断倍率为 `effective_account_cost / effective_theoretical_cost`。
8. 未命中账号统计定价规则的记录单独统计请求数、tokens、标准成本等覆盖率字段，不参与倍率。

## API Contract

Request:

`GET /api/v1/admin/accounts/pool-rate-analysis?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD&sort_by=inferred_multiplier&sort_order=asc`

Response:

```json
{
  "items": [
    {
      "group_id": 1,
      "group_name": "OpenAI Pool A",
      "platform": "openai",
      "configured_rate_multiplier": 1.0,
      "requests": 120,
      "valid_requests": 100,
      "uncovered_requests": 20,
      "input_tokens": 123,
      "output_tokens": 456,
      "valid_input_tokens": 100,
      "valid_output_tokens": 400,
      "theoretical_cost": 10.0,
      "account_cost": 1.8,
      "inferred_multiplier": 0.18,
      "coverage_rate": 0.8333
    }
  ],
  "summary": {
    "groups": 1,
    "valid_groups": 1,
    "requests": 120,
    "valid_requests": 100,
    "uncovered_requests": 20,
    "theoretical_cost": 10.0,
    "account_cost": 1.8
  }
}
```

## Sorting

Supported sort fields:

- `inferred_multiplier` default ascending, valid rows first.
- `account_cost`
- `theoretical_cost`
- `requests`
- `coverage_rate`

Rows with no valid theoretical cost sort after valid rows for multiplier sorting.

## UI

The page should be a dense admin table, not a marketing-style page:

- Header with date range picker and refresh action.
- Summary strip: valid groups, total requests, valid requests, uncovered requests, total theoretical cost, total account cost.
- Table columns: account pool, platform, current configured group rate, requests, coverage, input/output tokens, theoretical cost, account cost, inferred multiplier.
- Row state for no valid samples: show no multiplier and a clear "missing pricing coverage" state.
- Action link to `/admin/channels/pricing` for maintaining account stats pricing rules.

## Compatibility

- No schema migration required for MVP; reuse existing `usage_logs.account_stats_cost` and `usage_logs.account_rate_multiplier`.
- Historical rows with `account_stats_cost IS NULL` remain supported as uncovered samples.
- This feature does not change billing, usage recording, account scheduling, or existing rate multiplier fields.

## Trade-Offs

- Reusing `account_stats_cost` means changing account stats pricing rules will not recompute historical rows. This is intentional for MVP because it preserves the actual pricing snapshot at request time.
- Excluding uncovered rows reduces apparent coverage but keeps the multiplier mathematically clean.
