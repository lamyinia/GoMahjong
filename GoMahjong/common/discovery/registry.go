package discovery

import (
	"common/config"
	"common/log"
	"context"
	"encoding/json"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

/*
etcd 注册器
	1.服务端注册到 etcd，以供其它 grpc 客户端(gate)发现
	2.game 节点注册到 etcd，以供 march 节点负载均衡
	3.需要特别关注 etcd 服务的租约机制
*/

type Registry struct {
	etcdCli     *clientv3.Client
	leaseID     clientv3.LeaseID
	DialTimeout int
	keepAliveCh <-chan *clientv3.LeaseKeepAliveResponse
	info        Server
	closeCh     chan struct{}
}

func NewRegistry() *Registry {
	return &Registry{
		DialTimeout: 3,
	}
}

func (r *Registry) Register(conf config.EtcdConf) error {
	info := Server{
		Name:    conf.Register.Name,
		Addr:    conf.Register.Addr,
		Weight:  conf.Register.Weight,
		Version: conf.Register.Version,
		Ttl:     conf.Register.Ttl,
	}
	r.info = info

	var err error
	r.etcdCli, err = clientv3.New(clientv3.Config{
		Endpoints:   conf.Addrs,
		DialTimeout: time.Duration(r.DialTimeout) * time.Second,
	})
	if err != nil {
		return err
	}

	err = r.doRegister()
	if err != nil {
		return err
	}

	r.closeCh = make(chan struct{})
	go r.watch()
	return nil
}

func (r *Registry) doRegister() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.DialTimeout)*time.Second)
	defer cancel()

	err := r.grantLease(ctx, r.info.Ttl)
	if err != nil {
		return err
	}

	r.keepAliveCh, err = r.keepAlive(ctx)
	if err != nil {
		return err
	}

	data, _ := json.Marshal(r.info) // 存进去的 key 是 json，读出来的也应该要 json 解析
	return r.bindLease(ctx, r.info.buildKey(), string(data))
}

func (r *Registry) grantLease(ctx context.Context, ttl int) error {
	lease, err := r.etcdCli.Grant(ctx, int64(ttl))
	if err != nil {
		return err
	}
	r.leaseID = lease.ID
	return nil
}

func (r *Registry) bindLease(ctx context.Context, key, value string) error {
	_, err := r.etcdCli.Put(ctx, key, value, clientv3.WithLease(r.leaseID))
	if err != nil {
		log.Error("租约绑定失败: {}", err)
		return err
	}
	return nil
}

func (r *Registry) keepAlive(ctx context.Context) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	keepAliveCh, err := r.etcdCli.KeepAlive(ctx, r.leaseID)
	if err != nil {
		log.Error("租约续期失败: {}", err)
		return nil, err
	}
	return keepAliveCh, nil
}

func (r *Registry) watch() {
	ticker := time.NewTicker(time.Duration(r.info.Ttl) * time.Second)

	for {
		select {
		case res := <-r.keepAliveCh:
			// keepAliveCh 收到响应说明 etcd 已自动续期，不需要重新创建租约
			// 如果 res == nil，说明连接断开，需要重新注册
			if res == nil {
				log.Warn("keepAlive 连接断开，重新注册服务")
				if err := r.doRegister(); err != nil {
					log.Error("重新注册失败: {}", err)
				}
			}
			// res != nil 时，续期成功，不需要任何操作
		case <-ticker.C:
			// 定时器作为兜底检查：如果 keepAliveCh 为 nil（连接断开），才重新注册
			if r.keepAliveCh == nil {
				log.Warn("检测到 keepAlive 连接断开，重新注册服务")
				if err := r.doRegister(); err != nil {
					log.Error("定时器重新注册失败: {}", err)
				}
			}
		case <-r.closeCh:
			if err := r.unregister(); err != nil {
				log.Error("注销服务失败: {}", err)
			}
			if _, err := r.etcdCli.Revoke(context.Background(), r.leaseID); err != nil {
				log.Error("撤销租约失败: {}", err)
			}
			log.Info("关闭租约续期")
			return
		}
	}
}

func (r *Registry) unregister() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.DialTimeout)*time.Second)
	defer cancel()

	_, err := r.etcdCli.Delete(ctx, r.info.buildKey())
	if err != nil {
		log.Error("租约续期失败: {}", err)
		return err
	}
	return nil
}

// UpdateLoad 更新服务负载信息（不重新创建租约）
// load: 负载评分，值越小表示负载越低
func (r *Registry) UpdateLoad(load float64) error {
	r.info.Load = load
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.DialTimeout)*time.Second)
	defer cancel()

	data, err := json.Marshal(r.info)
	if err != nil {
		return err
	}

	// 使用现有租约更新服务信息
	_, err = r.etcdCli.Put(ctx, r.info.buildKey(), string(data), clientv3.WithLease(r.leaseID))
	if err != nil {
		log.Error("更新负载信息失败: {}", err)
		return err
	}
	return nil
}

func (r *Registry) Close() {
	r.closeCh <- struct{}{}
}
