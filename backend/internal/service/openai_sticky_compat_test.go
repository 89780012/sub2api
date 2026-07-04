package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestGetStickySessionAccountID_FallbackToLegacyKey(t *testing.T) {
	beforeFallbackTotal, beforeFallbackHit, _ := openAIStickyCompatStats()

	cache := &stubGatewayCache{
		sessionBindings: map[string]int64{
			"openai:legacy-hash": 42,
		},
	}
	svc := &OpenAIGatewayService{
		cache: cache,
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				OpenAIWS: config.GatewayOpenAIWSConfig{
					SessionHashReadOldFallback: true,
				},
			},
		},
	}

	ctx := withOpenAILegacySessionHash(context.Background(), "legacy-hash")
	accountID, err := svc.getStickySessionAccountID(ctx, nil, "new-hash")
	require.NoError(t, err)
	require.Equal(t, int64(42), accountID)

	afterFallbackTotal, afterFallbackHit, _ := openAIStickyCompatStats()
	require.Equal(t, beforeFallbackTotal+1, afterFallbackTotal)
	require.Equal(t, beforeFallbackHit+1, afterFallbackHit)
}

func TestSetStickySessionAccountID_DualWriteOldEnabled(t *testing.T) {
	_, _, beforeDualWriteTotal := openAIStickyCompatStats()

	cache := &stubGatewayCache{sessionBindings: map[string]int64{}}
	svc := &OpenAIGatewayService{
		cache: cache,
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				OpenAIWS: config.GatewayOpenAIWSConfig{
					SessionHashDualWriteOld: true,
				},
			},
		},
	}

	ctx := withOpenAILegacySessionHash(context.Background(), "legacy-hash")
	err := svc.setStickySessionAccountID(ctx, nil, "new-hash", 9, openaiStickySessionTTL)
	require.NoError(t, err)
	require.Equal(t, int64(9), cache.sessionBindings["openai:new-hash"])
	require.Equal(t, int64(9), cache.sessionBindings["openai:legacy-hash"])

	_, _, afterDualWriteTotal := openAIStickyCompatStats()
	require.Equal(t, beforeDualWriteTotal+1, afterDualWriteTotal)
}

func TestSetStickySessionAccountID_DualWriteOldDisabled(t *testing.T) {
	cache := &stubGatewayCache{sessionBindings: map[string]int64{}}
	svc := &OpenAIGatewayService{
		cache: cache,
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				OpenAIWS: config.GatewayOpenAIWSConfig{
					SessionHashDualWriteOld: false,
				},
			},
		},
	}

	ctx := withOpenAILegacySessionHash(context.Background(), "legacy-hash")
	err := svc.setStickySessionAccountID(ctx, nil, "new-hash", 9, openaiStickySessionTTL)
	require.NoError(t, err)
	require.Equal(t, int64(9), cache.sessionBindings["openai:new-hash"])
	_, exists := cache.sessionBindings["openai:legacy-hash"]
	require.False(t, exists)
}

func TestSnapshotOpenAICompatibilityFallbackMetrics(t *testing.T) {
	before := SnapshotOpenAICompatibilityFallbackMetrics()

	ctx := context.WithValue(context.Background(), ctxkey.ThinkingEnabled, true)
	_, _ = ThinkingEnabledFromContext(ctx)

	after := SnapshotOpenAICompatibilityFallbackMetrics()
	require.GreaterOrEqual(t, after.MetadataLegacyFallbackTotal, before.MetadataLegacyFallbackTotal+1)
	require.GreaterOrEqual(t, after.MetadataLegacyFallbackThinkingEnabledTotal, before.MetadataLegacyFallbackThinkingEnabledTotal+1)
}

func TestClearOpenAIStickyBindingsForGroup_ClearsSessionAndResponseSticky(t *testing.T) {
	ctx := context.Background()
	groupID := int64(301)
	cache := &stubGatewayCache{
		sessionBindings: map[string]int64{
			"openai:session-a":     1,
			"openai:legacy-a":      1,
			"openai:response:dead": 1,
			"gemini:session-a":     2,
			"plain-session-a":      3,
		},
	}
	svc := &OpenAIGatewayService{cache: cache}
	store := svc.getOpenAIWSStateStore()
	require.NoError(t, store.BindResponseAccount(ctx, groupID, "resp_local", 9, time.Hour))

	svc.ClearOpenAIStickyBindingsForGroup(ctx, groupID)

	for key := range cache.sessionBindings {
		require.NotContains(t, key, "openai:", "OpenAI sticky key should be cleared")
	}
	require.Equal(t, int64(2), cache.sessionBindings["gemini:session-a"])
	require.Equal(t, int64(3), cache.sessionBindings["plain-session-a"])
	accountID, err := store.GetResponseAccount(ctx, groupID, "resp_local")
	require.NoError(t, err)
	require.Zero(t, accountID)
}

