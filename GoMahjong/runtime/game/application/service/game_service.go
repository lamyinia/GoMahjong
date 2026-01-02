package service

import "context"

type GameService interface {
	CreateRoom(ctx context.Context, req *CreateRoomReq) (*CreateRoomResp, error)
}

type CreateRoomReq struct {
	Players    map[string]string `json:"players"`    // userID -> connectorTopic
	EngineType int32             `json:"engineType"` // 游戏引擎类型
}

type CreateRoomResp struct {
	Success bool   `json:"success"`
	RoomID  string `json:"roomID"`
	Message string `json:"message"`
}
