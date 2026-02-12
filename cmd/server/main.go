package main

import (
	"context"
	"fmt"
	"log"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/api"
	"github.com/qs3c/anal_go_server/internal/api/handler"
	"github.com/qs3c/anal_go_server/internal/database"
	"github.com/qs3c/anal_go_server/internal/pkg/cron"
	"github.com/qs3c/anal_go_server/internal/pkg/oauth"
	"github.com/qs3c/anal_go_server/internal/pkg/oss"
	"github.com/qs3c/anal_go_server/internal/pkg/pubsub"
	"github.com/qs3c/anal_go_server/internal/pkg/queue"
	"github.com/qs3c/anal_go_server/internal/pkg/ws"
	"github.com/qs3c/anal_go_server/internal/repository"
	"github.com/qs3c/anal_go_server/internal/service"
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

	// 初始化 WebSocket Hub
	wsHub := ws.NewHub()
	go wsHub.Run()
	log.Println("WebSocket hub started")

	// 初始化 OSS 客户端（可选）
	var ossClient *oss.Client
	if cfg.OSS.Endpoint != "" {
		var err error
		ossClient, err = oss.NewClient(&cfg.OSS)
		if err != nil {
			log.Printf("Warning: Failed to initialize OSS client: %v", err)
		} else {
			log.Println("OSS client initialized")
		}
	}

	// 初始化 Repository
	userRepo := repository.NewUserRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	jobRepo := repository.NewJobRepository(db)
	interactionRepo := repository.NewInteractionRepository(db)
	commentRepo := repository.NewCommentRepository(db)

	// 初始化 Service
	authService := service.NewAuthService(userRepo, cfg)
	userService := service.NewUserService(userRepo, ossClient, cfg)
	quotaService := service.NewQuotaService(userRepo, cfg)
	uploadService := service.NewUploadService(cfg)
	analysisService := service.NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, uploadService, ossClient, jobQueue, cfg)
	communityService := service.NewCommunityService(analysisRepo, interactionRepo, cfg)
	commentService := service.NewCommentService(commentRepo, analysisRepo, userRepo, cfg)

	// 初始化 OAuth StateStore
	stateStore := oauth.NewStateStore(rdb)

	// 初始化 Handler
	authHandler := handler.NewAuthHandler(authService, stateStore)
	userHandler := handler.NewUserHandler(userService)
	analysisHandler := handler.NewAnalysisHandler(analysisService)
	modelsHandler := handler.NewModelsHandler(cfg)
	websocketHandler := handler.NewWebSocketHandler(wsHub, cfg.JWT.Secret)
	communityHandler := handler.NewCommunityHandler(communityService)
	commentHandler := handler.NewCommentHandler(commentService)
	quotaHandler := handler.NewQuotaHandler(quotaService)
	uploadHandler := handler.NewUploadHandler(uploadService, cfg)

	// 初始化 Cron 服务
	cronService := cron.NewService(quotaService, analysisRepo, cfg.Upload.TempDir, cfg.Upload.ExpireHours)
	cronService.Start()
	log.Println("Cron service started")

	// 初始化 Router
	router := api.NewRouter(
		authHandler,
		userHandler,
		analysisHandler,
		modelsHandler,
		websocketHandler,
		communityHandler,
		commentHandler,
		quotaHandler,
		uploadHandler,
		cfg,
	)
	engine := router.Setup()

	// 启动 Redis 订阅，转发进度消息到 WebSocket (在所有依赖初始化之后)
	subscriber := pubsub.NewSubscriber(rdb)
	go func() {
		ctx := context.Background()
		log.Println("Starting Redis subscription...")
		err := subscriber.Subscribe(ctx, func(msg *pubsub.ProgressMessage) {
			log.Printf("📨 [Redis] Received progress: user=%d, analysis=%d, job=%d, status=%s, step=%s",
				msg.UserID, msg.AnalysisID, msg.JobID, msg.Status, msg.Step)

			// 转换消息格式以匹配前端期望
			msgType := "analysis_progress"
			if msg.Status == "completed" {
				msgType = "analysis_completed"
			} else if msg.Status == "failed" {
				msgType = "analysis_failed"
			}

			// 构建前端期望的数据格式
			data := map[string]interface{}{
				"job_id":         msg.JobID,
				"analysis_id":    msg.AnalysisID,
				"status":         msg.Status,
				"current_step":   msg.Step,
				"error_message":  msg.Error,
			}

			// 如果任务完成，获取 diagram_oss_url
			if msg.Status == "completed" {
				if analysis, err := analysisRepo.GetByID(msg.AnalysisID); err == nil && analysis != nil {
					data["diagram_oss_url"] = analysis.DiagramOSSURL
				}
			}

			log.Printf("📤 [WebSocket] Sending to user %d: type=%s", msg.UserID, msgType)
			wsHub.SendToUser(msg.UserID, &ws.Message{
				Type: msgType,
				Data: data,
			})
		})
		if err != nil {
			log.Printf("❌ Redis subscriber error: %v", err)
		}
		log.Println("⚠️ Redis subscriber goroutine exited")
	}()
	log.Println("Redis subscriber started")

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
