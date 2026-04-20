package rpc

import (
	"connector/infrastructure/config"
	"connector/infrastructure/discovery"
	"connector/infrastructure/log"
	matchpb "connector/pb"
	userpb "connector/pb"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

var (
	UserClient  userpb.UserServiceClient
	MatchClient matchpb.MatchServiceClient
)

func Init(domains map[string]config.Domain, etcdConf config.EtcdConf) {
	r := discovery.NewResolver(etcdConf)
	resolver.Register(r)

	authDomain, ok := domains["auth"]
	if !ok {
		log.Fatal("rpc 初始化失败: 未配置 auth domain")
	}
	initClient(authDomain.Name, authDomain.LoadBalance, &UserClient)
	log.Info(fmt.Sprintf("rpc 发现 auth 服务，%#v", authDomain))

	marchDomain, ok := domains["march"]
	if !ok {
		log.Fatal("rpc 初始化失败: 未配置 march domain")
	}
	initClient(marchDomain.Name, marchDomain.LoadBalance, &MatchClient)
	log.Info(fmt.Sprintf("rpc 发现 march 服务，%#v", marchDomain))
}

func initClient(name string, loadBalance bool, client interface{}) {
	addr := fmt.Sprintf("etcd:///%s", strings.TrimPrefix(name, "/"))
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	if loadBalance {
		opts = append(opts, grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, "round_robin")))
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		log.Fatal(fmt.Sprintf("rpc 连接 etcd 失败:%v", err))
	}
	switch c := client.(type) {
	case *userpb.UserServiceClient:
		*c = userpb.NewUserServiceClient(conn)
	case *matchpb.MatchServiceClient:
		*c = matchpb.NewMatchServiceClient(conn)
	default:
		log.Fatal(fmt.Sprintf("不支持的服务类型, %#v", c))
	}
}
