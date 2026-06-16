package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestComputeAccountQualityScoreWeightsFiveSecondTTFTHigher(t *testing.T) {
	scoreFast5s := ComputeAccountQualityScore(1, 1, 1, 0, 0, 0, 0, 12)
	scoreFast10sOnly := ComputeAccountQualityScore(1, 0, 1, 0, 0, 0, 0, 12)

	require.Greater(t, scoreFast5s.TTFT5sComponent, scoreFast10sOnly.TTFT5sComponent)
	require.Greater(t, scoreFast5s.FastBonus, scoreFast10sOnly.FastBonus)
	require.Equal(t, 0.0, scoreFast5s.TTFT10sComponent)
	require.Greater(t, scoreFast10sOnly.TTFT10sComponent, 0.0)
}

func TestComputeAccountQualityScoreLongTTFTPunishesHard(t *testing.T) {
	fastStable := ComputeAccountQualityScore(1, 1, 1, 0, 0, 0, 0, 12)
	oneVerySlow := ComputeAccountQualityScore(1, 0.8, 0.8, 0.2, 0.2, 0.2, 0, 12)

	require.Greater(t, oneVerySlow.SlowPenalty, 0.1)
	require.Greater(t, fastStable.TTFT5sComponent, oneVerySlow.TTFT5sComponent)
	require.Greater(t, fastStable.FastBonus, oneVerySlow.FastBonus)
}

func TestComputeAccountQualityScoreAvoidsEasySaturation(t *testing.T) {
	score := ComputeAccountQualityScore(1, 1, 1, 0, 0, 0, 0, 12)
	require.Less(t, score.FinalScore, 1.0)
}

func TestEffectiveAccountQualityScoreUsesNeutralBaseForUnknown(t *testing.T) {
	score, known := effectiveAccountQualityScore(&AccountQualitySnapshot{
		TotalRequests: accountQualityMinSamples - 1,
	})

	require.False(t, known)
	require.Equal(t, accountQualityNeutralBase, score)
}

func TestEffectiveAccountQualityScoreAllowsAuxiliaryOnlyEvidence(t *testing.T) {
	score, known := effectiveAccountQualityScore(&AccountQualitySnapshot{
		TotalRequests:          0,
		AuxiliaryTotalRequests: accountQualityAuxiliaryKnownSamples,
		EffectiveQualityScore:  0.82,
	})

	require.True(t, known)
	require.Equal(t, 0.82, score)
}

func TestEffectiveAccountQualityScoreUsesTransientPenaltyAdjustedScore(t *testing.T) {
	score, known := effectiveAccountQualityScore(&AccountQualitySnapshot{
		TotalRequests:         accountQualityMinSamples,
		BaseQualityScore:      0.92,
		TransientPenalty:      0.50,
		RecoveryCredit:        0.14,
		EffectiveQualityScore: 0.56,
	})

	require.True(t, known)
	require.Equal(t, 0.56, score)
}

func TestFilterByMaxQualityWithinSamePriority(t *testing.T) {
	accounts := []accountWithLoad{
		{account: &Account{ID: 1, Priority: 1}, loadInfo: &AccountLoadInfo{LoadRate: 0}},
		{account: &Account{ID: 2, Priority: 1}, loadInfo: &AccountLoadInfo{LoadRate: 0}},
	}
	snapshots := map[int64]*AccountQualitySnapshot{
		1: {AccountID: 1, TotalRequests: accountQualityMinSamples, QualityScore: 0.4},
		2: {AccountID: 2, TotalRequests: accountQualityMinSamples, QualityScore: 0.9},
	}

	filtered := filterByMaxQuality(accounts, snapshots)

	require.Len(t, filtered, 1)
	require.Equal(t, int64(2), filtered[0].account.ID)
}

func TestSortAccountsByPriorityQualityAndLastUsedRespectsPriorityFirst(t *testing.T) {
	now := time.Now()
	accounts := []*Account{
		{ID: 1, Priority: 1, LastUsedAt: &now},
		{ID: 2, Priority: 2, LastUsedAt: nil},
	}
	snapshots := map[int64]*AccountQualitySnapshot{
		1: {AccountID: 1, TotalRequests: accountQualityMinSamples, QualityScore: 0.1},
		2: {AccountID: 2, TotalRequests: accountQualityMinSamples, QualityScore: 1.0},
	}

	sortAccountsByPriorityQualityAndLastUsed(accounts, false, snapshots)

	require.Equal(t, int64(1), accounts[0].ID)
	require.Equal(t, int64(2), accounts[1].ID)
}

