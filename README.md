# 项目目录结构说明

## 目录树

```text
rule/ # 游戏逻辑手册
GoMahjong/ # 项目代码
  common/ # 通用基础能力，跨服务复用
    cache/ # 缓存
    config/ # 配置
    database/ # 数据库
    discovery/ # 服务发现
    jwts/ # jwt 封装
    log/ # 日志
    metrics/ # 监控
    rpc/ # rpc调用
    scripts/ # 测试相关
    utils/ # 工具类
  connector/ # 长连接网关暴露+启动入口
  core/ # 核心域与容器
    container/ # 容器初始化和注入
    domain/ # 领域模型
    infrastructure/ # 基础设施
  framework/ # 核心逻辑与框架封装
    conn/ # 长连接网关核心逻辑
    dto/ # 跨服务对象
    game/ # 游戏核心逻辑
    march/ # 匹配核心逻辑
    node/ # nats 节点封装
    protocol/ # 二进制协议
    stream/ # nats 推送封装
  game/ # 游戏服务暴露+启动入口
  gate/ # http 网关启动入口
  hall/ # 大厅服务暴露+启动入口
  march/ # 匹配服暴露+启动入口
  player/ # 用户服务暴露+启动入口
```




