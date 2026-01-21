package api

import (
	"github.com/gin-gonic/gin"

	"github.com/qs3c/anal_go_server/config"
	"github.com/qs3c/anal_go_server/internal/api/handler"
	"github.com/qs3c/anal_go_server/internal/api/middleware"
)

type Router struct {
	authHandler *handler.AuthHandler
	cfg         *config.Config
}

func NewRouter(authHandler *handler.AuthHandler, cfg *config.Config) *Router {
	return &Router{
		authHandler: authHandler,
		cfg:         cfg,
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
		// 公开接口 - 认证
		auth := api.Group("/auth")
		{
			auth.POST("/register", r.authHandler.Register)
			auth.POST("/login", r.authHandler.Login)
			auth.POST("/verify-email", r.authHandler.VerifyEmail)
			// TODO: GitHub OAuth
			// TODO: WeChat OAuth
		}

		// 需要认证的接口
		authenticated := api.Group("")
		authenticated.Use(middleware.Auth(r.cfg.JWT.Secret))
		{
			// TODO: User APIs
			// TODO: Analysis APIs
			// TODO: Comment APIs
			// TODO: Quota APIs
		}

		// 公开接口 - 社区（可选认证）
		community := api.Group("/community")
		community.Use(middleware.OptionalAuth(r.cfg.JWT.Secret))
		{
			// TODO: Community APIs
		}

		// 公开接口 - 其他
		api.GET("/models", func(c *gin.Context) {
			// TODO: Models API
		})
	}

	return engine
}
