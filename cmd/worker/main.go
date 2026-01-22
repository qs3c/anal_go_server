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
	"github.com/qs3c/anal_go_server/internal/pkg/queue"
	"github.com/qs3c/anal_go_server/internal/repository"
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

	// 初始化 Queue
	jobQueue := queue.NewQueue(rdb, cfg.Queue.AnalysisQueue)

	// 初始化 Repository
	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	userRepo := repository.NewUserRepository(db)
	_ = userRepo // TODO: 用于后续配额处理

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
					processJob(ctx, msg, analysisRepo, jobRepo, cfg)
				}
			}
		}(i)
	}

	// 等待 context 取消
	<-ctx.Done()
	log.Println("Worker shutdown complete")
}

func processJob(
	ctx context.Context,
	msg *queue.JobMessage,
	analysisRepo *repository.AnalysisRepository,
	jobRepo *repository.JobRepository,
	cfg *config.Config,
) {
	job, err := jobRepo.GetByID(msg.JobID)
	if err != nil {
		log.Printf("Failed to get job %d: %v", msg.JobID, err)
		return
	}

	// 更新状态为处理中
	now := time.Now()
	job.Status = "processing"
	job.StartedAt = &now
	jobRepo.Update(job)

	analysisRepo.UpdateStatus(job.AnalysisID, "analyzing")

	// TODO: 实际的分析逻辑
	// 1. Clone 仓库
	// 2. 调用 anal_go_agent 分析
	// 3. 上传结果到 OSS
	// 4. 更新数据库

	// 模拟处理
	time.Sleep(2 * time.Second)

	// 标记完成（这里是模拟）
	job.Status = "completed"
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	job.ElapsedSeconds = int(completedAt.Sub(*job.StartedAt).Seconds())
	jobRepo.Update(job)

	analysisRepo.UpdateStatus(job.AnalysisID, "completed")

	log.Printf("Job %d completed in %d seconds", job.ID, job.ElapsedSeconds)
}
