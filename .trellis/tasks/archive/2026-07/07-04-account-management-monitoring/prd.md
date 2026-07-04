# 账号管理上游监控功能

## Goal

在 sub2api 管理后台的账号管理板块增加只读的上游监控信息，让管理员在管理上游账号时能直接看到上游站点余额/额度、上游分组或模型倍率，并支持单账号手动刷新和后台自动定时刷新。

## Background

- 用户提供 `https://github.com/worryzyy/upstream-hub` 作为参考项目，希望借鉴其上游账号监控能力。
- 当前仓库是 sub2api，功能落点是账号管理页 [AccountsView.vue](F:/code/sub2api/frontend/src/views/admin/AccountsView.vue:187)，账号列表列定义在 [AccountsView.vue](F:/code/sub2api/frontend/src/views/admin/AccountsView.vue:1134)。
- 当前账号管理 API 已有账号列表、详情、测试、刷新、使用统计、今日统计、模型列表、上游模型同步等接口，见 [accounts.ts](F:/code/sub2api/frontend/src/api/admin/accounts.ts:26) 和 [account_handler.go](F:/code/sub2api/backend/internal/handler/admin/account_handler.go:226)。
- 账号路由集中注册在 `/api/v1/admin/accounts` 下，见 [admin.go](F:/code/sub2api/backend/internal/server/routes/admin.go:296)；已有上游模型同步接口 `POST /admin/accounts/:id/models/sync-upstream`，见 [account_handler.go](F:/code/sub2api/backend/internal/handler/admin/account_handler.go:2108)。
- 当前账号类型包含 `upstream`，语义是通过 Base URL + API Key 连接上游，见 [domain_constants.go](F:/code/sub2api/backend/internal/service/domain_constants.go:68)；但 Antigravity 创建“上游”账号时当前保存为 `apikey` 类型并带 `base_url` / `api_key`，见 [CreateAccountModal.vue](F:/code/sub2api/frontend/src/components/account/CreateAccountModal.vue:4548)。
- 当前账号模型有账号级 `rate_multiplier`，但它是 sub2api 账号计费倍率，不是上游站点分组倍率，见 [account.go](F:/code/sub2api/backend/ent/schema/account.go:110)。
- 当前分组模型有系统内 `groups.rate_multiplier`，这是 sub2api 分组计费倍率，不一定等于上游站点分组倍率，见 [group.go](F:/code/sub2api/backend/ent/schema/group.go:42)。
- 当前账号响应类型已有 `quota_limit` / `quota_used` 等本地配额字段，见 [index.ts](F:/code/sub2api/frontend/src/types/index.ts:1008)。
- 当前已有独立“渠道监控”模块和 API，用于探测渠道可用性/模型状态，见 [ChannelMonitorView.vue](F:/code/sub2api/frontend/src/views/admin/ChannelMonitorView.vue:18) 和 [channel_monitor_handler.go](F:/code/sub2api/backend/internal/handler/admin/channel_monitor_handler.go:219)。本功能不是替代该模块，而是账号管理中的上游账户状态快照。
- 当前已有后台定时机制可复用：`ScheduledTestRunnerService` 使用 cron 每分钟扫描到期任务并发执行，见 [scheduled_test_runner_service.go](F:/code/sub2api/backend/internal/service/scheduled_test_runner_service.go:45)；服务通过 `Start()` 接入进程生命周期，见 [wire.go](F:/code/sub2api/backend/internal/service/wire.go:377)。
- 当前 `FetchUpstreamSupportedModels` 只负责上游模型列表同步，不采集上游余额或上游分组倍率，见 [upstream_models.go](F:/code/sub2api/backend/internal/service/upstream_models.go:75)。
- upstream-hub 已实现类似目标能力：README 描述集中监控上游余额、消费、倍率、公告、API Key 等；其渠道模型保存 `last_balance` / `today_cost` / `total_cost`，倍率快照保存 `model_name` / `ratio` / `completion_ratio`，见 `.trellis/workspace/guhl/upstream-hub-src/README.md:5`, `:52`, `:61`, `:69` 和 `.trellis/workspace/guhl/upstream-hub-src/backend/storage/model.go:36`, `:112`。
- upstream-hub 的 Sub2API 连接器通过 `/api/v1/auth/me` 读取余额，通过 `/api/v1/groups/available` 与 `/api/v1/groups/rates` 读取分组倍率；NewAPI 连接器通过 `/api/user/self`、`/api/status` 和 `/api/user/self/groups` 读取余额和分组倍率，见 `.trellis/workspace/guhl/upstream-hub-src/backend/connector/sub2api/sub2api.go:140`, `:175` 和 `.trellis/workspace/guhl/upstream-hub-src/backend/connector/newapi/newapi.go:142`, `:221`。

