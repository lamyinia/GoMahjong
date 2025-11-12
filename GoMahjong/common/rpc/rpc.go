package rpc

import (
	"common/config"
	"common/discovery"
	"common/log"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"player/pb"
)

var (
	UserClient pb.UserServiceClient
)

func Init() {
	r := discovery.NewResolver(config.Conf.EtcdConf)
	resolver.Register(r)
	userDomain := config.Conf.Domain["user"]
	initClient(userDomain.Name, userDomain.LoadBalance, &UserClient)
	log.Info(fmt.Sprintf("rpc 发现服务，%#v", userDomain))
}

func initClient(name string, loadBalance bool, client interface{}) {
	//找服务的地址
	addr := fmt.Sprintf("etcd:///%s", name)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials())}
	if loadBalance {
		opts = append(opts, grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, "round_robin")))
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		log.Fatal(fmt.Sprintf("rpc 连接 etcd 失败:%v", err))
	}
	switch c := client.(type) {
	case *pb.UserServiceClient:
		*c = pb.NewUserServiceClient(conn)
	default:
		log.Fatal(fmt.Sprintf("不支持的服务类型, %#v", c))
	}
}
