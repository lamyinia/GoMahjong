package realtime

import (
	"common/database"
	"common/log"
	"context"
	"core/domain/repository"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// Redis Key 前缀
	marchQueueKey      = "march:queue"       // Sorted Set: 匹配队列
	marchPlayerInfoKey = "march:player:info" // Hash: 玩家信息 (userID -> connectorTopic)
	marchPlayerInfoTTL = 30 * time.Minute    // 玩家信息过期时间（比队列超时时间长）
)

// Lua 脚本：原子性地从队列中取出指定数量的玩家
// KEYS[1]: marchQueueKey (Sorted Set)
// KEYS[2]: marchPlayerInfoKey (Hash)
// ARGV[1]: count (需要取出的玩家数量)
// 返回：字符串数组，格式为 ["userID1", "connectorTopic1", "userID2", "connectorTopic2", ...]
// 在 Go 中解析为 map[userID]connectorTopic
var popPlayersScript = `
local queueKey = KEYS[1]
local infoKey = KEYS[2]
local count = tonumber(ARGV[1])
local result = {}

-- 从 Sorted Set 中取出 count 个玩家（按分数从低到高）
local players = redis.call('ZRANGE', queueKey, 0, count - 1, 'WITHSCORES')

if #players == 0 then
    return {}
end

-- 处理取出的玩家
for i = 1, #players, 2 do
    local userID = players[i]
    local score = players[i + 1]
    
    -- 从 Hash 中获取 connectorTopic
    local connectorTopic = redis.call('HGET', infoKey, userID)
    if connectorTopic == false then
        connectorTopic = ""
    end
    
    -- 从队列中移除
    redis.call('ZREM', queueKey, userID)
    redis.call('HDEL', infoKey, userID)
    
    -- 添加到结果数组（userID, connectorTopic 成对出现）
    table.insert(result, userID)
    table.insert(result, connectorTopic)
end

return result
`

// RedisMarchQueueRepository Redis 实现的匹配队列仓储
type RedisMarchQueueRepository struct {
	redis *database.RedisManager
	// 预编译的 Lua 脚本
	popPlayersSHA string
}

// NewRedisMarchQueueRepository 创建 Redis 匹配队列仓储
func NewRedisMarchQueueRepository(redis *database.RedisManager) repository.MarchQueueRepository {
	repo := &RedisMarchQueueRepository{
		redis: redis,
	}

	// 预编译 Lua 脚本
	ctx := context.Background()
	if redis.Cli != nil {
		sha, err := redis.Cli.ScriptLoad(ctx, popPlayersScript).Result()
		if err != nil {
			log.Error("预编译 Lua 脚本失败: %v", err)
		} else {
			repo.popPlayersSHA = sha
			log.Info("Lua 脚本预编译成功: %s", sha)
		}
	} else if redis.ClusterCli != nil {
		// 集群模式下，需要在每个节点上加载脚本，这里先不预编译
		// 后续使用 EVAL 而不是 EVALSHA
		log.Info("集群模式，Lua 脚本将在运行时加载")
	}

	return repo
}

// getClient 获取 Redis 客户端（单机或集群）
func (r *RedisMarchQueueRepository) getClient() (redis.Cmdable, error) {
	if r.redis.Cli != nil {
		return r.redis.Cli, nil
	}
	if r.redis.ClusterCli != nil {
		return r.redis.ClusterCli, nil
	}
	return nil, fmt.Errorf("redis 客户端未初始化")
}

