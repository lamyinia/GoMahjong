package discovery

import (
	"errors"
	"math/rand"
)

/*
	负载均衡选择器
	用于 march 服务根据负载信息选择 game 节点
*/

// LoadBalanceStrategy 负载均衡策略
type LoadBalanceStrategy int

const (
	// LeastLoad 最小负载优先（根据 Load 字段，值越小负载越低）
	LeastLoad LoadBalanceStrategy = iota
	// WeightedRoundRobin 加权轮询
	WeightedRoundRobin
	// Random 随机选择
	Random
)

// SelectServer 根据策略选择服务
func SelectServer(servers []Server, strategy LoadBalanceStrategy) (*Server, error) {
	if len(servers) == 0 {
		return nil, errors.New("服务列表为空")
	}

	if len(servers) == 1 {
		return &servers[0], nil
	}

	switch strategy {
	case LeastLoad:
		return selectLeastLoad(servers)
	case WeightedRoundRobin:
		return selectWeightedRoundRobin(servers)
	case Random:
		return selectRandom(servers)
	default:
		return selectLeastLoad(servers)
	}
}

// selectLeastLoad 选择负载最低的服务
func selectLeastLoad(servers []Server) (*Server, error) {
	if len(servers) == 0 {
		return nil, errors.New("服务列表为空")
	}

	selected := &servers[0]
	minLoad := selected.Load

	for i := 1; i < len(servers); i++ {
		if servers[i].Load < minLoad {
			minLoad = servers[i].Load
			selected = &servers[i]
		}
	}

	return selected, nil
}

// selectWeightedRoundRobin 加权轮询选择
// 注意：这里简化实现，实际应该维护轮询状态
// 如果需要真正的加权轮询，需要在调用方维护状态
func selectWeightedRoundRobin(servers []Server) (*Server, error) {
	if len(servers) == 0 {
		return nil, errors.New("服务列表为空")
	}

	// 计算总权重
	totalWeight := 0
	for _, s := range servers {
		if s.Weight <= 0 {
			s.Weight = 1 // 默认权重为1
		}
		totalWeight += s.Weight
	}

	if totalWeight == 0 {
		return selectRandom(servers)
	}

	// 随机选择（简化实现，真正的加权轮询需要维护状态）
	random := rand.Intn(totalWeight)

	currentWeight := 0
	for i := range servers {
		if servers[i].Weight <= 0 {
			servers[i].Weight = 1
		}
		currentWeight += servers[i].Weight
		if random < currentWeight {
			return &servers[i], nil
		}
	}

	// 兜底返回第一个
	return &servers[0], nil
}

// selectRandom 随机选择
func selectRandom(servers []Server) (*Server, error) {
	if len(servers) == 0 {
		return nil, errors.New("服务列表为空")
	}

	idx := rand.Intn(len(servers))
	return &servers[idx], nil
}
