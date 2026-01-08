package conn

import (
	"common/log"
	"core/infrastructure/message/protocol"
	"core/infrastructure/message/transfer"
	"encoding/json"
	"fmt"
)

// handlePush 处理所有 Push 类型消息
func (w *Worker) handlePush(users []string, body *protocol.Message, route string) {
	switch route {
	case transfer.MatchingSuccess:
		w.handleMatchSuccessPush(users, body)
	case transfer.GamePush:
		w.handleGamePush(users, body)
	default:
		log.Warn(fmt.Sprintf("connector handlePush 未知消息类型: %s", route))
	}
}

// handleMatchSuccessPush 处理匹配成功的 Push 消息
func (w *Worker) handleMatchSuccessPush(users []string, body *protocol.Message) {
	var failedUsers []error
	for _, userID := range users {
		if err := w.send(protocol.Push, userID, transfer.MatchingSuccess, body.Data); err != nil {
			failedUsers = append(failedUsers, err)
		}
	}

	if len(failedUsers) > 0 {
		log.Warn(fmt.Sprintf("connector handleMatchSuccessPush 发送失败的用户: %v", failedUsers))
	}
}

// handleGamePush 处理游戏推送的 Push 消息
func (w *Worker) handleGamePush(users []string, body *protocol.Message) {
	var failedUsers []error
	for _, userID := range users {
		if err := w.send(protocol.Push, userID, transfer.GamePush, body.Data); err != nil {
			failedUsers = append(failedUsers, err)
		}
	}

	if len(failedUsers) > 0 {
		log.Warn(fmt.Sprintf("connector handleGamePush 发送失败的用户: %v", failedUsers))
	}
}

// handlerMatchSuccess 处理 game.matchSuccess 的 Request 类型消息
func (w *Worker) handlerMatchSuccess(message []byte) any {
	var msg transfer.MatchSuccessDTO
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Error(fmt.Sprintf("connector 解析匹配成功消息失败: %v", err))
		return nil
	}

	for userID := range msg.Players {
		w.GameRouteCache.Set(userID, msg.GameNodeID)
		log.Info(fmt.Sprintf("connector 保存用户路由: %s -> %s", userID, msg.GameNodeID))
	}

	return nil
}
