package conn

import (
	"common/config"
	"common/jwts"
	"common/log"
	"common/utils"
	"context"
	"core/domain/repository"
	"core/infrastructure/cache"
	"core/infrastructure/message/node"
	"core/infrastructure/message/protocol"
	"core/infrastructure/message/transfer"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

/*
长连接网关职责：
 1. 连接事件：处理玩家长连接的生命周期、读写事件
 2. 游戏断线重连：配合游戏断线重连服务的实现
 3. 游戏逻辑通信：调用 nats 服务和 game 节点通信，nats 监听来自 game 节点的消息
 4. 大厅逻辑：nats 监听来自 hall 节点的消息
 5. 匹配逻辑：使用 rpc 服务调用 march 节点的方法，同时 nats 监听来自 march 节点的消息，配合匹配服务
    (1)用户在类似游戏对战开始(匹配、创建房间...)的逻辑之前，一定要有查路由的逻辑，检查有没有正在进行的游戏
 6. 路由转发，收到来自 game、hall、march 节点或者 user 的消息后，如果需要转发，转发给目标
 7. 通信协议协议：收到来自 game、hall、march 节点或者 user 的消息后，识别协议

可扩展点：
	支持 websocket、TCP、KCP、UDP 等
*/

type CheckOriginHandler func(r *http.Request) bool

type PacketTypeHandler func(packet *protocol.Packet, c Connection) error

type ClientBucket struct {
	sync.RWMutex
	clients map[string]Connection
}

type WorkerOption func(worker *Worker) error

type Worker struct {
	nodeID             string
	dataLock           sync.RWMutex // 仅用于保护data字段
	websocketUpgrade   *websocket.Upgrader
	upgradeOnce        sync.Once
	CheckOriginHandler CheckOriginHandler
	data               map[string]any

	clientBuckets         []*ClientBucket
	clientWorkers         []chan *ConnectionPack
	clientHandlers        map[protocol.PackageType]PacketTypeHandler
	bucketMask            uint32
	clientWorkerCount     int
	ConnectionRateLimiter *utils.RateLimiter
	MiddleWorker          *node.NatsWorker
	MessageTypeHandlers   MessageTypeHandler // see: pomelo_handler.go

	maxConnectionCount int
	connSemaphore      chan struct{} // 连接信号量
	stats              struct {
		messageProcessed   int64
		messageErrors      int64
		avgProcessingTime  int64
		currentConnections int32
	}

	connMap   sync.Map
	isRunning bool

	GameRouteCache *cache.GameRouteCache
	UserRouter     repository.UserRouterRepository
}

// NewWorkerWithDeps 接收依赖的构造函数（推荐用于生产环境）
func NewWorkerWithDeps(opts ...WorkerOption) *Worker {
	bucketCount := 32
	bucketMask := uint32(bucketCount - 1)
	workerCount := runtime.NumCPU() * 2

	w := &Worker{
		nodeID:              config.ConnectorConfig.ID,
		clientHandlers:      make(map[protocol.PackageType]PacketTypeHandler),
		data:                make(map[string]any),
		MessageTypeHandlers: make(MessageTypeHandler),
		bucketMask:          bucketMask,
		clientWorkerCount:   workerCount,
		maxConnectionCount:  100000,
		connSemaphore:       make(chan struct{}, 100000),
	}
	for _, opt := range opts {
		if err := opt(w); err != nil {
			log.Fatal("worker 启动错误: %v", err)
		}
	}

	// 初始化客户端分片
	w.clientBuckets = make([]*ClientBucket, bucketCount)
	for i := range bucketCount {
		w.clientBuckets[i] = NewClientBucket()
	}

	// 初始化客户端工作池
	w.clientWorkers = make([]chan *ConnectionPack, workerCount)
	for i := range workerCount {
		w.clientWorkers[i] = make(chan *ConnectionPack, 256)
	}

	w.CheckOriginHandler = func(r *http.Request) bool {
		return true
	}

	// 初始化用户路由缓存
	userRouteCache, err := cache.NewGameRouteCache()
	if err != nil {
		log.Fatal("创建用户路由缓存失败: %v", err)
		return nil
	}
	w.GameRouteCache = userRouteCache

	return w
}

func NewClientBucket() *ClientBucket {
	return &ClientBucket{
		clients: make(map[string]Connection),
	}
}

func (w *Worker) Run(topic string, addr string) error {
	if w.isRunning {
		return nil
	}

	log.Info("connector worker 组件正在配置")
	w.isRunning = true

	// 启动 NATS Worker
	err := w.MiddleWorker.Run(config.ConnectorConfig.NatsConfig.URL, topic)
	if err != nil {
		log.Fatal("nats 启动失败")
		return err
	}

	// 启动 WebSocket 服务
	log.Info("websocket worker 正在启动服务")
	for i := range w.clientWorkerCount {
		go w.clientWorkerRoutine(i)
	}

	go w.monitorPerformance()
	w.injectDefaultHandlers()
	w.injectMiddleWorkerHandler()

	http.HandleFunc("/ws/", w.upgradeFunc) // 注意匹配子路径
	log.Info("websocket worker 启动了 %d 个 worker 协程和 %d 个连接分片桶", w.clientWorkerCount, len(w.clientBuckets))
	log.Info("http 监听地址 %s", addr)
	return http.ListenAndServe(addr, nil)
}

func (w *Worker) upgradeFunc(writer http.ResponseWriter, r *http.Request) {
	userID, authMethod, err := w.identifyUser(r)
	if err != nil {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		log.Warn("连接鉴权失败 remote=%s err=%v", r.RemoteAddr, err)
		return
	}
	if !w.ConnectionRateLimiter.Allow() {
		http.Error(writer, "Too many connections", http.StatusTooManyRequests)
		log.Warn("连接速率限流 exceeded from %s", r.RemoteAddr)
		return
	}
	if atomic.LoadInt32(&w.stats.currentConnections) >= int32(w.maxConnectionCount) {
		http.Error(writer, "Server is at capacity", http.StatusServiceUnavailable)
		log.Warn("连接达到阈值 %s", r.RemoteAddr)
		return
	}

	var upgrade *websocket.Upgrader
	// 后续可以动态配置升级策略
	w.upgradeOnce.Do(w.InitUpgrade)
	upgrade = w.websocketUpgrade

	header := writer.Header()
	header.Add("Server", "go-mahjong-soul")
	log.Debug("WebSocket connection attempt from %s, User-Agent: %s", r.RemoteAddr, r.UserAgent())

	conn, err := upgrade.Upgrade(writer, r, nil)
	if err != nil {
		log.Warn("websocket 升级失败, err:%#v", err)
		return
	}

	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	client := takeLongConnection(conn, w)
	client.TakeSession().SetUserID(userID)
	w.BindUser(userID, client)
	w.addClient(client)
	client.Run()
	log.Info("WebSocket 建立连接: userID=%s, method=%s, connID=%s, remote=%s", userID, authMethod, client.ConnID, r.RemoteAddr)
}

func (w *Worker) addClient(client *LongConnection) {
	bucket := w.getBucket(client.ConnID)

	select {
	case w.connSemaphore <- struct{}{}:
		bucket.Lock()
		bucket.clients[client.ConnID] = client
		bucket.Unlock()

		w.dataLock.RLock()
		client.TakeSession().SetAll(w.data)
		w.dataLock.RUnlock()

		atomic.AddInt32(&w.stats.currentConnections, 1)
	default:
		log.Warn("addClient: 连接数达到上限")
		client.Close()
	}
}

func (w *Worker) removeClient(con *LongConnection) {
	bucket := w.getBucket(con.ConnID)
	removed := false

	bucket.Lock()
	if _, ok := bucket.clients[con.ConnID]; ok {
		delete(bucket.clients, con.ConnID)
		removed = true
	}
	bucket.Unlock()

	if !removed {
		return
	}

	if session := con.TakeSession(); session != nil {
		w.UnbindUser(session.GetUserID(), con)
	}

	con.Close()

	if w.connSemaphore != nil {
		select {
		case <-w.connSemaphore:
		default:
		}
	}

	atomic.AddInt32(&w.stats.currentConnections, -1)
}

func (w *Worker) clientWorkerRoutine(workerID int) {
	for messagePack := range w.clientWorkers[workerID] {
		startTime := time.Now()

		w.DecodeAndHandlePack(messagePack)

		processingTime := time.Since(startTime).Milliseconds()
		atomic.AddInt64(&w.stats.messageProcessed, 1)
		oldAvg := atomic.LoadInt64(&w.stats.avgProcessingTime)
		newAvg := (oldAvg*9 + processingTime) / 10
		atomic.StoreInt64(&w.stats.avgProcessingTime, newAvg)
	}
}

func (w *Worker) monitorPerformance() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		log.Debug("性能监控: connections=%d, messages_processed=%d, avg_processing_time=%dμs, errors=%d",
			atomic.LoadInt32(&w.stats.currentConnections),
			atomic.LoadInt64(&w.stats.messageProcessed),
			atomic.LoadInt64(&w.stats.avgProcessingTime),
			atomic.LoadInt64(&w.stats.messageErrors))
	}
}

