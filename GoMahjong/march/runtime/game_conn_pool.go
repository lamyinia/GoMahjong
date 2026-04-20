package runtime

import (
	"fmt"
	"march/infrastructure/log"
	pb "march/pb"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

type GameConnPool struct {
	conns map[string]*grpc.ClientConn
	mu    sync.RWMutex
}

func NewGameConnPool() *GameConnPool {
	return &GameConnPool{
		conns: make(map[string]*grpc.ClientConn),
	}
}

func (p *GameConnPool) GetClient(gameNodeAddr string) (pb.GameServiceClient, error) {
	p.mu.RLock()
	conn, exists := p.conns[gameNodeAddr]
	p.mu.RUnlock()

	if exists && conn != nil && conn.GetState() == connectivity.Ready {
		return pb.NewGameServiceClient(conn), nil
	}
	conn, err := grpc.NewClient(gameNodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接 Game 节点 %s 失败: %v", gameNodeAddr, err)
	}

	p.mu.Lock()
	p.conns[gameNodeAddr] = conn
	p.mu.Unlock()

	log.Info(fmt.Sprintf("GameConnPool 创建新连接: %s", gameNodeAddr))
	return pb.NewGameServiceClient(conn), nil
}

func (p *GameConnPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for gameNodeID, conn := range p.conns {
		if conn != nil {
			if err := conn.Close(); err != nil {
				log.Error(fmt.Sprintf("关闭连接 %s 失败: %v", gameNodeID, err))
				errs = append(errs, err)
			}
		}
	}

	p.conns = make(map[string]*grpc.ClientConn)

	if len(errs) > 0 {
		return fmt.Errorf("关闭连接池时发生 %d 个错误", len(errs))
	}

	log.Info("GameConnPool 已关闭")
	return nil
}
