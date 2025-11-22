package march

import (
	"common/config"
	"common/discovery"
	"common/log"
	"context"
	"errors"
	"fmt"
	"sync"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	gameServiceName = "game" // 固定为 game 服务
)

type NodeSelector struct {
	seeker      *discovery.Seeker
	gameServers []discovery.Server
	strategy    discovery.LoadBalanceStrategy
	mu          sync.RWMutex
	serviceName string
}

// NewNodeSelector 创建节点选择器
// strategy: 负载均衡策略
func NewNodeSelector(strategy discovery.LoadBalanceStrategy) (*NodeSelector, error) {
	seeker, err := discovery.NewSeeker(config.Conf.EtcdConf)
	if err != nil {
		return nil, fmt.Errorf("创建服务发现客户端失败: %v", err)
	}

	ns := &NodeSelector{
		seeker:      seeker,
		gameServers: make([]discovery.Server, 0),
		strategy:    strategy,
		serviceName: gameServiceName,
	}

	// 立即获取一次全量列表（初始化）
	servers, err := seeker.GetServers(gameServiceName)
	if err != nil {
		seeker.Close()
		return nil, fmt.Errorf("初始化获取 game 节点列表失败: %v", err)
	}

	ns.mu.Lock()
	ns.gameServers = servers
	ns.mu.Unlock()

	log.Info(fmt.Sprintf("NodeSelector 初始化成功, 发现 %d 个 game 节点", len(servers)))

	// 启动增量更新监听
	go ns.watchNodes()

	return ns, nil
}

// watchNodes 监听节点变化，实现增量更新
func (ns *NodeSelector) watchNodes() {
	watchCh := ns.seeker.Watch(ns.serviceName)

	for watchResp := range watchCh {
		if watchResp.Canceled {
			log.Warn("NodeSelector watch 被取消")
			return
		}
		// 处理增量更新
		ns.handleWatchEvents(watchResp.Events)
	}
}

// handleWatchEvents 处理 etcd Watch 事件，实现增量更新
func (ns *NodeSelector) handleWatchEvents(events []*clientv3.Event) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	updated := false
	for _, ev := range events {
		switch ev.Type {
		case clientv3.EventTypePut:
			// 添加或更新节点
			server, err := discovery.ParseValue(ev.Kv.Value)
			if err != nil {
				log.Error(fmt.Sprintf("NodeSelector 解析节点信息失败, key=%s, err=%v", string(ev.Kv.Key), err))
				continue
			}

			// 查找是否已存在（按 Addr 匹配）
			found := false
			for i := range ns.gameServers {
				if ns.gameServers[i].Addr == server.Addr {
					// 更新现有节点
					ns.gameServers[i] = server
					found = true
					updated = true
					log.Info(fmt.Sprintf("NodeSelector 更新 game 节点: %s, Load=%.2f", server.Addr, server.Load))
					break
				}
			}

			if !found {
				// 添加新节点
				ns.gameServers = append(ns.gameServers, server)
				updated = true
				log.Info(fmt.Sprintf("NodeSelector 新增 game 节点: %s, Load=%.2f", server.Addr, server.Load))
			}

		case clientv3.EventTypeDelete:
			// 删除节点
			server, err := discovery.ParseKey(string(ev.Kv.Key))
			if err != nil {
				log.Error(fmt.Sprintf("NodeSelector 解析删除节点 key 失败, key=%s, err=%v", string(ev.Kv.Key), err))
				continue
			}

			// 从列表中删除
			for i := range ns.gameServers {
				if ns.gameServers[i].Addr == server.Addr {
					// 使用切片技巧删除元素
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
		log.Info(fmt.Sprintf("NodeSelector 节点列表已更新, 当前共有 %d 个 game 节点", len(ns.gameServers)))
	}
}

// SelectGameNode 选择 game 节点（根据负载均衡策略）
func (ns *NodeSelector) SelectGameNode(ctx context.Context) (*discovery.Server, error) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	// 过滤掉 Load <= 0 的节点（不健康的节点）
	healthyServers := make([]discovery.Server, 0, len(ns.gameServers))
	for _, server := range ns.gameServers {
		if server.Load > 0 {
			healthyServers = append(healthyServers, server)
		}
	}

	if len(healthyServers) == 0 {
		return nil, errors.New("没有可用的 game 节点（所有节点负载 <= 0 或列表为空）")
	}

	// 使用负载均衡策略选择节点
	selected, err := discovery.SelectServer(healthyServers, ns.strategy)
	if err != nil {
		return nil, fmt.Errorf("选择 game 节点失败: %v", err)
	}

	log.Info(fmt.Sprintf("NodeSelector 选择 game 节点: %s, Load=%.2f, Strategy=%v", selected.Addr, selected.Load, ns.strategy))
	return selected, nil
}

// GetGameNodes 获取所有 game 节点列表（包括不健康的节点）
func (ns *NodeSelector) GetGameNodes() []discovery.Server {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	// 返回副本，避免外部修改
	result := make([]discovery.Server, len(ns.gameServers))
	copy(result, ns.gameServers)
	return result
}

// GetHealthyGameNodes 获取所有健康的 game 节点列表（Load > 0）
func (ns *NodeSelector) GetHealthyGameNodes() []discovery.Server {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	healthyServers := make([]discovery.Server, 0)
	for _, server := range ns.gameServers {
		if server.Load > 0 {
			healthyServers = append(healthyServers, server)
		}
	}
	return healthyServers
}

// UpdateStrategy 更新负载均衡策略
func (ns *NodeSelector) UpdateStrategy(strategy discovery.LoadBalanceStrategy) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.strategy = strategy
	log.Info(fmt.Sprintf("NodeSelector 更新负载均衡策略: %v", strategy))
}

// Close 关闭节点选择器
func (ns *NodeSelector) Close() error {
	if ns.seeker != nil {
		return ns.seeker.Close()
	}
	return nil
}
