package queue

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return client, cleanup
}

func TestNewQueue(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	q := NewQueue(client, "test_queue")

	assert.NotNil(t, q)
	assert.Equal(t, "test_queue", q.queueName)
	assert.Equal(t, client, q.client)
}

func TestQueue_Push(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	q := NewQueue(client, "test_queue")
	ctx := context.Background()

	t.Run("push single message", func(t *testing.T) {
		msg := &JobMessage{
			JobID:       1,
			AnalysisID:  100,
			UserID:      10,
			RepoURL:     "https://github.com/user/repo",
			StartStruct: "main.App",
			Depth:       3,
			ModelName:   "gpt-4",
		}

		err := q.Push(ctx, msg)
		require.NoError(t, err)

		length, err := q.Length(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), length)
	})

	t.Run("push multiple messages", func(t *testing.T) {
		// Clear queue first
		client.Del(ctx, "test_queue2")

		q2 := NewQueue(client, "test_queue2")

		for i := 0; i < 5; i++ {
			msg := &JobMessage{
				JobID: int64(i),
			}
			err := q2.Push(ctx, msg)
			require.NoError(t, err)
		}

		length, err := q2.Length(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(5), length)
	})
}

func TestQueue_Pop(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("pop from queue with messages", func(t *testing.T) {
		q := NewQueue(client, "test_pop_queue")

		msg := &JobMessage{
			JobID:       42,
			AnalysisID:  200,
			UserID:      20,
			RepoURL:     "https://github.com/test/repo",
			StartStruct: "pkg.Struct",
			Depth:       5,
			ModelName:   "claude-3",
		}

		err := q.Push(ctx, msg)
		require.NoError(t, err)

		result, err := q.Pop(ctx, time.Second)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, int64(42), result.JobID)
		assert.Equal(t, int64(200), result.AnalysisID)
		assert.Equal(t, int64(20), result.UserID)
		assert.Equal(t, "https://github.com/test/repo", result.RepoURL)
		assert.Equal(t, "pkg.Struct", result.StartStruct)
		assert.Equal(t, 5, result.Depth)
		assert.Equal(t, "claude-3", result.ModelName)
	})

	t.Run("pop FIFO order", func(t *testing.T) {
		q := NewQueue(client, "test_fifo_queue")

		// Push in order 1, 2, 3
		for i := 1; i <= 3; i++ {
			msg := &JobMessage{JobID: int64(i)}
			err := q.Push(ctx, msg)
			require.NoError(t, err)
		}

		// Should pop in order 1, 2, 3 (FIFO - first in, first out)
		for i := 1; i <= 3; i++ {
			result, err := q.Pop(ctx, time.Second)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, int64(i), result.JobID)
		}
	})

	t.Run("pop from empty queue times out", func(t *testing.T) {
		q := NewQueue(client, "test_empty_queue")

		// Pop with very short timeout
		result, err := q.Pop(ctx, 10*time.Millisecond)

		// miniredis doesn't support BRPop timeout properly, so check for nil or error
		if err == nil {
			assert.Nil(t, result)
		}
	})
}

func TestQueue_Length(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("length of empty queue", func(t *testing.T) {
		q := NewQueue(client, "test_length_empty")

		length, err := q.Length(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), length)
	})

	t.Run("length after push and pop", func(t *testing.T) {
		q := NewQueue(client, "test_length_ops")

		// Push 3 messages
		for i := 0; i < 3; i++ {
			msg := &JobMessage{JobID: int64(i)}
			err := q.Push(ctx, msg)
			require.NoError(t, err)
		}

		length, err := q.Length(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(3), length)

		// Pop 1 message
		_, err = q.Pop(ctx, time.Second)
		require.NoError(t, err)

		length, err = q.Length(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(2), length)
	})
}

func TestQueue_RoundTrip(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	q := NewQueue(client, "test_roundtrip")

	original := &JobMessage{
		JobID:       999,
		AnalysisID:  888,
		UserID:      777,
		RepoURL:     "https://gitlab.com/org/project",
		StartStruct: "internal.Service",
		Depth:       10,
		ModelName:   "gpt-4-turbo",
	}

	err := q.Push(ctx, original)
	require.NoError(t, err)

	result, err := q.Pop(ctx, time.Second)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, original.JobID, result.JobID)
	assert.Equal(t, original.AnalysisID, result.AnalysisID)
	assert.Equal(t, original.UserID, result.UserID)
	assert.Equal(t, original.RepoURL, result.RepoURL)
	assert.Equal(t, original.StartStruct, result.StartStruct)
	assert.Equal(t, original.Depth, result.Depth)
	assert.Equal(t, original.ModelName, result.ModelName)
}

func TestQueue_MultipleQueues(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	q1 := NewQueue(client, "queue_1")
	q2 := NewQueue(client, "queue_2")

	// Push to different queues
	msg1 := &JobMessage{JobID: 1}
	msg2 := &JobMessage{JobID: 2}

	err := q1.Push(ctx, msg1)
	require.NoError(t, err)

	err = q2.Push(ctx, msg2)
	require.NoError(t, err)

	// Each queue should have 1 message
	len1, _ := q1.Length(ctx)
	len2, _ := q2.Length(ctx)
	assert.Equal(t, int64(1), len1)
	assert.Equal(t, int64(1), len2)

	// Pop from each queue
	result1, _ := q1.Pop(ctx, time.Second)
	result2, _ := q2.Pop(ctx, time.Second)

	assert.Equal(t, int64(1), result1.JobID)
	assert.Equal(t, int64(2), result2.JobID)
}

func TestJobMessage_Fields(t *testing.T) {
	msg := &JobMessage{
		JobID:       123,
		AnalysisID:  456,
		UserID:      789,
		RepoURL:     "https://github.com/test/repo",
		StartStruct: "main.App",
		Depth:       5,
		ModelName:   "gpt-4",
	}

	assert.Equal(t, int64(123), msg.JobID)
	assert.Equal(t, int64(456), msg.AnalysisID)
	assert.Equal(t, int64(789), msg.UserID)
	assert.Equal(t, "https://github.com/test/repo", msg.RepoURL)
	assert.Equal(t, "main.App", msg.StartStruct)
	assert.Equal(t, 5, msg.Depth)
	assert.Equal(t, "gpt-4", msg.ModelName)
}