## Requirements

- R1: 功能必须落在 sub2api 管理后台账号管理上下文，而不是迁移或替换为 upstream-hub 项目。
- R2: 第一版只做只读展示、单账号手动刷新、后台自动定时刷新。
- R3: 第一版支持 NewAPI/Sub2API 兼容上游网关账号；支持条件是账号能提供可归一化为站点根地址的 `base_url`，并且现有凭据中有可用于上游监控端点的 `api_key` / `token` / `access_token`。不做账号密码登录、验证码、cookie 登录或独立上游 API Key 管理。
- R4: 其他账号类型或凭据不满足条件的账号必须显示“不支持/无数据”，不得影响账号列表加载。
- R5: UI 需要能在账号列表中看到上游监控摘要，至少包含最近上游余额/额度、上游分组或模型倍率、采集时间和错误状态。
- R6: 管理员可以对单个账号触发手动刷新；刷新只更新该账号的上游监控快照，不改变账号调度状态、账号余额、本地 quota 或用户余额。
- R7: 后台自动刷新需要可控的全局开关和刷新周期，默认开启，推荐默认周期 60 分钟；自动刷新要有超时、并发限制和失败隔离。
- R8: 设计必须区分三类倍率：sub2api 账号计费倍率 `account.rate_multiplier`、sub2api 系统分组倍率 `group.rate_multiplier`、上游站点返回的分组/模型倍率。
- R9: 设计必须区分三类余额/额度：用户余额、sub2api 账号本地 quota/usage、上游站点账户余额/额度。
- R10: 不能只改前端展示字段；必须新增后端采集、存储或透传接口，账号列表只展示本地快照，不在列表加载时直接请求上游。
- R11: 敏感凭据、token、cookie、API key 不得在列表、监控结果、API 响应或日志中明文展示。
- R12: 第一版不包含告警/通知、充值、兑换、公告、上游 API Key 管理、历史趋势图、倍率变化审计，除非后续明确扩展。

## Acceptance Criteria

- [ ] 账号管理页能展示支持账号的最近上游余额/额度状态、采集时间和错误状态；未知/不支持/失败均有明确文案。
- [ ] 账号管理页能展示支持账号的上游分组/模型倍率，文案和字段命名不与现有账号计费倍率或系统分组倍率混淆。
- [ ] 管理员可以手动刷新单个账号的上游监控数据，并看到刷新中、成功、失败状态。
- [ ] 后台能按全局配置周期自动刷新支持账号的上游监控数据，默认周期为 60 分钟。
- [ ] 自动刷新具备并发限制和请求超时；单个账号失败只更新该账号监控错误状态，不阻断其他账号。
- [ ] 不支持上游余额或倍率读取的账号类型不会阻断账号列表加载，并显示“不支持/无数据”。
- [ ] 上游监控刷新不会改变账号 `status`、`schedulable`、用户余额、sub2api 本地 quota/usage 或计费倍率。
- [ ] 后端返回的数据结构包含采集时间、最近成功时间、状态和安全错误信息，不包含敏感凭据。
- [ ] API 响应、日志和前端渲染中不会出现明文 token、cookie 或 API key。
- [ ] 第一版不发送低余额、倍率变化或失败告警通知。

## Out of Scope

- 不直接迁移 upstream-hub 的完整项目或 UI。
- 不在第一版实现上游账号密码登录、验证码处理、cookie 登录、充值、兑换、公告或上游 API Key 管理。
- 不在第一版实现低余额告警、倍率变化告警、通知冷却或通知日志。
- 不在第一版实现历史趋势图或倍率变化审计日志。
- 不改变现有网关调度计费逻辑。
- 不把上游余额自动同步为用户余额或 sub2api 本地 quota。
- 不把上游倍率自动同步为 sub2api 的账号计费倍率或系统分组倍率。

## Technical Notes

- 推荐使用本地快照表承载展示数据，账号列表通过批量快照 API 读取，不在账号列表接口中直接访问上游。
- 推荐对 `base_url` 做站点根地址归一化：去掉尾部 `/v1`、`/api/v1` 等推理接口路径后再请求 NewAPI/Sub2API 的管理/用户端点。
- 推荐第一版用兼容探测策略识别上游类型：优先尝试 Sub2API 端点，再尝试 NewAPI 端点；认证失败、404 或响应结构不匹配时记录安全错误信息。

## Open Questions

无。