func (w *Worker) Push(message *transfer.ServicePacket) {
	buf, err := protocol.MessageEncode(message.Body)
	if err != nil {
		log.Error("pushMessage 推送编码错误, err:%#v", err)
		return
	}

	res, err := protocol.Wrap(protocol.Data, buf)
	if err != nil {
		log.Error("pushMessage 打包类型错误, err:%#v", err)
		return
	}

	if message.Body.Type == protocol.Push {
		if len(message.PushUser) == 0 {
			return
		}

		for _, userID := range message.PushUser {
			if userID == "" {
				continue
			}
			connAny, ok := w.connMap.Load(userID)
			if !ok {
				log.Warn("pushMessage 找不到在线用户: %s", userID)
				continue
			}
			conn, ok := connAny.(Connection)
			if !ok {
				log.Warn("pushMessage 索引类型断言失败: %s", userID)
				continue
			}
			if err := w.doPush(res, conn); err != nil {
				log.Error("pushMessage 发送失败 userID=%s err=%v", userID, err)
			}
		}
	}
}

func (w *Worker) DecodeAndHandlePack(messagePack *ConnectionPack) {
	packet, err := protocol.Decode(messagePack.Body)
	if err != nil {
		atomic.AddInt64(&w.stats.messageErrors, 1)
		log.Warn("解码错误, pack: %#v, err: %#v", packet, err)
		return
	}
	if err := w.handleProtocolEvent(packet, messagePack.ConnID); err != nil {
		atomic.AddInt64(&w.stats.messageErrors, 1)
		log.Warn("事件处理错误, pack: %#v, err: %#v", packet, err)
		return
	}
}

