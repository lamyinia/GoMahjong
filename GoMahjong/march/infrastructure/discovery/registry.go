package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"march/infrastructure/config"
	"march/infrastructure/log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

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

func (r *Registry) Register(conf config.EtcdConf, nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("nodeID 不能为空，NATS 通信需要 nodeID")
	}

	info := Server{
		Domain:  conf.Register.Domain,
		Addr:    conf.Register.Addr,
		Weight:  conf.Register.Weight,
		Version: conf.Register.Version,
		Ttl:     conf.Register.Ttl,
		NodeID:  nodeID,
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

	data, _ := json.Marshal(r.info)
	err = r.bindLease(ctx, r.info.buildKey(), string(data))
	log.Info("etcd 注册信息: %s", r.info.buildKey())
	if err != nil {
		return err
	}

	r.keepAliveCh, err = r.keepAlive(context.Background())
	if err != nil {
		return err
	}

	return nil
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
		log.Error("租约绑定失败: %v", err)
		return err
	}
	return nil
}

func (r *Registry) keepAlive(ctx context.Context) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	keepAliveCh, err := r.etcdCli.KeepAlive(ctx, r.leaseID)
	if err != nil {
		log.Error("租约续期失败: %v", err)
		return nil, err
	}
	return keepAliveCh, nil
}

func (r *Registry) watch() {
	ticker := time.NewTicker(time.Duration(r.info.Ttl/2) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case res, ok := <-r.keepAliveCh:
			if !ok || res == nil {
				log.Warn("keepAlive 连接断开，重新注册服务")
				r.keepAliveCh = nil
				if err := r.doRegister(); err != nil {
					log.Error("重新注册失败: %v", err)
					time.Sleep(time.Duration(r.info.Ttl) * time.Second)
				} else {
					log.Info("重新注册成功")
				}
			}
		case <-ticker.C:
			if r.keepAliveCh == nil {
				log.Warn("定时器检测到 keepAlive 连接断开，重新注册服务")
				if err := r.doRegister(); err != nil {
					log.Error("定时器重新注册失败: %v", err)
				} else {
					log.Info("定时器重新注册成功")
				}
			}
		case <-r.closeCh:
			if err := r.unregister(); err != nil {
				log.Error("注销服务失败: %v", err)
			}
			if _, err := r.etcdCli.Revoke(context.Background(), r.leaseID); err != nil {
				log.Error("撤销租约失败: %v", err)
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
		log.Error("注销失败: %v", err)
		return err
	}
	return nil
}

func (r *Registry) UpdateLoad(load float64) error {
	r.info.Load = load
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.DialTimeout)*time.Second)
	defer cancel()

	data, err := json.Marshal(r.info)
	if err != nil {
		return err
	}

	_, err = r.etcdCli.Put(ctx, r.info.buildKey(), string(data), clientv3.WithLease(r.leaseID))
	if err != nil {
		log.Error("更新负载信息失败: %v", err)
		return err
	}
	return nil
}

func (r *Registry) Close() {
	close(r.closeCh)
}
