package discovery

import (
	"common/config"
	"common/log"
	"context"
	"encoding/json"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Register struct {
	etcdCli     *clientv3.Client
	leaseID     clientv3.LeaseID
	DialTimeout int
	keepAliveCh <-chan *clientv3.LeaseKeepAliveResponse
	info        Server
	closeCh     chan struct{}
}

func NewRegister() *Register {
	return &Register{
		DialTimeout: 3,
	}
}

func (r *Register) Register(conf config.EtcdConf) error {
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

	err = r.register()
	if err != nil {
		return err
	}

	r.closeCh = make(chan struct{})
	go r.watch()
	return nil
}

func (r *Register) register() error {
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

	data, _ := json.Marshal(r.info)
	return r.bindLease(ctx, r.info.buildKey(), string(data))
}

func (r *Register) grantLease(ctx context.Context, ttl int) error {
	lease, err := r.etcdCli.Grant(ctx, int64(ttl))
	if err != nil {
		return err
	}
	r.leaseID = lease.ID
	return nil
}

func (r *Register) bindLease(ctx context.Context, key, value string) error {
	_, err := r.etcdCli.Put(ctx, key, value, clientv3.WithLease(r.leaseID))
	if err != nil {
		log.Error("租约绑定失败: {}", err)
		return err
	}
	return nil
}

func (r *Register) keepAlive(ctx context.Context) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	keepAliveCh, err := r.etcdCli.KeepAlive(ctx, r.leaseID)
	if err != nil {
		log.Error("租约续期失败: {}", err)
		return nil, err
	}
	return keepAliveCh, nil
}

func (r *Register) watch() {
	ticker := time.NewTicker(time.Duration(r.info.Ttl) * time.Second)

	for {
		select {
		case res := <-r.keepAliveCh:
			if res != nil {
				if err := r.register(); err != nil {
					log.Error("租约续期失败: {}", err)
				}
			}
		case <-ticker.C:
			if err := r.register(); err != nil {
				log.Error("租约续期失败: {}", err)
			}
		case <-r.closeCh:
			if err := r.unregister(); err != nil {
				log.Error("租约续期失败: {}", err)
			}
			if _, err := r.etcdCli.Revoke(context.Background(), r.leaseID); err != nil {
				log.Error("租约续期失败: {}", err)
			}
			log.Info("关闭租约续期")
			return
		}
	}
}

func (r *Register) unregister() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.DialTimeout)*time.Second)
	defer cancel()

	_, err := r.etcdCli.Delete(ctx, r.info.buildKey())
	if err != nil {
		log.Error("租约续期失败: {}", err)
		return err
	}
	return nil
}

func (r *Register) Close() {
	r.closeCh <- struct{}{}
}
