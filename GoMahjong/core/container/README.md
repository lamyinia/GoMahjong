# 依赖注入容器架构

## 概述

这个目录实现了**服务隔离的依赖注入架构**，为每个微服务提供专用的容器，确保依赖清晰、易于测试和扩展。

## 架构设计

```
BaseContainer（基础容器）
  ├─ mongo: MongoManager
  ├─ redis: RedisManager
  └─ 所有服务共享的资源

PlayerContainer（player 服务容器）
  ├─ 继承 BaseContainer
  └─ userRepository: UserRepository

HallContainer（hall 服务容器）
  ├─ 继承 BaseContainer
  └─ userRepository: UserRepository
  └─ TODO: roomRepository、tableRepository

GameContainer（game 服务容器）
  ├─ 继承 BaseContainer
  └─ userRepository: UserRepository
  └─ TODO: gameRepository、ruleRepository

GateContainer（gate 服务容器）
  ├─ 继承 BaseContainer
  └─ userRepository: UserRepository
  └─ TODO: sessionRepository
```

## 文件说明

| 文件 | 职责 |
|------|------|
| `base.go` | 基础容器，管理数据库连接等共享资源 |
| `player_container.go` | player 服务专用容器 |
| `hall_container.go` | hall 服务专用容器 |
| `game_container.go` | game 服务专用容器 |
| `gate_container.go` | gate 服务专用容器 |

## 使用方法

### Player 服务

```go
import "core/container"

func Run(ctx context.Context) error {
    // 创建 player 服务容器
    playerContainer := container.NewPlayerContainer()
    defer playerContainer.Close()
    
    // 使用容器中的依赖
    userRepo := playerContainer.GetUserRepository()
    redis := playerContainer.GetRedis()
    
    // ... 业务逻辑
}
```

### Hall 服务

```go
import "core/container"

func Run(ctx context.Context) error {
    // 创建 hall 服务容器
    hallContainer := container.NewHallContainer()
    defer hallContainer.Close()
    
    // 使用容器中的依赖
    userRepo := hallContainer.GetUserRepository()
    
    // ... 业务逻辑
}
```

### Game 服务

```go
import "core/container"

func Run(ctx context.Context) error {
    // 创建 game 服务容器
    gameContainer := container.NewGameContainer()
    defer gameContainer.Close()
    
    // 使用容器中的依赖
    userRepo := gameContainer.GetUserRepository()
    
    // ... 业务逻辑
}
```

### Gate 服务

```go
import "core/container"

func Run(ctx context.Context) error {
    // 创建 gate 服务容器
    gateContainer := container.NewGateContainer()
    defer gateContainer.Close()
    
    // 使用容器中的依赖
    userRepo := gateContainer.GetUserRepository()
    
    // ... 业务逻辑
}
```

## 向后兼容

旧代码仍然可以使用 `core/infrastructure.New()`，但会收到弃用警告：

```go
// 不推荐（已弃用）
container := infrastructure.New()

// 推荐（新代码应使用）
playerContainer := container.NewPlayerContainer()
```

## 扩展新服务

当需要添加新的微服务时：

1. **创建新的容器文件**（例如 `xxx_container.go`）
2. **继承 BaseContainer**
3. **添加服务特定的仓储**
4. **实现必要的 Getter 方法**

示例：

```go
// core/container/xxx_container.go
package container

type XxxContainer struct {
    *BaseContainer
    // 添加 xxx 服务特定的依赖
}

func NewXxxContainer() *XxxContainer {
    base := NewBase()
    return &XxxContainer{
        BaseContainer: base,
        // 初始化依赖
    }
}
```

## 最佳实践

### 1. 依赖清晰
- 每个容器只暴露该服务需要的接口
- 避免过度暴露不必要的依赖

### 2. 资源管理
- 始终使用 `defer container.Close()` 确保资源被释放
- 在 Close() 中清理所有资源

### 3. 单例模式
- 数据库连接在 BaseContainer 中创建一次
- 所有服务容器共享同一个数据库连接

### 4. 测试友好
- 可以为每个容器创建 Mock 版本
- 便于单元测试

## 注意事项

- ⚠️ 避免容器之间的循环依赖
- ⚠️ 确保 Close() 方法被正确调用
- ⚠️ 不要在容器中存储业务状态，只存储基础设施依赖
