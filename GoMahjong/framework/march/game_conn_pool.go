package march

import (
	"common/log"
	"fmt"
	pb "game/pb"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// GameConnPool 管理与 Game 节点的 gRPC 连接
type GameConnPool struct {
	conns map[string]*grpc.ClientConn // gameNodeID -> connection
	mu    sync.RWMutex
}

// NewGameConnPool 创建连接池
func NewGameConnPool() *GameConnPool {
	return &GameConnPool{
		conns: make(map[string]*grpc.ClientConn),
	}
}

// GetClient 获取 Game 节点的 gRPC 客户端
// gameNodeAddr: game 节点的地址，格式为 "host:port"
func (p *GameConnPool) GetClient(gameNodeAddr string) (pb.GameServiceClient, error) {
	p.mu.RLock()
	conn, exists := p.conns[gameNodeAddr]
	p.mu.RUnlock()

	// 如果连接已存在且有效，直接返回
	if exists && conn != nil && conn.GetState() == connectivity.Ready {
		return pb.NewGameServiceClient(conn), nil
	}

	// 创建新连接
	conn, err := grpc.NewClient(gameNodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接 Game 节点 %s 失败: %v", gameNodeAddr, err)
	}

	// 保存连接到池中
	p.mu.Lock()
	p.conns[gameNodeAddr] = conn
	p.mu.Unlock()

	log.Info(fmt.Sprintf("GameConnPool 创建新连接: %s", gameNodeAddr))
	return pb.NewGameServiceClient(conn), nil
}

// Close 关闭所有连接
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
