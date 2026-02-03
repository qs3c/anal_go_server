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
	_ = jobQueue // TODO: 传给 analysis service

	// 初始化 WebSocket Hub
	wsHub := ws.NewHub()
	go wsHub.Run()
	log.Println("WebSocket hub started")

	// 启动 Redis 订阅，转发进度消息到 WebSocket
	subscriber := pubsub.NewSubscriber(rdb)
	go func() {
		ctx := context.Background()
		err := subscriber.Subscribe(ctx, func(msg *pubsub.ProgressMessage) {
			wsHub.SendToUser(msg.UserID, &ws.Message{
				Type: msg.Type,
				Data: msg,
			})
		})
		if err != nil {
			log.Printf("Redis subscriber error: %v", err)
		}
	}()
	log.Println("Redis subscriber started")

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
	analysisService := service.NewAnalysisService(analysisRepo, jobRepo, userRepo, quotaService, uploadService, ossClient, nil, cfg)
	communityService := service.NewCommunityService(analysisRepo, interactionRepo, cfg)
	commentService := service.NewCommentService(commentRepo, analysisRepo, userRepo, cfg)

	// 初始化 Handler
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	analysisHandler := handler.NewAnalysisHandler(analysisService)
	modelsHandler := handler.NewModelsHandler(cfg)
	websocketHandler := handler.NewWebSocketHandler(wsHub, cfg.JWT.Secret)
	communityHandler := handler.NewCommunityHandler(communityService)
	commentHandler := handler.NewCommentHandler(commentService)
	quotaHandler := handler.NewQuotaHandler(quotaService)
	uploadHandler := handler.NewUploadHandler(uploadService, cfg)

	// 初始化 Cron 服务
	cronService := cron.NewService(quotaService)
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

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
