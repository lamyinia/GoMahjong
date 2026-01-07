package api

import (
	"common/config"
	"common/http"
	"common/rpc"
)

// RegisterRoutes 注册所有路由，发现 rpc 服务
func RegisterRoutes(server *http.HttpServer) {
	server.GET("/ping", PingHandler)
	server.GET("/health", HealthHandler)
	rpc.Init(config.GateNodeConfig.Domains, config.GateNodeConfig.EtcdConf)
	// API v1 路由组
	v1 := server.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/login", LoginHandler)
			auth.POST("/register", RegisterHandler)
			auth.POST("/refresh", RefreshTokenHandler)
		}

		// 用户相关路由（需要认证）
		/*		user := v1.Group("/user", http.AuthMiddleware())
				{
					user.GET("/profile", GetProfileHandler)
					user.PUT("/profile", UpdateProfileHandler)
					user.POST("/logout", LogoutHandler)
				}
		*/
	}
}
