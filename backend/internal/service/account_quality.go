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
	accountQualityWindow                = 5 * time.Minute
	accountQualityRefreshInterval       = time.Minute
	accountQualityRefreshTimeout        = 20 * time.Second
	accountQualityMinSamples            = 3
	accountQualityAuxiliaryKnownSamples = 2
	accountQualityConfidenceMaxSamples  = 12

	accountQualityNeutralBase        = 0.25
	accountQualitySuccessWeight      = 0.20
	accountQualityTTFT5sWeight       = 0.28
	accountQualityTTFT10sWeight      = 0.12
	accountQualityFastBonusWeight    = 0.10
	accountQualitySlow10sPenalty     = 0.18
	accountQualitySlow20sPenalty     = 0.36
	accountQualitySlow40sPenalty     = 0.58
	accountQualityErrorPenaltyWeight = 0.78
	accountQualityBurstPenaltyCap    = 0.60
	accountQualityRecoveryCreditCap  = 0.45

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
	BaseQualityScore         float64
	EffectiveQualityScore    float64
	TransientPenalty         float64
	RecoveryCredit           float64
	SlowStreak               int64
	ErrorStreak              int64
	RecoverySuccessStreak    int64
	RecoveryFastStreak       int64
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
	NeutralBase           float64
	SuccessComponent      float64
	TTFT5sComponent       float64
	TTFT10sComponent      float64
	FastBonus             float64
	SlowPenalty           float64
	ErrorPenalty          float64
	SampleConfidence      float64
	BaseQualityScore      float64
	EffectiveQualityScore float64
	TransientPenalty      float64
	RecoveryCredit        float64
	AppliedPenalty        float64
	FinalScore            float64
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
	ttft5To10sRate := clampQualityRate(ttftLE10sRate - ttftLE5sRate)
	successComponent := accountQualitySuccessWeight * clampQualityRate(successRate)
	ttft5sComponent := accountQualityTTFT5sWeight * clampQualityRate(ttftLE5sRate)
	ttft10sComponent := accountQualityTTFT10sWeight * ttft5To10sRate
	fastBonus := accountQualityFastBonusWeight * confidence * clampQualityRate((ttftLE5sRate*0.85)+(ttft5To10sRate*0.15))
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
			NeutralBase:           accountQualityNeutralBase,
			SampleConfidence:      0,
			BaseQualityScore:      accountQualityNeutralBase,
			EffectiveQualityScore: accountQualityNeutralBase,
			FinalScore:            accountQualityNeutralBase,
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
	components.BaseQualityScore = clampQualityRate(snapshot.BaseQualityScore)
	components.TransientPenalty = clampQualityRate(snapshot.TransientPenalty)
	components.RecoveryCredit = clampQualityRate(snapshot.RecoveryCredit)
	components.AppliedPenalty = appliedAccountQualityPenalty(snapshot)

	score, known := effectiveAccountQualityScore(snapshot)
	components.EffectiveQualityScore = score
	components.FinalScore = score
	if !known {
		components.EffectiveQualityScore = accountQualityNeutralBase
		components.FinalScore = accountQualityNeutralBase
		return accountQualityNeutralBase, false, fmt.Sprintf(
			"quality unknown: neutral score %.4f until at least %d recent samples",
			accountQualityNeutralBase,
			accountQualityMinSamples,
		), components
	}

	return score, true, fmt.Sprintf(
		"effective=%.4f = base=%.4f - burst_penalty=%.4f + recovery_credit=%.4f (applied_penalty=%.4f, slow_streak=%d, error_streak=%d, recovery_success=%d, recovery_fast=%d; aggregate: neutral=%.4f + success=%.4f + ttft5=%.4f + ttft10=%.4f + fast_bonus=%.4f - slow_penalty=%.4f - error_penalty=%.4f, confidence=%.4f)",
		score,
		components.BaseQualityScore,
		components.TransientPenalty,
		components.RecoveryCredit,
		components.AppliedPenalty,
		snapshot.SlowStreak,
		snapshot.ErrorStreak,
		snapshot.RecoverySuccessStreak,
		snapshot.RecoveryFastStreak,
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
	if !accountQualityHasEvidence(snapshot) {
		return accountQualityNeutralBase, false
	}
	score := snapshot.EffectiveQualityScore
	if score == 0 && snapshot.QualityScore != 0 {
		score = snapshot.QualityScore
	}
	return clampQualityRate(score), true
}

func accountQualityHasEvidence(snapshot *AccountQualitySnapshot) bool {
	if snapshot == nil {
		return false
	}
	if snapshot.TotalRequests >= accountQualityMinSamples {
		return true
	}
	if snapshot.AuxiliaryTotalRequests >= accountQualityAuxiliaryKnownSamples {
		return true
	}
	if snapshot.TransientPenalty > 0 || snapshot.RecoveryCredit > 0 {
		return true
	}
	if snapshot.SlowStreak >= 3 || snapshot.ErrorStreak >= 2 {
		return true
	}
	if snapshot.RecoverySuccessStreak > 0 || snapshot.RecoveryFastStreak > 0 {
		return true
	}
	return false
}

func appliedAccountQualityPenalty(snapshot *AccountQualitySnapshot) float64 {
	if snapshot == nil {
		return 0
	}
	return clampQualityRate(math.Max(0, snapshot.TransientPenalty-snapshot.RecoveryCredit))
}

func accountQualityRecoveryTieBreak(snapshot *AccountQualitySnapshot) float64 {
	if snapshot == nil {
		return 0
	}
	recovery := clampQualityRate(snapshot.RecoveryCredit)
	if snapshot.RecoveryFastStreak > 0 {
		recovery += math.Min(float64(snapshot.RecoveryFastStreak)*0.02, 0.08)
	}
	if snapshot.RecoverySuccessStreak > 0 {
		recovery += math.Min(float64(snapshot.RecoverySuccessStreak)*0.01, 0.04)
	}
	return recovery
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
	aPenalty := appliedAccountQualityPenalty(snapshots[a.ID])
	bPenalty := appliedAccountQualityPenalty(snapshots[b.ID])
	if aPenalty < bPenalty {
		return -1
	}
	if aPenalty > bPenalty {
		return 1
	}
	aRecovery := accountQualityRecoveryTieBreak(snapshots[a.ID])
	bRecovery := accountQualityRecoveryTieBreak(snapshots[b.ID])
	if aRecovery > bRecovery {
		return -1
	}
	if aRecovery < bRecovery {
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
	if appliedAccountQualityPenalty(snapshot) >= 0.30 {
		return true, "transient_penalty"
	}
	if snapshot.AuxiliaryTotalRequests == 0 &&
		snapshot.TotalRequests >= accountQualityMinSamples &&
		snapshot.RecentSuccessRate < accountQualityStickyEscapeSuccessRateThreshold {
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
		score, _ := effectiveAccountQualityScore(snapshot)
		logger.LegacyPrintf(logKey,
			"sticky account quality escape: account_id=%d reason=%s total=%d success_rate=%.4f ttft_5s_rate=%.4f ttft_10s_rate=%.4f score=%.4f penalty=%.4f recovery=%.4f",
			accountID,
			reason,
			snapshot.TotalRequests,
			snapshot.RecentSuccessRate,
			snapshot.TTFTLE5sRate,
			snapshot.TTFTLE10sRate,
			score,
			snapshot.TransientPenalty,
			snapshot.RecoveryCredit,
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
