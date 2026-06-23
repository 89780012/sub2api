package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GroupHandler handles admin group management
type GroupHandler struct {
	adminService         service.AdminService
	dashboardService     *service.DashboardService
	groupCapacityService *service.GroupCapacityService
	accountQualityReader service.AccountQualityReader
	usageService         *service.UsageService
}

type groupAccountQualityItem struct {
	AccountID                int64      `json:"account_id"`
	AccountName              string     `json:"account_name"`
	Platform                 string     `json:"platform"`
	AccountType              string     `json:"account_type"`
	Status                   string     `json:"status"`
	Schedulable              bool       `json:"schedulable"`
	Priority                 int        `json:"priority"`
	Concurrency              int        `json:"concurrency"`
	GroupIDs                 []int64    `json:"group_ids,omitempty"`
	LastUsedAt               *time.Time `json:"last_used_at,omitempty"`
	UpdatedAt                time.Time  `json:"updated_at"`
	QualityKnown             bool       `json:"quality_known"`
	QualityScore             float64    `json:"quality_score"`
	ScoreBreakdown           string     `json:"score_breakdown"`
	WindowStart              *time.Time `json:"window_start,omitempty"`
	WindowEnd                *time.Time `json:"window_end,omitempty"`
	SnapshotUpdatedAt        *time.Time `json:"snapshot_updated_at,omitempty"`
	TotalRequests            int64      `json:"total_requests"`
	SuccessRequests          int64      `json:"success_requests"`
	FailureRequests          int64      `json:"failure_requests"`
	RecentSuccessRate        float64    `json:"recent_success_rate"`
	ErrorRate                float64    `json:"error_rate"`
	TTFTSampleCount          int64      `json:"ttft_sample_count"`
	TTFTLE5sCount            int64      `json:"ttft_le_5s_count"`
	TTFTLE10sCount           int64      `json:"ttft_le_10s_count"`
	TTFTGT10sCount           int64      `json:"ttft_gt_10s_count"`
	TTFTGT20sCount           int64      `json:"ttft_gt_20s_count"`
	TTFTGT40sCount           int64      `json:"ttft_gt_40s_count"`
	TTFTLE5sRate             float64    `json:"ttft_le_5s_rate"`
	TTFTLE10sRate            float64    `json:"ttft_le_10s_rate"`
	TTFTGT10sRate            float64    `json:"ttft_gt_10s_rate"`
	TTFTGT20sRate            float64    `json:"ttft_gt_20s_rate"`
	TTFTGT40sRate            float64    `json:"ttft_gt_40s_rate"`
	SampleConfidence         float64    `json:"sample_confidence"`
	NeutralBase              float64    `json:"neutral_base"`
	SuccessComponent         float64    `json:"success_component"`
	TTFT5sComponent          float64    `json:"ttft_5s_component"`
	TTFT10sComponent         float64    `json:"ttft_10s_component"`
	FastBonus                float64    `json:"fast_bonus"`
	SlowPenalty              float64    `json:"slow_penalty"`
	ErrorPenalty             float64    `json:"error_penalty"`
	BaseQualityScore         float64    `json:"base_quality_score"`
	EffectiveQualityScore    float64    `json:"effective_quality_score"`
	TransientPenalty         float64    `json:"transient_penalty"`
	RecoveryCredit           float64    `json:"recovery_credit"`
	AppliedPenalty           float64    `json:"applied_penalty"`
	SlowStreak               int64      `json:"slow_streak"`
	ErrorStreak              int64      `json:"error_streak"`
	RecoverySuccessStreak    int64      `json:"recovery_success_streak"`
	RecoveryFastStreak       int64      `json:"recovery_fast_streak"`
	AuxiliaryTotalRequests   int64      `json:"auxiliary_total_requests"`
	AuxiliarySuccessRequests int64      `json:"auxiliary_success_requests"`
	AuxiliaryFailureRequests int64      `json:"auxiliary_failure_requests"`
	AuxiliaryTTFTSampleCount int64      `json:"auxiliary_ttft_sample_count"`
	AuxiliaryTTFTLE5sCount   int64      `json:"auxiliary_ttft_le_5s_count"`
	AuxiliaryTTFTLE10sCount  int64      `json:"auxiliary_ttft_le_10s_count"`
	AuxiliaryTTFTGT10sCount  int64      `json:"auxiliary_ttft_gt_10s_count"`
	AuxiliaryTTFTGT20sCount  int64      `json:"auxiliary_ttft_gt_20s_count"`
	AuxiliaryTTFTGT40sCount  int64      `json:"auxiliary_ttft_gt_40s_count"`
	AuxiliaryWeight          float64    `json:"auxiliary_weight"`
	ScheduleRank             int        `json:"schedule_rank"`
	ScheduleSortKey          string     `json:"schedule_sort_key"`
	IsManualPrimary          bool       `json:"is_manual_primary"`
	IsActivePrimary          bool       `json:"is_active_primary"`
	IsPoolMode               bool       `json:"is_pool_mode"`
	PoolModeRetryCount       int        `json:"pool_mode_retry_count"`
	PoolModeRetryStatusCodes []int      `json:"pool_mode_retry_status_codes,omitempty"`
	PrimaryEligible          bool       `json:"primary_eligible"`
	PrimaryIneligibleReason  string     `json:"primary_ineligible_reason,omitempty"`
}

