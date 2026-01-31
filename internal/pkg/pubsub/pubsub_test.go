package pubsub

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepProgress(t *testing.T) {
	// Verify all steps have progress values
	steps := []string{StepCloning, StepParsing, StepAnalyzing, StepUploading, StepDone}

	for _, step := range steps {
		progress, ok := StepProgress[step]
		assert.True(t, ok, "Step %s should have progress value", step)
		assert.Greater(t, progress, 0, "Progress for %s should be > 0", step)
		assert.LessOrEqual(t, progress, 100, "Progress for %s should be <= 100", step)
	}

	// Verify progress is monotonically increasing
	assert.Less(t, StepProgress[StepCloning], StepProgress[StepParsing])
	assert.Less(t, StepProgress[StepParsing], StepProgress[StepAnalyzing])
	assert.Less(t, StepProgress[StepAnalyzing], StepProgress[StepUploading])
	assert.Less(t, StepProgress[StepUploading], StepProgress[StepDone])
	assert.Equal(t, 100, StepProgress[StepDone])
}

func TestStepMessages(t *testing.T) {
	// Verify all steps have messages
	steps := []string{StepCloning, StepParsing, StepAnalyzing, StepUploading, StepDone}

	for _, step := range steps {
		msg, ok := StepMessages[step]
		assert.True(t, ok, "Step %s should have message", step)
		assert.NotEmpty(t, msg, "Message for %s should not be empty", step)
	}
}

func TestProgressMessage_JSON(t *testing.T) {
	msg := &ProgressMessage{
		Type:       "job_progress",
		UserID:     1,
		AnalysisID: 2,
		JobID:      3,
		Status:     "processing",
		Step:       StepAnalyzing,
		Progress:   60,
		Message:    "Analyzing",
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	require.NoError(t, err)

	// Verify snake_case keys
	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "user_id")
	assert.Contains(t, raw, "analysis_id")
	assert.Contains(t, raw, "job_id")

	// Unmarshal back
	var decoded ProgressMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, msg.UserID, decoded.UserID)
	assert.Equal(t, msg.AnalysisID, decoded.AnalysisID)
	assert.Equal(t, msg.JobID, decoded.JobID)
}

func TestProgressMessage_OmitEmpty(t *testing.T) {
	msg := &ProgressMessage{
		UserID: 1,
		Status: "processing",
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	// Message and Error should be omitted when empty
	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	_, hasMessage := raw["message"]
	_, hasError := raw["error"]
	assert.False(t, hasMessage, "empty message should be omitted")
	assert.False(t, hasError, "empty error should be omitted")
}

// Integration tests with real Redis (skip if not available)
func TestPublisherSubscriber_Integration(t *testing.T) {
	// Try to connect to Redis
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
	}
	defer client.Close()

	publisher := NewPublisher(client)
	subscriber := NewSubscriber(client)

	// Use a unique channel to avoid interference
	testCtx, testCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer testCancel()

	received := make(chan *ProgressMessage, 1)

	// Start subscriber in goroutine
	go func() {
		subscriber.Subscribe(testCtx, func(msg *ProgressMessage) {
			received <- msg
		})
	}()

	// Give subscriber time to connect
	time.Sleep(100 * time.Millisecond)

	// Publish a message
	msg := &ProgressMessage{
		UserID:     123,
		AnalysisID: 456,
		JobID:      789,
		Status:     "processing",
		Step:       StepAnalyzing,
	}

	err := publisher.PublishProgress(testCtx, msg)
	require.NoError(t, err)

	// Wait for message
	select {
	case receivedMsg := <-received:
		assert.Equal(t, msg.UserID, receivedMsg.UserID)
		assert.Equal(t, msg.AnalysisID, receivedMsg.AnalysisID)
		assert.Equal(t, msg.JobID, receivedMsg.JobID)
		assert.Equal(t, "job_progress", receivedMsg.Type)
		assert.Equal(t, 60, receivedMsg.Progress) // Auto-filled from step
		assert.NotEmpty(t, receivedMsg.Message)   // Auto-filled from step
	case <-testCtx.Done():
		t.Fatal("Timeout waiting for message")
	}
}

func TestPublisher_AutoFillProgress(t *testing.T) {
	// This test verifies the auto-fill logic without actually publishing
	msg := &ProgressMessage{
		UserID: 1,
		Step:   StepParsing,
	}

	// Simulate the auto-fill logic from PublishProgress
	if msg.Progress == 0 && msg.Step != "" {
		if progress, ok := StepProgress[msg.Step]; ok {
			msg.Progress = progress
		}
	}
	if msg.Message == "" && msg.Step != "" {
		if message, ok := StepMessages[msg.Step]; ok {
			msg.Message = message
		}
	}

	assert.Equal(t, 40, msg.Progress)
	assert.Equal(t, StepMessages[StepParsing], msg.Message)
}

func TestNewPublisher(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	publisher := NewPublisher(client)
	assert.NotNil(t, publisher)
}

func TestNewSubscriber(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()

	subscriber := NewSubscriber(client)
	assert.NotNil(t, subscriber)
}
