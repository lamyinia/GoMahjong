package discovery

import (
	"connector/infrastructure/config"
	"connector/infrastructure/log"
	"context"
	"fmt"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
)

type Resolver struct {
	conf        config.EtcdConf
	etcdCli     *clientv3.Client
	DialTimeout int
	closeCh     chan struct{}
	key         string
	clientConn  resolver.ClientConn
	srvAddrList []resolver.Address
	watchCh     clientv3.WatchChan
}

func NewResolver(conf config.EtcdConf) *Resolver {
	return &Resolver{
		conf:        conf,
		DialTimeout: conf.DialTimeout,
	}
}

func (r *Resolver) Build(target resolver.Target, clientConn resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r.clientConn = clientConn

	var err error
	r.etcdCli, err = clientv3.New(clientv3.Config{
		Endpoints:   r.conf.Addrs,
		DialTimeout: time.Duration(r.DialTimeout) * time.Second,
	})
	if err != nil {
		log.Fatal(fmt.Sprintf("Build grpc 客户端连接 etcd 失败:%v", err))
	}
	r.closeCh = make(chan struct{})

	r.key = strings.TrimPrefix(target.URL.Path, "/")
	if err = r.sync(); err != nil {
		return nil, err
	}

	go r.watch()
	return nil, nil
}

func (r *Resolver) Scheme() string {
	return "etcd"
}

func (r *Resolver) watch() {
	ticker := time.NewTicker(time.Minute)
	r.watchCh = r.etcdCli.Watch(context.Background(), r.key, clientv3.WithPrefix())

	for {
		select {
		case <-r.closeCh:
			r.Close()
		case info, ok := <-r.watchCh:
			if ok {
				r.update(info.Events)
			}
		case <-ticker.C:
			if err := r.sync(); err != nil {
				log.Error(fmt.Sprintf("watch sync 失败,err:%v", err))
			}
		}
	}
}

func (r *Resolver) sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.conf.RWTimeout)*time.Second)
	defer cancel()

	res, err := r.etcdCli.Get(ctx, r.key, clientv3.WithPrefix())
	if err != nil {
		log.Error("sync() grpc 客户端获取 etcd 错误")
		return err
	}
	log.Debug(fmt.Sprintf("sync() etcd 客户端同步结果：%+v", res))

	r.srvAddrList = []resolver.Address{}
	for _, kv := range res.Kvs {
		keyStr := string(kv.Key)
		expectedPrefix := r.key + "/"
		if !strings.HasPrefix(keyStr, expectedPrefix) {
			if keyStr != r.key {
				log.Warn(fmt.Sprintf("sync() 跳过不匹配的键: %s (期望前缀: %s)", keyStr, expectedPrefix))
				continue
			}
		}

		server, err := ParseValue(kv.Value)
		if err != nil {
			log.Error(fmt.Sprintf("sync() grpc 客户端解析 etcd 失败, name=%s,err:%v", r.key, err))
			continue
		}
		r.srvAddrList = append(r.srvAddrList, resolver.Address{
			Addr:       server.Addr,
			Attributes: attributes.New("weight", server.Weight),
		})
	}

	if len(r.srvAddrList) == 0 {
		log.Error("sync() 未发现服务")
		return nil
	}

	err = r.clientConn.UpdateState(resolver.State{
		Addresses: r.srvAddrList,
	})
	if err != nil {
		log.Error(fmt.Sprintf("sync() grpc 客户端全量更新服务失败, name=%s, err: %v", r.key, err))
		return err
	}
	return nil
}

func (r *Resolver) update(events []*clientv3.Event) {
	for _, ev := range events {
		keyStr := string(ev.Kv.Key)
		expectedPrefix := r.key + "/"

		if !strings.HasPrefix(keyStr, expectedPrefix) {
			if keyStr != r.key {
				log.Warn(fmt.Sprintf("update() 跳过不匹配的键: %s (期望前缀: %s)", keyStr, expectedPrefix))
				continue
			}
		}

		switch ev.Type {
		case clientv3.EventTypePut:
			server, err := ParseValue(ev.Kv.Value)
			if err != nil {
				log.Error(fmt.Sprintf("update() grpc 客户端 update(EventTypePut) parse etcd 失败, name=%s,err:%v", r.key, err))
				continue
			}
			addr := resolver.Address{
				Addr:       server.Addr,
				Attributes: attributes.New("weight", server.Weight),
			}
			if idx := findIndex(r.srvAddrList, addr); idx >= 0 {
				r.srvAddrList[idx] = addr
			} else {
				r.srvAddrList = append(r.srvAddrList, addr)
			}
			err = r.clientConn.UpdateState(resolver.State{
				Addresses: r.srvAddrList,
			})
			if err != nil {
				log.Error(fmt.Sprintf("update() grpc 客户端 update(EventTypePut) UpdateState 失败, name=%s,err:%v", r.key, err))
			}
		case clientv3.EventTypeDelete:
			server, err := ParseKey(keyStr)
			if err != nil {
				log.Error(fmt.Sprintf("update() grpc 客户端 update(EventTypeDelete) parse etcd 失败, name=%s,err:%v", r.key, err))
				continue
			}
			addr := resolver.Address{Addr: server.Addr}
			if list, ok := remove(r.srvAddrList, addr); ok {
				r.srvAddrList = list
				err = r.clientConn.UpdateState(resolver.State{
					Addresses: r.srvAddrList,
				})
				if err != nil {
					log.Error(fmt.Sprintf("update() grpc 客户端 update(EventTypeDelete) UpdateState 失败, name=%s,err:%v", r.key, err))
				}
			}
		}
	}
}

func findIndex(list []resolver.Address, addr resolver.Address) int {
	for i := range list {
		if list[i].Addr == addr.Addr {
			return i
		}
	}
	return -1
}

func remove(list []resolver.Address, tar resolver.Address) ([]resolver.Address, bool) {
	for i, item := range list {
		if item.Addr == tar.Addr {
			list[i] = list[len(list)-1]
			return list[:len(list)-1], true
		}
	}
	return nil, false
}

func (r *Resolver) Close() {
	if r.etcdCli != nil {
		err := r.etcdCli.Close()
		if err != nil {
			log.Error(fmt.Sprintf("Resolver 关闭 etcd 错误:%v", err))
		}
		log.Info("成功关闭 etcd 连接...")
	}
}
