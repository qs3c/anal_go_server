package oauth

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return client, cleanup
}

func TestStateStore_GenerateState(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	store := NewStateStore(rdb)
	ctx := context.Background()

	state, err := store.GenerateState(ctx, "http://localhost:3000")
	require.NoError(t, err)
	assert.Len(t, state, 64) // 32 bytes = 64 hex chars
}

func TestStateStore_ValidateState_Success(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	store := NewStateStore(rdb)
	ctx := context.Background()

	redirectURI := "http://localhost:3000"
	state, err := store.GenerateState(ctx, redirectURI)
	require.NoError(t, err)

	// Validate should return the redirect URI
	result, err := store.ValidateState(ctx, state)
	require.NoError(t, err)
	assert.Equal(t, redirectURI, result)
}

func TestStateStore_ValidateState_Consumed(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	store := NewStateStore(rdb)
	ctx := context.Background()

	state, err := store.GenerateState(ctx, "http://localhost:3000")
	require.NoError(t, err)

	// First validation should succeed
	_, err = store.ValidateState(ctx, state)
	require.NoError(t, err)

	// Second validation should fail (state consumed)
	_, err = store.ValidateState(ctx, state)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or expired")
}

func TestStateStore_ValidateState_Invalid(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	store := NewStateStore(rdb)
	ctx := context.Background()

	_, err := store.ValidateState(ctx, "invalid-state")
	assert.Error(t, err)
}

func TestStateStore_ValidateState_Empty(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	store := NewStateStore(rdb)
	ctx := context.Background()

	_, err := store.ValidateState(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty state")
}

func TestStateStore_GenerateState_Unique(t *testing.T) {
	rdb, cleanup := setupTestRedis(t)
	defer cleanup()

	store := NewStateStore(rdb)
	ctx := context.Background()

	states := make(map[string]bool)
	for i := 0; i < 100; i++ {
		state, err := store.GenerateState(ctx, "http://localhost:3000")
		require.NoError(t, err)
		assert.False(t, states[state], "duplicate state generated")
		states[state] = true
	}
}
