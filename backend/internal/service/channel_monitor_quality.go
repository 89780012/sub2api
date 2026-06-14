package service

import (
	"context"
	"log/slog"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	channelMonitorQualityRecentLimit     = 7
	channelMonitorQualityRefreshInterval = 10 * time.Second
)

type ChannelMonitorQualityScorer struct {
	repo ChannelMonitorRepository

	mu        sync.RWMutex
	snapshot  *monitorQualitySnapshot
	refreshMu sync.Mutex

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
}

type monitorQualitySnapshot struct {
	monitorByKey map[string]*ChannelMonitor
	historyByID  map[int64][]*ChannelMonitorHistoryEntry
	createdAt    time.Time
}

type channelMonitorQualityScore struct {
	Known        bool
	MonitorID    int64
	MonitorName  string
	PrimaryModel string
	SnapshotAt   time.Time
	Counts3      monitorStatusCounts
	Counts5      monitorStatusCounts
	Counts7      monitorStatusCounts
	Recent7      []monitorStatusSample
}

type monitorStatusCounts struct {
	Green   int
	Orange  int
	Red     int
	Unknown int
}

type monitorStatusSample struct {
	Status    string
	CheckedAt time.Time
}

func NewChannelMonitorQualityScorer(repo ChannelMonitorRepository) *ChannelMonitorQualityScorer {
	scorer := &ChannelMonitorQualityScorer{
		repo:   repo,
		stopCh: make(chan struct{}),
	}
	scorer.Start()
	return scorer
}

func (s *ChannelMonitorQualityScorer) Start() {
	if s == nil || s.repo == nil {
		return
	}
	s.startOnce.Do(func() {
		go s.refreshLoop()
	})
}

func (s *ChannelMonitorQualityScorer) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

func (s *ChannelMonitorQualityScorer) refreshLoop() {
	s.refresh(context.Background())
	ticker := time.NewTicker(channelMonitorQualityRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.refresh(context.Background())
		case <-s.stopCh:
			return
		}
	}
}

func (s *ChannelMonitorQualityScorer) Snapshot(ctx context.Context) *monitorQualitySnapshot {
	if s == nil || s.repo == nil {
		return nil
	}
	if snapshot := s.currentSnapshot(); snapshot != nil {
		return snapshot
	}
	return s.refresh(ctx)
}

func (s *ChannelMonitorQualityScorer) currentSnapshot() *monitorQualitySnapshot {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot
}

func (s *ChannelMonitorQualityScorer) refresh(ctx context.Context) *monitorQualitySnapshot {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	snapshot, err := s.loadSnapshot(ctx)
	if err != nil {
		slog.Warn("channel_monitor_quality: refresh failed", "error", err)
		return s.currentSnapshot()
	}
	s.mu.Lock()
	s.snapshot = snapshot
	s.mu.Unlock()
	return snapshot
}

func (s *ChannelMonitorQualityScorer) loadSnapshot(ctx context.Context) (*monitorQualitySnapshot, error) {
	monitors, err := s.repo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if len(monitors) == 0 {
		return &monitorQualitySnapshot{
			monitorByKey: map[string]*ChannelMonitor{},
			historyByID:  map[int64][]*ChannelMonitorHistoryEntry{},
			createdAt:    time.Now(),
		}, nil
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
		return &monitorQualitySnapshot{
			monitorByKey: map[string]*ChannelMonitor{},
			historyByID:  map[int64][]*ChannelMonitorHistoryEntry{},
			createdAt:    time.Now(),
		}, nil
	}

	historyByID, err := s.repo.ListRecentHistoryForMonitors(ctx, ids, primaryModels, channelMonitorQualityRecentLimit)
	if err != nil {
		return nil, err
	}

	return &monitorQualitySnapshot{
		monitorByKey: monitorByKey,
		historyByID:  historyByID,
		createdAt:    time.Now(),
	}, nil
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
		return channelMonitorQualityScore{SnapshotAt: s.createdAt}
	}
	history := s.historyByID[monitor.ID]
	if len(history) == 0 {
		return channelMonitorQualityScore{
			Known:        true,
			MonitorID:    monitor.ID,
			MonitorName:  monitor.Name,
			PrimaryModel: strings.TrimSpace(monitor.PrimaryModel),
			SnapshotAt:   s.createdAt,
		}
	}
	score := buildMonitorQualityScore(history)
	score.MonitorID = monitor.ID
	score.MonitorName = monitor.Name
	score.PrimaryModel = strings.TrimSpace(monitor.PrimaryModel)
	score.SnapshotAt = s.createdAt
	return score
}

func buildMonitorQualityScore(history []*ChannelMonitorHistoryEntry) channelMonitorQualityScore {
	score := channelMonitorQualityScore{Known: true}
	recentFirst := sortMonitorHistoryRecentFirst(history)
	score.Counts3 = countMonitorStatuses(recentFirst, 3)
	score.Counts5 = countMonitorStatuses(recentFirst, 5)
	score.Counts7 = countMonitorStatuses(recentFirst, 7)
	score.Recent7 = monitorStatusSamples(recentFirst, 7)
	return score
}

func sortMonitorHistoryRecentFirst(history []*ChannelMonitorHistoryEntry) []*ChannelMonitorHistoryEntry {
	recentFirst := append([]*ChannelMonitorHistoryEntry(nil), history...)
	sort.SliceStable(recentFirst, func(i, j int) bool {
		left := recentFirst[i]
		right := recentFirst[j]
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		if !left.CheckedAt.Equal(right.CheckedAt) {
			return left.CheckedAt.After(right.CheckedAt)
		}
		return left.ID > right.ID
	})
	return recentFirst
}

func monitorStatusSamples(history []*ChannelMonitorHistoryEntry, limit int) []monitorStatusSample {
	if limit <= 0 || len(history) == 0 {
		return nil
	}
	samples := make([]monitorStatusSample, 0, min(limit, len(history)))
	for i, entry := range history {
		if i >= limit {
			break
		}
		if entry == nil {
			samples = append(samples, monitorStatusSample{Status: "unknown"})
			continue
		}
		samples = append(samples, monitorStatusSample{
			Status:    strings.TrimSpace(entry.Status),
			CheckedAt: entry.CheckedAt,
		})
	}
	return samples
}

func countMonitorStatuses(history []*ChannelMonitorHistoryEntry, limit int) monitorStatusCounts {
	counts := monitorStatusCounts{}
	for i, entry := range history {
		if i >= limit {
			break
		}
		if entry == nil {
			counts.Unknown++
			continue
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
