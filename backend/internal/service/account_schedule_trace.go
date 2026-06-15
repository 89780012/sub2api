package service

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
