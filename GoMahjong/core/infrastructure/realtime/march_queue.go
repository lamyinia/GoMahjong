package realtime

import (
	"common/database"
	"common/log"
	"context"
	"core/domain/repository"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// 玩家信息过期时间（比队列超时时间长）
	marchPlayerInfoTTL = 30 * time.Minute
	// Redis Key 前缀
	queueKeyPrefix = "march:queue"     // Sorted Set: march:queue:{poolID}
	userPoolKey    = "march:user:pool" // Hash: userID -> poolID
)

// getQueueKey 生成队列 Key
func getQueueKey(poolID string) string {
	return fmt.Sprintf("%s:%s", queueKeyPrefix, poolID)
}

// joinQueueScript 加入队列的 Lua 脚本（原子操作）
// KEYS[1] = queueKey (march:queue:{poolID})
// KEYS[2] = userPoolKey (march:user:pool)
// ARGV[1] = userID
// ARGV[2] = score
// ARGV[3] = poolID
// 返回：1 表示成功，-1 表示已在其他池中，-2 表示已在同一池的队列中
var joinQueueScript = `
local queueKey = KEYS[1]
local userPoolKey = KEYS[2]
local userID = ARGV[1]
local score = tonumber(ARGV[2])
local poolID = ARGV[3]

-- 1. 检查用户是否已在其他池中
local existingPool = redis.call('HGET', userPoolKey, userID)
if existingPool ~= false and existingPool ~= nil and existingPool ~= "" then
	if existingPool ~= poolID then
		-- 用户已在其他池中
		return -1
	end
	-- 用户已在同一池中，检查是否在队列中
	local existingScore = redis.call('ZSCORE', queueKey, userID)
	if existingScore ~= false and existingScore ~= nil then
		-- 用户已在队列中
		return -2
	end
end

-- 2. 原子操作：加入队列 + 更新映射
redis.call('ZADD', queueKey, score, userID)
redis.call('HSET', userPoolKey, userID, poolID)

return 1
`

// popPlayersScript 取出玩家的 Lua 脚本（原子操作：取出 + 清理映射）
// KEYS[1] = queueKey (march:queue:{poolID})
// KEYS[2] = userPoolKey (march:user:pool)
// ARGV[1] = count
var popPlayersScript = `
local queueKey = KEYS[1]
local userPoolKey = KEYS[2]
local count = tonumber(ARGV[1])

-- 1. 检查队列大小
local queueSize = redis.call('ZCARD', queueKey)
if queueSize < count then
	return {}
end

-- 2. 取出玩家
local players = redis.call('ZRANGE', queueKey, 0, count - 1)
if #players == 0 then
	return {}
end

-- 3. 原子操作：从队列移除 + 清理映射
local result = {}
for i = 1, #players do
	local userID = players[i]
	redis.call('ZREM', queueKey, userID)
	redis.call('HDEL', userPoolKey, userID)
	table.insert(result, userID)
end

return result
`

// removeFromQueueScript 从队列移除玩家的 Lua 脚本（原子操作）
// KEYS[1] = userPoolKey (march:user:pool)
// ARGV[1] = userID
// 返回：1 表示成功，0 表示用户不在任何池中（幂等）
var removeFromQueueScript = `
local userPoolKey = KEYS[1]
local userID = ARGV[1]

-- 1. 从映射中获取 poolID
local poolID = redis.call('HGET', userPoolKey, userID)
if poolID == false or poolID == nil or poolID == "" then
	-- 用户不在任何池中，返回成功（幂等）
	return 0
end

-- 2. 构建队列 Key
local queueKey = "march:queue:" .. poolID

-- 3. 原子操作：从队列移除 + 删除映射
redis.call('ZREM', queueKey, userID)
redis.call('HDEL', userPoolKey, userID)

return 1
`

// RedisMarchQueueRepository Redis 实现的匹配队列仓储
type RedisMarchQueueRepository struct {
	redis *database.RedisManager
}

// NewRedisMarchQueueRepository 创建 Redis 匹配队列仓储
func NewRedisMarchQueueRepository(redis *database.RedisManager) repository.MarchQueueRepository {
	return &RedisMarchQueueRepository{redis: redis}
}

