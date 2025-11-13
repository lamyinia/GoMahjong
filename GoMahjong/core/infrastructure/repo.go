package infrastructure

import (
	"common/log"
	"core/container"
)

// Container 是依赖注入容器（已弃用，保留用于向后兼容）
// 新代码应该使用 core/container 中的服务特定容器
// 例如：container.NewPlayerContainer()、container.NewHallContainer() 等
type Container = container.PlayerContainer

// New 创建容器并初始化所有依赖（已弃用，保留用于向后兼容）
// 推荐使用：container.NewPlayerContainer()
func New() *Container {
	log.Warn("使用已弃用的 infrastructure.New()，建议改用 container.NewPlayerContainer()")
	return container.NewPlayerContainer()
}
