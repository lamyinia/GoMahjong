package api

import "common/http"

// GetRoomsHandler 获取房间列表
func GetRoomsHandler(c *http.Context) error {
	userID := c.GetString("userID")
	if userID == "" {
		c.Unauthorized("用户未认证")
		return nil
	}

	// TODO: 获取房间列表
	rooms := getRoomList()

	c.Success(map[string]interface{}{
		"rooms": rooms,
		"total": len(rooms),
	})
	return nil
}

// CreateRoomHandler 创建房间
func CreateRoomHandler(c *http.Context) error {
	userID := c.GetString("userID")
	if userID == "" {
		c.Unauthorized("用户未认证")
		return nil
	}

	var req struct {
		Name     string `json:"name" binding:"required"`
		MaxUsers int    `json:"max_users" binding:"required"`
		IsPublic bool   `json:"is_public"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.BadRequest("请求参数错误")
		return nil
	}

	// TODO: 创建房间
	roomID, err := createGameRoom(userID, req.Name, req.MaxUsers, req.IsPublic)
	if err != nil {
		c.InternalServerError("创建房间失败")
		return nil
	}

	c.SuccessWithMessage("房间创建成功", map[string]interface{}{
		"room_id": roomID,
		"name":    req.Name,
	})
	return nil
}

// JoinRoomHandler 加入房间
func JoinRoomHandler(c *http.Context) error {
	userID := c.GetString("userID")
	if userID == "" {
		c.Unauthorized("用户未认证")
		return nil
	}

	roomID := c.GetParam("id")
	if roomID == "" {
		c.BadRequest("房间ID不能为空")
		return nil
	}

	// TODO: 加入房间逻辑
	err := joinGameRoom(userID, roomID)
	if err != nil {
		c.ErrorWithCode(40001, "加入房间失败: "+err.Error())
		return nil
	}

	c.SuccessWithMessage("成功加入房间", map[string]string{
		"room_id": roomID,
	})
	return nil
}

// LeaveRoomHandler 离开房间
func LeaveRoomHandler(c *http.Context) error {
	userID := c.GetString("userID")
	if userID == "" {
		c.Unauthorized("用户未认证")
		return nil
	}

	roomID := c.GetParam("id")
	if roomID == "" {
		c.BadRequest("房间ID不能为空")
		return nil
	}

	// TODO: 离开房间逻辑
	err := leaveGameRoom(userID, roomID)
	if err != nil {
		c.InternalServerError("离开房间失败")
		return nil
	}

	c.SuccessWithMessage("已离开房间", nil)
	return nil
}

// 辅助函数（TODO: 实现具体逻辑）
func getRoomList() []map[string]interface{} {
	// TODO: 从游戏服务获取房间列表
	return []map[string]interface{}{
		{
			"id":            "room_1",
			"name":          "麻将房间1",
			"current_users": 2,
			"max_users":     4,
			"is_public":     true,
		},
		{
			"id":            "room_2",
			"name":          "麻将房间2",
			"current_users": 1,
			"max_users":     4,
			"is_public":     false,
		},
	}
}

func createGameRoom(userID, name string, maxUsers int, isPublic bool) (string, error) {
	// TODO: 调用游戏服务创建房间
	return "room_" + userID + "_" + name, nil
}

func joinGameRoom(userID, roomID string) error {
	// TODO: 调用游戏服务加入房间
	return nil
}

func leaveGameRoom(userID, roomID string) error {
	// TODO: 调用游戏服务离开房间
	return nil
}