// JoinQueue 加入匹配队列（使用 Lua 脚本保证原子性）
func (q *RedisMarchQueueRepository) JoinQueue(ctx context.Context, poolID, userID string, score float64) error {
	if poolID == "" || userID == "" {
		return fmt.Errorf("poolID 和 userID 不能为空")
	}

	queueKey := getQueueKey(poolID)

	// 执行 Lua 脚本
	anyResult, err := q.redis.EvalScript(ctx, "joinQueueScript", joinQueueScript,
		[]string{queueKey, userPoolKey}, userID, score, poolID)
	if err != nil {
		return fmt.Errorf("执行 joinQueue Lua 脚本失败: %w", err)
	}

	// 处理返回结果
	result, ok := anyResult.(int64)
	if !ok {
		// 尝试 float64（某些 Redis 客户端可能返回 float64）
		if f, ok := anyResult.(float64); ok {
			result = int64(f)
		} else {
			return fmt.Errorf("joinQueue 返回类型错误: 期望 int64，实际 %T", anyResult)
		}
	}

	switch result {
	case 1:
		// 成功
		log.Debug("玩家 %s 加入匹配池 %s，分数: %.2f", userID, poolID, score)
		return nil
	case -1:
		// 用户已在其他池中
		existingPool, _ := q.GetUserPool(ctx, userID)
		return fmt.Errorf("用户已在匹配池 %s 中，无法加入 %s", existingPool, poolID)
	case -2:
		// 用户已在同一池的队列中
		return repository.ErrPlayerAlreadyInQueue
	default:
		return fmt.Errorf("joinQueue 返回未知结果: %d", result)
	}
}

// RemoveFromQueue 从队列中移除玩家（使用 Lua 脚本保证原子性）
func (q *RedisMarchQueueRepository) RemoveFromQueue(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("userID 不能为空")
	}

	// 执行 Lua 脚本
	anyResult, err := q.redis.EvalScript(ctx, "removeFromQueueScript", removeFromQueueScript,
		[]string{userPoolKey}, userID)
	if err != nil {
		return fmt.Errorf("执行 removeFromQueue Lua 脚本失败: %w", err)
	}

	// 处理返回结果
	result, ok := anyResult.(int64)
	if !ok {
		// 尝试 float64（某些 Redis 客户端可能返回 float64）
		if f, ok := anyResult.(float64); ok {
			result = int64(f)
		} else {
			return fmt.Errorf("removeFromQueue 返回类型错误: 期望 int64，实际 %T", anyResult)
		}
	}

	switch result {
	case 1:
		// 成功移除
		log.Debug("玩家 %s 从匹配队列移除", userID)
		return nil
	case 0:
		// 用户不在任何池中（幂等，返回成功）
		return nil
	default:
		return fmt.Errorf("removeFromQueue 返回未知结果: %d", result)
	}
}

// IsInQueue 检查玩家是否在队列中（自动查找 poolID）
func (q *RedisMarchQueueRepository) IsInQueue(ctx context.Context, userID string) (bool, string, error) {
	if userID == "" {
		return false, "", fmt.Errorf("userID 不能为空")
	}

	cli, err := q.redis.GetClient()
	if err != nil {
		return false, "", err
	}

	// 1. 从映射中获取 poolID
	poolID, err := cli.HGet(ctx, userPoolKey, userID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, "", nil // 用户不在任何池中
		}
		return false, "", fmt.Errorf("获取用户池映射失败: %w", err)
	}

	if poolID == "" {
		return false, "", nil
	}

	// 2. 检查是否在队列中
	queueKey := getQueueKey(poolID)
	score, err := cli.ZScore(ctx, queueKey, userID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, poolID, nil // 在映射中但不在队列中（异常情况，但返回 false）
		}
		return false, poolID, fmt.Errorf("检查队列状态失败: %w", err)
	}

	return score > 0, poolID, nil
}

// GetUserPool 获取用户所在的匹配池
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
			return "", nil // 用户不在任何池中
		}
		return "", fmt.Errorf("获取用户池映射失败: %w", err)
	}

	return poolID, nil
}

// PopPlayers 从队列中取出指定数量的玩家（使用 Lua 脚本保证原子性）
func (q *RedisMarchQueueRepository) PopPlayers(ctx context.Context, poolID string, count int) ([]string, error) {
	if count <= 0 {
		return []string{}, nil
	}

	if poolID == "" {
		return nil, fmt.Errorf("poolID 不能为空")
	}

	queueKey := getQueueKey(poolID)

	// 执行 Lua 脚本
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

	// 类型转换
	arr, ok := anyResult.([]interface{})
	if !ok {
		return nil, fmt.Errorf("popPlayers 返回类型错误: 期望 []interface{}，实际 %T", anyResult)
	}

	// 转换为 []string
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

// GetQueueSize 获取队列当前大小
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
