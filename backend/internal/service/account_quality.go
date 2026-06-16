package service

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	accountQualityWindow               = 5 * time.Minute
	accountQualityRefreshInterval      = time.Minute
	accountQualityRefreshTimeout       = 20 * time.Second
	accountQualityMinSamples           = 3
	accountQualityConfidenceMaxSamples = 12

	accountQualityNeutralBase        = 0.60
	accountQualitySuccessWeight      = 0.25
	accountQualityTTFT5sWeight       = 0.35
	accountQualityTTFT10sWeight      = 0.20
	accountQualityFastBonusWeight    = 0.20
	accountQualitySlow10sPenalty     = 0.10
	accountQualitySlow20sPenalty     = 0.30
	accountQualitySlow40sPenalty     = 0.60
	accountQualityErrorPenaltyWeight = 0.70

	accountQualityStickyEscapeScoreThreshold       = 0.70
	accountQualityStickyEscapeSuccessRateThreshold = 0.80
)

type AccountQualitySnapshot struct {
	AccountID                int64
	WindowStart              time.Time
	WindowEnd                time.Time
	TotalRequests            int64
	SuccessRequests          int64
	FailureRequests          int64
	RecentSuccessRate        float64
	ErrorRate                float64
	TTFTSampleCount          int64
	TTFTLE5sCount            int64
	TTFTLE10sCount           int64
	TTFTGT10sCount           int64
	TTFTGT20sCount           int64
	TTFTGT40sCount           int64
	TTFTLE5sRate             float64
	TTFTLE10sRate            float64
	TTFTGT10sRate            float64
	TTFTGT20sRate            float64
	TTFTGT40sRate            float64
	SampleConfidence         float64
	FastBonus                float64
	SlowPenalty              float64
	ErrorPenalty             float64
	NeutralBase              float64
	QualityScore             float64
	AuxiliaryTotalRequests   int64
	AuxiliarySuccessRequests int64
	AuxiliaryFailureRequests int64
	AuxiliaryTTFTSampleCount int64
	AuxiliaryTTFTLE5sCount   int64
	AuxiliaryTTFTLE10sCount  int64
	AuxiliaryTTFTGT10sCount  int64
	AuxiliaryTTFTGT20sCount  int64
	AuxiliaryTTFTGT40sCount  int64
	AuxiliaryWeight          float64
	UpdatedAt                time.Time
}

type AccountQualityRepository interface {
	RefreshSnapshots(ctx context.Context, windowStart, windowEnd time.Time) error
	GetSnapshotsByAccountIDs(ctx context.Context, accountIDs []int64) (map[int64]*AccountQualitySnapshot, error)
}

type AccountQualityReader interface {
	GetSnapshotsByAccountIDs(ctx context.Context, accountIDs []int64) (map[int64]*AccountQualitySnapshot, error)
}

var (
	defaultAccountQualityReaderMu sync.RWMutex
	defaultAccountQualityReader   AccountQualityReader
)

type AccountQualityComponents struct {
	NeutralBase      float64
	SuccessComponent float64
	TTFT5sComponent  float64
	TTFT10sComponent float64
	FastBonus        float64
	SlowPenalty      float64
	ErrorPenalty     float64
	SampleConfidence float64
	FinalScore       float64
}

func ComputeAccountQualityScore(
	successRate float64,
	ttftLE5sRate float64,
	ttftLE10sRate float64,
	ttftGT10sRate float64,
	ttftGT20sRate float64,
	ttftGT40sRate float64,
	errorRate float64,
	totalRequests int64,
) AccountQualityComponents {
	confidence := 0.0
	if totalRequests > 0 {
		confidence = clampQualityRate(float64(totalRequests) / float64(accountQualityConfidenceMaxSamples))
	}
	successComponent := accountQualitySuccessWeight * clampQualityRate(successRate)
	ttft5sComponent := accountQualityTTFT5sWeight * clampQualityRate(ttftLE5sRate)
	ttft10sComponent := accountQualityTTFT10sWeight * clampQualityRate(ttftLE10sRate)
	fastBonus := accountQualityFastBonusWeight * confidence * clampQualityRate((ttftLE5sRate*0.7)+(ttftLE10sRate*0.3))
	slowPenalty := confidence * (accountQualitySlow10sPenalty*clampQualityRate(ttftGT10sRate) +
		accountQualitySlow20sPenalty*clampQualityRate(ttftGT20sRate) +
		accountQualitySlow40sPenalty*clampQualityRate(ttftGT40sRate))
	errorPenalty := accountQualityErrorPenaltyWeight * clampQualityRate(errorRate)
	finalScore := accountQualityNeutralBase + successComponent + ttft5sComponent + ttft10sComponent + fastBonus - slowPenalty - errorPenalty

	return AccountQualityComponents{
		NeutralBase:      accountQualityNeutralBase,
		SuccessComponent: math.Round(successComponent*10000) / 10000,
		TTFT5sComponent:  math.Round(ttft5sComponent*10000) / 10000,
		TTFT10sComponent: math.Round(ttft10sComponent*10000) / 10000,
		FastBonus:        math.Round(fastBonus*10000) / 10000,
		SlowPenalty:      math.Round(slowPenalty*10000) / 10000,
		ErrorPenalty:     math.Round(errorPenalty*10000) / 10000,
		SampleConfidence: math.Round(confidence*10000) / 10000,
		FinalScore:       math.Round(clampQualityRate(finalScore)*10000) / 10000,
	}
}

