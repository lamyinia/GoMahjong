package api

import (
	"common/http"
)

// LoginHandler 用户登录
func LoginHandler(c *http.Context) error {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.BadRequest("请求参数错误")
		return nil
	}

	return nil
}

// RegisterHandler 用户注册
func RegisterHandler(c *http.Context) error {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.BadRequest("请求参数错误")
		return nil
	}

	return nil
}

// RefreshTokenHandler 刷新令牌
func RefreshTokenHandler(c *http.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.BadRequest("请求参数错误")
		return nil
	}

	return nil
}