type groupAccountQualityResponse struct {
	GroupID                        int64                     `json:"group_id"`
	GroupName                      string                    `json:"group_name"`
	GroupPlatform                  string                    `json:"group_platform"`
	PrimaryAccountMode             string                    `json:"primary_account_mode"`
	ManualPrimaryAccountID         *int64                    `json:"manual_primary_account_id"`
	ActivePrimaryAccountID         *int64                    `json:"active_primary_account_id"`
	ActivePrimarySource            string                    `json:"active_primary_source"`
	ActivePrimaryReason            string                    `json:"active_primary_reason"`
	ActivePrimarySwitchedAt        *time.Time                `json:"active_primary_switched_at"`
	PrimaryFailoverCooldownSeconds int                       `json:"primary_failover_cooldown_seconds"`
	PrimaryAllowManualAutoReplace  bool                      `json:"primary_allow_manual_auto_replace"`
	RecentScheduleTrace            *dto.UsageScheduleTrace   `json:"recent_schedule_trace,omitempty"`
	RecentScheduleRequestID        string                    `json:"recent_schedule_request_id,omitempty"`
	RecentScheduleCreatedAt        *time.Time                `json:"recent_schedule_created_at,omitempty"`
	AccountCount                   int                       `json:"account_count"`
	KnownAccountCount              int                       `json:"known_account_count"`
	UnknownAccountCount            int                       `json:"unknown_account_count"`
	Items                          []groupAccountQualityItem `json:"items"`
}

type optionalLimitField struct {
	set   bool
	value *float64
}

func (f *optionalLimitField) UnmarshalJSON(data []byte) error {
	f.set = true

	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		f.value = nil
		return nil
	}

	var number float64
	if err := json.Unmarshal(trimmed, &number); err == nil {
		f.value = &number
		return nil
	}

	var text string
	if err := json.Unmarshal(trimmed, &text); err == nil {
		text = strings.TrimSpace(text)
		if text == "" {
			f.value = nil
			return nil
		}
		number, err = strconv.ParseFloat(text, 64)
		if err != nil {
			return fmt.Errorf("invalid numeric limit value %q: %w", text, err)
		}
		f.value = &number
		return nil
	}

	return fmt.Errorf("invalid limit value: %s", string(trimmed))
}

func (f optionalLimitField) ToServiceInput() *float64 {
	if !f.set {
		return nil
	}
	if f.value != nil {
		return f.value
	}
	zero := 0.0
	return &zero
}

// NewGroupHandler creates a new admin group handler
func NewGroupHandler(adminService service.AdminService, dashboardService *service.DashboardService, groupCapacityService *service.GroupCapacityService, usageService *service.UsageService) *GroupHandler {
	return &GroupHandler{
		adminService:         adminService,
		dashboardService:     dashboardService,
		groupCapacityService: groupCapacityService,
		usageService:         usageService,
	}
}

func (h *GroupHandler) latestGroupScheduleTrace(ctx *gin.Context, groupID int64) (*dto.UsageScheduleTrace, string, *time.Time) {
	if h == nil || h.usageService == nil || groupID <= 0 {
		return nil, "", nil
	}
	logs, _, err := h.usageService.ListWithFilters(ctx.Request.Context(), pagination.PaginationParams{
		Page:      1,
		PageSize:  1,
		SortBy:    "created_at",
		SortOrder: "desc",
	}, usagestats.UsageLogFilters{
		GroupID: groupID,
	})
	if err != nil || len(logs) == 0 {
		return nil, "", nil
	}
	log := logs[0]
	return dto.UsageLogFromServiceAdmin(&log).ScheduleTrace, log.RequestID, &log.CreatedAt
}