func clampQualityRate(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func SetDefaultAccountQualityReader(reader AccountQualityReader) {
	defaultAccountQualityReaderMu.Lock()
	defer defaultAccountQualityReaderMu.Unlock()
	defaultAccountQualityReader = reader
}

func DefaultAccountQualityReader() AccountQualityReader {
	defaultAccountQualityReaderMu.RLock()
	defer defaultAccountQualityReaderMu.RUnlock()
	return defaultAccountQualityReader
}

func DescribeAccountQualitySnapshot(snapshot *AccountQualitySnapshot) (float64, bool, string, AccountQualityComponents) {
	if snapshot == nil {
		components := AccountQualityComponents{
			NeutralBase:      accountQualityNeutralBase,
			SampleConfidence: 0,
			FinalScore:       accountQualityNeutralBase,
		}
		return accountQualityNeutralBase, false, fmt.Sprintf(
			"quality unknown: neutral score %.4f until at least %d recent samples",
			accountQualityNeutralBase,
			accountQualityMinSamples,
		), components
	}

	components := ComputeAccountQualityScore(
		snapshot.RecentSuccessRate,
		snapshot.TTFTLE5sRate,
		snapshot.TTFTLE10sRate,
		snapshot.TTFTGT10sRate,
		snapshot.TTFTGT20sRate,
		snapshot.TTFTGT40sRate,
		snapshot.ErrorRate,
		snapshot.TotalRequests,
	)

	score, known := effectiveAccountQualityScore(snapshot)
	if !known {
		components.FinalScore = accountQualityNeutralBase
		return accountQualityNeutralBase, false, fmt.Sprintf(
			"quality unknown: neutral score %.4f until at least %d recent samples",
			accountQualityNeutralBase,
			accountQualityMinSamples,
		), components
	}

	return score, true, fmt.Sprintf(
		"final=%.4f = neutral=%.4f + success=%.4f + ttft5=%.4f + ttft10=%.4f + fast_bonus=%.4f - slow_penalty=%.4f - error_penalty=%.4f (confidence=%.4f)",
		score,
		components.NeutralBase,
		components.SuccessComponent,
		components.TTFT5sComponent,
		components.TTFT10sComponent,
		components.FastBonus,
		components.SlowPenalty,
		components.ErrorPenalty,
		components.SampleConfidence,
	), components
}

func effectiveAccountQualityScore(snapshot *AccountQualitySnapshot) (float64, bool) {
	if snapshot == nil || snapshot.TotalRequests < accountQualityMinSamples {
		return accountQualityNeutralBase, false
	}
	return clampQualityRate(snapshot.QualityScore), true
}

func accountIDsFromAccounts(accounts []*Account) []int64 {
	if len(accounts) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(accounts))
	ids := make([]int64, 0, len(accounts))
	for _, account := range accounts {
		if account == nil || account.ID <= 0 {
			continue
		}
		if _, ok := seen[account.ID]; ok {
			continue
		}
		seen[account.ID] = struct{}{}
		ids = append(ids, account.ID)
	}
	return ids
}

func accountIDsFromAccountValues(accounts []Account) []int64 {
	if len(accounts) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(accounts))
	ids := make([]int64, 0, len(accounts))
	for i := range accounts {
		account := &accounts[i]
		if account.ID <= 0 {
			continue
		}
		if _, ok := seen[account.ID]; ok {
			continue
		}
		seen[account.ID] = struct{}{}
		ids = append(ids, account.ID)
	}
	return ids
}

func accountIDsFromAccountsWithLoad(accounts []accountWithLoad) []int64 {
	if len(accounts) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(accounts))
	ids := make([]int64, 0, len(accounts))
	for _, item := range accounts {
		if item.account == nil || item.account.ID <= 0 {
			continue
		}
		if _, ok := seen[item.account.ID]; ok {
			continue
		}
		seen[item.account.ID] = struct{}{}
		ids = append(ids, item.account.ID)
	}
	return ids
}

func filterByMaxQuality(accounts []accountWithLoad, snapshots map[int64]*AccountQualitySnapshot) []accountWithLoad {
	if len(accounts) <= 1 || len(snapshots) == 0 {
		return accounts
	}
	maxScore := -1.0
	for _, item := range accounts {
		score, _ := effectiveAccountQualityScore(snapshots[item.account.ID])
		if score > maxScore {
			maxScore = score
		}
	}
	result := make([]accountWithLoad, 0, len(accounts))
	for _, item := range accounts {
		score, _ := effectiveAccountQualityScore(snapshots[item.account.ID])
		if score == maxScore {
			result = append(result, item)
		}
	}
	return result
}

