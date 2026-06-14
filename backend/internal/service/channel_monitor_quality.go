package service

import (
	"context"
	"net/url"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

const (
	channelMonitorQualityRecentLimit = 7
	channelMonitorQualityCacheTTL    = 15 * time.Second
)

type ChannelMonitorQualityScorer struct {
	repo  ChannelMonitorRepository
	cache *gocache.Cache
}

type monitorQualitySnapshot struct {
	monitorByKey map[string]*ChannelMonitor
	historyByID  map[int64][]*ChannelMonitorHistoryEntry
}

type channelMonitorQualityScore struct {
	Known   bool
	Counts3 monitorStatusCounts
	Counts5 monitorStatusCounts
	Counts7 monitorStatusCounts
}

type monitorStatusCounts struct {
	Green   int
	Orange  int
	Red     int
	Unknown int
}

func NewChannelMonitorQualityScorer(repo ChannelMonitorRepository) *ChannelMonitorQualityScorer {
	return &ChannelMonitorQualityScorer{
		repo:  repo,
		cache: gocache.New(channelMonitorQualityCacheTTL, time.Minute),
	}
}

func (s *ChannelMonitorQualityScorer) Snapshot(ctx context.Context) *monitorQualitySnapshot {
	if s == nil || s.repo == nil {
		return nil
	}
	if cached, ok := s.cache.Get("snapshot"); ok {
		if snapshot, ok := cached.(*monitorQualitySnapshot); ok {
			return snapshot
		}
	}

	monitors, err := s.repo.ListEnabled(ctx)
	if err != nil || len(monitors) == 0 {
		return nil
	}

	monitorByKey := make(map[string]*ChannelMonitor, len(monitors))
	ids := make([]int64, 0, len(monitors))
	primaryModels := make(map[int64]string, len(monitors))
	for _, monitor := range monitors {
		if monitor == nil {
			continue
		}
		key := monitorQualityKey(monitor.Provider, monitor.Endpoint)
		if key == "" {
			continue
		}
		if existing := monitorByKey[key]; existing != nil {
			if monitor.ID <= existing.ID {
				continue
			}
		}
		monitorByKey[key] = monitor
		ids = append(ids, monitor.ID)
		primaryModels[monitor.ID] = strings.TrimSpace(monitor.PrimaryModel)
	}
	if len(monitorByKey) == 0 {
		return nil
	}

	historyByID, err := s.repo.ListRecentHistoryForMonitors(ctx, ids, primaryModels, channelMonitorQualityRecentLimit)
	if err != nil {
		historyByID = map[int64][]*ChannelMonitorHistoryEntry{}
	}

	snapshot := &monitorQualitySnapshot{
		monitorByKey: monitorByKey,
		historyByID:  historyByID,
	}
	s.cache.Set("snapshot", snapshot, channelMonitorQualityCacheTTL)
	return snapshot
}

func (s *ChannelMonitorQualityScorer) CompareAccounts(ctx context.Context, a, b *Account) int {
	if a == nil || b == nil {
		return 0
	}
	return compareMonitorQualityScores(s.ScoreAccount(ctx, a), s.ScoreAccount(ctx, b))
}

func (s *ChannelMonitorQualityScorer) ScoreAccount(ctx context.Context, account *Account) channelMonitorQualityScore {
	snapshot := s.Snapshot(ctx)
	if snapshot == nil {
		return channelMonitorQualityScore{}
	}
	return snapshot.ScoreAccount(account)
}

func (s *monitorQualitySnapshot) ScoreAccount(account *Account) channelMonitorQualityScore {
	if s == nil || account == nil {
		return channelMonitorQualityScore{}
	}
	key := monitorQualityKey(monitorProviderForAccount(account), monitorEndpointForAccount(account))
	if key == "" {
		return channelMonitorQualityScore{}
	}
	monitor := s.monitorByKey[key]
	if monitor == nil {
		return channelMonitorQualityScore{}
	}
	history := s.historyByID[monitor.ID]
	if len(history) == 0 {
		return channelMonitorQualityScore{Known: true}
	}
	return buildMonitorQualityScore(history)
}

func buildMonitorQualityScore(history []*ChannelMonitorHistoryEntry) channelMonitorQualityScore {
	score := channelMonitorQualityScore{Known: true}
	score.Counts3 = countMonitorStatuses(history, 3)
	score.Counts5 = countMonitorStatuses(history, 5)
	score.Counts7 = countMonitorStatuses(history, 7)
	return score
}

func countMonitorStatuses(history []*ChannelMonitorHistoryEntry, limit int) monitorStatusCounts {
	counts := monitorStatusCounts{}
	for i, entry := range history {
		if i >= limit {
			break
		}
		switch strings.TrimSpace(entry.Status) {
		case MonitorStatusOperational:
			counts.Green++
		case MonitorStatusDegraded:
			counts.Orange++
		case MonitorStatusFailed, MonitorStatusError:
			counts.Red++
		default:
			counts.Unknown++
		}
	}
	return counts
}

func compareMonitorQualityScores(a, b channelMonitorQualityScore) int {
	if a.Known != b.Known {
		if a.Known {
			return -1
		}
		return 1
	}
	for _, pair := range [][2]monitorStatusCounts{
		{a.Counts3, b.Counts3},
		{a.Counts5, b.Counts5},
		{a.Counts7, b.Counts7},
	} {
		if cmp := compareMonitorStatusCounts(pair[0], pair[1]); cmp != 0 {
			return cmp
		}
	}
	return 0
}

func compareMonitorStatusCounts(a, b monitorStatusCounts) int {
	if a.Green != b.Green {
		if a.Green > b.Green {
			return -1
		}
		return 1
	}
	if a.Orange != b.Orange {
		if a.Orange < b.Orange {
			return -1
		}
		return 1
	}
	if a.Red != b.Red {
		if a.Red < b.Red {
			return -1
		}
		return 1
	}
	if a.Unknown != b.Unknown {
		if a.Unknown < b.Unknown {
			return -1
		}
		return 1
	}
	return 0
}

func monitorProviderForAccount(account *Account) string {
	if account == nil {
		return ""
	}
	switch account.Platform {
	case PlatformAnthropic, PlatformAntigravity:
		return MonitorProviderAnthropic
	case PlatformOpenAI:
		return MonitorProviderOpenAI
	case PlatformGemini:
		return MonitorProviderGemini
	default:
		return strings.TrimSpace(account.Platform)
	}
}

func monitorEndpointForAccount(account *Account) string {
	if account == nil {
		return ""
	}
	switch account.Platform {
	case PlatformOpenAI:
		return account.GetOpenAIBaseURL()
	case PlatformGemini:
		return account.GetGeminiBaseURL("https://generativelanguage.googleapis.com")
	case PlatformAnthropic:
		if account.IsCustomBaseURLEnabled() && account.GetCustomBaseURL() != "" {
			return account.GetCustomBaseURL()
		}
		return account.GetBaseURL()
	case PlatformAntigravity:
		if account.Type == AccountTypeAPIKey {
			return account.GetCredential("base_url")
		}
	}
	return strings.TrimSpace(account.GetCredential("base_url"))
}

func monitorQualityKey(provider, rawEndpoint string) string {
	provider = strings.TrimSpace(provider)
	endpoint := normalizeMonitorQualityEndpoint(rawEndpoint)
	if provider == "" || endpoint == "" {
		return ""
	}
	return provider + "|" + endpoint
}

func normalizeMonitorQualityEndpoint(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimRight(strings.ToLower(raw), "/")
	}
	scheme := strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Host)
	return scheme + "://" + host
}
