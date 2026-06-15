package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestComputeAccountQualityScoreWeightsFiveSecondTTFTHigher(t *testing.T) {
	scoreFast5s := ComputeAccountQualityScore(1, 1, 1)
	scoreFast10sOnly := ComputeAccountQualityScore(1, 0, 1)

	require.Equal(t, 1.0, scoreFast5s)
	require.Equal(t, 0.6, scoreFast10sOnly)
	require.Greater(t, scoreFast5s-scoreFast10sOnly, 0.2)
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
		QualityScore:      ComputeAccountQualityScore(0.5, 1, 1),
	})

	require.True(t, escape)
	require.Equal(t, "recent_success_rate", reason)
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