// CreateGroupRequest represents create group request
type CreateGroupRequest struct {
	Name             string             `json:"name" binding:"required"`
	Description      string             `json:"description"`
	Platform         string             `json:"platform" binding:"omitempty,oneof=anthropic openai gemini antigravity"`
	RateMultiplier   float64            `json:"rate_multiplier"`
	IsExclusive      bool               `json:"is_exclusive"`
	SubscriptionType string             `json:"subscription_type" binding:"omitempty,oneof=standard subscription"`
	DailyLimitUSD    optionalLimitField `json:"daily_limit_usd"`
	WeeklyLimitUSD   optionalLimitField `json:"weekly_limit_usd"`
	MonthlyLimitUSD  optionalLimitField `json:"monthly_limit_usd"`
	// 图片生成计费配置（antigravity 和 gemini 平台使用，负数表示清除配置）
	AllowImageGeneration            bool     `json:"allow_image_generation"`
	ImageRateIndependent            bool     `json:"image_rate_independent"`
	ImageRateMultiplier             *float64 `json:"image_rate_multiplier"`
	ImagePrice1K                    *float64 `json:"image_price_1k"`
	ImagePrice2K                    *float64 `json:"image_price_2k"`
	ImagePrice4K                    *float64 `json:"image_price_4k"`
	ClaudeCodeOnly                  bool     `json:"claude_code_only"`
	FallbackGroupID                 *int64   `json:"fallback_group_id"`
	FallbackGroupIDOnInvalidRequest *int64   `json:"fallback_group_id_on_invalid_request"`
	// 模型路由配置（仅 anthropic 平台使用）
	ModelRouting        map[string][]int64 `json:"model_routing"`
	ModelRoutingEnabled bool               `json:"model_routing_enabled"`
	MCPXMLInject        *bool              `json:"mcp_xml_inject"`
	// 支持的模型系列（仅 antigravity 平台使用）
	SupportedModelScopes []string `json:"supported_model_scopes"`
	// OpenAI Messages 调度配置（仅 openai 平台使用）
	AllowMessagesDispatch       bool                                      `json:"allow_messages_dispatch"`
	RequireOAuthOnly            bool                                      `json:"require_oauth_only"`
	RequirePrivacySet           bool                                      `json:"require_privacy_set"`
	DefaultMappedModel          string                                    `json:"default_mapped_model"`
	MessagesDispatchModelConfig service.OpenAIMessagesDispatchModelConfig `json:"messages_dispatch_model_config"`
	ModelsListConfig            service.GroupModelsListConfig             `json:"models_list_config"`
	// 分组 RPM 上限（0 = 不限制）
	RPMLimit int `json:"rpm_limit"`
	// 从指定分组复制账号（创建后自动绑定）
	CopyAccountsFromGroupIDs []int64 `json:"copy_accounts_from_group_ids"`
}

// UpdateGroupRequest represents update group request
type UpdateGroupRequest struct {
	Name             string             `json:"name"`
	Description      *string            `json:"description"`
	Platform         string             `json:"platform" binding:"omitempty,oneof=anthropic openai gemini antigravity"`
	RateMultiplier   *float64           `json:"rate_multiplier"`
	IsExclusive      *bool              `json:"is_exclusive"`
	Status           string             `json:"status" binding:"omitempty,oneof=active inactive"`
	SubscriptionType string             `json:"subscription_type" binding:"omitempty,oneof=standard subscription"`
	DailyLimitUSD    optionalLimitField `json:"daily_limit_usd"`
	WeeklyLimitUSD   optionalLimitField `json:"weekly_limit_usd"`
	MonthlyLimitUSD  optionalLimitField `json:"monthly_limit_usd"`
	// 图片生成计费配置（antigravity 和 gemini 平台使用，负数表示清除配置）
	AllowImageGeneration            *bool    `json:"allow_image_generation"`
	ImageRateIndependent            *bool    `json:"image_rate_independent"`
	ImageRateMultiplier             *float64 `json:"image_rate_multiplier"`
	ImagePrice1K                    *float64 `json:"image_price_1k"`
	ImagePrice2K                    *float64 `json:"image_price_2k"`
	ImagePrice4K                    *float64 `json:"image_price_4k"`
	ClaudeCodeOnly                  *bool    `json:"claude_code_only"`
	FallbackGroupID                 *int64   `json:"fallback_group_id"`
	FallbackGroupIDOnInvalidRequest *int64   `json:"fallback_group_id_on_invalid_request"`
	// 模型路由配置（仅 anthropic 平台使用）
	ModelRouting        map[string][]int64 `json:"model_routing"`
	ModelRoutingEnabled *bool              `json:"model_routing_enabled"`
	MCPXMLInject        *bool              `json:"mcp_xml_inject"`
	// 支持的模型系列（仅 antigravity 平台使用）
	SupportedModelScopes *[]string `json:"supported_model_scopes"`
	// OpenAI Messages 调度配置（仅 openai 平台使用）
	AllowMessagesDispatch       *bool                                      `json:"allow_messages_dispatch"`
	RequireOAuthOnly            *bool                                      `json:"require_oauth_only"`
	RequirePrivacySet           *bool                                      `json:"require_privacy_set"`
	DefaultMappedModel          *string                                    `json:"default_mapped_model"`
	MessagesDispatchModelConfig *service.OpenAIMessagesDispatchModelConfig `json:"messages_dispatch_model_config"`
	ModelsListConfig            *service.GroupModelsListConfig             `json:"models_list_config"`
	// 分组 RPM 上限（0 = 不限制）；nil 表示未提供不改动
	RPMLimit *int `json:"rpm_limit"`
	// 从指定分组复制账号（同步操作：先清空当前分组的账号绑定，再绑定源分组的账号）
	CopyAccountsFromGroupIDs []int64 `json:"copy_accounts_from_group_ids"`
}

