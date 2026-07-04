# 账号池倍率分析页面

## Goal

新增一个管理员页面，用使用记录中已预计算的账号统计定价成本反推各账号池/分组的实际成本倍率，并按倍率排序，帮助管理员识别不同账号池的真实成本水平。

## Background

- 管理员的账号管理方式是把账号组织成账号池/分组；单账号维度对本分析场景没有意义。
- 账号池倍率来自账号侧成本和理论成本的比例，而不是用户实际扣费倍率。
- 模型输入价、输出价复用渠道管理中的“账号统计定价规则”；分析页不新增临时价格配置，也不新增价格持久化表。
- 页面默认统计近 7 天，并允许管理员通过日期范围选择器切换统计窗口。

## Confirmed Facts

- 前端已有管理员账号管理页 `frontend/src/views/admin/AccountsView.vue`，路由为 `/admin/accounts`。
- 前端已有管理员账号 API `frontend/src/api/admin/accounts.ts`，包含账号列表、单账号统计、今日统计、可用模型等接口封装。
- 后端已有账号统计接口 `GET /api/v1/admin/accounts/:id/stats`，由 `backend/internal/handler/admin/account_handler.go` 调用 `AccountUsageService.GetAccountUsageStats`。
- 使用记录表 `usage_logs` 已有倍率分析所需字段：`group_id`、`input_tokens`、`output_tokens`、`total_cost`、`actual_cost`、`account_rate_multiplier`、`account_stats_cost`、`created_at`。
- 分组/账号池管理已存在：`/admin/groups`，`Group`/`AdminGroup` 类型包含 `rate_multiplier`、`account_count` 等字段。
- 渠道管理已有模型定价结构 `ChannelModelPricing`，支持 `input_price`、`output_price`、`cache_write_price`、`cache_read_price` 等字段。
- 渠道管理已有账号统计定价规则 `AccountStatsPricingRule`，可绑定 `group_ids` 并把预计算结果写入 `usage_logs.account_stats_cost`。
- 现有渠道定价 UI 位于 `frontend/src/views/admin/ChannelsView.vue`，支持编辑模型定价和账号统计定价规则，但当前没有账号池倍率排行页面。
- 已决定有效倍率只纳入 `account_stats_cost IS NOT NULL AND account_stats_cost > 0` 的记录；缺少 `account_stats_cost` 的历史记录作为未覆盖样本展示，不参与倍率计算或排序。

## Requirements

- R1: 提供一个管理员可访问的账号池/分组倍率分析页面。
- R2: 页面按账号池/分组维度聚合使用记录，展示请求数、有效请求数、未覆盖请求数、输入 tokens、输出 tokens、理论成本、账号侧成本、推断倍率和覆盖率。
- R3: 推断倍率公式为 `SUM(account_stats_cost * COALESCE(account_rate_multiplier, 1)) / SUM(account_stats_cost)`，仅对有效记录计算。
- R4: 缺少账号统计定价覆盖、有效理论成本为 0、或样本不足的账号池必须明确标记，不显示误导性倍率。
- R5: 推断倍率支持排序，至少支持按倍率升序和降序排序；无有效倍率的行应排在有效行之后。
- R6: 页面支持日期范围过滤，默认近 7 天。
- R7: 页面提供入口或提示，引导管理员到渠道管理维护账号统计定价规则。
- R8: MVP 不支持在分析页输入临时价格后实时重算历史区间；价格变更只影响后续写入的 `usage_logs.account_stats_cost`。

## Acceptance Criteria

- [ ] 管理员能进入倍率分析页面，并看到按倍率排序的账号池/分组列表。
- [ ] 首次进入页面默认查询近 7 天数据，并可切换日期范围后重新加载。
- [ ] 每一行展示统计窗口、样本量、tokens、理论成本、账号侧成本、推断倍率和覆盖率。
- [ ] 倍率分子使用账号侧成本 `account_stats_cost * account_rate_multiplier`，不使用用户实际扣费 `actual_cost`。
- [ ] `account_stats_cost` 为空的记录不会参与有效倍率计算，并会在未覆盖样本统计中体现。
- [ ] 缺少账号统计定价覆盖或理论成本为 0 的行不会显示有效倍率。
- [ ] 页面和接口只对管理员开放。
- [ ] 新增接口或前端逻辑有针对核心倍率公式、未覆盖样本、零理论成本和排序边界的测试。

## Out of Scope

- 自动修改账号或分组现有 `rate_multiplier` 字段。
- 单账号维度排行或单账号倍率分析。
- 分析页临时价格配置、临时价格持久化、或按新价格实时重算历史日志。
- 自动抓取第三方官方模型价格。
- 替代现有用户扣费、分组倍率、账号倍率结算逻辑。
