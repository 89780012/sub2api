# 账号管理上游监控实施计划

## Pre-Development

- 当前任务仍是 `planning`，实现前必须先获得用户确认，再执行 `python ./.trellis/scripts/task.py start .trellis/tasks/07-04-account-management-monitoring`。
- 进入实现前加载 `trellis-before-dev`，读取 backend/frontend 相关规范。
- 因为 Codex 当前是 inline 模式，不需要补充 `implement.jsonl` / `check.jsonl` 的 sub-agent manifest。

## Implementation Checklist

1. Backend data model
   - 新增 Ent schema：`AccountUpstreamMonitorSnapshot`、`AccountUpstreamMonitorRate`。
   - 建立 `account_id` 唯一索引和 `(account_id, rate_key)` 唯一索引。
   - 生成 Ent 代码：`cd backend; go generate ./ent`。
   - 如仓库迁移机制要求手写 SQL，同步新增 migration。

2. Backend repository
   - 新增 snapshot/rate repository 接口和 Ent 实现。
   - 实现按账号 ID 批量读取快照。
   - 实现刷新成功 upsert snapshot/rates、刷新失败只更新 snapshot 错误状态。

3. Backend monitor service
   - 新增 `AccountUpstreamMonitorService`。
   - 实现账号支持性判断、base_url 归一化、token 选择、错误脱敏。
   - 实现 Sub2API 采集器：`/api/v1/auth/me`、`/api/v1/usage/dashboard/stats`、`/api/v1/groups/available`、`/api/v1/groups/rates`。
   - 实现 NewAPI 采集器：`/api/status`、`/api/user/self`、`/api/user/self/groups`。
   - 增加单元测试覆盖成功、认证失败、不支持、超时、非数字倍率、敏感信息不泄露。

4. Backend settings and runner
   - 新增 setting constants：`account_upstream_monitor_enabled`、`account_upstream_monitor_interval_minutes`。
   - 在 SettingService 中提供 runtime 读取方法，默认 enabled=true、interval=60，范围 `[15,1440]`。
   - 如需要 SettingsView 展示，扩展 admin settings request/response。
   - 新增 `AccountUpstreamMonitorRunnerService`，每分钟扫描到期账号，默认并发 5、单账号超时 15s。
   - 接入 wire/provider lifecycle，支持 `Start()`。

5. Backend handlers and routes
   - 新增 handler 方法：
     - `POST /api/v1/admin/accounts/upstream-monitor/batch`
     - `GET /api/v1/admin/accounts/:id/upstream-monitor`
     - `POST /api/v1/admin/accounts/:id/upstream-monitor/refresh`
   - 注册到 `backend/internal/server/routes/admin.go` 的 account group。
   - 为 handler 增加单元测试，确保 batch 不访问上游，refresh 返回最新快照。

6. Frontend API and types
   - 在 `frontend/src/types/index.ts` 增加上游监控类型。
   - 在 `frontend/src/api/admin/accounts.ts` 增加 batch/get/refresh API。
   - 确保 TypeScript 类型不包含敏感凭据字段。

7. Frontend account list
   - 在 `AccountsView.vue` 增加 `upstream_monitor` 可切换列。
   - 账号列表加载后按当前页账号 ID 调用 batch 快照 API。
   - 实现监控列渲染：余额、状态、最近更新时间、倍率 chips、失败/不支持文案。
   - 实现单账号刷新按钮和局部刷新状态。
   - 增加“更多倍率”轻量弹层或复用现有 tooltip/popover 展示全部 rates。

8. Frontend settings
   - 在 admin Settings 页增加上游监控开关和刷新周期，或放入现有监控相关区域。
   - 扩展 `frontend/src/api/admin/settings.ts` 类型和默认 form 值。
   - 增加必要 i18n 文案。

9. Documentation and polish
   - 检查所有用户可见文案，避免“账号倍率”“分组倍率”“上游倍率”混淆。
   - 检查日志和错误展示，确认不会输出 token/API key/cookie。
   - 确认自动刷新不会改变账号状态、调度状态、用户余额或本地 quota。

## Validation Commands

Backend:

```powershell
cd backend
go generate ./ent
go generate ./cmd/server
go test ./internal/service ./internal/handler/admin ./internal/server/routes ./internal/repository
```

Frontend:

```powershell
pnpm --dir frontend run typecheck
pnpm --dir frontend run lint:check
pnpm --dir frontend exec vitest run src/views/admin/__tests__ src/components/account/__tests__
```

Full checks if time allows:

```powershell
make test-backend
make test-frontend
```

## Risk Points

- NewAPI/Sub2API 管理端点不一定接受推理 API key；MVP 只支持上游兼容这些监控端点且凭据可用的账号，不做登录/cookie 管理。
- `base_url` 可能保存为推理接口地址，例如 `/v1`；归一化错误会导致误判不支持，需要单元测试覆盖。
- Ent schema 变更需要同步生成代码和 migration，遗漏会导致构建或运行时失败。
- 账号列表已有 ETag；本方案用独立 batch 快照 API，避免监控快照频繁变化破坏账号列表缓存语义。
- SettingsView 文件较大，扩展时要保持局部修改，避免无关格式化。
- 自动 runner 多实例可能重复刷新；第一版接受重复刷新，但要限制并发和超时。

## Review Gate

实现前需要用户确认以下最终规划：

- MVP 只支持 NewAPI/Sub2API 兼容上游监控端点。
- 只读展示 + 单账号手动刷新 + 后台自动定时刷新。
- 默认自动刷新开启，周期 60 分钟，可在设置中调整。
- 不做告警、充值、公告、登录/cookie 管理、历史趋势。
