package mahjong

import (
	"common/log"
	"encoding/json"
)

// broadcastOperations 下发操作给客户端
func (eg *RiichiMahjong4p) broadcastOperations(reactions map[int]*PlayerReaction) {
	for seatIndex, reaction := range reactions {
		log.Info("玩家 %d 可用操作数: %d", seatIndex, len(reaction.Operations))
		data, err := json.Marshal(reaction.Operations)
		if err != nil {
			log.Error("JSON序列化失败: %v", err)
			continue
		}

		connector := eg.getConnector(seatIndex)
		if connector == "" {
			log.Warn("玩家 %d 没有连接信息", seatIndex)
			continue
		}

		err = eg.Worker.PushConnector(connector, DispatchViceReaction, data)
		if err != nil {
			log.Warn("玩家 %d 推送失败", seatIndex)
		}
	}
}

func (eg *RiichiMahjong4p) getConnector(seatIndex int) string {
	id := eg.Players[seatIndex].UserID
	user, ok := eg.UserMap[id]
	if ok {
		return user.ConnectorNodeID
	} else {
		return ""
	}
}
