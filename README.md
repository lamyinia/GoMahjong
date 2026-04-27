# **[日麻开发文档.md](./rule/README.md)**
- 游戏逻辑参考手册


# 开发环境搭建

## 系统要求

- **OS**: Linux (Ubuntu 22.04+ / 其他发行版) / macOS / WSL2
- **Git**: 2.x+

## 语言与构建工具

### Go

- **版本**: 1.24.0+（项目使用 `go.work` 多模块工作区，需 Go 1.24+）
- **安装**: https://go.dev/dl/

```bash
go version  # 确认 >= 1.24
```

项目使用 Go Workspace 管理多模块，顶层 `go.work` 包含以下模块：

```
common/    connector/    core/    game/    gate/
hall/      march/        runtime/ user/    test/webtest
```

### C++

- **C++ 标准**: C++20
- **CMake**: 3.20+
- **编译器**: GCC 12+ / Clang 15+ (需支持 C++20)

```bash
cmake --version   # 确认 >= 3.20
g++ --version     # 确认 >= 12
```

### Protobuf

- **protoc**: 3.x+（Go 和 C++ 共用）
- **protoc-gen-go**: Go Protobuf 插件
- **protoc-gen-go-grpc**: Go gRPC 插件（部分服务需要）

```bash
protoc --version                    # 确认可用
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## C++ 第三方依赖

`game-cpp` 通过 CMake `FetchContent` 自动拉取的依赖（无需手动安装）：

| 依赖 | 版本 | 用途 |
|---|---|---|
| spdlog | v1.14.1 | 日志 |
| nlohmann_json | v3.11.3 | JSON 解析 |

需系统安装的依赖：

| 依赖 | 用途 | Ubuntu 安装 |
|---|---|---|
| Boost | 网络/asio 等 | `sudo apt install libboost-all-dev` |
| protobuf | Protobuf 运行时 | `sudo apt install libprotobuf-dev protobuf-compiler` |
| gRPC + grpc_cpp_plugin | gRPC 框架 | 从源码编译或使用 vcpkg/conan |
| etcd-cpp-apiv3 | etcd C++ 客户端 | 从源码编译 |
| mongocxx | MongoDB C++ 驱动 | `sudo apt install libmongocxx-dev` |

> **注意**: gRPC C++ 和 etcd-cpp-apiv3 的安装较为复杂，建议参考各自官方文档。后续会提供一键安装脚本。

## 基础设施服务

开发环境需要以下服务运行：

| 服务 | 用途 | 默认端口 | 安装方式 |
|---|---|---|---|
| **etcd** | 服务注册/发现 | 2379 | `sudo apt install etcd` 或 Docker |
| **Redis** | 缓存/会话 | 6379 | `sudo apt install redis-server` |
| **MongoDB** | 数据持久化 | 27017 | `sudo apt install mongosh mongodb-server` |
| **NATS** | 消息中间件 | 4222 | 从 https://nats.io 下载或 Docker |

### 快速启动基础设施（Docker 方式）

```bash
docker run -d --name etcd -p 2379:2379 quay.io/coreos/etcd:v3.5 \
  etcd -advertise-client-urls=http://0.0.0.0:2379 \
       -listen-client-urls=http://0.0.0.0:2379

docker run -d --name redis -p 6379:6379 redis:7-alpine

docker run -d --name mongo -p 27017:27017 \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=admin123 \
  mongo:7

docker run -d --name nats -p 4222:4222 nats:2-alpine
```

### 本地安装（Ubuntu 24.04 示例）

```bash
# etcd
sudo apt install etcd-server etcd-client
sudo systemctl start etcd

# Redis
sudo apt install redis-server
sudo systemctl start redis-server

# MongoDB
sudo apt install mongodb-server mongosh
sudo systemctl start mongodb

# NATS（需手动下载）
curl -L https://github.com/nats-io/nats-server/releases/download/v2.10.24/nats-server-v2.10.24-linux-amd64.tar.gz | tar xz
sudo mv nats-server-v2.10.24-linux-amd64/nats-server /usr/local/bin/
nats-server &

# 验证
etcdctl endpoint health
redis-cli ping          # 应返回 PONG
mongosh --eval "db.runCommand({ping:1})"
```

## Node.js（可选）

`test/webtest` 的前端 UI 需要 Node.js 构建：

- **Node.js**: 18+
- **npm**: 9+

```bash
node --version   # 确认 >= 18
cd test/webtest/webui && npm install && npm run build
```

## 构建项目

### Go 服务

```bash
# 在项目根目录
go work sync       # 同步工作区依赖

# 构建单个服务
go build -o bin/gate ./gate
go build -o bin/connector ./connector
go build -o bin/hall ./hall
go build -o bin/march ./march
go build -o bin/user ./user
go build -o bin/game ./game

# 或构建全部
for svc in gate connector hall march user game; do
  go build -o bin/$svc ./$svc
done
```

### C++ 游戏服务器

```bash
cd game-cpp
mkdir build && cd build
cmake .. -DCMAKE_BUILD_TYPE=Release
make -j$(nproc)

# 产出可执行文件: gomahjong_server
```

### 测试工具

```bash
cd test/webtest
go build -o bin/webtest .
# 前端构建
cd webui && npm install && npm run build
```

## 环境验证清单

启动所有服务前，确认以下条件满足：

- [ ] Go >= 1.24 已安装
- [ ] CMake >= 3.20 + C++20 编译器已安装
- [ ] protoc 已安装且版本兼容
- [ ] etcd 运行在 2379 端口
- [ ] Redis 运行在 6379 端口
- [ ] MongoDB 运行在 27017 端口
- [ ] NATS 运行在 4222 端口
- [ ] C++ 第三方依赖已安装（Boost、gRPC、etcd-cpp-api、mongocxx）
- [ ] `game-cpp` 构建成功产出 `gomahjong_server`
- [ ] Go 各服务构建成功
