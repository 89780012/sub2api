package service

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

type groupPrimaryCandidate struct {
	Account *Account
}

func (s *GatewayService) getGroupPrimaryAccount(ctx context.Context, groupID *int64) *Group {
	if s == nil || s.groupRepo == nil || groupID == nil || *groupID <= 0 {
		return nil
	}
	if group := s.groupFromContext(ctx, *groupID); group != nil {
		return group
	}
	group, err := s.groupRepo.GetByIDLite(ctx, *groupID)
	if err != nil || group == nil {
		return nil
	}
	return group
}

func (s *OpenAIGatewayService) getGroupPrimaryConfig(ctx context.Context, groupID *int64) *Group {
	if s == nil || s.groupRepo == nil || groupID == nil || *groupID <= 0 {
		return nil
	}
	group, err := s.groupRepo.GetByIDLite(ctx, *groupID)
	if err != nil || group == nil {
		return nil
	}
	return group
}

func (g *Group) currentPreferredPrimaryAccountID() int64 {
	if g == nil {
		return 0
	}
	mode := g.EffectivePrimaryAccountMode()
	switch mode {
	case GroupPrimaryAccountModeManual:
		if g.ManualPrimaryAccountID != nil && *g.ManualPrimaryAccountID > 0 {
			return *g.ManualPrimaryAccountID
		}
		if g.ActivePrimaryAccountID != nil && *g.ActivePrimaryAccountID > 0 {
			return *g.ActivePrimaryAccountID
		}
	case GroupPrimaryAccountModeAuto:
		if g.ActivePrimaryAccountID != nil && *g.ActivePrimaryAccountID > 0 {
			return *g.ActivePrimaryAccountID
		}
	}
	return 0
}

func (g *Group) shouldPersistPrimaryPromotion(promotedAccountID int64) bool {
	if g == nil || promotedAccountID <= 0 {
		return false
	}
	mode := g.EffectivePrimaryAccountMode()
	switch mode {
	case GroupPrimaryAccountModeAuto:
		return true
	case GroupPrimaryAccountModeManual:
		return g.PrimaryAllowManualAutoReplace
	default:
		return false
	}
}

func (g *Group) shouldBypassPrimaryCooldown(lastSwitchedAt time.Time) bool {
	if g == nil || g.PrimaryFailoverCooldownSeconds <= 0 {
		return false
	}
	return time.Since(lastSwitchedAt) < time.Duration(g.PrimaryFailoverCooldownSeconds)*time.Second
}

func persistGroupPrimarySelection(
	ctx context.Context,
	repo GroupRepository,
	group *Group,
	accountID int64,
	source string,
	reason string,
) {
	if repo == nil || group == nil || accountID <= 0 {
		return
	}
	currentID := int64(0)
	if group.ActivePrimaryAccountID != nil {
		currentID = *group.ActivePrimaryAccountID
	}
	if currentID == accountID && strings.TrimSpace(group.ActivePrimarySource) == source && strings.TrimSpace(group.ActivePrimaryReason) == reason {
		return
	}
	now := time.Now()
	group.ActivePrimaryAccountID = &accountID
	group.ActivePrimarySource = source
	group.ActivePrimaryReason = reason
	group.ActivePrimarySwitchedAt = &now
	if err := repo.Update(ctx, group); err != nil {
		slog.Warn("group primary account update failed",
			"group_id", group.ID,
			"account_id", accountID,
			"source", source,
			"reason", reason,
			"error", err,
		)
	}
}

func attachPrimaryTrace(trace *UsageScheduleTrace, group *Group, hit bool, bypassReason string) {
	if trace == nil || group == nil {
		return
	}
	trace.PrimaryHit = hit
	trace.PrimarySource = strings.TrimSpace(group.ActivePrimarySource)
	trace.PrimaryBypassReason = strings.TrimSpace(bypassReason)
}

func (s *GatewayService) persistPrimaryPromotionFromSelection(
	ctx context.Context,
	group *Group,
	selectedAccountID int64,
	hadExcludedFailures bool,
) {
	if s == nil || group == nil || !hadExcludedFailures || selectedAccountID <= 0 {
		return
	}
	if !group.shouldPersistPrimaryPromotion(selectedAccountID) {
		return
	}
	persistGroupPrimarySelection(ctx, s.groupRepo, group, selectedAccountID, "failover_promoted", "same_account_retry_exhausted")
}
