package conn

import (
	"common/log"
	"common/rpc"
	"context"
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

func joinQueueHandler(session *Session, body []byte) (any, error) {
	userID := session.GetUserID()
	if userID == "" {
		return failMessage("用户ID未检测"), nil
	}
	nodeID := session.worker.nodeID
	if nodeID == "" {
		return failMessage("nodeID未检测"), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req := &matchpb.JoinQueueRequest{
		UserID: userID,
		NodeID: nodeID,
	}

	resp, err := rpc.MatchClient.JoinQueue(ctx, req)
	if err != nil {
		log.Error("JoinQueue RPC 调用失败: userID=%s, err=%v", userID, err)
		return failMessage(fmt.Sprintf("加入匹配队列失败: %v", err)), nil
	}

	result := map[string]interface{}{
		"success": resp.GetSuccess(),
		"message": resp.GetMessage(),
	}

	if resp.GetQueueID() != "" {
		result["queueID"] = resp.GetQueueID()
	}
	if resp.GetEstimatedSeconds() > 0 {
		result["estimatedSeconds"] = resp.GetEstimatedSeconds()
	}

	log.Info("用户加入匹配队列: userID=%s, connectorID=%s, success=%v, queueID=%s", userID, nodeID, resp.GetSuccess(), resp.GetQueueID())

	return result, nil
}

func redirectGame(session *Session, body []byte) (any, error) {
	return nil, nil
}
