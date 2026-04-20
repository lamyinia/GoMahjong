package http

import "github.com/gin-gonic/gin"

type Context struct {
	*gin.Context
}

func (c *Context) BindJSON(obj interface{}) error {
	return c.ShouldBindJSON(obj)
}

func (c *Context) GetString(key string) string {
	val, _ := c.Context.Get(key)
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

func (c *Context) GetParam(key string) string {
	return c.Param(key)
}

func (c *Context) Success(data interface{}) {
	Success(c.Context, data)
}

func (c *Context) SuccessWithMessage(message string, data interface{}) {
	SuccessWithMessage(c.Context, message, data)
}

func (c *Context) ErrorWithCode(code int, message string) {
	ErrorWithCode(c.Context, code, message)
}

func (c *Context) BadRequest(message string) {
	BadRequest(c.Context, message)
}

func (c *Context) Unauthorized(message string) {
	Unauthorized(c.Context, message)
}

func (c *Context) NotFound(message string) {
	NotFound(c.Context, message)
}

func (c *Context) InternalServerError(message string) {
	InternalServerError(c.Context, message)
}

type HandlerFunc func(c *Context) error

func WrapHandler(h HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := &Context{Context: c}
		if err := h(ctx); err != nil {
			InternalServerError(c, err.Error())
		}
	}
}
