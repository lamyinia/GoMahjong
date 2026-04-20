package realtime

import (
	"context"
	"errors"
	"fmt"
	"march/domain/repository"
	"march/infrastructure/database"
	"march/infrastructure/log"
	"march/infrastructure/message/transfer"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	marchPlayerInfoTTL = 30 * time.Minute
	queueKeyPrefix     = "march:queue"
	userPoolKey        = "march:user:pool"
)

func getQueueKey(poolID string) string {
	return fmt.Sprintf("%s:%s", queueKeyPrefix, poolID)
}

var joinQueueScript = `
local queueKey = KEYS[1]
local userPoolKey = KEYS[2]
local userID = ARGV[1]
local score = tonumber(ARGV[2])
local poolID = ARGV[3]

local existingPool = redis.call('HGET', userPoolKey, userID)
if existingPool ~= false and existingPool ~= nil and existingPool ~= "" then
	if existingPool ~= poolID then
		return -1
	end
	local existingScore = redis.call('ZSCORE', queueKey, userID)
	if existingScore ~= false and existingScore ~= nil then
		return -2
	end
end

redis.call('ZADD', queueKey, score, userID)
redis.call('HSET', userPoolKey, userID, poolID)

return 1
`

var popPlayersScript = `
local queueKey = KEYS[1]
local userPoolKey = KEYS[2]
local count = tonumber(ARGV[1])

local queueSize = redis.call('ZCARD', queueKey)
if queueSize < count then
	return {}
end

local players = redis.call('ZRANGE', queueKey, 0, count - 1)
if #players == 0 then
	return {}
end

local result = {}
for i = 1, #players do
	local userID = players[i]
	redis.call('ZREM', queueKey, userID)
	redis.call('HDEL', userPoolKey, userID)
	table.insert(result, userID)
end

return result
`

var removeFromQueueScript = `
local userPoolKey = KEYS[1]
local userID = ARGV[1]

local poolID = redis.call('HGET', userPoolKey, userID)
if poolID == false or poolID == nil or poolID == "" then
	return 0
end

local queueKey = "march:queue:" .. poolID

redis.call('ZREM', queueKey, userID)
redis.call('HDEL', userPoolKey, userID)

return 1
`

type RedisMarchQueueRepository struct {
	redis *database.RedisManager
}

func NewRedisMarchQueueRepository(redis *database.RedisManager) repository.MarchQueueRepository {
	return &RedisMarchQueueRepository{redis: redis}
}

func (q *RedisMarchQueueRepository) JoinQueue(ctx context.Context, poolID, userID string, score float64) error {
	if poolID == "" || userID == "" {
		return fmt.Errorf("poolID 和 userID 不能为空")
	}

	queueKey := getQueueKey(poolID)

	anyResult, err := q.redis.EvalScript(ctx, "joinQueueScript", joinQueueScript,
		[]string{queueKey, userPoolKey}, userID, score, poolID)
	if err != nil {
		return fmt.Errorf("执行 joinQueue Lua 脚本失败: %w", err)
	}

	result, ok := anyResult.(int64)
	if !ok {
		if f, ok := anyResult.(float64); ok {
			result = int64(f)
		} else {
			return fmt.Errorf("joinQueue 返回类型错误: 期望 int64，实际 %T", anyResult)
		}
	}

	switch result {
	case 1:
		log.Debug("玩家 %s 加入匹配池 %s，分数: %.2f", userID, poolID, score)
		return nil
	case -1:
		existingPool, _ := q.GetUserPool(ctx, userID)
		return fmt.Errorf("用户已在匹配池 %s 中，无法加入 %s", existingPool, poolID)
	case -2:
		return transfer.ErrPlayerAlreadyInQueue
	default:
		return fmt.Errorf("joinQueue 返回未知结果: %d", result)
	}
}

func (q *RedisMarchQueueRepository) RemoveFromQueue(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("userID 不能为空")
	}

	anyResult, err := q.redis.EvalScript(ctx, "removeFromQueueScript", removeFromQueueScript,
		[]string{userPoolKey}, userID)
	if err != nil {
		return fmt.Errorf("执行 removeFromQueue Lua 脚本失败: %w", err)
	}

	result, ok := anyResult.(int64)
	if !ok {
		if f, ok := anyResult.(float64); ok {
			result = int64(f)
		} else {
			return fmt.Errorf("removeFromQueue 返回类型错误: 期望 int64，实际 %T", anyResult)
		}
	}

	switch result {
	case 1:
		log.Debug("玩家 %s 从匹配队列移除", userID)
		return nil
	case 0:
		return nil
	default:
		return fmt.Errorf("removeFromQueue 返回未知结果: %d", result)
	}
}

func (q *RedisMarchQueueRepository) IsInQueue(ctx context.Context, userID string) (bool, string, error) {
	if userID == "" {
		return false, "", fmt.Errorf("userID 不能为空")
	}

	cli, err := q.redis.GetClient()
	if err != nil {
		return false, "", err
	}

	poolID, err := cli.HGet(ctx, userPoolKey, userID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("获取用户池映射失败: %w", err)
	}

	if poolID == "" {
		return false, "", nil
	}

	queueKey := getQueueKey(poolID)
	score, err := cli.ZScore(ctx, queueKey, userID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, poolID, nil
		}
		return false, poolID, fmt.Errorf("检查队列状态失败: %w", err)
	}

	return score > 0, poolID, nil
}

func (q *RedisMarchQueueRepository) GetUserPool(ctx context.Context, userID string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("userID 不能为空")
	}

	cli, err := q.redis.GetClient()
	if err != nil {
		return "", err
	}

	poolID, err := cli.HGet(ctx, userPoolKey, userID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", fmt.Errorf("获取用户池映射失败: %w", err)
	}

	return poolID, nil
}

func (q *RedisMarchQueueRepository) PopPlayers(ctx context.Context, poolID string, count int) ([]string, error) {
	if count <= 0 {
		return []string{}, nil
	}

	if poolID == "" {
		return nil, fmt.Errorf("poolID 不能为空")
	}

	queueKey := getQueueKey(poolID)

	anyResult, err := q.redis.EvalScript(ctx, "popPlayersScript", popPlayersScript, []string{queueKey, userPoolKey}, count)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("执行 popPlayers Lua 脚本失败: %w", err)
	}

	if anyResult == nil {
		return []string{}, nil
	}

	arr, ok := anyResult.([]interface{})
	if !ok {
		return nil, fmt.Errorf("popPlayers 返回类型错误: 期望 []interface{}，实际 %T", anyResult)
	}

	userIDs := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			userIDs = append(userIDs, s)
		} else {
			log.Warn("popPlayers 返回了非字符串元素: %v (类型: %T)", v, v)
		}
	}

	log.Debug("从匹配池 %s 取出 %d 个玩家", poolID, len(userIDs))
	return userIDs, nil
}

func (q *RedisMarchQueueRepository) GetQueueSize(ctx context.Context, poolID string) (int, error) {
	if poolID == "" {
		return 0, fmt.Errorf("poolID 不能为空")
	}

	cli, err := q.redis.GetClient()
	if err != nil {
		return 0, err
	}
	return int(cli.ZCard(ctx, getQueueKey(poolID)).Val()), nil
}
