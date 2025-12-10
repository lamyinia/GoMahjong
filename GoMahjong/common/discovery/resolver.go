package discovery

import (
	"common/config"
	"common/log"
	"context"
	"fmt"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
)

/*
	grpc 和 gin 组合使用的服务
	srvAddrList 会把服务器信息装载到内存
*/

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

// Build grpc.Dial 的时候就会同步调用此方法
func (r *Resolver) Build(target resolver.Target, clientConn resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r.clientConn = clientConn

	// 建立etcd的连接
	var err error
	r.etcdCli, err = clientv3.New(clientv3.Config{
		Endpoints:   r.conf.Addrs,
		DialTimeout: time.Duration(r.DialTimeout) * time.Second,
	})
	if err != nil {
		log.Fatal(fmt.Sprintf("Build grpc 客户端连接 etcd 失败:%v", err))
	}
	r.closeCh = make(chan struct{})

	// 根据 key 获取 value，去掉前导 "/"
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

// 1.定时 1分钟同步一次数据。2.监听 etcd 的事件 从而触发不同的操作。3.监听 Close 事件 关闭 etcd。
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

// 全量同步，初始化定时器
func (r *Resolver) sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.conf.RWTimeout)*time.Second)
	defer cancel()

	// 示例：user/v1/xxx:2222
	// 使用 WithPrefix() 获取所有匹配的键，但需要精确过滤以避免误选
	// 例如：如果 r.key = "user/v1"，不应该匹配到 "user/v10"
	res, err := r.etcdCli.Get(ctx, r.key, clientv3.WithPrefix())
	if err != nil {
		log.Error("sync() grpc 客户端获取 etcd 错误")
		return err
	}
	log.Debug(fmt.Sprintf("sync() etcd 客户端同步结果：%+v", res))

	r.srvAddrList = []resolver.Address{}
	for _, kv := range res.Kvs {
		// 精确匹配：确保 key 以 r.key + "/" 开头，避免前缀误选
		// 例如：r.key = "user/v1" 时，只匹配 "user/v1/" 开头的键，不匹配 "user/v10/"
		keyStr := string(kv.Key)
		expectedPrefix := r.key + "/"
		if !strings.HasPrefix(keyStr, expectedPrefix) {
			// 如果 key 正好等于 r.key（不应该发生，但防御性检查）
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

// 增量同步
func (r *Resolver) update(events []*clientv3.Event) {
	for _, ev := range events {
		keyStr := string(ev.Kv.Key)
		expectedPrefix := r.key + "/"

		// 精确匹配：确保 key 以 r.key + "/" 开头，避免前缀误选，防御性检查
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
			// 如果服务已存在，更新它；否则添加新服务
			if idx := findIndex(r.srvAddrList, addr); idx >= 0 {
				// 更新现有服务
				r.srvAddrList[idx] = addr
			} else {
				// 添加新服务
				r.srvAddrList = append(r.srvAddrList, addr)
			}
			err = r.clientConn.UpdateState(resolver.State{
				Addresses: r.srvAddrList,
			})
			if err != nil {
				log.Error(fmt.Sprintf("update() grpc 客户端 update(EventTypePut) UpdateState 失败, name=%s,err:%v", r.key, err))
			}
		case clientv3.EventTypeDelete:
			//接收到delete操作 删除r.srvAddrList其中匹配的
			// user/v1/127.0.0.1:12000
			server, err := ParseKey(keyStr)
			if err != nil {
				log.Error(fmt.Sprintf("update() grpc 客户端 update(EventTypeDelete) parse etcd 失败, name=%s,err:%v", r.key, err))
				continue
			}
			addr := resolver.Address{Addr: server.Addr}
			//r.srvAddrList remove操作
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
