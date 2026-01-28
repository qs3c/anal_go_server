package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
)

const (
	ChannelAnalysisProgress = "analysis_progress"
)

// ProgressMessage 进度消息
type ProgressMessage struct {
	Type       string `json:"type"`
	UserID     int64  `json:"user_id"`
	AnalysisID int64  `json:"analysis_id"`
	JobID      int64  `json:"job_id"`
	Status     string `json:"status"`
	Step       string `json:"step"`
	Progress   int    `json:"progress"`
	Message    string `json:"message,omitempty"`
	Error      string `json:"error,omitempty"`
}

// 进度阶段常量
const (
	StepCloning   = "cloning"
	StepParsing   = "parsing"
	StepAnalyzing = "analyzing"
	StepUploading = "uploading"
	StepDone      = "done"
)

// 阶段对应的进度百分比
var StepProgress = map[string]int{
	StepCloning:   20,
	StepParsing:   40,
	StepAnalyzing: 60,
	StepUploading: 80,
	StepDone:      100,
}

// 阶段对应的消息
var StepMessages = map[string]string{
	StepCloning:   "正在克隆仓库",
	StepParsing:   "正在解析项目结构",
	StepAnalyzing: "正在进行 AI 分析",
	StepUploading: "正在上传结果",
	StepDone:      "分析完成",
}

// Publisher Redis 发布者
type Publisher struct {
	client *redis.Client
}

// NewPublisher 创建发布者
func NewPublisher(client *redis.Client) *Publisher {
	return &Publisher{client: client}
}

// PublishProgress 发布进度消息
func (p *Publisher) PublishProgress(ctx context.Context, msg *ProgressMessage) error {
	msg.Type = "job_progress"

	// 自动填充进度和消息
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

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal progress message: %w", err)
	}

	return p.client.Publish(ctx, ChannelAnalysisProgress, data).Err()
}

// Subscriber Redis 订阅者
type Subscriber struct {
	client *redis.Client
}

// NewSubscriber 创建订阅者
func NewSubscriber(client *redis.Client) *Subscriber {
	return &Subscriber{client: client}
}

// Subscribe 订阅进度消息
func (s *Subscriber) Subscribe(ctx context.Context, handler func(*ProgressMessage)) error {
	pubsub := s.client.Subscribe(ctx, ChannelAnalysisProgress)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}

			var progressMsg ProgressMessage
			if err := json.Unmarshal([]byte(msg.Payload), &progressMsg); err != nil {
				continue // 忽略解析错误
			}

			handler(&progressMsg)
		}
	}
}
