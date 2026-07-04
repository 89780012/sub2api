package service

import (
	"fmt"
	"strings"
	"time"
)

const (
	BillingTypeBalance      int8 = 0 // 钱包余额
	BillingTypeSubscription int8 = 1 // 订阅套餐
)

type RequestType int16

const (
	RequestTypeUnknown RequestType = 0
	RequestTypeSync    RequestType = 1
	RequestTypeStream  RequestType = 2
	RequestTypeWSV2    RequestType = 3
)

func (t RequestType) IsValid() bool {
	switch t {
	case RequestTypeUnknown, RequestTypeSync, RequestTypeStream, RequestTypeWSV2:
		return true
	default:
		return false
	}
}

func (t RequestType) Normalize() RequestType {
	if t.IsValid() {
		return t
	}
	return RequestTypeUnknown
}

func (t RequestType) String() string {
	switch t.Normalize() {
	case RequestTypeSync:
		return "sync"
	case RequestTypeStream:
		return "stream"
	case RequestTypeWSV2:
		return "ws_v2"
	default:
		return "unknown"
	}
}

func RequestTypeFromInt16(v int16) RequestType {
	return RequestType(v).Normalize()
}

func ParseUsageRequestType(value string) (RequestType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "unknown":
		return RequestTypeUnknown, nil
	case "sync":
		return RequestTypeSync, nil
	case "stream":
		return RequestTypeStream, nil
	case "ws_v2":
		return RequestTypeWSV2, nil
	default:
		return RequestTypeUnknown, fmt.Errorf("invalid request_type, allowed values: unknown, sync, stream, ws_v2")
	}
}

func RequestTypeFromLegacy(stream bool, openAIWSMode bool) RequestType {
	if openAIWSMode {
		return RequestTypeWSV2
	}
	if stream {
		return RequestTypeStream
	}
	return RequestTypeSync
}

func ApplyLegacyRequestFields(requestType RequestType, fallbackStream bool, fallbackOpenAIWSMode bool) (stream bool, openAIWSMode bool) {
	switch requestType.Normalize() {
	case RequestTypeSync:
		return false, false
	case RequestTypeStream:
		return true, false
	case RequestTypeWSV2:
		return true, true
	default:
		return fallbackStream, fallbackOpenAIWSMode
	}
}

type UsageLog struct {
	ID        int64
	UserID    int64
	APIKeyID  int64
	AccountID int64
	RequestID string
	Model     string
	// RequestedModel is the client-requested model name recorded for stable user/admin display.
	// Empty should be treated as Model for backward compatibility with historical rows.
	RequestedModel string
	// UpstreamModel is the actual model sent to the upstream provider after mapping.
	// Nil means no mapping was applied (requested model was used as-is).
	UpstreamModel *string
	// ChannelID 渠道 ID
	ChannelID *int64
	// ModelMappingChain 模型映射链，如 "a→b→c"
	ModelMappingChain *string
	// BillingTier 计费层级标签（per_request/image 模式）
	BillingTier *string
	// BillingMode 计费模式：token/image
	BillingMode *string
	// ServiceTier records the OpenAI service tier used for billing, e.g. "priority" / "flex".
	ServiceTier *string
	// ReasoningEffort is the request's reasoning effort level.
	// OpenAI: "low" / "medium" / "high" / "xhigh"; Claude: "low" / "medium" / "high" / "max".
	// Nil means not provided / not applicable.
	ReasoningEffort *string
	// InboundEndpoint is the client-facing API endpoint path, e.g. /v1/chat/completions.
	InboundEndpoint *string
	// UpstreamEndpoint is the normalized upstream endpoint path, e.g. /v1/responses.
	UpstreamEndpoint *string

	GroupID        *int64
	SubscriptionID *int64

	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int

	CacheCreation5mTokens int `gorm:"column:cache_creation_5m_tokens"`
	CacheCreation1hTokens int `gorm:"column:cache_creation_1h_tokens"`

	ImageOutputTokens int
	ImageOutputCost   float64

	InputCost         float64
	OutputCost        float64
	CacheCreationCost float64
	CacheReadCost     float64
	TotalCost         float64
	ActualCost        float64
	RateMultiplier    float64
	// AccountRateMultiplier 账号计费倍率快照（nil 表示历史数据，按 1.0 处理）
	AccountRateMultiplier *float64
	// AccountStatsCost 账号统计定价预计算费用（nil = 使用默认公式 total_cost × account_rate_multiplier）
	AccountStatsCost *float64

	BillingType   int8
	RequestType   RequestType
	Stream        bool
	OpenAIWSMode  bool
	DurationMs    *int
	FirstTokenMs  *int
	UserAgent     *string
	IPAddress     *string
	ScheduleTrace *UsageScheduleTrace

	// Cache TTL Override 标记（管理员强制替换了缓存 TTL 计费）
	CacheTTLOverridden bool

	// 图片生成字段
	ImageCount         int
	ImageSize          *string
	ImageInputSize     *string
	ImageOutputSize    *string
	ImageSizeSource    *string
	ImageSizeBreakdown map[string]int
	MediaType          *string

	CreatedAt time.Time

	User         *User
	APIKey       *APIKey
	Account      *Account
	Group        *Group
	Subscription *UserSubscription
}

