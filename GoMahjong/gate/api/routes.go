package api

import (
	"common/http"
	"common/rpc"
)

// RegisterRoutes 注册所有路由，发现 rpc 服务
func RegisterRoutes(server *http.HttpServer) {
	// 健康检查
	server.GET("/ping", PingHandler)
	server.GET("/health", HealthHandler)

	// 发现 rpc 服务
	rpc.Init()
	// API v1 路由组
	v1 := server.Group("/api/v1")
	{
		// 认证相关路由（无需认证）
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

				// 游戏相关路由（需要认证）
				game := v1.Group("/game", http.AuthMiddleware())
				{
					game.GET("/rooms", GetRoomsHandler)
					game.POST("/rooms", CreateRoomHandler)
					game.POST("/rooms/:id/join", JoinRoomHandler)
					game.DELETE("/rooms/:id/leave", LeaveRoomHandler)
				}*/
	}
}
