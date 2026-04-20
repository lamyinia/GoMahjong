package discovery

import (
	"context"
	"fmt"
	"march/infrastructure/config"
	"march/infrastructure/log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

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
