package conn

import (
	"common/log"
	"encoding/json"
	"fmt"
	"framework/dto"
	"framework/protocol"
)

// handlePush 处理所有 Push 类型消息
func (w *Worker) handlePush(users []string, body *protocol.Message, route string) {
	// 对路由过滤，不同路由有不同的错误处理等级
	switch route {
	case dto.MatchingSuccess:
		w.handleMatchSuccessPush(users, body)
	default:
		log.Warn(fmt.Sprintf("connector handlePush 未知消息类型: %s", route))
	}
}

// handleMatchSuccessPush 处理匹配成功的 Push 消息
func (w *Worker) handleMatchSuccessPush(users []string, body *protocol.Message) {
	var failedUsers []error
	for _, userID := range users {
		if err := w.send(protocol.Push, userID, dto.MatchingSuccess, body.Data); err != nil {
			failedUsers = append(failedUsers, err)
		}
	}

	if len(failedUsers) > 0 {
		log.Warn(fmt.Sprintf("connector handleMatchSuccessPush 发送失败的用户: %v", failedUsers))
	}
}

// handlerMatchSuccess 处理 game.matchSuccess 的 Request 类型消息
func (w *Worker) handlerMatchSuccess(message []byte) any {
	var msg dto.MatchSuccessDTO
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Error(fmt.Sprintf("connector 解析匹配成功消息失败: %v", err))
		return nil
	}

	for userID := range msg.Players {
		w.UserRouteCache.Set(userID, msg.GameNodeID)
		log.Info(fmt.Sprintf("connector 保存用户路由: %s -> %s", userID, msg.GameNodeID))
	}

	return nil
}
