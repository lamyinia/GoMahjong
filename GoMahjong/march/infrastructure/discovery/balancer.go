package discovery

import (
	"errors"
	"math/rand"
)

type LoadBalanceStrategy int

const (
	LeastLoad LoadBalanceStrategy = iota
	WeightedRoundRobin
	Random
)

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

func selectWeightedRoundRobin(servers []Server) (*Server, error) {
	if len(servers) == 0 {
		return nil, errors.New("服务列表为空")
	}

	totalWeight := 0
	for _, s := range servers {
		if s.Weight <= 0 {
			s.Weight = 1
		}
		totalWeight += s.Weight
	}

	if totalWeight == 0 {
		return selectRandom(servers)
	}

	randomVal := rand.Intn(totalWeight)

	currentWeight := 0
	for i := range servers {
		if servers[i].Weight <= 0 {
			servers[i].Weight = 1
		}
		currentWeight += servers[i].Weight
		if randomVal < currentWeight {
			return &servers[i], nil
		}
	}

	return &servers[0], nil
}

func selectRandom(servers []Server) (*Server, error) {
	if len(servers) == 0 {
		return nil, errors.New("服务列表为空")
	}

	idx := rand.Intn(len(servers))
	return &servers[idx], nil
}
