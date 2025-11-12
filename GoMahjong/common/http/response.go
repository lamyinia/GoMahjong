package http

import "net/http"

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 预定义的响应码
const (
	CodeSuccess      = 0     // 成功
	CodeError        = -1    // 通用错误
	CodeInvalidParam = 10001 // 参数错误
	CodeUnauthorized = 10002 // 未授权
	CodeForbidden    = 10003 // 禁止访问
	CodeNotFound     = 10004 // 资源不存在
	CodeServerError  = 10005 // 服务器内部错误
)

// 预定义的响应消息
const (
	MsgSuccess      = "success"
	MsgError        = "error"
	MsgInvalidParam = "invalid parameters"
	MsgUnauthorized = "unauthorized"
	MsgForbidden    = "forbidden"
	MsgNotFound     = "not found"
	MsgServerError  = "internal server error"
)

// NewResponse 创建响应
func NewResponse(code int, message string, data interface{}) *Response {
	return &Response{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// Context 响应方法扩展

// Success 成功响应
func (c *Context) Success(data interface{}) {
	c.JSON(http.StatusOK, NewResponse(CodeSuccess, MsgSuccess, data))
}

// SuccessWithMessage 成功响应（自定义消息）
func (c *Context) SuccessWithMessage(message string, data interface{}) {
	c.JSON(http.StatusOK, NewResponse(CodeSuccess, message, data))
}

// Error 错误响应
func (c *Context) Error(message string) {
	c.JSON(http.StatusOK, NewResponse(CodeError, message, nil))
}

// ErrorWithCode 错误响应（自定义错误码）
func (c *Context) ErrorWithCode(code int, message string) {
	c.JSON(http.StatusOK, NewResponse(code, message, nil))
}

// BadRequest 400 错误请求
func (c *Context) BadRequest(message string) {
	if message == "" {
		message = MsgInvalidParam
	}
	c.JSON(http.StatusBadRequest, NewResponse(CodeInvalidParam, message, nil))
}

// Unauthorized 401 未授权
func (c *Context) Unauthorized(message string) {
	if message == "" {
		message = MsgUnauthorized
	}
	c.JSON(http.StatusUnauthorized, NewResponse(CodeUnauthorized, message, nil))
}

// Forbidden 403 禁止访问
func (c *Context) Forbidden(message string) {
	if message == "" {
		message = MsgForbidden
	}
	c.JSON(http.StatusForbidden, NewResponse(CodeForbidden, message, nil))
}

// NotFound 404 资源不存在
func (c *Context) NotFound(message string) {
	if message == "" {
		message = MsgNotFound
	}
	c.JSON(http.StatusNotFound, NewResponse(CodeNotFound, message, nil))
}

// InternalServerError 500 服务器内部错误
func (c *Context) InternalServerError(message string) {
	if message == "" {
		message = MsgServerError
	}
	c.JSON(http.StatusInternalServerError, NewResponse(CodeServerError, message, nil))
}

// 分页响应结构
type PageResponse struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

// NewPageResponse 创建分页响应
func NewPageResponse(list interface{}, total int64, page, size int) *PageResponse {
	return &PageResponse{
		List:  list,
		Total: total,
		Page:  page,
		Size:  size,
	}
}

// SuccessWithPage 分页成功响应
func (c *Context) SuccessWithPage(list interface{}, total int64, page, size int) {
	pageResp := NewPageResponse(list, total, page, size)
	c.JSON(http.StatusOK, NewResponse(CodeSuccess, MsgSuccess, pageResp))
}