type adminPrimaryStickyGroupRepo struct {
	GroupRepository
	group   *Group
	updated *Group
}

func (r *adminPrimaryStickyGroupRepo) GetByID(context.Context, int64) (*Group, error) {
	return r.group, nil
}

func (r *adminPrimaryStickyGroupRepo) Update(_ context.Context, group *Group) error {
	r.updated = group
	return nil
}

type adminPrimaryStickyAccountRepo struct {
	AccountRepository
	account *Account
}

func (r adminPrimaryStickyAccountRepo) GetByID(context.Context, int64) (*Account, error) {
	return r.account, nil
}

type stickyClearRuntimeBlocker struct {
	clearedGroups []int64
}

func (b *stickyClearRuntimeBlocker) BlockAccountScheduling(*Account, time.Time, string) {}

func (b *stickyClearRuntimeBlocker) ClearAccountSchedulingBlock(int64) {}

func (b *stickyClearRuntimeBlocker) ClearOpenAIStickyBindingsForGroup(_ context.Context, groupID int64) {
	b.clearedGroups = append(b.clearedGroups, groupID)
}

func TestAdminUpdateGroupPrimaryAccount_ClearsOpenAIStickyOnPrimarySwitch(t *testing.T) {
	groupID := int64(302)
	oldID := int64(10)
	newID := int64(11)
	groupRepo := &adminPrimaryStickyGroupRepo{
		group: &Group{
			ID:                     groupID,
			Platform:               PlatformOpenAI,
			PrimaryAccountMode:     GroupPrimaryAccountModeManual,
			ManualPrimaryAccountID: &oldID,
			ActivePrimaryAccountID: &oldID,
		},
	}
	clearer := &stickyClearRuntimeBlocker{}
	svc := &adminServiceImpl{
		groupRepo: groupRepo,
		accountRepo: adminPrimaryStickyAccountRepo{account: &Account{
			ID:       newID,
			GroupIDs: []int64{groupID},
		}},
		runtimeBlocker: clearer,
	}

	group, err := svc.UpdateGroupPrimaryAccount(context.Background(), groupID, &UpdateGroupPrimaryAccountInput{
		PrimaryAccountMode:     GroupPrimaryAccountModeManual,
		ManualPrimaryAccountID: &newID,
	})

	require.NoError(t, err)
	require.NotNil(t, group)
	require.NotNil(t, groupRepo.updated)
	require.Equal(t, []int64{groupID}, clearer.clearedGroups)
}

type openAIPrimaryStickyGroupRepo struct {
	GroupRepository
	updated *Group
}

func (r *openAIPrimaryStickyGroupRepo) Update(_ context.Context, group *Group) error {
	r.updated = group
	return nil
}

func TestOpenAIPrimaryPromotion_ClearsOpenAIStickyOnPrimarySwitch(t *testing.T) {
	ctx := context.Background()
	groupID := int64(303)
	oldID := int64(20)
	newID := int64(21)
	group := &Group{
		ID:                     groupID,
		Platform:               PlatformOpenAI,
		PrimaryAccountMode:     GroupPrimaryAccountModeAuto,
		ActivePrimaryAccountID: &oldID,
	}
	cache := &stubGatewayCache{sessionBindings: map[string]int64{
		"openai:session-b": newID,
		"gemini:session-b": oldID,
	}}
	groupRepo := &openAIPrimaryStickyGroupRepo{}
	svc := &OpenAIGatewayService{
		groupRepo: groupRepo,
		cache:     cache,
	}

	svc.persistPrimaryPromotionFromSelection(ctx, group, newID, true)

	require.NotNil(t, groupRepo.updated)
	require.Equal(t, newID, *groupRepo.updated.ActivePrimaryAccountID)
	_, openAIExists := cache.sessionBindings["openai:session-b"]
	require.False(t, openAIExists)
	require.Equal(t, oldID, cache.sessionBindings["gemini:session-b"])
}
