package discovery

import (
	"context"
	"fmt"
	"gate/infrastructure/config"
	"gate/infrastructure/log"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
)

const schema = "etcd"

type Resolver struct {
	etcdConf config.EtcdConf
	client   *clientv3.Client
	cc       resolver.ClientConn
	addrList []resolver.Address
	services map[string][]string
	mu       sync.Mutex
	domain   string
}

func NewResolver(etcdConf config.EtcdConf) *Resolver {
	return &Resolver{
		etcdConf: etcdConf,
		services: make(map[string][]string),
	}
}

func (r *Resolver) Scheme() string {
	return schema
}

func (r *Resolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r.cc = cc
	r.domain = target.Endpoint()

	var err error
	r.client, err = clientv3.New(clientv3.Config{
		Endpoints:   r.etcdConf.Addrs,
		DialTimeout: time.Duration(r.etcdConf.DialTimeout) * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("etcd resolver: failed to connect: %w", err)
	}

	go r.watch(r.domain)
	return r, nil
}

func (r *Resolver) ResolveNow(resolver.ResolveNowOptions) {}

func (r *Resolver) Close() {
	if r.client != nil {
		r.client.Close()
	}
}

func (r *Resolver) watch(domain string) {
	prefix := fmt.Sprintf("%s/", domain)
	resp, err := r.client.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		log.Error("etcd resolver: get failed: %v", err)
		return
	}

	r.mu.Lock()
	var addrs []resolver.Address
	for _, kv := range resp.Kvs {
		server, err := ParseValue(kv.Value)
		if err != nil {
			continue
		}
		addr := server.Addr
		addrs = append(addrs, resolver.Address{Addr: addr})
		r.services[domain] = append(r.services[domain], addr)
	}
	r.addrList = addrs
	r.mu.Unlock()

	if len(addrs) > 0 {
		r.cc.UpdateState(resolver.State{Addresses: addrs})
	}

	watchCh := r.client.Watch(context.Background(), prefix, clientv3.WithPrefix())
	for {
		select {
		case watchResp, ok := <-watchCh:
			if !ok {
				return
			}
			for _, event := range watchResp.Events {
				switch event.Type {
				case clientv3.EventTypePut:
					server, err := ParseValue(event.Kv.Value)
					if err != nil {
						continue
					}
					r.mu.Lock()
					r.services[domain] = append(r.services[domain], server.Addr)
					r.updateState(domain)
					r.mu.Unlock()
				case clientv3.EventTypeDelete:
					r.mu.Lock()
					key := string(event.Kv.Key)
					server, err := ParseKey(key)
					if err != nil {
						r.mu.Unlock()
						continue
					}
					var newAddrs []string
					for _, addr := range r.services[domain] {
						if addr != server.Addr {
							newAddrs = append(newAddrs, addr)
						}
					}
					r.services[domain] = newAddrs
					r.updateState(domain)
					r.mu.Unlock()
				}
			}
		}
	}
}

func (r *Resolver) updateState(domain string) {
	var addrs []resolver.Address
	for _, addr := range r.services[domain] {
		addrs = append(addrs, resolver.Address{Addr: addr})
	}
	r.addrList = addrs
	if len(addrs) > 0 {
		r.cc.UpdateState(resolver.State{Addresses: addrs})
	}
}
