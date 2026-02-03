package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type Queue struct {
	client    *redis.Client
	queueName string
}

type JobMessage struct {
	JobID       int64  `json:"job_id"`
	AnalysisID  int64  `json:"analysis_id"`
	UserID      int64  `json:"user_id"`
	SourceType  string `json:"source_type"`
	RepoURL     string `json:"repo_url"`
	UploadID    string `json:"upload_id"`
	StartFile   string `json:"start_file"`
	StartStruct string `json:"start_struct"`
	Depth       int    `json:"depth"`
	ModelName   string `json:"model_name"`
}

func NewQueue(client *redis.Client, queueName string) *Queue {
	return &Queue{
		client:    client,
		queueName: queueName,
	}
}

// Push 将任务加入队列
func (q *Queue) Push(ctx context.Context, msg *JobMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return q.client.LPush(ctx, q.queueName, data).Err()
}

// Pop 从队列获取任务（阻塞）
func (q *Queue) Pop(ctx context.Context, timeout time.Duration) (*JobMessage, error) {
	result, err := q.client.BRPop(ctx, timeout, q.queueName).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 超时，无任务
		}
		return nil, fmt.Errorf("failed to pop from queue: %w", err)
	}

	if len(result) < 2 {
		return nil, nil
	}

	var msg JobMessage
	if err := json.Unmarshal([]byte(result[1]), &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &msg, nil
}

// Length 获取队列长度
func (q *Queue) Length(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, q.queueName).Result()
}
