package http

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HttpServer struct {
	engine *gin.Engine
	port   int
	server *http.Server
}

type Option func(*HttpServer)

func WithPort(port int) Option {
	return func(s *HttpServer) {
		s.port = port
	}
}

func WithMode(mode string) Option {
	return func(s *HttpServer) {
		switch mode {
		case "debug":
			gin.SetMode(gin.DebugMode)
		case "warn", "error":
			gin.SetMode(gin.ReleaseMode)
		default:
			gin.SetMode(gin.TestMode)
		}
	}
}

func NewHttpServer(opts ...Option) *HttpServer {
	s := &HttpServer{
		engine: gin.New(),
		port:   8080,
	}

	for _, opt := range opts {
		opt(s)
	}

	s.engine.Use(gin.Recovery())

	return s
}

func (s *HttpServer) Use(middleware ...gin.HandlerFunc) {
	s.engine.Use(middleware...)
}

func (s *HttpServer) GET(relativePath string, handler HandlerFunc) {
	s.engine.GET(relativePath, WrapHandler(handler))
}

func (s *HttpServer) POST(relativePath string, handler HandlerFunc) {
	s.engine.POST(relativePath, WrapHandler(handler))
}

func (s *HttpServer) PUT(relativePath string, handler HandlerFunc) {
	s.engine.PUT(relativePath, WrapHandler(handler))
}

func (s *HttpServer) DELETE(relativePath string, handler HandlerFunc) {
	s.engine.DELETE(relativePath, WrapHandler(handler))
}

func (s *HttpServer) Group(relativePath string, handlers ...gin.HandlerFunc) *RouterGroup {
	return &RouterGroup{
		group: s.engine.Group(relativePath, handlers...),
	}
}

type RouterGroup struct {
	group *gin.RouterGroup
}

func (g *RouterGroup) GET(relativePath string, handler HandlerFunc) {
	g.group.GET(relativePath, WrapHandler(handler))
}

func (g *RouterGroup) POST(relativePath string, handler HandlerFunc) {
	g.group.POST(relativePath, WrapHandler(handler))
}

func (g *RouterGroup) PUT(relativePath string, handler HandlerFunc) {
	g.group.PUT(relativePath, WrapHandler(handler))
}

func (g *RouterGroup) DELETE(relativePath string, handler HandlerFunc) {
	g.group.DELETE(relativePath, WrapHandler(handler))
}

func (g *RouterGroup) Group(relativePath string, handlers ...gin.HandlerFunc) *RouterGroup {
	return &RouterGroup{
		group: g.group.Group(relativePath, handlers...),
	}
}

func (s *HttpServer) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}
	return s.server.ListenAndServe()
}

func (s *HttpServer) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}
