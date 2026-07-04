package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestGatewayCache_DeleteSessionAccountIDsByGroupPrefix(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	cache := &gatewayCache{
		rdb: redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	}

	require.NoError(t, cache.SetSessionAccountID(ctx, 7, "openai:session-a", 1, time.Hour))
	require.NoError(t, cache.SetSessionAccountID(ctx, 7, "openai:response:a", 1, time.Hour))
	require.NoError(t, cache.SetSessionAccountID(ctx, 7, "gemini:session-a", 2, time.Hour))
	require.NoError(t, cache.SetSessionAccountID(ctx, 8, "openai:session-a", 3, time.Hour))

	require.NoError(t, cache.DeleteSessionAccountIDsByGroupPrefix(ctx, 7, "openai:"))

	_, err := cache.GetSessionAccountID(ctx, 7, "openai:session-a")
	require.Error(t, err)
	_, err = cache.GetSessionAccountID(ctx, 7, "openai:response:a")
	require.Error(t, err)
	accountID, err := cache.GetSessionAccountID(ctx, 7, "gemini:session-a")
	require.NoError(t, err)
	require.Equal(t, int64(2), accountID)
	accountID, err = cache.GetSessionAccountID(ctx, 8, "openai:session-a")
	require.NoError(t, err)
	require.Equal(t, int64(3), accountID)
}