type UsageScheduleTrace struct {
	Layer                string                        `json:"layer,omitempty"`
	Reason               string                        `json:"reason,omitempty"`
	Platform             string                        `json:"platform,omitempty"`
	AccountType          string                        `json:"account_type,omitempty"`
	SelectedAccountID    int64                         `json:"selected_account_id,omitempty"`
	Priority             *int                          `json:"priority,omitempty"`
	PrimaryHit           bool                          `json:"primary_hit,omitempty"`
	PrimaryCandidateID   int64                         `json:"primary_candidate_id,omitempty"`
	PrimaryCandidateName string                        `json:"primary_candidate_name,omitempty"`
	PrimarySource        string                        `json:"primary_source,omitempty"`
	PrimaryBypassReason  string                        `json:"primary_bypass_reason,omitempty"`
	StickyHit            bool                          `json:"sticky_hit,omitempty"`
	StickyEscape         bool                          `json:"sticky_escape,omitempty"`
	StickyEscapeReason   string                        `json:"sticky_escape_reason,omitempty"`
	WaitPlan             bool                          `json:"wait_plan,omitempty"`
	CandidateCount       int                           `json:"candidate_count,omitempty"`
	TopK                 int                           `json:"top_k,omitempty"`
	QualityScore         *float64                      `json:"quality_score,omitempty"`
	RecentSuccessRate    *float64                      `json:"recent_success_rate,omitempty"`
	TTFTLE5sRate         *float64                      `json:"ttft_le_5s_rate,omitempty"`
	TTFTLE10sRate        *float64                      `json:"ttft_le_10s_rate,omitempty"`
	TTFTGT10sRate        *float64                      `json:"ttft_gt_10s_rate,omitempty"`
	TTFTGT20sRate        *float64                      `json:"ttft_gt_20s_rate,omitempty"`
	TTFTGT40sRate        *float64                      `json:"ttft_gt_40s_rate,omitempty"`
	TTFTSampleCount      int64                         `json:"ttft_sample_count,omitempty"`
	TotalRequests        int64                         `json:"total_requests,omitempty"`
	FailureRequests      int64                         `json:"failure_requests,omitempty"`
	ErrorRate            *float64                      `json:"error_rate,omitempty"`
	SampleConfidence     *float64                      `json:"sample_confidence,omitempty"`
	FastBonus            *float64                      `json:"fast_bonus,omitempty"`
	SlowPenalty          *float64                      `json:"slow_penalty,omitempty"`
	ErrorPenalty         *float64                      `json:"error_penalty,omitempty"`
	NeutralBase          *float64                      `json:"neutral_base,omitempty"`
	LoadRate             *float64                      `json:"load_rate,omitempty"`
	LoadSkew             *float64                      `json:"load_skew,omitempty"`
	ScoreFormula         string                        `json:"score_formula,omitempty"`
	Score                *float64                      `json:"score,omitempty"`
	Candidates           []UsageScheduleCandidateScore `json:"candidates,omitempty"`
}

type UsageScheduleCandidateScore struct {
	AccountID         int64      `json:"account_id,omitempty"`
	AccountName       string     `json:"account_name,omitempty"`
	Platform          string     `json:"platform,omitempty"`
	AccountType       string     `json:"account_type,omitempty"`
	Priority          int        `json:"priority,omitempty"`
	Selected          bool       `json:"selected,omitempty"`
	QualityKnown      bool       `json:"quality_known,omitempty"`
	QualityScore      *float64   `json:"quality_score,omitempty"`
	RecentSuccessRate *float64   `json:"recent_success_rate,omitempty"`
	TTFTLE5sRate      *float64   `json:"ttft_le_5s_rate,omitempty"`
	TTFTLE10sRate     *float64   `json:"ttft_le_10s_rate,omitempty"`
	TTFTGT10sRate     *float64   `json:"ttft_gt_10s_rate,omitempty"`
	TTFTGT20sRate     *float64   `json:"ttft_gt_20s_rate,omitempty"`
	TTFTGT40sRate     *float64   `json:"ttft_gt_40s_rate,omitempty"`
	TTFTSampleCount   int64      `json:"ttft_sample_count,omitempty"`
	TotalRequests     int64      `json:"total_requests,omitempty"`
	FailureRequests   int64      `json:"failure_requests,omitempty"`
	ErrorRate         *float64   `json:"error_rate,omitempty"`
	SampleConfidence  *float64   `json:"sample_confidence,omitempty"`
	FastBonus         *float64   `json:"fast_bonus,omitempty"`
	SlowPenalty       *float64   `json:"slow_penalty,omitempty"`
	ErrorPenalty      *float64   `json:"error_penalty,omitempty"`
	NeutralBase       *float64   `json:"neutral_base,omitempty"`
	LoadRate          *float64   `json:"load_rate,omitempty"`
	WaitingCount      *int       `json:"waiting_count,omitempty"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty"`
	ComputedScore     *float64   `json:"computed_score,omitempty"`
	ScoreBreakdown    string     `json:"score_breakdown,omitempty"`
	SelectionStage    string     `json:"selection_stage,omitempty"`
}

func (u *UsageLog) TotalTokens() int {
	return u.InputTokens + u.OutputTokens + u.CacheCreationTokens + u.CacheReadTokens
}

func (u *UsageLog) EffectiveRequestType() RequestType {
	if u == nil {
		return RequestTypeUnknown
	}
	if normalized := u.RequestType.Normalize(); normalized != RequestTypeUnknown {
		return normalized
	}
	return RequestTypeFromLegacy(u.Stream, u.OpenAIWSMode)
}

func (u *UsageLog) SyncRequestTypeAndLegacyFields() {
	if u == nil {
		return
	}
	requestType := u.EffectiveRequestType()
	u.RequestType = requestType
	u.Stream, u.OpenAIWSMode = ApplyLegacyRequestFields(requestType, u.Stream, u.OpenAIWSMode)
}