// JoinQueue 加入匹配队列
func (r *RedisMarchQueueRepository) JoinQueue(ctx context.Context, userID, connectorTopic string, score float64) error {
	cli, err := r.getClient()
	if err != nil {
		return err
	}

	// 检查是否已在队列中
	exists, err := cli.ZScore(ctx, marchQueueKey, userID).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("检查队列状态失败: %w", err)
	}
	if err == nil && exists > 0 {
		return repository.ErrPlayerAlreadyInQueue
	}

	// 使用 Pipeline 保证原子性
	pipe := cli.Pipeline()
	// 1. 添加到 Sorted Set（分数为等待时间戳或段位分数）
	pipe.ZAdd(ctx, marchQueueKey, redis.Z{
		Score:  score,
		Member: userID,
	})
	// 2. 保存玩家信息到 Hash
	pipe.HSet(ctx, marchPlayerInfoKey, userID, connectorTopic)
	// 3. 设置 Hash 过期时间（防止内存泄漏）
	pipe.Expire(ctx, marchPlayerInfoKey, marchPlayerInfoTTL)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("加入队列失败: %w", err)
	}

	log.Debug(fmt.Sprintf("玩家 %s 加入匹配队列，分数: %.2f", userID, score))
	return nil
}

// RemoveFromQueue 从队列中移除玩家
func (r *RedisMarchQueueRepository) RemoveFromQueue(ctx context.Context, userID string) error {
	cli, err := r.getClient()
	if err != nil {
		return err
	}

	pipe := cli.Pipeline()
	pipe.ZRem(ctx, marchQueueKey, userID)
	pipe.HDel(ctx, marchPlayerInfoKey, userID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("从队列移除玩家失败: %w", err)
	}

	log.Debug(fmt.Sprintf("玩家 %s 从匹配队列移除", userID))
	return nil
}

// PopPlayers 从队列中取出指定数量的玩家（使用 Lua 脚本保证原子性）
func (r *RedisMarchQueueRepository) PopPlayers(ctx context.Context, count int) (map[string]string, error) {
	if count <= 0 {
		return make(map[string]string), nil
	}

	var strArray []string
	var err error

	// 优先使用预编译的脚本（仅单机模式）
	if r.popPlayersSHA != "" && r.redis.Cli != nil {
		result, evalErr := r.redis.Cli.EvalSha(ctx, r.popPlayersSHA, []string{marchQueueKey, marchPlayerInfoKey}, count).Result()
		if evalErr != nil {
			errStr := evalErr.Error()
			if errStr == "NOSCRIPT No matching script. Use EVAL." {
				// 脚本未找到，重新加载
				sha, loadErr := r.redis.Cli.ScriptLoad(ctx, popPlayersScript).Result()
				if loadErr != nil {
					return nil, fmt.Errorf("重新加载 Lua 脚本失败: %w", loadErr)
				}
				r.popPlayersSHA = sha
				result, evalErr = r.redis.Cli.EvalSha(ctx, r.popPlayersSHA, []string{marchQueueKey, marchPlayerInfoKey}, count).Result()
			}
			if evalErr != nil {
				err = evalErr
			} else {
				// 类型断言：Lua 脚本返回数组
				if arr, ok := result.([]interface{}); ok {
					strArray = make([]string, 0, len(arr))
					for _, v := range arr {
						if s, ok := v.(string); ok {
							strArray = append(strArray, s)
						}
					}
				}
			}
		} else {
			// 类型断言：Lua 脚本返回数组
			if arr, ok := result.([]interface{}); ok {
				strArray = make([]string, 0, len(arr))
				for _, v := range arr {
					if s, ok := v.(string); ok {
						strArray = append(strArray, s)
					}
				}
			}
		}
	} else {
		// 使用原始脚本（集群模式或未预编译）
		cli, cliErr := r.getClient()
		if cliErr != nil {
			return nil, cliErr
		}

		result, evalErr := cli.Eval(ctx, popPlayersScript, []string{marchQueueKey, marchPlayerInfoKey}, count).Result()
		if evalErr != nil {
			err = evalErr
		} else {
			// 类型断言：Lua 脚本返回数组
			if arr, ok := result.([]interface{}); ok {
				strArray = make([]string, 0, len(arr))
				for _, v := range arr {
					if s, ok := v.(string); ok {
						strArray = append(strArray, s)
					}
				}
			}
		}
	}

	if err != nil {
		if err == redis.Nil {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("执行 Lua 脚本失败: %w", err)
	}

	// 转换为 map（成对解析）
	resultMap := make(map[string]string, len(strArray)/2)
	for i := 0; i < len(strArray); i += 2 {
		if i+1 < len(strArray) {
			userID := strArray[i]
			connectorTopic := strArray[i+1]
			resultMap[userID] = connectorTopic
		}
	}

	log.Debug(fmt.Sprintf("从匹配队列取出 %d 个玩家", len(resultMap)))
	return resultMap, nil
}

// GetQueueSize 获取队列当前大小
func (r *RedisMarchQueueRepository) GetQueueSize(ctx context.Context) (int, error) {
	cli, err := r.getClient()
	if err != nil {
		return 0, err
	}

	count, err := cli.ZCard(ctx, marchQueueKey).Result()
	if err != nil {
		return 0, fmt.Errorf("获取队列大小失败: %w", err)
	}

	return int(count), nil
}

// IsInQueue 检查玩家是否在队列中
func (r *RedisMarchQueueRepository) IsInQueue(ctx context.Context, userID string) (bool, error) {
	cli, err := r.getClient()
	if err != nil {
		return false, err
	}

	score, err := cli.ZScore(ctx, marchQueueKey, userID).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, fmt.Errorf("检查队列状态失败: %w", err)
	}

	return score > 0, nil
}

