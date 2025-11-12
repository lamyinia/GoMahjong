package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Context 封装 gin.Context，提供统一的请求/响应接口
type Context struct {
	ginCtx *gin.Context
}

// newContext 创建新的上下文实例
func newContext(c *gin.Context) *Context {
	return &Context{ginCtx: c}
}

// Request 相关方法

// GetParam 获取路径参数
func (c *Context) GetParam(key string) string {
	return c.ginCtx.Param(key)
}

// GetQuery 获取查询参数
func (c *Context) GetQuery(key string) string {
	return c.ginCtx.Query(key)
}

// GetQueryWithDefault 获取查询参数，带默认值
func (c *Context) GetQueryWithDefault(key, defaultValue string) string {
	return c.ginCtx.DefaultQuery(key, defaultValue)
}

// GetHeader 获取请求头
func (c *Context) GetHeader(key string) string {
	return c.ginCtx.GetHeader(key)
}

// BindJSON 绑定 JSON 请求体
func (c *Context) BindJSON(obj interface{}) error {
	return c.ginCtx.ShouldBindJSON(obj)
}

// BindQuery 绑定查询参数
func (c *Context) BindQuery(obj interface{}) error {
	return c.ginCtx.ShouldBindQuery(obj)
}

// Response 相关方法

// JSON 返回 JSON 响应
func (c *Context) JSON(code int, obj interface{}) {
	c.ginCtx.JSON(code, obj)
}

// String 返回字符串响应
func (c *Context) String(code int, format string, values ...interface{}) {
	c.ginCtx.String(code, format, values...)
}

// HTML 返回 HTML 响应
func (c *Context) HTML(code int, name string, obj interface{}) {
	c.ginCtx.HTML(code, name, obj)
}

// Redirect 重定向
func (c *Context) Redirect(code int, location string) {
	c.ginCtx.Redirect(code, location)
}

// SetHeader 设置响应头
func (c *Context) SetHeader(key, value string) {
	c.ginCtx.Header(key, value)
}

// SetCookie 设置 Cookie
func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	c.ginCtx.SetCookie(name, value, maxAge, path, domain, secure, httpOnly)
}

// GetCookie 获取 Cookie
func (c *Context) GetCookie(name string) (string, error) {
	return c.ginCtx.Cookie(name)
}

// 工具方法

// ClientIP 获取客户端 IP
func (c *Context) ClientIP() string {
	return c.ginCtx.ClientIP()
}

// UserAgent 获取 User-Agent
func (c *Context) UserAgent() string {
	return c.ginCtx.GetHeader("User-Agent")
}

// Method 获取请求方法
func (c *Context) Method() string {
	return c.ginCtx.Request.Method
}

// Path 获取请求路径
func (c *Context) Path() string {
	return c.ginCtx.Request.URL.Path
}

// Set 设置上下文值
func (c *Context) Set(key string, value interface{}) {
	c.ginCtx.Set(key, value)
}

// Get 获取上下文值
func (c *Context) Get(key string) (interface{}, bool) {
	return c.ginCtx.Get(key)
}

// GetString 获取字符串类型的上下文值
func (c *Context) GetString(key string) string {
	return c.ginCtx.GetString(key)
}

// Abort 中止请求处理
func (c *Context) Abort() {
	c.ginCtx.Abort()
}

// AbortWithStatus 中止请求并设置状态码
func (c *Context) AbortWithStatus(code int) {
	c.ginCtx.AbortWithStatus(code)
}

// AbortWithStatusJSON 中止请求并返回 JSON 错误
func (c *Context) AbortWithStatusJSON(code int, jsonObj interface{}) {
	c.ginCtx.AbortWithStatusJSON(code, jsonObj)
}

// IsAborted 检查请求是否已被中止
func (c *Context) IsAborted() bool {
	return c.ginCtx.IsAborted()
}

// Status 设置响应状态码
func (c *Context) Status(code int) {
	c.ginCtx.Status(code)
}

// GetRawData 获取原始请求体数据
func (c *Context) GetRawData() ([]byte, error) {
	return c.ginCtx.GetRawData()
}

// Request 获取原始 http.Request（谨慎使用）
func (c *Context) Request() *http.Request {
	return c.ginCtx.Request
}
