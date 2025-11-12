package http

// 这个文件展示了如何使用封装的 HTTP 库
// 在实际项目中，这些代码应该放在具体的业务模块中

import (
	"context"
	"time"
)

// ExampleServer 示例服务器
func ExampleServer() {
	// 创建 HTTP 服务器
	server := NewHttpServer(
		WithPort(8080),
		WithMode("debug"), // gin.DebugMode
	)

	// 添加全局中间件
	server.Use(
		CorsMiddleware(),
		LoggerMiddleware(),
		RecoveryMiddleware(),
		RequestIDMiddleware(),
	)

	// 注册基本路由
	server.GET("/ping", PingHandler)
	server.POST("/login", LoginHandler)
	server.GET("/user/:id", GetUserHandler)

	// 创建需要认证的路由组
	authGroup := server.Group("/api/v1", AuthMiddleware())
	{
		authGroup.GET("/profile", GetProfileHandler)
		authGroup.PUT("/profile", UpdateProfileHandler)
		authGroup.DELETE("/logout", LogoutHandler)
	}

	// 创建管理员路由组
	adminGroup := server.Group("/admin", AuthMiddleware(), AdminMiddleware())
	{
		adminGroup.GET("/users", GetUsersHandler)
		adminGroup.POST("/users", CreateUserHandler)
		adminGroup.DELETE("/users/:id", DeleteUserHandler)
	}

	// 静态文件服务
	server.Static("/static", "./static")
	server.StaticFile("/favicon.ico", "./assets/favicon.ico")

	// 启动服务器
	if err := server.Start(); err != nil {
		panic(err)
	}
}

// 示例处理函数

// PingHandler ping 处理函数
func PingHandler(c *Context) error {
	c.Success(map[string]string{
		"message": "pong",
		"time":    time.Now().Format(time.RFC3339),
	})
	return nil
}

// LoginHandler 登录处理函数
func LoginHandler(c *Context) error {
	// 绑定请求参数
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.BadRequest("Invalid request parameters")
		return nil
	}

	// 验证用户名密码（示例）
	if req.Username != "admin" || req.Password != "password" {
		c.Unauthorized("Invalid username or password")
		return nil
	}

	// 生成 token（示例）
	token := "fake-jwt-token-" + time.Now().Format("20060102150405")

	c.Success(map[string]string{
		"token": token,
		"user":  req.Username,
	})
	return nil
}

// GetUserHandler 获取用户信息
func GetUserHandler(c *Context) error {
	userID := c.GetParam("id")
	if userID == "" {
		c.BadRequest("User ID is required")
		return nil
	}

	// 模拟从数据库获取用户信息
	user := map[string]interface{}{
		"id":       userID,
		"username": "user" + userID,
		"email":    "user" + userID + "@example.com",
		"created":  time.Now().AddDate(-1, 0, 0).Format(time.RFC3339),
	}

	c.Success(user)
	return nil
}

// GetProfileHandler 获取当前用户资料
func GetProfileHandler(c *Context) error {
	// 从上下文中获取用户 ID（由认证中间件设置）
	userID := c.GetString("userID")
	if userID == "" {
		c.Unauthorized("User not authenticated")
		return nil
	}

	// 模拟获取用户资料
	profile := map[string]interface{}{
		"id":       userID,
		"username": "current_user",
		"email":    "current@example.com",
		"profile": map[string]interface{}{
			"nickname": "Current User",
			"avatar":   "https://example.com/avatar.jpg",
		},
	}

	c.Success(profile)
	return nil
}

// UpdateProfileHandler 更新用户资料
func UpdateProfileHandler(c *Context) error {
	userID := c.GetString("userID")
	if userID == "" {
		c.Unauthorized("User not authenticated")
		return nil
	}

	var req struct {
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.BadRequest("Invalid request parameters")
		return nil
	}

	// 模拟更新用户资料
	c.SuccessWithMessage("Profile updated successfully", map[string]interface{}{
		"id":       userID,
		"nickname": req.Nickname,
		"avatar":   req.Avatar,
		"updated":  time.Now().Format(time.RFC3339),
	})
	return nil
}

// LogoutHandler 登出
func LogoutHandler(c *Context) error {
	// 实际应该清理 token 或 session
	c.Success(map[string]string{
		"message": "Logged out successfully",
	})
	return nil
}

// GetUsersHandler 获取用户列表（管理员）
func GetUsersHandler(c *Context) error {
	// 获取分页参数
	//page := c.GetQueryWithDefault("page", "1")
	//size := c.GetQueryWithDefault("size", "10")

	// 模拟用户列表
	users := []map[string]interface{}{
		{"id": "1", "username": "user1", "email": "user1@example.com"},
		{"id": "2", "username": "user2", "email": "user2@example.com"},
		{"id": "3", "username": "user3", "email": "user3@example.com"},
	}

	// 返回分页数据
	c.SuccessWithPage(users, 100, 1, 10)
	return nil
}

// CreateUserHandler 创建用户（管理员）
func CreateUserHandler(c *Context) error {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.BadRequest("Invalid request parameters")
		return nil
	}

	// 模拟创建用户
	user := map[string]interface{}{
		"id":       "new_user_id",
		"username": req.Username,
		"email":    req.Email,
		"created":  time.Now().Format(time.RFC3339),
	}

	c.SuccessWithMessage("User created successfully", user)
	return nil
}

// DeleteUserHandler 删除用户（管理员）
func DeleteUserHandler(c *Context) error {
	userID := c.GetParam("id")
	if userID == "" {
		c.BadRequest("User ID is required")
		return nil
	}

	// 模拟删除用户
	c.SuccessWithMessage("User deleted successfully", map[string]string{
		"id": userID,
	})
	return nil
}

// AdminMiddleware 管理员权限中间件
func AdminMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		userID := c.GetString("userID")
		if userID == "" {
			c.Unauthorized("User not authenticated")
			return nil
		}

		// 检查是否为管理员（示例）
		if !isAdmin(userID) {
			c.Forbidden("Admin access required")
			return nil
		}

		return nil
	}
}

// isAdmin 检查是否为管理员（示例实现）
func isAdmin(userID string) bool {
	// TODO: 实现真正的管理员检查逻辑
	return userID == "admin" || userID == "user123"
}

// ExampleGracefulShutdown 优雅关闭示例
func ExampleGracefulShutdown() {
	server := NewHttpServer(WithPort(8080))

	// 注册路由...
	server.GET("/ping", PingHandler)

	// 在 goroutine 中启动服务器
	go func() {
		if err := server.Start(); err != nil {
			panic(err)
		}
	}()

	// 等待关闭信号...
	// 这里应该监听 SIGINT, SIGTERM 等信号

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		panic(err)
	}
}