// GetPlayerScore 获取玩家在队列中的分数
func (r *RedisMarchQueueRepository) GetPlayerScore(ctx context.Context, userID string) (float64, error) {
	cli, err := r.getClient()
	if err != nil {
		return 0, err
	}

	score, err := cli.ZScore(ctx, marchQueueKey, userID).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, repository.ErrPlayerNotInQueue
		}
		return 0, fmt.Errorf("获取玩家分数失败: %w", err)
	}

	return score, nil
}

// UpdatePlayerScore 更新玩家在队列中的分数
func (r *RedisMarchQueueRepository) UpdatePlayerScore(ctx context.Context, userID string, score float64) error {
	cli, err := r.getClient()
	if err != nil {
		return err
	}

	// 检查玩家是否在队列中
	exists, err := cli.ZScore(ctx, marchQueueKey, userID).Result()
	if err != nil {
		if err == redis.Nil {
			return repository.ErrPlayerNotInQueue
		}
		return fmt.Errorf("检查队列状态失败: %w", err)
	}
	if exists == 0 {
		return repository.ErrPlayerNotInQueue
	}

	// 更新分数
	err = cli.ZAdd(ctx, marchQueueKey, redis.Z{
		Score:  score,
		Member: userID,
	}).Err()
	if err != nil {
		return fmt.Errorf("更新玩家分数失败: %w", err)
	}

	log.Debug(fmt.Sprintf("更新玩家 %s 的队列分数为: %.2f", userID, score))
	return nil
}

// RemoveExpiredPlayers 移除过期的玩家（等待时间超过指定时间）
func (r *RedisMarchQueueRepository) RemoveExpiredPlayers(ctx context.Context, maxWaitTime time.Duration) ([]string, error) {
	cli, err := r.getClient()
	if err != nil {
		return nil, err
	}

	// 计算过期时间戳（当前时间 - 最大等待时间）
	expiredScore := float64(time.Now().Add(-maxWaitTime).Unix())

	// 获取所有分数小于过期分数的玩家（使用 ZRANGEBYSCORE）
	expiredPlayers, err := cli.ZRangeByScore(ctx, marchQueueKey, &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%.0f", expiredScore),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("查询过期玩家失败: %w", err)
	}

	if len(expiredPlayers) == 0 {
		return []string{}, nil
	}

	// 批量移除
	pipe := cli.Pipeline()
	for _, userID := range expiredPlayers {
		pipe.ZRem(ctx, marchQueueKey, userID)
		pipe.HDel(ctx, marchPlayerInfoKey, userID)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("移除过期玩家失败: %w", err)
	}

	log.Info(fmt.Sprintf("移除 %d 个过期玩家（等待时间超过 %v）", len(expiredPlayers), maxWaitTime))
	return expiredPlayers, nil
}
