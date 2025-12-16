package game

import (
	"common/log"
	"encoding/json"
	"fmt"
	"framework/game/share"
)

// handleReconnect 处理断线重连消息
func (w *Worker) handleReconnect(data []byte) interface{} {
	return nil
}

func (w *Worker) handleDropTileHandler(data []byte) any {
	var event share.DropTileEvent
	err := json.Unmarshal(data, &event)
	if err != nil {
		log.Warn("handleDropTileHandler json 解析失败")
		return nil
	}
	room, exists := w.RoomManager.GetPlayerRoom(event.GetUserID())
	if !exists {
		log.Warn(fmt.Sprintf("Game Worker 玩家 %s 不在任何房间中", event.GetUserID()))
		return nil
	}

	room.Engine.DriveEngine(&event)
	return nil
}

func (w *Worker) handlePengTileHandler(data []byte) any {
	var event share.PengTileEvent
	err := json.Unmarshal(data, &event)
	if err != nil {
		log.Warn("handlePengTileHandler json 解析失败")
		return nil
	}
	room, exists := w.RoomManager.GetPlayerRoom(event.GetUserID())
	if !exists {
		log.Warn(fmt.Sprintf("Game Worker 玩家 %s 不在任何房间中", event.GetUserID()))
		return nil
	}

	room.Engine.DriveEngine(&event)
	return nil
}

func (w *Worker) handleGangTileHandler(data []byte) any {
	var event share.GangEvent
	err := json.Unmarshal(data, &event)
	if err != nil {
		log.Warn("handleGangTileHandler json 解析失败")
		return nil
	}
	room, exists := w.RoomManager.GetPlayerRoom(event.GetUserID())
	if !exists {
		log.Warn(fmt.Sprintf("Game Worker 玩家 %s 不在任何房间中", event.GetUserID()))
		return nil
	}

	room.Engine.DriveEngine(&event)
	return nil
}
