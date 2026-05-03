# GoMahjong

基于微服务架构的在线日麻游戏平台。

- 🎮 **混合语言架构**：游戏逻辑核心 C++ 实现，业务服务 Go 构建
- 🔗 **多元网络协议**：支持 TCP / UDP / KCP / QUIC 直连游戏服务器
- 🏗️ **云原生设计**：etcd 服务发现 + NATS 消息总线 + Docker Compose 一键部署

> 📖 游戏规则与牌理逻辑参考：[日麻开发文档](./rule/README.md)

## 架构概览

```
                              ┌──────────┐
                         gRPC │   auth   │
                    ┌────────►└──────────┘
                    │         ┌──────────┐
              gRPC  │    NATS │   gate   │
                    │    ┌───►└──────────┘
                    │    │    ┌──────────┐
                    │    │    │  march   │
                    │    │    └────┬─────┘
                    │    │         │ gRPC
┌─────────┐  WS  ┌──┴────┴──┐    │
│  Client  ├────►│ connector │    │
└────┬─────┘     └───────────┘    │
     │                             │
     └── TCP/UDP/KCP/QUIC ────────┤
                                   ▼
                          ┌──────────────────┐
                          │   game-cpp (C++)  │
                          │  游戏逻辑核心引擎  │
                          └────────┬─────────┘
                                   │
                          etcd / MongoDB / Redis
```

| 服务 | 语言 | 用途 |
|---|---|---|
| `auth` | Go | 认证 / JWT 签发 |
| `connector` | Go | WebSocket 连接器，维护客户端长连接 |
| `gate` | Go | HTTP API 网关 |
| `march` | Go | 匹配服务（排队 / 分房） |
| `game-cpp` | C++ | 游戏逻辑核心（牌局状态机、AI、胡牌判定） |
| `test/webtest` | Go | 集成测试工具 |

## 快速开始

### 环境要求

| 工具 | 版本 | 说明 |
|---|---|---|
| Go | ≥ 1.24 | 项目使用 `go.work` 多模块工作区 |
| GCC / Clang | ≥ 12 / ≥ 15 | 需支持 C++20 |
| CMake | ≥ 3.20 | C++ 构建系统 |
| Conan | 2.x | C++ 依赖管理 |
| Docker + Compose | 最新 | 基础设施服务 |
| protoc | ≥ 3.x | Go 和 C++ 共用 |

### 1. 启动基础设施

```bash
cd GoMahjong
docker compose up -d
docker compose ps   # 确认 etcd/Redis/MongoDB/NATS 均运行
```

| 服务 | 端口 | 用途 |
|---|---|---|
| etcd | 2379 | 服务注册与发现 |
| Redis | 6379 | 缓存 / 会话 |
| MongoDB | 27017 | 数据持久化 |
| NATS | 4222 | 消息中间件 |

### 2. 构建 Go 服务

```bash
cd GoMahjong
go work sync

# 构建全部
for svc in auth connector gate march; do
  go build -o $svc/bin/$svc ./$svc
done
```

### 3. 构建 C++ 游戏服务器

```bash
cd GoMahjong/game-cpp

# Conan 拉取依赖（首次）
conan install . --build=missing -s build_type=Release

# CMake 配置 & 构建
cmake --preset conan-release
cmake --build build/Release -j$(nproc)

# 产出: build/Release/gomahjong_server
```

C++ 依赖通过 Conan 管理（`conanfile.txt`），主要包括：

| 依赖 | 用途 |
|---|---|
| gRPC + protobuf | RPC 通信 |
| Boost | Asio 网络 / 容器 |
| spdlog | 日志 |
| nlohmann_json | JSON 解析 |
| mongocxx | MongoDB 驱动 |

> 项目使用纯 gRPC 客户端直连 etcd v3 API（proto 定义见 `proto/etcd/`），无需 `etcd-cpp-apiv3`。

### 4. 运行

```bash
# Go 服务（各服务需要 --configFile 指定配置）
./auth/bin/auth --configFile auth/config/dev/auth.yml
./connector/bin/connector --configFile connector/config/dev/connector.yml
./march/bin/march --configFile march/config/dev/march.yml
./gate/bin/gate --configFile gate/config/dev/gate.yml

# C++ 游戏服务器
cd game-cpp && ./build/Release/gomahjong_server
```

## 开发指南

### Protobuf 代码生成

项目包含 Go 和 C++ 共用的 proto 定义：

```bash
# Go 插件
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# C++ 由 CMake 自动生成（见 CMakeLists.txt 中 etcd / game_service codegen）
```

### 国内网络加速

```bash
# Go 模块代理
go env -w GOPROXY=https://goproxy.cn,direct

# Git SSH 重定向（GitHub）
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

### 本地安装基础设施（Ubuntu 24.04）

如果不使用 Docker，可本地安装：

```bash
# etcd
sudo apt install etcd-server etcd-client && sudo systemctl start etcd

# Redis
sudo apt install redis-server && sudo systemctl start redis-server

# MongoDB
sudo apt install mongodb-server mongosh && sudo systemctl start mongodb

# NATS
curl -L https://github.com/nats-io/nats-server/releases/latest/download/nats-server-linux-amd64.tar.gz | tar xz
sudo mv nats-server /usr/local/bin/ && nats-server &

# 验证
etcdctl endpoint health && redis-cli ping && mongosh --eval "db.runCommand({ping:1})"
```

### Node.js（可选）

`test/webtest` 的前端 UI 需要 Node.js 18+：

```bash
cd GoMahjong/test/webtest/webui && npm install && npm run build
```

## 环境验证清单

- [ ] Go ≥ 1.24 已安装
- [ ] CMake ≥ 3.20 + C++20 编译器已安装
- [ ] Conan 2.x 已安装
- [ ] protoc 已安装且版本兼容
- [ ] etcd 运行在 2379 端口
- [ ] Redis 运行在 6379 端口
- [ ] MongoDB 运行在 27017 端口
- [ ] NATS 运行在 4222 端口
- [ ] `game-cpp` 构建成功产出 `gomahjong_server`
- [ ] Go 各服务构建成功
