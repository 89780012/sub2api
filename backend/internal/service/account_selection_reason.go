package service

import (
	"context"
	"strings"
	"time"
)

const (
	accountSelectionRuleMonitorQualityPriorityLoadLRU = "monitor_quality_priority_load_lru"
	accountSelectionRuleMonitorQualityPriorityLRU     = "monitor_quality_priority_lru"
	accountSelectionRuleSticky                        = "sticky"
	accountSelectionRuleAdvancedOpenAI                = "openai_advanced_scheduler"
	accountSelectionRuleLegacy                        = "legacy_order"
)

type accountSelectionReasonInput struct {
	Layer          string
	Rule           string
	Account        *Account
	LoadInfo       *AccountLoadInfo
	MonitorScorer  *ChannelMonitorQualityScorer
	Acquired       bool
	WaitPlan       *AccountWaitPlan
	CandidateCount int
	TopK           int
	LoadSkew       float64
	TieBreakers    []string
	Notes          map[string]any
}

func buildAccountSelectionReason(ctx context.Context, in accountSelectionReasonInput) map[string]any {
	if in.Account == nil {
		return nil
	}
	rule := strings.TrimSpace(in.Rule)
	if rule == "" {
		rule = accountSelectionRuleMonitorQualityPriorityLoadLRU
	}
	layer := strings.TrimSpace(in.Layer)
	if layer == "" {
		layer = "unknown"
	}

	reason := map[string]any{
		"summary":      accountSelectionSummary(layer, rule, in.Acquired, in.WaitPlan != nil),
		"layer":        layer,
		"rule":         rule,
		"account_id":   in.Account.ID,
		"account_name": in.Account.Name,
		"platform":     in.Account.Platform,
		"provider":     monitorProviderForAccount(in.Account),
		"endpoint":     normalizeMonitorQualityEndpoint(monitorEndpointForAccount(in.Account)),
		"priority":     in.Account.Priority,
		"acquired":     in.Acquired,
	}
	if len(in.TieBreakers) > 0 {
		reason["tie_breakers"] = in.TieBreakers
	}
	if in.CandidateCount > 0 {
		reason["candidate_count"] = in.CandidateCount
	}
	if in.TopK > 0 {
		reason["top_k"] = in.TopK
	}
	if in.LoadSkew > 0 {
		reason["load_skew"] = in.LoadSkew
	}
	if in.LoadInfo != nil {
		reason["load"] = map[string]any{
			"load_rate":           in.LoadInfo.LoadRate,
			"current_concurrency": in.LoadInfo.CurrentConcurrency,
			"waiting_count":       in.LoadInfo.WaitingCount,
		}
	}
	if in.WaitPlan != nil {
		reason["wait_plan"] = map[string]any{
			"account_id":       in.WaitPlan.AccountID,
			"max_concurrency":  in.WaitPlan.MaxConcurrency,
			"timeout_ms":       int64(in.WaitPlan.Timeout / time.Millisecond),
			"max_waiting":      in.WaitPlan.MaxWaiting,
			"selected_to_wait": true,
		}
	}
	if scoreReason := accountMonitorQualityReason(ctx, in.MonitorScorer, in.Account); scoreReason != nil {
		reason["monitor_quality"] = scoreReason
	}
	for k, v := range in.Notes {
		if strings.TrimSpace(k) == "" || v == nil {
			continue
		}
		reason[k] = v
	}
	return reason
}

func accountSelectionSummary(layer, rule string, acquired bool, hasWaitPlan bool) string {
	status := "selected"
	if hasWaitPlan && !acquired {
		status = "selected for wait queue"
	}
	switch rule {
	case accountSelectionRuleSticky:
		return status + " by sticky binding"
	case accountSelectionRuleAdvancedOpenAI:
		return status + " by OpenAI advanced scheduler"
	case accountSelectionRuleLegacy:
		return status + " by monitor quality, then priority/LRU legacy order"
	default:
		return status + " by monitor quality, then priority/load/LRU"
	}
}

func accountMonitorQualityReason(ctx context.Context, scorer *ChannelMonitorQualityScorer, account *Account) map[string]any {
	if account == nil {
		return nil
	}
	score := channelMonitorQualityScore{}
	if scorer != nil {
		score = scorer.ScoreAccount(ctx, account)
	}
	return map[string]any{
		"known":          score.Known,
		"order":          "checked_at_desc",
		"latest_first":   true,
		"recent_7":       monitorStatusSamplesReason(score.Recent7),
		"counts_3":       monitorStatusCountsReason(score.Counts3),
		"counts_5":       monitorStatusCountsReason(score.Counts5),
		"counts_7":       monitorStatusCountsReason(score.Counts7),
	}
}

func monitorStatusCountsReason(counts monitorStatusCounts) map[string]any {
	return map[string]any{
		"green":   counts.Green,
		"orange":  counts.Orange,
		"red":     counts.Red,
		"unknown": counts.Unknown,
	}
}

func monitorStatusSamplesReason(samples []monitorStatusSample) []map[string]any {
	if len(samples) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(samples))
	for _, sample := range samples {
		item := map[string]any{
			"status": sample.Status,
		}
		if !sample.CheckedAt.IsZero() {
			item["checked_at"] = sample.CheckedAt.Format(time.RFC3339Nano)
		}
		out = append(out, item)
	}
	return out
}
