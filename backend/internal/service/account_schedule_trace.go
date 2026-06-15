package service

import (
	"fmt"
	"time"
)

func newUsageScheduleTrace(layer string, account *Account) *UsageScheduleTrace {
	trace := &UsageScheduleTrace{Layer: layer}
	trace.applyAccount(account)
	return trace
}

func attachUsageScheduleTrace(selection *AccountSelectionResult, trace *UsageScheduleTrace) *AccountSelectionResult {
	if selection == nil {
		return nil
	}
	selection.ScheduleTrace = trace
	return selection
}

func (t *UsageScheduleTrace) clone() *UsageScheduleTrace {
	if t == nil {
		return nil
	}
	out := *t
	out.Priority = cloneIntPtr(t.Priority)
	out.QualityScore = cloneFloat64Ptr(t.QualityScore)
	out.RecentSuccessRate = cloneFloat64Ptr(t.RecentSuccessRate)
	out.TTFTLE5sRate = cloneFloat64Ptr(t.TTFTLE5sRate)
	out.TTFTLE10sRate = cloneFloat64Ptr(t.TTFTLE10sRate)
	out.LoadRate = cloneFloat64Ptr(t.LoadRate)
	out.LoadSkew = cloneFloat64Ptr(t.LoadSkew)
	out.Score = cloneFloat64Ptr(t.Score)
	if len(t.Candidates) > 0 {
		out.Candidates = make([]UsageScheduleCandidateScore, len(t.Candidates))
		for i := range t.Candidates {
			out.Candidates[i] = cloneUsageScheduleCandidateScore(t.Candidates[i])
		}
	}
	return &out
}

func (t *UsageScheduleTrace) applyAccount(account *Account) {
	if t == nil || account == nil {
		return
	}
	t.SelectedAccountID = account.ID
	t.Platform = account.Platform
	t.AccountType = account.Type
	priority := account.Priority
	t.Priority = &priority
}

func (t *UsageScheduleTrace) applyQuality(snapshot *AccountQualitySnapshot) {
	if t == nil || snapshot == nil {
		return
	}
	t.QualityScore = float64Ptr(snapshot.QualityScore)
	t.RecentSuccessRate = float64Ptr(snapshot.RecentSuccessRate)
	t.TTFTLE5sRate = float64Ptr(snapshot.TTFTLE5sRate)
	t.TTFTLE10sRate = float64Ptr(snapshot.TTFTLE10sRate)
	t.TTFTSampleCount = snapshot.TTFTSampleCount
	t.TotalRequests = snapshot.TotalRequests
}

func (t *UsageScheduleTrace) applyLoad(loadInfo *AccountLoadInfo) {
	if t == nil || loadInfo == nil {
		return
	}
	rate := float64(loadInfo.LoadRate)
	t.LoadRate = &rate
}

func cloneIntPtr(v *int) *int {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func cloneFloat64Ptr(v *float64) *float64 {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func cloneTimePtr(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

func cloneUsageScheduleCandidateScore(in UsageScheduleCandidateScore) UsageScheduleCandidateScore {
	out := in
	out.QualityScore = cloneFloat64Ptr(in.QualityScore)
	out.RecentSuccessRate = cloneFloat64Ptr(in.RecentSuccessRate)
	out.TTFTLE5sRate = cloneFloat64Ptr(in.TTFTLE5sRate)
	out.TTFTLE10sRate = cloneFloat64Ptr(in.TTFTLE10sRate)
	out.LoadRate = cloneFloat64Ptr(in.LoadRate)
	out.ComputedScore = cloneFloat64Ptr(in.ComputedScore)
	out.LastUsedAt = cloneTimePtr(in.LastUsedAt)
	if in.WaitingCount != nil {
		waiting := *in.WaitingCount
		out.WaitingCount = &waiting
	}
	return out
}

func newUsageScheduleCandidateScore(account *Account) UsageScheduleCandidateScore {
	score := UsageScheduleCandidateScore{}
	if account == nil {
		return score
	}
	score.AccountID = account.ID
	score.AccountName = account.Name
	score.Platform = account.Platform
	score.AccountType = account.Type
	score.Priority = account.Priority
	score.LastUsedAt = cloneTimePtr(account.LastUsedAt)
	return score
}

func (t *UsageScheduleTrace) setCandidates(candidates []UsageScheduleCandidateScore) {
	if t == nil {
		return
	}
	if len(candidates) == 0 {
		t.Candidates = nil
		return
	}
	t.Candidates = make([]UsageScheduleCandidateScore, len(candidates))
	for i := range candidates {
		t.Candidates[i] = cloneUsageScheduleCandidateScore(candidates[i])
	}
}

func formatScheduleScore(v float64) string {
	return fmt.Sprintf("%.4f", v)
}

func newScheduleCandidateFromQuality(account *Account, snapshot *AccountQualitySnapshot, loadInfo *AccountLoadInfo, selected bool, stage string) UsageScheduleCandidateScore {
	candidate := newUsageScheduleCandidateScore(account)
	candidate.Selected = selected
	candidate.SelectionStage = stage
	score, known := effectiveAccountQualityScore(snapshot)
	candidate.QualityKnown = known
	candidate.QualityScore = float64Ptr(score)
	candidate.ComputedScore = float64Ptr(score)
	if snapshot != nil {
		candidate.RecentSuccessRate = float64Ptr(snapshot.RecentSuccessRate)
		candidate.TTFTLE5sRate = float64Ptr(snapshot.TTFTLE5sRate)
		candidate.TTFTLE10sRate = float64Ptr(snapshot.TTFTLE10sRate)
		candidate.TTFTSampleCount = snapshot.TTFTSampleCount
		candidate.TotalRequests = snapshot.TotalRequests
	}
	if known {
		candidate.ScoreBreakdown = fmt.Sprintf("quality=%s = 0.40*success_rate + 0.40*ttft<=5s + 0.20*ttft<=10s", formatScheduleScore(score))
	} else {
		candidate.ScoreBreakdown = "quality unknown: treated as 1.0000 until at least 3 recent samples"
	}
	if loadInfo != nil {
		loadRate := float64(loadInfo.LoadRate)
		candidate.LoadRate = &loadRate
		candidate.WaitingCount = intPtr(loadInfo.WaitingCount)
	}
	return candidate
}

func buildQualityScheduleCandidates(accounts []accountWithLoad, snapshots map[int64]*AccountQualitySnapshot, selectedID int64, stage string) []UsageScheduleCandidateScore {
	out := make([]UsageScheduleCandidateScore, 0, len(accounts))
	for _, item := range accounts {
		if item.account == nil {
			continue
		}
		out = append(out, newScheduleCandidateFromQuality(item.account, snapshots[item.account.ID], item.loadInfo, item.account.ID == selectedID, stage))
	}
	return out
}

func buildQualityScheduleCandidatesFromAccounts(accounts []*Account, snapshots map[int64]*AccountQualitySnapshot, selectedID int64, stage string) []UsageScheduleCandidateScore {
	out := make([]UsageScheduleCandidateScore, 0, len(accounts))
	for _, account := range accounts {
		if account == nil {
			continue
		}
		out = append(out, newScheduleCandidateFromQuality(account, snapshots[account.ID], nil, account.ID == selectedID, stage))
	}
	return out
}