func compareAccountsByQuality(a, b *Account, snapshots map[int64]*AccountQualitySnapshot) int {
	if len(snapshots) == 0 || a == nil || b == nil {
		return 0
	}
	aScore, aKnown := effectiveAccountQualityScore(snapshots[a.ID])
	bScore, bKnown := effectiveAccountQualityScore(snapshots[b.ID])
	if aScore > bScore {
		return -1
	}
	if aScore < bScore {
		return 1
	}
	if aKnown && !bKnown {
		return -1
	}
	if !aKnown && bKnown {
		return 1
	}
	return 0
}

func accountQualitySnapshotsForReader(ctx context.Context, reader AccountQualityReader, accountIDs []int64, logKey string) map[int64]*AccountQualitySnapshot {
	if reader == nil || len(accountIDs) == 0 {
		return nil
	}
	snapshots, err := reader.GetSnapshotsByAccountIDs(ctx, accountIDs)
	if err != nil {
		logger.LegacyPrintf(logKey, "load account quality snapshots failed: %v", err)
		return nil
	}
	return snapshots
}

func accountQualitySnapshotForAccount(ctx context.Context, reader AccountQualityReader, accountID int64, logKey string) *AccountQualitySnapshot {
	snapshots := accountQualitySnapshotsForReader(ctx, reader, []int64{accountID}, logKey)
	if len(snapshots) == 0 {
		return nil
	}
	return snapshots[accountID]
}

func shouldEscapeStickyByAccountQuality(snapshot *AccountQualitySnapshot) (bool, string) {
	score, known := effectiveAccountQualityScore(snapshot)
	if !known {
		return false, ""
	}
	if snapshot.RecentSuccessRate < accountQualityStickyEscapeSuccessRateThreshold {
		return true, "recent_success_rate"
	}
	if score < accountQualityStickyEscapeScoreThreshold {
		return true, "quality_score"
	}
	return false, ""
}

func shouldEscapeStickyAccountByQuality(ctx context.Context, reader AccountQualityReader, accountID int64, logKey string) (bool, string, *AccountQualitySnapshot) {
	if reader == nil || accountID <= 0 {
		return false, "", nil
	}
	snapshot := accountQualitySnapshotForAccount(ctx, reader, accountID, logKey)
	escape, reason := shouldEscapeStickyByAccountQuality(snapshot)
	if escape && snapshot != nil {
		logger.LegacyPrintf(logKey,
			"sticky account quality escape: account_id=%d reason=%s total=%d success_rate=%.4f ttft_5s_rate=%.4f ttft_10s_rate=%.4f score=%.4f",
			accountID,
			reason,
			snapshot.TotalRequests,
			snapshot.RecentSuccessRate,
			snapshot.TTFTLE5sRate,
			snapshot.TTFTLE10sRate,
			snapshot.QualityScore,
		)
	}
	return escape, reason, snapshot
}

type AccountQualityService struct {
	repo AccountQualityRepository

	stopCh  chan struct{}
	doneCh  chan struct{}
	start   sync.Once
	stop    sync.Once
	started atomic.Bool
}

func NewAccountQualityService(repo AccountQualityRepository) *AccountQualityService {
	return &AccountQualityService{
		repo:   repo,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

func (s *AccountQualityService) Start() {
	if s == nil || s.repo == nil {
		return
	}
	s.start.Do(func() {
		s.started.Store(true)
		go s.run()
	})
}

func (s *AccountQualityService) Stop() {
	if s == nil {
		return
	}
	if !s.started.Load() {
		return
	}
	s.stop.Do(func() {
		close(s.stopCh)
		<-s.doneCh
	})
}

func (s *AccountQualityService) GetSnapshotsByAccountIDs(ctx context.Context, accountIDs []int64) (map[int64]*AccountQualitySnapshot, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	return s.repo.GetSnapshotsByAccountIDs(ctx, accountIDs)
}

func (s *AccountQualityService) run() {
	defer close(s.doneCh)
	s.refreshOnce()

	ticker := time.NewTicker(accountQualityRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.refreshOnce()
		case <-s.stopCh:
			return
		}
	}
}

func (s *AccountQualityService) refreshOnce() {
	if s == nil || s.repo == nil {
		return
	}
	now := time.Now().UTC()
	ctx, cancel := context.WithTimeout(context.Background(), accountQualityRefreshTimeout)
	defer cancel()

	if err := s.repo.RefreshSnapshots(ctx, now.Add(-accountQualityWindow), now); err != nil {
		logger.LegacyPrintf("service.account_quality", "refresh account quality snapshots failed: %v", err)
	}
}

func ProvideAccountQualityService(repo AccountQualityRepository, gateway *GatewayService, openaiGateway *OpenAIGatewayService) *AccountQualityService {
	svc := NewAccountQualityService(repo)
	if gateway != nil {
		gateway.SetAccountQualityReader(svc)
	}
	if openaiGateway != nil {
		openaiGateway.SetAccountQualityReader(svc)
	}
	SetDefaultAccountQualityReader(svc)
	svc.Start()
	return svc
}
