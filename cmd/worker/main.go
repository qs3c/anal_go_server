package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/database"
	"github.com/qs3c/anal_go_server/internal/pkg/oss"
	"github.com/qs3c/anal_go_server/internal/pkg/pubsub"
	"github.com/qs3c/anal_go_server/internal/pkg/queue"
	"github.com/qs3c/anal_go_server/internal/repository"
	"github.com/qs3c/anal_go_server/internal/worker"
)

func main() {
	// 加载配置
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库
	db, err := database.NewMySQL(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}
	log.Println("Database connected")

	// 初始化 Redis
	rdb, err := database.NewRedis(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect redis: %v", err)
	}
	log.Println("Redis connected")

	// 初始化 OSS（可选）
	var ossClient *oss.Client
	if cfg.OSS.Endpoint != "" && cfg.OSS.AccessKeyID != "" {
		ossClient, err = oss.NewClient(&cfg.OSS)
		if err != nil {
			log.Printf("Warning: Failed to init OSS client: %v", err)
		} else {
			log.Println("OSS client initialized")
		}
	}

	// 初始化 Queue 和 Pub/Sub
	jobQueue := queue.NewQueue(rdb, cfg.Queue.AnalysisQueue)
	publisher := pubsub.NewPublisher(rdb)

	// 初始化 Repository
	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)

	// 创建任务处理器
	processor := worker.NewProcessor(jobRepo, analysisRepo, ossClient, publisher, cfg)

	// 创建 context 用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	log.Printf("Worker started, max workers: %d", cfg.Queue.MaxWorkers)

	// 启动 worker 循环
	for i := 0; i < cfg.Queue.MaxWorkers; i++ {
		go func(workerID int) {
			for {
				select {
				case <-ctx.Done():
					log.Printf("Worker %d shutting down", workerID)
					return
				default:
					// 从队列获取任务
					msg, err := jobQueue.Pop(ctx, 5*time.Second)
					if err != nil {
						if ctx.Err() != nil {
							return
						}
						log.Printf("Worker %d: failed to pop job: %v", workerID, err)
						continue
					}

					if msg == nil {
						continue // 超时，继续等待
					}

					log.Printf("Worker %d: processing job %d", workerID, msg.JobID)
					if err := processor.Process(ctx, msg); err != nil {
						log.Printf("Worker %d: job %d failed: %v", workerID, msg.JobID, err)
					}
				}
			}
		}(i)
	}

	// 等待 context 取消
	<-ctx.Done()
	log.Println("Worker shutdown complete")
}
