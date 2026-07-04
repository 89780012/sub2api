package service

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

const (
	groupPrimarySourceManualSet         = "manual_override"
	groupPrimarySourceFailoverCandidate = "failover_candidate"
	groupPrimarySourceFailoverPromoted  = "failover_promoted"

	groupPrimaryReasonManualSet          = "manual_set"
	groupPrimaryReasonRetryExhausted     = "same_account_retry_exhausted"
	groupPrimaryReasonFailoverStabilized = "failover_stabilized"
)

type groupPrimaryCandidate struct {
	Account *Account
}

type groupPrimarySelectionUpdater interface {
	Update(ctx context.Context, group *Group) error
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
		if g.PrimaryAllowManualAutoReplace &&
			g.ActivePrimaryAccountID != nil && *g.ActivePrimaryAccountID > 0 &&
			(strings.TrimSpace(g.ActivePrimarySource) == groupPrimarySourceFailoverPromoted ||
				strings.TrimSpace(g.ActivePrimarySource) == groupPrimarySourceFailoverCandidate) {
			return *g.ActivePrimaryAccountID
		}
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

func (g *Group) shouldPreferPrimaryBeforeStickySession() bool {
	if g == nil {
		return false
	}
	return g.EffectivePrimaryAccountMode() == GroupPrimaryAccountModeManual &&
		g.currentPreferredPrimaryAccountID() > 0
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

func (g *Group) isFailoverCandidateCoolingDown(now time.Time) bool {
	if g == nil || g.ActivePrimaryAccountID == nil || *g.ActivePrimaryAccountID <= 0 {
		return false
	}
	if strings.TrimSpace(g.ActivePrimarySource) != groupPrimarySourceFailoverCandidate {
		return false
	}
	if g.ActivePrimarySwitchedAt == nil {
		return false
	}
	return g.shouldBypassPrimaryCooldown(*g.ActivePrimarySwitchedAt)
}

func maybePromoteGroupPrimaryCandidate(
	ctx context.Context,
	repo groupPrimarySelectionUpdater,
	group *Group,
	accountID int64,
) bool {
	if repo == nil || group == nil || accountID <= 0 {
		return false
	}
	if group.ActivePrimaryAccountID == nil || *group.ActivePrimaryAccountID != accountID {
		return false
	}
	if strings.TrimSpace(group.ActivePrimarySource) != groupPrimarySourceFailoverCandidate {
		return false
	}
	if group.ActivePrimarySwitchedAt != nil && group.shouldBypassPrimaryCooldown(*group.ActivePrimarySwitchedAt) {
		return false
	}
	return persistGroupPrimarySelection(
		ctx,
		repo,
		group,
		accountID,
		groupPrimarySourceFailoverPromoted,
		groupPrimaryReasonFailoverStabilized,
	)
}

func persistGroupPrimarySelection(
	ctx context.Context,
	repo groupPrimarySelectionUpdater,
	group *Group,
	accountID int64,
	source string,
	reason string,
) bool {
	if repo == nil || group == nil || accountID <= 0 {
		return false
	}
	shouldTakeOverManual := group.EffectivePrimaryAccountMode() == GroupPrimaryAccountModeManual &&
		group.PrimaryAllowManualAutoReplace &&
		source == groupPrimarySourceFailoverPromoted
	currentID := int64(0)
	if group.ActivePrimaryAccountID != nil {
		currentID = *group.ActivePrimaryAccountID
	}
	if currentID == accountID &&
		strings.TrimSpace(group.ActivePrimarySource) == source &&
		strings.TrimSpace(group.ActivePrimaryReason) == reason &&
		(!shouldTakeOverManual || (group.ManualPrimaryAccountID == nil && group.EffectivePrimaryAccountMode() == GroupPrimaryAccountModeAuto)) {
		return false
	}
	now := time.Now()
	group.ActivePrimaryAccountID = &accountID
	group.ActivePrimarySource = source
	group.ActivePrimaryReason = reason
	group.ActivePrimarySwitchedAt = &now
	if shouldTakeOverManual {
		group.ManualPrimaryAccountID = nil
		group.PrimaryAccountMode = GroupPrimaryAccountModeAuto
	}
	if err := repo.Update(ctx, group); err != nil {
		slog.Warn("group primary account update failed",
			"group_id", group.ID,
			"account_id", accountID,
			"source", source,
			"reason", reason,
			"error", err,
		)
		return false
	}
	return currentID != accountID
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
	now := time.Now()
	activeID := int64(0)
	if group.ActivePrimaryAccountID != nil {
		activeID = *group.ActivePrimaryAccountID
	}
	activeSource := strings.TrimSpace(group.ActivePrimarySource)
	if activeID == selectedAccountID && activeSource == groupPrimarySourceFailoverCandidate {
		if group.ActivePrimarySwitchedAt != nil && group.shouldBypassPrimaryCooldown(*group.ActivePrimarySwitchedAt) {
			return
		}
		_ = persistGroupPrimarySelection(
			ctx,
			s.groupRepo,
			group,
			selectedAccountID,
			groupPrimarySourceFailoverPromoted,
			groupPrimaryReasonFailoverStabilized,
		)
		return
	}
	if activeID == selectedAccountID && activeSource == groupPrimarySourceFailoverPromoted {
		return
	}
	group.ActivePrimaryAccountID = &selectedAccountID
	group.ActivePrimarySource = groupPrimarySourceFailoverCandidate
	group.ActivePrimaryReason = groupPrimaryReasonRetryExhausted
	group.ActivePrimarySwitchedAt = &now
	if err := s.groupRepo.Update(ctx, group); err != nil {
		slog.Warn("group primary candidate update failed",
			"group_id", group.ID,
			"account_id", selectedAccountID,
			"source", groupPrimarySourceFailoverCandidate,
			"reason", groupPrimaryReasonRetryExhausted,
			"error", err,
		)
	}
}

func (s *GatewayService) promotePrimaryCandidateIfReady(ctx context.Context, group *Group, accountID int64) {
	if s == nil || group == nil || accountID <= 0 {
		return
	}
	_ = maybePromoteGroupPrimaryCandidate(ctx, s.groupRepo, group, accountID)
}
