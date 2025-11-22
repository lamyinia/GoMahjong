package discovery

import (
	"common/config"
	"common/log"
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

/*
	服务发现客户端，用于主动获取服务列表
	主要用于 march 服务选择 game 节点做负载均衡
*/

type Seeker struct {
	etcdCli     *clientv3.Client
	DialTimeout int
	conf        config.EtcdConf
}

func NewSeeker(conf config.EtcdConf) (*Seeker, error) {
	etcdCli, err := clientv3.New(clientv3.Config{
		Endpoints:   conf.Addrs,
		DialTimeout: time.Duration(conf.DialTimeout) * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 etcd 客户端失败: %v", err)
	}

	return &Seeker{
		etcdCli:     etcdCli,
		DialTimeout: conf.DialTimeout,
		conf:        conf,
	}, nil
}

// GetServers 获取指定服务名称的所有服务实例
// serviceName: 服务名称，如 "game"
func (seeker *Seeker) GetServers(serviceName string) ([]Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(seeker.conf.RWTimeout)*time.Second)
	defer cancel()

	key := serviceName + "/"
	res, err := seeker.etcdCli.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("从 etcd 获取服务列表失败: %v", err)
	}

	servers := make([]Server, 0, len(res.Kvs))
	for _, kv := range res.Kvs {
		server, err := ParseValue(kv.Value)
		if err != nil {
			log.Error(fmt.Sprintf("解析服务信息失败, key=%s, err=%v", string(kv.Key), err))
			continue
		}
		servers = append(servers, server)
	}

	return servers, nil
}

// WatchServers 监听服务变化，当服务列表发生变化时调用回调函数
// serviceName: 服务名称
// callback: 服务列表变化时的回调函数
func (seeker *Seeker) WatchServers(serviceName string, callback func([]Server)) error {
	key := serviceName + "/"
	watchCh := seeker.etcdCli.Watch(context.Background(), key, clientv3.WithPrefix())

	go func() {
		for watchResp := range watchCh {
			if watchResp.Canceled {
				log.Warn(fmt.Sprintf("watch 被取消, serviceName=%s", serviceName))
				return
			}

			// 重新获取服务列表
			servers, err := seeker.GetServers(serviceName)
			if err != nil {
				log.Error(fmt.Sprintf("watch 获取服务列表失败, serviceName=%s, err=%v", serviceName, err))
				continue
			}

			callback(servers)
		}
	}()

	return nil
}

// Watch 返回 etcd Watch 通道，用于增量更新
// serviceName: 服务名称，如 "game"
func (seeker *Seeker) Watch(serviceName string) clientv3.WatchChan {
	key := serviceName + "/"
	return seeker.etcdCli.Watch(context.Background(), key, clientv3.WithPrefix())
}

func (seeker *Seeker) Close() error {
	if seeker.etcdCli != nil {
		return seeker.etcdCli.Close()
	}
	return nil
}
