package api

import "common/http"

// GetProfileHandler 获取用户资料
func GetProfileHandler(c *http.Context) error {
	userID := c.GetString("userID")
	if userID == "" {
		c.Unauthorized("用户未认证")
		return nil
	}

	// TODO: 从用户服务获取用户信息
	profile, err := getUserProfile(userID)
	if err != nil {
		c.InternalServerError("获取用户信息失败")
		return nil
	}

	c.Success(profile)
	return nil
}

// UpdateProfileHandler 更新用户资料
func UpdateProfileHandler(c *http.Context) error {
	userID := c.GetString("userID")
	if userID == "" {
		c.Unauthorized("用户未认证")
		return nil
	}

	var req struct {
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
		Email    string `json:"email"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.BadRequest("请求参数错误")
		return nil
	}

	// TODO: 更新用户信息
	err := updateUserProfile(userID, req.Nickname, req.Avatar, req.Email)
	if err != nil {
		c.InternalServerError("更新用户信息失败")
		return nil
	}

	c.SuccessWithMessage("更新成功", nil)
	return nil
}

// LogoutHandler 用户登出
func LogoutHandler(c *http.Context) error {
	userID := c.GetString("userID")
	if userID == "" {
		c.Unauthorized("用户未认证")
		return nil
	}

	// TODO: 清理用户会话/令牌
	err := clearUserSession(userID)
	if err != nil {
		c.InternalServerError("登出失败")
		return nil
	}

	c.SuccessWithMessage("登出成功", nil)
	return nil
}

// 辅助函数（TODO: 实现具体逻辑）
func getUserProfile(userID string) (map[string]interface{}, error) {
	// TODO: 调用用户服务
	return map[string]interface{}{
		"user_id":  userID,
		"username": "test_user",
		"nickname": "测试用户",
		"email":    "test@example.com",
		"avatar":   "https://example.com/avatar.jpg",
	}, nil
}

func updateUserProfile(userID, nickname, avatar, email string) error {
	// TODO: 调用用户服务更新
	return nil
}

func clearUserSession(userID string) error {
	// TODO: 清理会话
	return nil
}