func TestSortAccountsByPriorityQualityAndLastUsedUsesQualityBeforeLRU(t *testing.T) {
	now := time.Now()
	older := now.Add(-time.Hour)
	accounts := []*Account{
		{ID: 1, Priority: 1, LastUsedAt: &older},
		{ID: 2, Priority: 1, LastUsedAt: &now},
	}
	snapshots := map[int64]*AccountQualitySnapshot{
		1: {AccountID: 1, TotalRequests: accountQualityMinSamples, QualityScore: 0.4},
		2: {AccountID: 2, TotalRequests: accountQualityMinSamples, QualityScore: 0.9},
	}

	sortAccountsByPriorityQualityAndLastUsed(accounts, false, snapshots)

	require.Equal(t, int64(2), accounts[0].ID)
	require.Equal(t, int64(1), accounts[1].ID)
}

func TestSortCandidatesForFallbackRandomKeepsQualityAheadOfShuffle(t *testing.T) {
	accounts := []*Account{
		{ID: 1, Priority: 1},
		{ID: 2, Priority: 1},
		{ID: 3, Priority: 2},
	}
	snapshots := map[int64]*AccountQualitySnapshot{
		1: {AccountID: 1, TotalRequests: accountQualityMinSamples, QualityScore: 0.3},
		2: {AccountID: 2, TotalRequests: accountQualityMinSamples, QualityScore: 0.8},
		3: {AccountID: 3, TotalRequests: accountQualityMinSamples, QualityScore: 1.0},
	}

	(&GatewayService{}).sortCandidatesForFallback(accounts, false, "random", snapshots)

	require.Equal(t, int64(2), accounts[0].ID)
	require.Equal(t, int64(1), accounts[1].ID)
	require.Equal(t, int64(3), accounts[2].ID)
}

func TestShouldEscapeStickyByAccountQualityOnRecentFailure(t *testing.T) {
	escape, reason := shouldEscapeStickyByAccountQuality(&AccountQualitySnapshot{
		TotalRequests:     accountQualityMinSamples,
		RecentSuccessRate: 0.5,
		TTFTLE5sRate:      1,
		TTFTLE10sRate:     1,
		QualityScore:      ComputeAccountQualityScore(0.5, 1, 1, 0, 0, 0, 0, accountQualityMinSamples).FinalScore,
	})

	require.True(t, escape)
	require.Equal(t, "recent_success_rate", reason)
}

func TestShouldEscapeStickyByAccountQualityOnTransientPenalty(t *testing.T) {
	escape, reason := shouldEscapeStickyByAccountQuality(&AccountQualitySnapshot{
		TotalRequests:          accountQualityMinSamples,
		AuxiliaryTotalRequests: 3,
		TransientPenalty:       0.42,
		RecoveryCredit:         0.05,
		EffectiveQualityScore:  0.72,
	})

	require.True(t, escape)
	require.Equal(t, "transient_penalty", reason)
}

func TestShouldEscapeStickyByAccountQualitySkipsRecentSuccessRuleWhenAuxiliaryExists(t *testing.T) {
	escape, reason := shouldEscapeStickyByAccountQuality(&AccountQualitySnapshot{
		TotalRequests:          accountQualityMinSamples,
		AuxiliaryTotalRequests: 3,
		RecentSuccessRate:      0.5,
		EffectiveQualityScore:  0.88,
	})

	require.False(t, escape)
	require.Empty(t, reason)
}

func TestShouldEscapeStickyByAccountQualityIgnoresSmallSamples(t *testing.T) {
	escape, reason := shouldEscapeStickyByAccountQuality(&AccountQualitySnapshot{
		TotalRequests:     accountQualityMinSamples - 1,
		RecentSuccessRate: 0,
		QualityScore:      0,
	})

	require.False(t, escape)
	require.Empty(t, reason)
}

func TestCompareAccountsByQualityPrefersLowerPenaltyWhenScoresEqual(t *testing.T) {
	a := &Account{ID: 1}
	b := &Account{ID: 2}
	snapshots := map[int64]*AccountQualitySnapshot{
		1: {AccountID: 1, TotalRequests: accountQualityMinSamples, EffectiveQualityScore: 0.82, TransientPenalty: 0.30, RecoveryCredit: 0.00, QualityScore: 0.82},
		2: {AccountID: 2, TotalRequests: accountQualityMinSamples, EffectiveQualityScore: 0.82, TransientPenalty: 0.10, RecoveryCredit: 0.00, QualityScore: 0.82},
	}

	require.Equal(t, 1, compareAccountsByQuality(a, b, snapshots))
	require.Equal(t, -1, compareAccountsByQuality(b, a, snapshots))
}
