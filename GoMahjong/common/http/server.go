package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HandlerFunc func(*Context) error
type MiddlewareFunc func(*Context) error

// HttpServer HTTP 服务器封装
type HttpServer struct {
	engine   *gin.Engine
	server   *http.Server
	port     int
	handlers map[string]HandlerFunc
}

// ServerOption 服务器配置选项
type ServerOption func(*HttpServer)

// WithPort 设置端口
func WithPort(port int) ServerOption {
	return func(s *HttpServer) {
		s.port = port
	}
}

// WithMode 设置运行模式
func WithMode(mode string) ServerOption {
	return func(s *HttpServer) {
		gin.SetMode(mode)
	}
}

// NewHttpServer 创建 HTTP 服务器
func NewHttpServer(opts ...ServerOption) *HttpServer {
	server := &HttpServer{
		engine:   gin.New(),
		port:     8080,
		handlers: make(map[string]HandlerFunc),
	}

	// 应用配置选项
	for _, opt := range opts {
		opt(server)
	}

	server.engine.Use(gin.Logger())
	server.engine.Use(gin.Recovery())

	return server
}

// wrapHandler 包装处理函数
func (s *HttpServer) wrapHandler(handler HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := newContext(c)
		if err := handler(ctx); err != nil {
			// 统一错误处理
			ctx.InternalServerError(err.Error())
		}
	}
}

// wrapMiddleware 包装中间件
func (s *HttpServer) wrapMiddleware(middleware MiddlewareFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := newContext(c)
		if err := middleware(ctx); err != nil {
			ctx.InternalServerError(err.Error())
			c.Abort()
			return
		}
		c.Next()
	}
}

// 路由注册方法

// GET 注册 GET 路由
func (s *HttpServer) GET(path string, handler HandlerFunc) {
	s.engine.GET(path, s.wrapHandler(handler))
}

// POST 注册 POST 路由
func (s *HttpServer) POST(path string, handler HandlerFunc) {
	s.engine.POST(path, s.wrapHandler(handler))
}

// PUT 注册 PUT 路由
func (s *HttpServer) PUT(path string, handler HandlerFunc) {
	s.engine.PUT(path, s.wrapHandler(handler))
}

// DELETE 注册 DELETE 路由
func (s *HttpServer) DELETE(path string, handler HandlerFunc) {
	s.engine.DELETE(path, s.wrapHandler(handler))
}

// PATCH 注册 PATCH 路由
func (s *HttpServer) PATCH(path string, handler HandlerFunc) {
	s.engine.PATCH(path, s.wrapHandler(handler))
}

// OPTIONS 注册 OPTIONS 路由
func (s *HttpServer) OPTIONS(path string, handler HandlerFunc) {
	s.engine.OPTIONS(path, s.wrapHandler(handler))
}

// HEAD 注册 HEAD 路由
func (s *HttpServer) HEAD(path string, handler HandlerFunc) {
	s.engine.HEAD(path, s.wrapHandler(handler))
}

// Any 注册所有 HTTP 方法的路由
func (s *HttpServer) Any(path string, handler HandlerFunc) {
	s.engine.Any(path, s.wrapHandler(handler))
}

// 路由组

// Group 创建路由组
func (s *HttpServer) Group(relativePath string, middlewares ...MiddlewareFunc) *RouterGroup {
	ginGroup := s.engine.Group(relativePath)

	// 添加中间件
	for _, middleware := range middlewares {
		ginGroup.Use(s.wrapMiddleware(middleware))
	}

	return &RouterGroup{
		group:  ginGroup,
		server: s,
	}
}

// RouterGroup 路由组封装
type RouterGroup struct {
	group  *gin.RouterGroup
	server *HttpServer
}

// GET 路由组 GET 方法
func (rg *RouterGroup) GET(path string, handler HandlerFunc) {
	rg.group.GET(path, rg.server.wrapHandler(handler))
}

// POST 路由组 POST 方法
func (rg *RouterGroup) POST(path string, handler HandlerFunc) {
	rg.group.POST(path, rg.server.wrapHandler(handler))
}

// PUT 路由组 PUT 方法
func (rg *RouterGroup) PUT(path string, handler HandlerFunc) {
	rg.group.PUT(path, rg.server.wrapHandler(handler))
}

// DELETE 路由组 DELETE 方法
func (rg *RouterGroup) DELETE(path string, handler HandlerFunc) {
	rg.group.DELETE(path, rg.server.wrapHandler(handler))
}

// PATCH 路由组 PATCH 方法
func (rg *RouterGroup) PATCH(path string, handler HandlerFunc) {
	rg.group.PATCH(path, rg.server.wrapHandler(handler))
}

// OPTIONS 路由组 OPTIONS 方法
func (rg *RouterGroup) OPTIONS(path string, handler HandlerFunc) {
	rg.group.OPTIONS(path, rg.server.wrapHandler(handler))
}

// HEAD 路由组 HEAD 方法
func (rg *RouterGroup) HEAD(path string, handler HandlerFunc) {
	rg.group.HEAD(path, rg.server.wrapHandler(handler))
}

// Any 路由组所有 HTTP 方法
func (rg *RouterGroup) Any(path string, handler HandlerFunc) {
	rg.group.Any(path, rg.server.wrapHandler(handler))
}

// Use 添加中间件到路由组
func (rg *RouterGroup) Use(middlewares ...MiddlewareFunc) {
	for _, middleware := range middlewares {
		rg.group.Use(rg.server.wrapMiddleware(middleware))
	}
}

// Group 创建子路由组
func (rg *RouterGroup) Group(relativePath string, middlewares ...MiddlewareFunc) *RouterGroup {
	ginGroup := rg.group.Group(relativePath)

	// 添加中间件
	for _, middleware := range middlewares {
		ginGroup.Use(rg.server.wrapMiddleware(middleware))
	}

	return &RouterGroup{
		group:  ginGroup,
		server: rg.server,
	}
}

// 中间件管理

// Use 添加全局中间件
func (s *HttpServer) Use(middlewares ...MiddlewareFunc) {
	for _, middleware := range middlewares {
		s.engine.Use(s.wrapMiddleware(middleware))
	}
}

// 静态文件服务

// Static 静态文件服务
func (s *HttpServer) Static(relativePath, root string) {
	s.engine.Static(relativePath, root)
}

// StaticFile 单个静态文件
func (s *HttpServer) StaticFile(relativePath, filepath string) {
	s.engine.StaticFile(relativePath, filepath)
}

// 服务器控制

// Start 启动服务器
func (s *HttpServer) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}

	return s.server.ListenAndServe()
}

// StartTLS 启动 HTTPS 服务器
func (s *HttpServer) StartTLS(certFile, keyFile string) error {
	addr := fmt.Sprintf(":%d", s.port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}

	return s.server.ListenAndServeTLS(certFile, keyFile)
}

// Shutdown 优雅关闭服务器
func (s *HttpServer) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// GetEngine 获取原始 gin.Engine（谨慎使用）
func (s *HttpServer) GetEngine() *gin.Engine {
	return s.engine
}

// SetPort 设置端口
func (s *HttpServer) SetPort(port int) {
	s.port = port
}

// GetPort 获取端口
func (s *HttpServer) GetPort() int {
	return s.port
}
