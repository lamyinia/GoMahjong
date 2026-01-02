package container

import (
	"common/config"
	"common/log"
)

type GateContainer struct {
	*BaseContainer
}

// NewGateContainer 创建 gate 服务容器
func NewGateContainer() *GateContainer {
	base := NewBase(config.GateNodeConfig.DatabaseConf)
	if base == nil {
		log.Fatal("基础容器初始化失败")
		return nil
	}

	return &GateContainer{
		BaseContainer: base,
	}
}

// Close 关闭容器资源
func (c *GateContainer) Close() error {
	return c.BaseContainer.Close()
}