type UpdateGroupPrimaryAccountRequest struct {
	PrimaryAccountMode             string `json:"primary_account_mode" binding:"required,oneof=off auto manual"`
	ManualPrimaryAccountID         *int64 `json:"manual_primary_account_id"`
	PrimaryFailoverCooldownSeconds *int   `json:"primary_failover_cooldown_seconds"`
	PrimaryAllowManualAutoReplace  *bool  `json:"primary_allow_manual_auto_replace"`
}

// List handles listing all groups with pagination
// GET /api/v1/admin/groups
func (h *GroupHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	platform := c.Query("platform")
	status := c.Query("status")
	search := c.Query("search")
	// 标准化和验证 search 参数
	search = strings.TrimSpace(search)
	if len(search) > 100 {
		search = search[:100]
	}
	isExclusiveStr := c.Query("is_exclusive")
	sortBy := c.DefaultQuery("sort_by", "sort_order")
	sortOrder := c.DefaultQuery("sort_order", "asc")

	var isExclusive *bool
	if isExclusiveStr != "" {
		val := isExclusiveStr == "true"
		isExclusive = &val
	}

	groups, total, err := h.adminService.ListGroups(c.Request.Context(), page, pageSize, platform, status, search, isExclusive, sortBy, sortOrder)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	outGroups := make([]dto.AdminGroup, 0, len(groups))
	for i := range groups {
		outGroups = append(outGroups, *dto.GroupFromServiceAdmin(&groups[i]))
	}
	response.Paginated(c, outGroups, total, page, pageSize)
}

// GetAll handles getting all active groups without pagination.
// Pass ?include_inactive=true to also include disabled groups (used by the
// API Key group filter, which needs to surface groups that still have API keys
// bound to them even after the group is disabled).
// GET /api/v1/admin/groups/all
func (h *GroupHandler) GetAll(c *gin.Context) {
	platform := c.Query("platform")
	includeInactive := c.Query("include_inactive") == "true"

	var groups []service.Group
	var err error

	if includeInactive {
		groups, err = h.adminService.GetAllGroupsIncludingInactive(c.Request.Context())
	} else if platform != "" {
		groups, err = h.adminService.GetAllGroupsByPlatform(c.Request.Context(), platform)
	} else {
		groups, err = h.adminService.GetAllGroups(c.Request.Context())
	}

	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	outGroups := make([]dto.AdminGroup, 0, len(groups))
	for i := range groups {
		outGroups = append(outGroups, *dto.GroupFromServiceAdmin(&groups[i]))
	}
	response.Success(c, outGroups)
}

// GetByID handles getting a group by ID
// GET /api/v1/admin/groups/:id
func (h *GroupHandler) GetByID(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	group, err := h.adminService.GetGroup(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.GroupFromServiceAdmin(group))
}

