package discovery

import (
	"context"
	"errors"
	"fmt"
	"march/infrastructure/config"
	"march/infrastructure/log"
	"sync"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	gameServiceName = "game"
)

type NodeSelector struct {
	seeker      *Seeker
	gameServers []Server
	strategy    LoadBalanceStrategy
	mu          sync.RWMutex
	serviceName string
}

func NewNodeSelector(strategy LoadBalanceStrategy, etcdConf config.EtcdConf) (*NodeSelector, error) {
	seeker, err := NewSeeker(etcdConf)
	if err != nil {
		return nil, fmt.Errorf("创建服务发现客户端失败: %v", err)
	}

	ns := &NodeSelector{
		seeker:      seeker,
		gameServers: make([]Server, 0),
		strategy:    strategy,
		serviceName: gameServiceName,
	}

	servers, err := seeker.GetServers(gameServiceName)
	log.Info("全量拉取游戏节点: %#v", servers)
	if err != nil {
		seeker.Close()
		return nil, fmt.Errorf("初始化获取 game 节点列表失败: %v", err)
	}

	ns.mu.Lock()
	ns.gameServers = servers
	ns.mu.Unlock()

	log.Info(fmt.Sprintf("NodeSelector 初始化成功, 发现 %d 个 game 节点", len(servers)))

	go ns.watchNodes()

	return ns, nil
}

func (ns *NodeSelector) watchNodes() {
	watchCh := ns.seeker.Watch(ns.serviceName)

	for watchResp := range watchCh {
		if watchResp.Canceled {
			log.Warn("NodeSelector watch 被取消")
			return
		}
		ns.handleWatchEvents(watchResp.Events)
	}
}

func (ns *NodeSelector) handleWatchEvents(events []*clientv3.Event) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	updated := false
	for _, ev := range events {
		switch ev.Type {
		case clientv3.EventTypePut:
			server, err := ParseValue(ev.Kv.Value)
			if err != nil {
				log.Error(fmt.Sprintf("NodeSelector 解析节点信息失败, key=%s, err=%v", string(ev.Kv.Key), err))
				continue
			}

			found := false
			for i := range ns.gameServers {
				if ns.gameServers[i].Addr == server.Addr {
					ns.gameServers[i] = server
					found = true
					updated = true
					log.Debug(fmt.Sprintf("NodeSelector 更新 game 节点: %s, Load=%.2f", server.Addr, server.Load))
					break
				}
			}

			if !found {
				ns.gameServers = append(ns.gameServers, server)
				updated = true
				log.Info(fmt.Sprintf("NodeSelector 新增 game 节点: %s, Load=%.2f", server.Addr, server.Load))
			}

		case clientv3.EventTypeDelete:
			server, err := ParseKey(string(ev.Kv.Key))
			if err != nil {
				log.Error(fmt.Sprintf("NodeSelector 解析删除节点 key 失败, key=%s, err=%v", string(ev.Kv.Key), err))
				continue
			}

			for i := range ns.gameServers {
				if ns.gameServers[i].Addr == server.Addr {
					ns.gameServers[i] = ns.gameServers[len(ns.gameServers)-1]
					ns.gameServers = ns.gameServers[:len(ns.gameServers)-1]
					updated = true
					log.Info(fmt.Sprintf("NodeSelector 删除 game 节点: %s", server.Addr))
					break
				}
			}
		}
	}

	if updated {
		log.Debug(fmt.Sprintf("NodeSelector 节点列表已更新, 当前共有 %d 个 game 节点", len(ns.gameServers)))
	}
}

func (ns *NodeSelector) SelectGameNode(ctx context.Context) (*Server, error) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	healthyServers := make([]Server, 0, len(ns.gameServers))
	for _, server := range ns.gameServers {
		if server.Load > 0 {
			healthyServers = append(healthyServers, server)
		}
	}

	if len(healthyServers) == 0 {
		return nil, errors.New("没有可用的 game 节点（所有节点负载 <= 0 或列表为空）")
	}

	selected, err := SelectServer(healthyServers, ns.strategy)
	if err != nil {
		return nil, fmt.Errorf("选择 game 节点失败: %v", err)
	}

	log.Info(fmt.Sprintf("NodeSelector 选择 game 节点: %s, Load=%.2f, Strategy=%v", selected.Addr, selected.Load, ns.strategy))
	return selected, nil
}

func (ns *NodeSelector) GetGameNodes() []Server {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	result := make([]Server, len(ns.gameServers))
	copy(result, ns.gameServers)
	return result
}

func (ns *NodeSelector) GetHealthyGameNodes() []Server {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	healthyServers := make([]Server, 0)
	for _, server := range ns.gameServers {
		if server.Load > 0 {
			healthyServers = append(healthyServers, server)
		}
	}
	return healthyServers
}

func (ns *NodeSelector) UpdateStrategy(strategy LoadBalanceStrategy) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.strategy = strategy
	log.Info(fmt.Sprintf("NodeSelector 更新负载均衡策略: %v", strategy))
}

func (ns *NodeSelector) Close() error {
	if ns.seeker != nil {
		return ns.seeker.Close()
	}
	return nil
}
