package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type channelMonitorQualityRepoStub struct {
	ChannelMonitorRepository
	monitors []*ChannelMonitor
	history  map[int64][]*ChannelMonitorHistoryEntry
}

func (r channelMonitorQualityRepoStub) ListEnabled(ctx context.Context) ([]*ChannelMonitor, error) {
	return r.monitors, nil
}

func (r channelMonitorQualityRepoStub) ListRecentHistoryForMonitors(ctx context.Context, ids []int64, primaryModels map[int64]string, perMonitorLimit int) (map[int64][]*ChannelMonitorHistoryEntry, error) {
	return r.history, nil
}

func monitorHistory(statuses ...string) []*ChannelMonitorHistoryEntry {
	now := time.Now()
	out := make([]*ChannelMonitorHistoryEntry, 0, len(statuses))
	for i, status := range statuses {
		out = append(out, &ChannelMonitorHistoryEntry{
			Status:    status,
			CheckedAt: now.Add(-time.Duration(i) * time.Minute),
		})
	}
	return out
}

func TestCompareMonitorQualityScoresUses3Then5Then7(t *testing.T) {
	scoreA := buildMonitorQualityScore(monitorHistory(
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusFailed,
		MonitorStatusFailed,
	))
	scoreB := buildMonitorQualityScore(monitorHistory(
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusDegraded,
		MonitorStatusOperational,
		MonitorStatusOperational,
	))

	require.Less(t, compareMonitorQualityScores(scoreA, scoreB), 0, "same first 3: more green in first 5 wins")

	scoreC := buildMonitorQualityScore(monitorHistory(
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusFailed,
		MonitorStatusFailed,
	))
	scoreD := buildMonitorQualityScore(monitorHistory(
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusOperational,
		MonitorStatusOperational,
	))

	require.Greater(t, compareMonitorQualityScores(scoreC, scoreD), 0, "same first 5: first 7 breaks the tie")
}

func TestChannelMonitorQualityScorerMatchesAccountByEndpointOrigin(t *testing.T) {
	repo := channelMonitorQualityRepoStub{
		monitors: []*ChannelMonitor{
			{ID: 1, Provider: MonitorProviderOpenAI, Endpoint: "https://API.Example.com/v1/chat/completions", Enabled: true},
			{ID: 2, Provider: MonitorProviderOpenAI, Endpoint: "https://bad.example.com/v1", Enabled: true},
		},
		history: map[int64][]*ChannelMonitorHistoryEntry{
			1: monitorHistory(MonitorStatusOperational, MonitorStatusOperational, MonitorStatusOperational),
			2: monitorHistory(MonitorStatusFailed, MonitorStatusFailed, MonitorStatusFailed),
		},
	}
	scorer := NewChannelMonitorQualityScorer(repo)
	good := &Account{
		ID:       1,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "https://api.example.com",
		},
	}
	bad := &Account{
		ID:       2,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "https://bad.example.com/v1/",
		},
	}

	require.Less(t, scorer.CompareAccounts(context.Background(), good, bad), 0)
}

func TestMonitorEndpointForAntigravityAPIKeyUsesRawBaseURL(t *testing.T) {
	account := &Account{
		Platform: PlatformAntigravity,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "https://antigravity.example.com",
		},
	}

	require.Equal(t, "https://antigravity.example.com", monitorEndpointForAccount(account))
}