// GetAccountQuality handles getting current quality snapshots for all accounts in a group.
// GET /api/v1/admin/groups/:id/account-quality
func (h *GroupHandler) GetAccountQuality(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	group, err := h.adminService.GetGroup(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	const pageSize = 1000
	page := 1
	var (
		accounts []service.Account
		total    int64
	)
	for {
		batch, batchTotal, err := h.adminService.ListAccounts(c.Request.Context(), page, pageSize, "", "", "", "", groupID, "", "priority", "asc")
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if page == 1 {
			total = batchTotal
		}
		accounts = append(accounts, batch...)
		if int64(len(accounts)) >= total || len(batch) == 0 {
			break
		}
		page++
	}

	accountIDs := make([]int64, 0, len(accounts))
	for i := range accounts {
		accountIDs = append(accountIDs, accounts[i].ID)
	}

	reader := h.accountQualityReader
	if reader == nil {
		reader = service.DefaultAccountQualityReader()
	}

	snapshots := map[int64]*service.AccountQualitySnapshot{}
	if reader != nil && len(accountIDs) > 0 {
		snapshots, err = reader.GetSnapshotsByAccountIDs(c.Request.Context(), accountIDs)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}

	items := make([]groupAccountQualityItem, 0, len(accounts))
	knownCount := 0
	for i := range accounts {
		account := &accounts[i]
		snapshot := snapshots[account.ID]
		score, known, breakdown, components := service.DescribeAccountQualitySnapshot(snapshot)
		if known {
			knownCount++
		}
		item := groupAccountQualityItem{
			AccountID:             account.ID,
			AccountName:           account.Name,
			Platform:              account.Platform,
			AccountType:           account.Type,
			Status:                account.Status,
			Schedulable:           account.Schedulable,
			Priority:              account.Priority,
			Concurrency:           account.Concurrency,
			GroupIDs:              append([]int64(nil), account.GroupIDs...),
			LastUsedAt:            account.LastUsedAt,
			UpdatedAt:             account.UpdatedAt,
			QualityKnown:          known,
			QualityScore:          score,
			ScoreBreakdown:        breakdown,
			NeutralBase:           components.NeutralBase,
			SuccessComponent:      components.SuccessComponent,
			TTFT5sComponent:       components.TTFT5sComponent,
			TTFT10sComponent:      components.TTFT10sComponent,
			FastBonus:             components.FastBonus,
			SlowPenalty:           components.SlowPenalty,
			ErrorPenalty:          components.ErrorPenalty,
			SampleConfidence:      components.SampleConfidence,
			BaseQualityScore:      components.BaseQualityScore,
			EffectiveQualityScore: components.EffectiveQualityScore,
			TransientPenalty:      components.TransientPenalty,
			RecoveryCredit:        components.RecoveryCredit,
			AppliedPenalty:        components.AppliedPenalty,
			IsManualPrimary:       group.ManualPrimaryAccountID != nil && *group.ManualPrimaryAccountID == account.ID,
			IsActivePrimary:       group.ActivePrimaryAccountID != nil && *group.ActivePrimaryAccountID == account.ID,
			IsPoolMode:            account.IsPoolMode(),
			PoolModeRetryCount:    account.GetPoolModeRetryCount(),
			PrimaryEligible:       account.Schedulable && account.Status == service.StatusActive,
		}
		if item.IsPoolMode {
			item.PoolModeRetryStatusCodes = account.GetPoolModeRetryStatusCodes()
			if len(item.PoolModeRetryStatusCodes) == 0 {
				item.PoolModeRetryStatusCodes = []int{401, 403, 429}
			}
		}
		if !item.PrimaryEligible {
			if !account.Schedulable {
				item.PrimaryIneligibleReason = "unschedulable"
			} else if account.Status != service.StatusActive {
				item.PrimaryIneligibleReason = account.Status
			}
		}
		if snapshot != nil {
			item.WindowStart = &snapshot.WindowStart
			item.WindowEnd = &snapshot.WindowEnd
			item.SnapshotUpdatedAt = &snapshot.UpdatedAt
			item.TotalRequests = snapshot.TotalRequests
			item.SuccessRequests = snapshot.SuccessRequests
			item.FailureRequests = snapshot.FailureRequests
			item.RecentSuccessRate = snapshot.RecentSuccessRate
			item.ErrorRate = snapshot.ErrorRate
			item.TTFTSampleCount = snapshot.TTFTSampleCount
			item.TTFTLE5sCount = snapshot.TTFTLE5sCount
			item.TTFTLE10sCount = snapshot.TTFTLE10sCount
			item.TTFTGT10sCount = snapshot.TTFTGT10sCount
			item.TTFTGT20sCount = snapshot.TTFTGT20sCount
			item.TTFTGT40sCount = snapshot.TTFTGT40sCount
			item.TTFTLE5sRate = snapshot.TTFTLE5sRate
			item.TTFTLE10sRate = snapshot.TTFTLE10sRate
			item.TTFTGT10sRate = snapshot.TTFTGT10sRate
			item.TTFTGT20sRate = snapshot.TTFTGT20sRate
			item.TTFTGT40sRate = snapshot.TTFTGT40sRate
			item.SlowStreak = snapshot.SlowStreak
			item.ErrorStreak = snapshot.ErrorStreak
			item.RecoverySuccessStreak = snapshot.RecoverySuccessStreak
			item.RecoveryFastStreak = snapshot.RecoveryFastStreak
			item.AuxiliaryTotalRequests = snapshot.AuxiliaryTotalRequests
			item.AuxiliarySuccessRequests = snapshot.AuxiliarySuccessRequests
			item.AuxiliaryFailureRequests = snapshot.AuxiliaryFailureRequests
			item.AuxiliaryTTFTSampleCount = snapshot.AuxiliaryTTFTSampleCount
			item.AuxiliaryTTFTLE5sCount = snapshot.AuxiliaryTTFTLE5sCount
			item.AuxiliaryTTFTLE10sCount = snapshot.AuxiliaryTTFTLE10sCount
			item.AuxiliaryTTFTGT10sCount = snapshot.AuxiliaryTTFTGT10sCount
			item.AuxiliaryTTFTGT20sCount = snapshot.AuxiliaryTTFTGT20sCount
			item.AuxiliaryTTFTGT40sCount = snapshot.AuxiliaryTTFTGT40sCount
			item.AuxiliaryWeight = snapshot.AuxiliaryWeight
		}
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		a := &items[i]
		b := &items[j]
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		if cmp := compareQualityItemsForSchedule(a, b); cmp != 0 {
			return cmp < 0
		}
		switch {
		case a.LastUsedAt == nil && b.LastUsedAt != nil:
			return true
		case a.LastUsedAt != nil && b.LastUsedAt == nil:
			return false
		case a.LastUsedAt == nil && b.LastUsedAt == nil:
			return a.AccountID < b.AccountID
		default:
			if !a.LastUsedAt.Equal(*b.LastUsedAt) {
				return a.LastUsedAt.Before(*b.LastUsedAt)
			}
			return a.AccountID < b.AccountID
		}
	})

	for i := range items {
		items[i].ScheduleRank = i + 1
		items[i].ScheduleSortKey = buildScheduleSortKey(&items[i])
	}

	recentScheduleTrace, recentScheduleRequestID, recentScheduleCreatedAt := h.latestGroupScheduleTrace(c, group.ID)

	response.Success(c, groupAccountQualityResponse{
		GroupID:                        group.ID,
		GroupName:                      group.Name,
		GroupPlatform:                  group.Platform,
		PrimaryAccountMode:             group.PrimaryAccountMode,
		ManualPrimaryAccountID:         group.ManualPrimaryAccountID,
		ActivePrimaryAccountID:         group.ActivePrimaryAccountID,
		ActivePrimarySource:            group.ActivePrimarySource,
		ActivePrimaryReason:            group.ActivePrimaryReason,
		ActivePrimarySwitchedAt:        group.ActivePrimarySwitchedAt,
		PrimaryFailoverCooldownSeconds: group.PrimaryFailoverCooldownSeconds,
		PrimaryAllowManualAutoReplace:  group.PrimaryAllowManualAutoReplace,
		RecentScheduleTrace:            recentScheduleTrace,
		RecentScheduleRequestID:        recentScheduleRequestID,
		RecentScheduleCreatedAt:        recentScheduleCreatedAt,
		AccountCount:                   len(items),
		KnownAccountCount:              knownCount,
		UnknownAccountCount:            len(items) - knownCount,
		Items:                          items,
	})
}

