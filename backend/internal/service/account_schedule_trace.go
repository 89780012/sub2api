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
	t.TTFTGT10sRate = float64Ptr(snapshot.TTFTGT10sRate)
	t.TTFTGT20sRate = float64Ptr(snapshot.TTFTGT20sRate)
	t.TTFTGT40sRate = float64Ptr(snapshot.TTFTGT40sRate)
	t.TTFTSampleCount = snapshot.TTFTSampleCount
	t.TotalRequests = snapshot.TotalRequests
	t.FailureRequests = snapshot.FailureRequests
	t.ErrorRate = float64Ptr(snapshot.ErrorRate)
	t.SampleConfidence = float64Ptr(snapshot.SampleConfidence)
	t.FastBonus = float64Ptr(snapshot.FastBonus)
	t.SlowPenalty = float64Ptr(snapshot.SlowPenalty)
	t.ErrorPenalty = float64Ptr(snapshot.ErrorPenalty)
	t.NeutralBase = float64Ptr(snapshot.NeutralBase)
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
	out.TTFTGT10sRate = cloneFloat64Ptr(in.TTFTGT10sRate)
	out.TTFTGT20sRate = cloneFloat64Ptr(in.TTFTGT20sRate)
	out.TTFTGT40sRate = cloneFloat64Ptr(in.TTFTGT40sRate)
	out.LoadRate = cloneFloat64Ptr(in.LoadRate)
	out.ComputedScore = cloneFloat64Ptr(in.ComputedScore)
	out.ErrorRate = cloneFloat64Ptr(in.ErrorRate)
	out.SampleConfidence = cloneFloat64Ptr(in.SampleConfidence)
	out.FastBonus = cloneFloat64Ptr(in.FastBonus)
	out.SlowPenalty = cloneFloat64Ptr(in.SlowPenalty)
	out.ErrorPenalty = cloneFloat64Ptr(in.ErrorPenalty)
	out.NeutralBase = cloneFloat64Ptr(in.NeutralBase)
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
		candidate.TTFTGT10sRate = float64Ptr(snapshot.TTFTGT10sRate)
		candidate.TTFTGT20sRate = float64Ptr(snapshot.TTFTGT20sRate)
		candidate.TTFTGT40sRate = float64Ptr(snapshot.TTFTGT40sRate)
		candidate.TTFTSampleCount = snapshot.TTFTSampleCount
		candidate.TotalRequests = snapshot.TotalRequests
		candidate.FailureRequests = snapshot.FailureRequests
		candidate.ErrorRate = float64Ptr(snapshot.ErrorRate)
		candidate.SampleConfidence = float64Ptr(snapshot.SampleConfidence)
		candidate.FastBonus = float64Ptr(snapshot.FastBonus)
		candidate.SlowPenalty = float64Ptr(snapshot.SlowPenalty)
		candidate.ErrorPenalty = float64Ptr(snapshot.ErrorPenalty)
		candidate.NeutralBase = float64Ptr(snapshot.NeutralBase)
	}
	if known {
		candidate.ScoreBreakdown = fmt.Sprintf(
			"final=%s = neutral=%s + success=%s + ttft5=%s + ttft10=%s + fast_bonus=%s - slow_penalty=%s - error_penalty=%s (confidence=%s)",
			formatScheduleScore(score),
			formatScheduleScore(snapshot.NeutralBase),
			formatScheduleScore(0.25*clampQualityRate(snapshot.RecentSuccessRate)),
			formatScheduleScore(0.35*clampQualityRate(snapshot.TTFTLE5sRate)),
			formatScheduleScore(0.20*clampQualityRate(snapshot.TTFTLE10sRate)),
			formatScheduleScore(snapshot.FastBonus),
			formatScheduleScore(snapshot.SlowPenalty),
			formatScheduleScore(snapshot.ErrorPenalty),
			formatScheduleScore(snapshot.SampleConfidence),
		)
	} else {
		candidate.ScoreBreakdown = fmt.Sprintf("quality unknown: neutral score %s until at least %d recent samples", formatScheduleScore(accountQualityNeutralBase), accountQualityMinSamples)
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
