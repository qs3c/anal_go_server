package api

import (
	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/api/handler"
	"github.com/qs3c/anal_go_server/internal/api/middleware"
)

type Router struct {
	authHandler      *handler.AuthHandler
	userHandler      *handler.UserHandler
	analysisHandler  *handler.AnalysisHandler
	modelsHandler    *handler.ModelsHandler
	websocketHandler *handler.WebSocketHandler
	communityHandler *handler.CommunityHandler
	commentHandler   *handler.CommentHandler
	quotaHandler     *handler.QuotaHandler
	uploadHandler    *handler.UploadHandler
	cfg              *config.Config
}

func NewRouter(
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	analysisHandler *handler.AnalysisHandler,
	modelsHandler *handler.ModelsHandler,
	websocketHandler *handler.WebSocketHandler,
	communityHandler *handler.CommunityHandler,
	commentHandler *handler.CommentHandler,
	quotaHandler *handler.QuotaHandler,
	uploadHandler *handler.UploadHandler,
	cfg *config.Config,
) *Router {
	return &Router{
		authHandler:      authHandler,
		userHandler:      userHandler,
		analysisHandler:  analysisHandler,
		modelsHandler:    modelsHandler,
		websocketHandler: websocketHandler,
		communityHandler: communityHandler,
		commentHandler:   commentHandler,
		quotaHandler:     quotaHandler,
		uploadHandler:    uploadHandler,
		cfg:              cfg,
	}
}

func (r *Router) Setup() *gin.Engine {
	if r.cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.CORS(r.cfg.CORS))

	api := engine.Group("/api/v1")
	{
		// WebSocket
		api.GET("/ws", r.websocketHandler.Handle)

		// 公开接口 - 认证
		auth := api.Group("/auth")
		{
			auth.POST("/register", r.authHandler.Register)
			auth.POST("/login", r.authHandler.Login)
			auth.POST("/verify-email", r.authHandler.VerifyEmail)
			auth.GET("/github", r.authHandler.GithubAuth)
			auth.GET("/github/callback", r.authHandler.GithubCallback)
			// TODO: WeChat OAuth
		}

		// 公开接口 - 模型
		api.GET("/models", r.modelsHandler.List)

		// 需要认证的接口
		authenticated := api.Group("")
		authenticated.Use(middleware.Auth(r.cfg.JWT.Secret))
		{
			// 用户
			user := authenticated.Group("/user")
			{
				user.GET("/profile", r.userHandler.GetProfile)
				user.PUT("/profile", r.userHandler.UpdateProfile)
				user.POST("/avatar", r.userHandler.UploadAvatar)
				user.GET("/quota", r.quotaHandler.GetQuota)
			}

			// 分析
			analyses := authenticated.Group("/analyses")
			{
				analyses.POST("", r.analysisHandler.Create)
				analyses.GET("", r.analysisHandler.List)
				analyses.GET("/:id", r.analysisHandler.Get)
				analyses.PUT("/:id", r.analysisHandler.Update)
				analyses.DELETE("/:id", r.analysisHandler.Delete)
				analyses.POST("/:id/share", r.analysisHandler.Share)
				analyses.DELETE("/:id/share", r.analysisHandler.Unshare)
				analyses.GET("/:id/job-status", r.analysisHandler.GetJobStatus)
				analyses.GET("/:id/diagram", r.analysisHandler.GetDiagram)
			}

			// 上传相关
			authenticated.POST("/upload/parse", r.uploadHandler.Parse)
		}

		// 公开接口 - 社区（可选认证）
		community := api.Group("/community")
		community.Use(middleware.OptionalAuth(r.cfg.JWT.Secret))
		{
			community.GET("/analyses", r.communityHandler.List)
			community.GET("/analyses/:id", r.communityHandler.Get)
		}

		// 社区互动（需要认证）
		communityAuth := api.Group("/community")
		communityAuth.Use(middleware.Auth(r.cfg.JWT.Secret))
		{
			communityAuth.POST("/analyses/:id/like", r.communityHandler.Like)
			communityAuth.DELETE("/analyses/:id/like", r.communityHandler.Unlike)
			communityAuth.POST("/analyses/:id/bookmark", r.communityHandler.Bookmark)
			communityAuth.DELETE("/analyses/:id/bookmark", r.communityHandler.Unbookmark)
		}

		// 评论 - 公开读取（可选认证）
		commentsPublic := api.Group("/analyses")
		commentsPublic.Use(middleware.OptionalAuth(r.cfg.JWT.Secret))
		{
			commentsPublic.GET("/:id/comments", r.commentHandler.List)
		}

		// 评论 - 需要认证
		commentsAuth := api.Group("")
		commentsAuth.Use(middleware.Auth(r.cfg.JWT.Secret))
		{
			commentsAuth.POST("/analyses/:id/comments", r.commentHandler.Create)
			commentsAuth.DELETE("/comments/:id", r.commentHandler.Delete)
		}
	}

	return engine
}