// handleProtocolEvent 处理协议层的事件
func (w *Worker) handleProtocolEvent(packet *protocol.Packet, connID string) error {
	bucket := w.getBucket(connID)
	bucket.RLock()
	conn, ok := bucket.clients[connID] // 不会拷贝新的连接对象，只是拷贝了一份接口值（很小的头部），底层还是同一个连接
	bucket.RUnlock()
	if !ok {
		return errors.New("找不到客户端桶")
	}
	handler, ok := w.clientHandlers[packet.Type]
	if !ok {
		return errors.New("找不到处理器")
	}
	return handler(packet, conn)
}

func (w *Worker) getBucket(connID string) *ClientBucket {
	hash := fnv32(connID)
	index := hash & w.bucketMask
	return w.clientBuckets[index]
}

func fnv32(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

// identifyUser 验证请求 URL 的 path
func (w *Worker) identifyUser(r *http.Request) (string, string, error) {
	if userID, ok := w.extractUserIDFromTestPath(r.URL.Path); ok {
		// 测试白名单
		return userID, "test-path", nil
	}

	token := r.URL.Query().Get("barrier")
	if token == "" {
		return "", "", errors.New("缺少 barrier token")
	}

	secret := config.ConnectorConfig.JwtConf.Secret
	if secret == "" {
		return "", "", errors.New("未配置 jwt secret")
	}

	userID, err := jwts.ParseToken(token, secret)
	if err != nil {
		return "", "", err
	}
	if userID == "" {
		return "", "", errors.New("token 中 userID 为空")
	}
	return userID, "token", nil
}

func (w *Worker) BindUser(userID string, conn Connection) {
	if userID == "" || conn == nil {
		return
	}

	if oldConn, ok := w.connMap.Load(userID); ok {
		if existing, ok := oldConn.(Connection); ok && existing != conn {
			log.Info("用户 %s 已有连接，踢出旧连接", userID)
			existing.Close()
		}
	}

	w.connMap.Store(userID, conn)
	go func() {
		// 更新路由错误不用处理
		_ = w.UserRouter.SaveConnectorRouter(context.Background(), userID, w.nodeID, 2*time.Hour)
	}()
}

func (w *Worker) UnbindUser(userID string, conn Connection) {
	if userID == "" {
		return
	}

	if stored, ok := w.connMap.Load(userID); ok {
		if conn == nil || stored == conn {
			w.connMap.Delete(userID)
		}
	}
	go func() {
		// 更新路由错误不用处理
		_ = w.UserRouter.DeleteConnectorRouter(context.Background(), userID)
	}()
}

func (w *Worker) send(messageType protocol.MessageType, userID string, route string, body any) error {
	connAny, ok := w.connMap.Load(userID)
	if !ok {
		return fmt.Errorf("玩家 %s 连接不存在", userID)
	}
	conn, ok := connAny.(Connection)
	if !ok {
		return fmt.Errorf("玩家 %s 连接类型断言失败", userID)
	}

	// 1. 构建 pomelo Message
	var data []byte
	if body != nil {
		if bodyBytes, ok := body.([]byte); ok {
			data = bodyBytes
		} else {
			// 如果是其他类型，序列化为 JSON
			jsonData, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("%s 序列化消息失败: %w", userID, err)
			}
			data = jsonData
		}
	}

	msg := &protocol.Message{
		Type:  messageType,
		ID:    0,
		Route: route,
		Data:  data,
		Error: false,
	}

	// 2. 编码 Message
	msgEncoded, err := protocol.MessageEncode(msg)
	if err != nil {
		return fmt.Errorf("%s 编码消息失败: %w", userID, err)
	}

	// 3. 包装成 pomelo Packet
	packet, err := protocol.Wrap(protocol.Data, msgEncoded)
	if err != nil {
		return fmt.Errorf("%s 打包消息失败: %w", userID, err)
	}

	// 4. 发送给客户端
	if err := conn.SendMessage(packet); err != nil {
		return fmt.Errorf("发送消息给玩家 %s 失败: %w", userID, err)
	}

	log.Info(fmt.Sprintf("connector send 发送消息给玩家 %s, route: %s", userID, route))
	return nil
}

func (w *Worker) doPush(bye []byte, conn Connection) error {
	return conn.SendMessage(bye)
}

func (w *Worker) InitUpgrade() {
	w.websocketUpgrade = &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		EnableCompression: true,
	}
}

func (w *Worker) Close() {
	if w.isRunning {
		if w.MiddleWorker != nil {
			w.MiddleWorker.Close()
		}
		if w.GameRouteCache != nil {
			w.GameRouteCache.Close()
		}
		w.isRunning = false
	}
}