func (h *GroupHandler) UpdatePrimaryAccount(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	var req UpdateGroupPrimaryAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	group, err := h.adminService.UpdateGroupPrimaryAccount(c.Request.Context(), groupID, &service.UpdateGroupPrimaryAccountInput{
		PrimaryAccountMode:             req.PrimaryAccountMode,
		ManualPrimaryAccountID:         req.ManualPrimaryAccountID,
		PrimaryFailoverCooldownSeconds: req.PrimaryFailoverCooldownSeconds,
		PrimaryAllowManualAutoReplace:  req.PrimaryAllowManualAutoReplace,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.GroupFromServiceAdmin(group))
}

func compareQualityItemsForSchedule(a, b *groupAccountQualityItem) int {
	if a == nil || b == nil {
		return 0
	}
	if a.EffectiveQualityScore > b.EffectiveQualityScore {
		return -1
	}
	if a.EffectiveQualityScore < b.EffectiveQualityScore {
		return 1
	}
	if a.AppliedPenalty < b.AppliedPenalty {
		return -1
	}
	if a.AppliedPenalty > b.AppliedPenalty {
		return 1
	}
	aRecovery := a.RecoveryCredit + minFloat64(float64(a.RecoverySuccessStreak)*0.01, 0.04) + minFloat64(float64(a.RecoveryFastStreak)*0.02, 0.08)
	bRecovery := b.RecoveryCredit + minFloat64(float64(b.RecoverySuccessStreak)*0.01, 0.04) + minFloat64(float64(b.RecoveryFastStreak)*0.02, 0.08)
	if aRecovery > bRecovery {
		return -1
	}
	if aRecovery < bRecovery {
		return 1
	}
	if a.QualityKnown && !b.QualityKnown {
		return -1
	}
	if !a.QualityKnown && b.QualityKnown {
		return 1
	}
	return 0
}

func buildScheduleSortKey(item *groupAccountQualityItem) string {
	if item == nil {
		return ""
	}
	lastUsed := "never"
	if item.LastUsedAt != nil {
		lastUsed = item.LastUsedAt.UTC().Format(time.RFC3339)
	}
	return fmt.Sprintf(
		"priority=%d -> quality=%.4f -> penalty=%.4f -> recovery=%.4f -> last_used=%s",
		item.Priority,
		item.EffectiveQualityScore,
		item.AppliedPenalty,
		item.RecoveryCredit+minFloat64(float64(item.RecoverySuccessStreak)*0.01, 0.04)+minFloat64(float64(item.RecoveryFastStreak)*0.02, 0.08),
		lastUsed,
	)
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// GetModelsListCandidates handles getting candidate model IDs for custom /v1/models list.
// GET /api/v1/admin/groups/:id/models-list-candidates
func (h *GroupHandler) GetModelsListCandidates(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || groupID < 0 {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	models, err := h.adminService.GetGroupModelsListCandidates(
		c.Request.Context(),
		groupID,
		c.Query("platform"),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"models": models})
}

// Create handles creating a new group
// POST /api/v1/admin/groups
func (h *GroupHandler) Create(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	group, err := h.adminService.CreateGroup(c.Request.Context(), &service.CreateGroupInput{
		Name:                            req.Name,
		Description:                     req.Description,
		Platform:                        req.Platform,
		RateMultiplier:                  req.RateMultiplier,
		IsExclusive:                     req.IsExclusive,
		SubscriptionType:                req.SubscriptionType,
		DailyLimitUSD:                   req.DailyLimitUSD.ToServiceInput(),
		WeeklyLimitUSD:                  req.WeeklyLimitUSD.ToServiceInput(),
		MonthlyLimitUSD:                 req.MonthlyLimitUSD.ToServiceInput(),
		AllowImageGeneration:            req.AllowImageGeneration,
		ImageRateIndependent:            req.ImageRateIndependent,
		ImageRateMultiplier:             req.ImageRateMultiplier,
		ImagePrice1K:                    req.ImagePrice1K,
		ImagePrice2K:                    req.ImagePrice2K,
		ImagePrice4K:                    req.ImagePrice4K,
		ClaudeCodeOnly:                  req.ClaudeCodeOnly,
		FallbackGroupID:                 req.FallbackGroupID,
		FallbackGroupIDOnInvalidRequest: req.FallbackGroupIDOnInvalidRequest,
		ModelRouting:                    req.ModelRouting,
		ModelRoutingEnabled:             req.ModelRoutingEnabled,
		MCPXMLInject:                    req.MCPXMLInject,
		SupportedModelScopes:            req.SupportedModelScopes,
		AllowMessagesDispatch:           req.AllowMessagesDispatch,
		RequireOAuthOnly:                req.RequireOAuthOnly,
		RequirePrivacySet:               req.RequirePrivacySet,
		DefaultMappedModel:              req.DefaultMappedModel,
		MessagesDispatchModelConfig:     req.MessagesDispatchModelConfig,
		ModelsListConfig:                req.ModelsListConfig,
		RPMLimit:                        req.RPMLimit,
		CopyAccountsFromGroupIDs:        req.CopyAccountsFromGroupIDs,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.GroupFromServiceAdmin(group))
}

// Update handles updating a group
// PUT /api/v1/admin/groups/:id
func (h *GroupHandler) Update(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	group, err := h.adminService.UpdateGroup(c.Request.Context(), groupID, &service.UpdateGroupInput{
		Name:                            req.Name,
		Description:                     req.Description,
		Platform:                        req.Platform,
		RateMultiplier:                  req.RateMultiplier,
		IsExclusive:                     req.IsExclusive,
		Status:                          req.Status,
		SubscriptionType:                req.SubscriptionType,
		DailyLimitUSD:                   req.DailyLimitUSD.ToServiceInput(),
		WeeklyLimitUSD:                  req.WeeklyLimitUSD.ToServiceInput(),
		MonthlyLimitUSD:                 req.MonthlyLimitUSD.ToServiceInput(),
		AllowImageGeneration:            req.AllowImageGeneration,
		ImageRateIndependent:            req.ImageRateIndependent,
		ImageRateMultiplier:             req.ImageRateMultiplier,
		ImagePrice1K:                    req.ImagePrice1K,
		ImagePrice2K:                    req.ImagePrice2K,
		ImagePrice4K:                    req.ImagePrice4K,
		ClaudeCodeOnly:                  req.ClaudeCodeOnly,
		FallbackGroupID:                 req.FallbackGroupID,
		FallbackGroupIDOnInvalidRequest: req.FallbackGroupIDOnInvalidRequest,
		ModelRouting:                    req.ModelRouting,
		ModelRoutingEnabled:             req.ModelRoutingEnabled,
		MCPXMLInject:                    req.MCPXMLInject,
		SupportedModelScopes:            req.SupportedModelScopes,
		AllowMessagesDispatch:           req.AllowMessagesDispatch,
		RequireOAuthOnly:                req.RequireOAuthOnly,
		RequirePrivacySet:               req.RequirePrivacySet,
		DefaultMappedModel:              req.DefaultMappedModel,
		MessagesDispatchModelConfig:     req.MessagesDispatchModelConfig,
		ModelsListConfig:                req.ModelsListConfig,
		RPMLimit:                        req.RPMLimit,
		CopyAccountsFromGroupIDs:        req.CopyAccountsFromGroupIDs,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.GroupFromServiceAdmin(group))
}

// Delete handles deleting a group
// DELETE /api/v1/admin/groups/:id
func (h *GroupHandler) Delete(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	err = h.adminService.DeleteGroup(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Group deleted successfully"})
}

// GetStats handles getting group statistics
// GET /api/v1/admin/groups/:id/stats
func (h *GroupHandler) GetStats(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	// Return mock data for now
	response.Success(c, gin.H{
		"total_api_keys":  0,
		"active_api_keys": 0,
		"total_requests":  0,
		"total_cost":      0.0,
	})
	_ = groupID // TODO: implement actual stats
}

// GetUsageSummary returns today's and cumulative cost for all groups.
// GET /api/v1/admin/groups/usage-summary?timezone=Asia/Shanghai
func (h *GroupHandler) GetUsageSummary(c *gin.Context) {
	userTZ := c.Query("timezone")
	now := timezone.NowInUserLocation(userTZ)
	todayStart := timezone.StartOfDayInUserLocation(now, userTZ)

	results, err := h.dashboardService.GetGroupUsageSummary(c.Request.Context(), todayStart)
	if err != nil {
		response.Error(c, 500, "Failed to get group usage summary")
		return
	}

	response.Success(c, results)
}

// GetCapacitySummary returns aggregated capacity (concurrency/sessions/RPM) for all active groups.
// GET /api/v1/admin/groups/capacity-summary
func (h *GroupHandler) GetCapacitySummary(c *gin.Context) {
	results, err := h.groupCapacityService.GetAllGroupCapacity(c.Request.Context())
	if err != nil {
		response.Error(c, 500, "Failed to get group capacity summary")
		return
	}
	response.Success(c, results)
}

// GetGroupAPIKeys handles getting API keys in a group
// GET /api/v1/admin/groups/:id/api-keys
func (h *GroupHandler) GetGroupAPIKeys(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	page, pageSize := response.ParsePagination(c)

	keys, total, err := h.adminService.GetGroupAPIKeys(c.Request.Context(), groupID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	outKeys := make([]dto.APIKey, 0, len(keys))
	for i := range keys {
		outKeys = append(outKeys, *dto.APIKeyFromService(&keys[i]))
	}
	response.Paginated(c, outKeys, total, page, pageSize)
}

// GetGroupRateMultipliers handles getting rate multipliers for users in a group
// GET /api/v1/admin/groups/:id/rate-multipliers
func (h *GroupHandler) GetGroupRateMultipliers(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	entries, err := h.adminService.GetGroupRateMultipliers(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if entries == nil {
		entries = []service.UserGroupRateEntry{}
	}
	response.Success(c, entries)
}

// ClearGroupRateMultipliers handles clearing all rate multipliers for a group
// DELETE /api/v1/admin/groups/:id/rate-multipliers
func (h *GroupHandler) ClearGroupRateMultipliers(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	if err := h.adminService.ClearGroupRateMultipliers(c.Request.Context(), groupID); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Rate multipliers cleared successfully"})
}

// BatchSetGroupRateMultipliersRequest represents batch set rate multipliers request
type BatchSetGroupRateMultipliersRequest struct {
	Entries []service.GroupRateMultiplierInput `json:"entries" binding:"required"`
}

// BatchSetGroupRateMultipliers handles batch setting rate multipliers for a group
// PUT /api/v1/admin/groups/:id/rate-multipliers
func (h *GroupHandler) BatchSetGroupRateMultipliers(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	var req BatchSetGroupRateMultipliersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.adminService.BatchSetGroupRateMultipliers(c.Request.Context(), groupID, req.Entries); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Rate multipliers updated successfully"})
}

// BatchSetGroupRPMOverridesRequest represents batch set rpm_override request
type BatchSetGroupRPMOverridesRequest struct {
	Entries []service.GroupRPMOverrideInput `json:"entries" binding:"required"`
}

// BatchSetGroupRPMOverrides handles batch setting rpm_override for users in a group
// PUT /api/v1/admin/groups/:id/rpm-overrides
func (h *GroupHandler) BatchSetGroupRPMOverrides(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	var req BatchSetGroupRPMOverridesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.adminService.BatchSetGroupRPMOverrides(c.Request.Context(), groupID, req.Entries); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "RPM overrides updated successfully"})
}

// ClearGroupRPMOverrides handles clearing all rpm_override for a group
// DELETE /api/v1/admin/groups/:id/rpm-overrides
func (h *GroupHandler) ClearGroupRPMOverrides(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid group ID")
		return
	}

	if err := h.adminService.ClearGroupRPMOverrides(c.Request.Context(), groupID); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "RPM overrides cleared successfully"})
}

// UpdateSortOrderRequest represents the request to update group sort orders
type UpdateSortOrderRequest struct {
	Updates []struct {
		ID        int64 `json:"id" binding:"required"`
		SortOrder int   `json:"sort_order"`
	} `json:"updates" binding:"required,min=1"`
}

// UpdateSortOrder handles updating group sort orders
// PUT /api/v1/admin/groups/sort-order
func (h *GroupHandler) UpdateSortOrder(c *gin.Context) {
	var req UpdateSortOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	updates := make([]service.GroupSortOrderUpdate, 0, len(req.Updates))
	for _, u := range req.Updates {
		updates = append(updates, service.GroupSortOrderUpdate{
			ID:        u.ID,
			SortOrder: u.SortOrder,
		})
	}

	if err := h.adminService.UpdateGroupSortOrders(c.Request.Context(), updates); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Sort order updated successfully"})
}
