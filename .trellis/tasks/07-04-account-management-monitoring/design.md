# 账号管理上游监控技术设计

## Scope

第一版在账号管理页展示 NewAPI/Sub2API 兼容上游网关的只读监控快照。系统从已保存账号凭据中读取 `base_url` 和可用 token，主动采集上游余额/额度与分组倍率，保存为本地快照；前端只读展示快照并提供单账号刷新按钮。

不迁移 upstream-hub 的登录、充值、公告、API Key 管理、历史趋势和告警系统。

## Architecture

新增一个后端领域服务：`AccountUpstreamMonitorService`。

职责：

- 判断账号是否满足监控条件。
- 将账号 `base_url` 归一化为上游站点根地址。
- 对 NewAPI/Sub2API 兼容端点执行采集。
- 将采集结果写入本地快照表。
- 为账号列表提供批量快照查询。
- 为后台 runner 提供“哪些账号到期需要刷新”的能力。

新增一个后台 runner：`AccountUpstreamMonitorRunnerService`。

职责：

- 每分钟 tick 一次，读取全局设置。
- 找出支持且快照过期的账号。
- 以固定并发刷新，单账号超时，失败隔离。
- 只写监控快照，不修改账号调度状态。

前端账号页保持当前账号列表加载逻辑，加载列表后调用批量快照 API 获取当前页账号的监控数据。手动刷新只刷新单个账号快照并局部更新 UI。

## Data Model

新增 Ent schema，推荐表名：

- `account_upstream_monitor_snapshots`
- `account_upstream_monitor_rates`

`account_upstream_monitor_snapshots` 字段：

- `account_id`：唯一索引，关联账号。
- `provider`：`sub2api` / `newapi` / `unsupported` / `unknown`。
- `status`：`unknown` / `unsupported` / `success` / `failed`。
- `site_url`：归一化后的站点根地址，非敏感。
- `balance`：上游余额，nullable。
- `balance_unit`：默认 `USD`，NewAPI 由 quota 转换后也以 USD 展示。
- `quota`、`quota_used`：预留 nullable，用于兼容返回额度/用量的上游。
- `today_cost`、`total_cost`：预留 nullable，Sub2API/NewAPI 可采集时保存。
- `last_checked_at`：最近一次尝试采集时间。
- `last_success_at`：最近一次成功采集时间。
- `last_error`：安全错误信息，禁止包含响应体原文和凭据。
- `raw_meta`：JSONB，保存非敏感扩展，例如 `quota_per_unit`、响应来源等。
- `created_at`、`updated_at`。

`account_upstream_monitor_rates` 字段：

- `account_id`：关联账号。
- `provider`：`sub2api` / `newapi`。
- `rate_key`：上游分组名、模型名或上游分组 ID 字符串。
- `display_name`：展示名。
- `description`：上游描述，nullable。
- `ratio`：上游输入倍率。
- `completion_ratio`：上游输出倍率，nullable；第一版 NewAPI/Sub2API 分组倍率通常为空。
- `first_seen_at`、`last_seen_at`。
- 唯一索引：`(account_id, rate_key)`。

刷新成功时 upsert snapshot 和 rates，并清理该账号本次未返回的旧 rates，避免展示过期倍率。刷新失败时只更新 snapshot 的 `status=failed`、`last_checked_at`、`last_error`，保留上一批成功 rates 但 UI 需要标明快照失败和数据时间。

## Compatibility Detection

支持条件：

- 账号 `credentials` 中存在可用 `base_url`。
- 凭据中存在 `access_token`、`token` 或 `api_key` 之一。
- 账号不是 Bedrock、Service Account、官方 OAuth 等需要专用授权流程的类型。

站点根地址归一化：

- trim 空格和尾部 `/`。
- 若尾部为 `/v1`、`/api/v1`、`/v1beta`、`/antigravity`，移除到站点根。
- 校验 scheme 必须是 `http` 或 `https`，生产环境建议拒绝空 host。

采集顺序：

1. Sub2API probe：
   - `GET {site}/api/v1/auth/me`，`Authorization: Bearer <token>`，解析 `balance`。
   - `GET {site}/api/v1/usage/dashboard/stats`，解析 `today_actual_cost`、`total_actual_cost`，失败不影响余额和倍率。
   - `GET {site}/api/v1/groups/available`，解析 `id`、`name`、`description`、`rate_multiplier`。
   - `GET {site}/api/v1/groups/rates`，成功时用返回 override 覆盖对应分组倍率。
