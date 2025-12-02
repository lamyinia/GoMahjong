package rpc

import (
	"common/config"
	"common/discovery"
	"common/log"
	"fmt"
	matchpb "march/pb"
	userpb "player/pb"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

var (
	UserClient  userpb.UserServiceClient
	MatchClient matchpb.MatchServiceClient
)

func Init() {
	r := discovery.NewResolver(config.Conf.EtcdConf)
	resolver.Register(r)
	userDomain := config.Conf.Domain["user"]
	initClient(userDomain.Name, userDomain.LoadBalance, &UserClient)
	log.Info(fmt.Sprintf("rpc 发现 user 服务，%#v", userDomain))

	marchDomain, ok := config.Conf.Domain["march"]
	if !ok {
		log.Fatal("rpc 初始化失败: 未配置 march domain")
	}
	initClient(marchDomain.Name, marchDomain.LoadBalance, &MatchClient)
	log.Info(fmt.Sprintf("rpc 发现 march 服务，%#v", marchDomain))
}

// client 结构体指针，大小 8 字节
func initClient(name string, loadBalance bool, client interface{}) {
	// 找服务的地址，强制 host 为空
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
	// 指针被装进接口，类型断言提取出来，地址信息完全保留
	switch c := client.(type) {
	case *userpb.UserServiceClient:
		*c = userpb.NewUserServiceClient(conn)
	case *matchpb.MatchServiceClient:
		*c = matchpb.NewMatchServiceClient(conn)
	default:
		log.Fatal(fmt.Sprintf("不支持的服务类型, %#v", c))
	}
}
