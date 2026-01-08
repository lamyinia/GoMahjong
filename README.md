# 项目目录结构说明

## 目录树

```text
rule/                                    # 游戏逻辑文档
GoMahjong/                               # 项目代码
  common/                                # 通用基础能力，跨服务复用
    cache/                               # 缓存
    config/                              # 配置
      profile/                           # 配置文件
    database/                            # 数据库
    discovery/                           # 服务发现
    jwts/                                # JWT 封装
    log/                                 # 日志
    metrics/                             # 监控
    rpc/                                 # RPC 调用
    scripts/                             # 测试相关
    utils/                               # 工具类
  connector/                             # 长连接网关服务
    app/                                 # 应用启动
  core/                                  # 核心域与容器
    container/                           # 容器初始化和依赖注入
    domain/                              # 领域模型
      entity/                            # 实体
      repository/                        # 仓储
      vo/                                # 值对象
    infrastructure/                      # 基础设施
      cache/                             # 缓存实现
      message/                           # 消息
      persistence/                       # 持久化
      realtime/                          # 实时数据
  game/                                  # 游戏服务
    api/                                 # API 定义
    app/                                 # 应用启动
    interfaces/                          # 接口
    pb/                                  # 协议缓冲
  gate/                                  # HTTP 网关服务
    api/                                 # API 路由
    app/                                 # 应用启动
  hall/                                  # 大厅服务
    api/                                 # API 定义
    app/                                 # 应用启动
    pb/                                  # 协议缓冲
  march/                                 # 匹配服务
    api/                                 # API 定义
    app/                                 # 应用启动
    interfaces/                          # 接口
    pb/                                  # 协议缓冲
  runtime/                               # 运行时核心逻辑
    conn/                                # 长连接网关核心逻辑
    dto/                                 # 跨服务对象
    game/                                # 游戏核心逻辑
      application/                       # 应用层
        service/                         # 服务
          game_service.go                # 游戏服务接口
          impl/                          # 实现
            game_service_impl.go         # 游戏服务实现
      engines/                           # 游戏引擎
        engine.go                        # 引擎接口
        mahjong/                         # 麻将游戏引擎实现
          checker.go                     # 牌型检查器
          material.go                    # 牌材料定义
          opt_selector.go                # 操作选择器
          persist.go                     # 持久化
          player_image.go                # 玩家镜像
          riichi_mahjong_4p_engine.go    # 立直麻将4人引擎
          router.go                      # 路由
          searcher.go                    # 搜索器
          turn_manager.go                # 回合管理器
          yaku.go                        # 役（和牌类型）
      share/                             # 共享模块
      engine_handler.go                  # 引擎事件处理器
      load_info.go                       # 负载信息计算
      monitor.go                         # 监控器，收集负载信息
      room_manager.go                    # 房间管理器
      worker.go                          # 工作器，处理消息和房间管理
    march/                               # 匹配核心逻辑
    	march_pool.go				   # 匹配池
  user/                                  # 用户服务
    api/                                 # API 定义
    app/                                 # 应用启动
    application/                         # 应用层
    interfaces/                          # 接口
    pb/                                  # 协议缓冲
```



