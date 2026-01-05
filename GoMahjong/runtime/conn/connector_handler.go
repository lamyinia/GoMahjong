package conn

import (
	"common/log"
	"common/rpc"
	"context"
	"encoding/json"
	"fmt"
	matchpb "march/pb"
	"time"
)

func failMessage(tip string) map[string]any {
	return map[string]any{
		"success": false,
		"message": tip,
	}
}

// joinQueueRequest 客户端请求结构
type joinQueueRequest struct {
	PoolID string `json:"poolID"` // 匹配池ID（如 "classic:rank4", "classic:casual4", "classic:casual3"）
}

func joinQueueHandler(session *Session, body []byte) (any, error) {
	userID := session.GetUserID()
	if userID == "" {
		return failMessage("用户ID未检测"), nil
	}

	var clientReq joinQueueRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &clientReq); err != nil {
			log.Warn("解析 joinQueue 请求失败: %v, body=%s", err, string(body))
			return failMessage("请求参数格式错误"), nil
		}
	}
	if clientReq.PoolID == "" {
		return failMessage("poolID 不能为空"), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req := &matchpb.JoinQueueRequest{
		UserID: userID,
		PoolID: clientReq.PoolID,
	}

	resp, err := rpc.MatchClient.JoinQueue(ctx, req)
	if err != nil {
		log.Error("JoinQueue RPC 调用失败: userID=%s, poolID=%s, err=%v", userID, clientReq.PoolID, err)
		return failMessage(fmt.Sprintf("加入匹配队列失败: %v", err)), nil
	}

	result := map[string]any{
		"message":          resp.GetMessage(),
		"estimatedSeconds": resp.GetEstimatedSeconds(),
	}

	log.Info("用户加入匹配队列: userID=%s, poolID=%s, resp=%v", userID, clientReq.PoolID, resp)

	return result, nil
}

func redirectGame(session *Session, body []byte) (any, error) {
	return nil, nil
}
