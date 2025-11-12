package http

import (
	"strings"
	"time"

	"common/log"
)

// CORS 跨域中间件
func CorsMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		method := c.Method()
		origin := c.GetHeader("Origin")

		if origin != "" {
			c.SetHeader("Access-Control-Allow-Origin", "*")
			c.SetHeader("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
			c.SetHeader("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, Token, X-Token")
			c.SetHeader("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
			c.SetHeader("Access-Control-Allow-Credentials", "true")
		}

		// 处理预检请求
		if method == "OPTIONS" {
			c.AbortWithStatus(204)
			return nil
		}

		return nil
	}
}

// Logger 日志中间件
func LoggerMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		start := time.Now()
		path := c.Path()
		method := c.Method()
		clientIP := c.ClientIP()
		userAgent := c.UserAgent()

		// 记录请求开始
		log.Info("HTTP Request: %s %s from %s, User-Agent: %s", method, path, clientIP, userAgent)

		// 处理请求后记录响应时间
		defer func() {
			latency := time.Since(start)
			log.Info("HTTP Response: %s %s completed in %v", method, path, latency)
		}()

		return nil
	}
}

// Recovery 恢复中间件
func RecoveryMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		defer func() {
			if err := recover(); err != nil {
				log.Error("Panic recovered: %v", err)
				c.InternalServerError("Internal Server Error")
			}
		}()
		return nil
	}
}

// Auth 认证中间件
func AuthMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		token := c.GetHeader("Authorization")
		if token == "" {
			token = c.GetHeader("Token")
		}
		if token == "" {
			token = c.GetHeader("X-Token")
		}

		if token == "" {
			c.Unauthorized("Missing authorization token")
			return nil
		}

		// 移除 Bearer 前缀
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}

		// TODO: 这里应该验证 token 的有效性
		// 可以调用 JWT 验证服务或查询数据库
		if !validateToken(token) {
			c.Unauthorized("Invalid token")
			return nil
		}

		// 将用户信息存储到上下文中
		userID := getUserIDFromToken(token)
		c.Set("userID", userID)

		return nil
	}
}

// validateToken 验证 token（示例实现）
func validateToken(token string) bool {
	// TODO: 实现真正的 token 验证逻辑
	// 这里只是示例，实际应该验证 JWT 或查询数据库
	return token != "" && len(token) > 10
}

// getUserIDFromToken 从 token 中获取用户 ID（示例实现）
func getUserIDFromToken(token string) string {
	// TODO: 实现从 token 中解析用户 ID 的逻辑
	// 这里只是示例，实际应该解析 JWT 或查询数据库
	return "user123"
}

// RateLimit 限流中间件（简单实现）
func RateLimitMiddleware(maxRequests int, duration time.Duration) MiddlewareFunc {
	// 简单的内存限流，生产环境建议使用 Redis
	requestCounts := make(map[string][]time.Time)

	return func(c *Context) error {
		clientIP := c.ClientIP()
		now := time.Now()

		// 清理过期的请求记录
		if requests, exists := requestCounts[clientIP]; exists {
			validRequests := make([]time.Time, 0)
			for _, reqTime := range requests {
				if now.Sub(reqTime) < duration {
					validRequests = append(validRequests, reqTime)
				}
			}
			requestCounts[clientIP] = validRequests
		}

		// 检查是否超过限制
		if len(requestCounts[clientIP]) >= maxRequests {
			c.JSON(429, map[string]string{
				"error": "Too Many Requests",
			})
			return nil
		}

		// 记录当前请求
		requestCounts[clientIP] = append(requestCounts[clientIP], now)

		return nil
	}
}

// RequestID 请求 ID 中间件
func RequestIDMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// 生成新的请求 ID
			requestID = generateRequestID()
		}

		c.Set("requestID", requestID)
		c.SetHeader("X-Request-ID", requestID)

		return nil
	}
}

// generateRequestID 生成请求 ID
func generateRequestID() string {
	// 简单的请求 ID 生成，生产环境可以使用更复杂的算法
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

// randomString 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// Timeout 超时中间件
func TimeoutMiddleware(timeout time.Duration) MiddlewareFunc {
	return func(c *Context) error {
		// 设置请求超时
		// 注意：这是一个简化的实现，实际的超时处理会更复杂
		c.Set("timeout", timeout)

		start := time.Now()
		defer func() {
			if time.Since(start) > timeout {
				log.Warn("Request timeout: %s %s took %v", c.Method(), c.Path(), time.Since(start))
			}
		}()

		return nil
	}
}

// ContentType 内容类型中间件
func ContentTypeMiddleware(contentType string) MiddlewareFunc {
	return func(c *Context) error {
		c.SetHeader("Content-Type", contentType)
		return nil
	}
}

// Security 安全头中间件
func SecurityMiddleware() MiddlewareFunc {
	return func(c *Context) error {
		c.SetHeader("X-Content-Type-Options", "nosniff")
		c.SetHeader("X-Frame-Options", "DENY")
		c.SetHeader("X-XSS-Protection", "1; mode=block")
		c.SetHeader("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		return nil
	}
}