2. NewAPI probe：
   - `GET {site}/api/status`，解析 `quota_per_unit`，无效时按 upstream-hub 经验回退 `500000`。
   - `GET {site}/api/user/self`，`Authorization: Bearer <token>`，解析 `quota`、`used_quota`，换算余额和总消耗。
   - `GET {site}/api/user/self/groups`，解析分组倍率；`ratio` 非数字的分组跳过。

如果 Sub2API 返回 401/403/404 或结构不匹配，再尝试 NewAPI。两者都失败时记录 `unsupported` 或 `failed`：

- 缺少 base_url/token：`unsupported`。
- 端点不存在或结构不匹配：`unsupported`。
- 认证失败、超时、5xx：`failed`。

错误信息只记录归类后的短消息，例如 `upstream authentication failed`、`upstream monitor endpoint not supported`、`upstream request timed out`。

## API Contract

新增账号管理 API：

- `POST /api/v1/admin/accounts/upstream-monitor/batch`
  - request：`{ "ids": [1, 2, 3] }`
  - response：`{ "items": { "1": AccountUpstreamMonitorSnapshot, "2": ... } }`
  - 用于当前账号列表页批量读取本地快照，不访问上游。

- `GET /api/v1/admin/accounts/:id/upstream-monitor`
  - response：单账号快照，包含 rates。
  - 用于详情或刷新后兜底。

- `POST /api/v1/admin/accounts/:id/upstream-monitor/refresh`
  - 同步触发单账号采集，成功或失败都返回最新快照。
  - 配置/不支持类错误返回 200 + `status=unsupported`，上游请求失败返回 200 + `status=failed`；只有账号不存在、权限、服务内部错误走 HTTP 错误。

前端类型建议：

```ts
export interface AccountUpstreamMonitorRate {
  rate_key: string
  display_name: string
  description?: string | null
  ratio: number
  completion_ratio?: number | null
  first_seen_at?: string | null
  last_seen_at?: string | null
}

export interface AccountUpstreamMonitorSnapshot {
  account_id: number
  provider: 'sub2api' | 'newapi' | 'unsupported' | 'unknown'
  status: 'unknown' | 'unsupported' | 'success' | 'failed'
  site_url?: string | null
  balance?: number | null
  balance_unit?: string | null
  quota?: number | null
  quota_used?: number | null
  today_cost?: number | null
  total_cost?: number | null
  last_checked_at?: string | null
  last_success_at?: string | null
  last_error?: string | null
  rates: AccountUpstreamMonitorRate[]
}
```

## Settings

新增 DB-backed settings：

- `account_upstream_monitor_enabled`，默认 `true`。
- `account_upstream_monitor_interval_minutes`，默认 `60`，建议限制 `[15, 1440]`。

Settings 页新增一个小节或复用监控相关区域，提供开关和刷新周期输入。Runner 读取 setting 时 fail-open：读取失败按启用和 60 分钟处理；周期非法时 clamp 到默认值。

## Frontend UX

账号列表新增可切换列：`上游监控`。

列内容：

- 支持且成功：显示余额、上游类型、最近成功时间、最多 2-3 个倍率 chip，以及“更多”入口查看全部倍率。
- 未采集：显示“未采集”，提供刷新按钮。
- 不支持：显示“不支持”，tooltip 说明缺少兼容 base_url/token 或协议不支持。
- 失败：显示“采集失败”，展示安全错误摘要和上次成功时间。

操作：

- 单账号刷新按钮放在监控列或行操作菜单内，使用刷新图标和 tooltip。
- 刷新中只锁定该账号刷新按钮，不锁整页。
- 刷新完成后局部更新 `monitorSnapshots[account.id]`。

前端不要显示、缓存或透传明文凭据。

## Security

- 后端日志不得打印 token、cookie、API key 或完整上游响应体。
- API 响应只返回快照和安全错误摘要。
- 手动刷新和批量读取只允许管理员账号使用现有 admin 中间件。
- 上游请求使用现有 HTTP client/proxy 能力时，不能把请求体或 headers 作为错误详情输出。

## Operational Behavior

- 自动刷新按账号维度隔离失败。
- Runner 每轮设置总超时，例如 5 分钟；单账号请求超时建议 15 秒。
- 并发建议默认 5，避免批量打爆上游。
- 对没有快照的支持账号优先刷新；已有快照按 `last_checked_at + interval` 到期刷新。
- 多实例部署第一版可以接受重复刷新；如果后续需要强一致，可复用 idempotency/system lock。

## Rollback

- 前端列可通过隐藏列避免影响常规账号管理。
- 后端 runner 可通过 `account_upstream_monitor_enabled=false` 关闭自动刷新。
- 快照表与账号核心调度、计费、余额无写入耦合，禁用功能不影响网关请求路径。
