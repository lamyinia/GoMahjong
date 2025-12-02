package conn

import (
	"common/config"
	"common/jwts"
	"common/log"
	"common/utils"
	"encoding/json"
	"errors"
	"fmt"
	"framework/node"
	"framework/protocal"
	"framework/stream"
	"hash/fnv"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var (
	websocketUpgrade = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		EnableCompression: true,
	}

	connectionRateLimiter = utils.NewRateLimiter(100, 1)
)

type CheckOriginHandler func(r *http.Request) bool

type PacketTypeHandler func(packet *protocal.Packet, c Connection) error

type ClientBucket struct {
	sync.RWMutex
	clients map[string]Connection
}

/*
	连接器职责：
	1. 正确处理玩家长连接的生命周期、读写事件
	2. 调用 rpc 服务，实现游戏断线重连机制
	3. 调用 nats 服务和 game 节点通信，nats 监听来自 game 节点的消息，实现游戏逻辑
	4. 使用 rpc 服务调用 game 节点的方法，同时 nats 监听来自 hall 节点的消息，实现大厅逻辑
	5. 使用 rpc 服务调用 march 节点的方法，同时 nats 监听来自 march 节点的消息，实现匹配逻辑
			(1)用户在类似游戏对战开始(匹配、创建房间...)的逻辑之前，一定要有查路由的逻辑，检查有没有正在进行的游戏

	6. 设计正确的处理器和路由，收到来自 game、hall、march 节点或者 player 的消息后，如果需要转发，正确转发给目标
	7. 设计正确的处理器，收到来自 game、hall、march 节点或者 player 的消息后，如果需要操作本地内存，正确操作本地内存

	数据流向 LongConnection.ReadChan=Worker.ClientReadChan -> clientBuckets[connID]
*/

type Worker struct {
	dataLock           sync.RWMutex // 仅用于保护data字段
	websocketUpgrade   *websocket.Upgrader
	nodeID             string
	CheckOriginHandler CheckOriginHandler
	data               map[string]any

	clientBuckets     []*ClientBucket
	ClientReadChan    chan *ConnectionPack
	clientWorkers     []chan *ConnectionPack
	clientHandlers    map[protocal.PackageType]PacketTypeHandler
	bucketMask        uint32
	clientWorkerCount int

	MiddleWorker *node.NatsWorker

	LocalHandlers LogicHandler

	maxConnectionCount int
	connSemaphore      chan struct{}
	stats              struct {
		messageProcessed   int64
		messageErrors      int64
		avgProcessingTime  int64
		currentConnections int32
	}

	connMap        sync.Map
	isRunning      bool
	UserRouteCache *UserRouteCache
}

// NewWorkerWithDeps 接收依赖的构造函数（推荐用于生产环境）
func NewWorkerWithDeps(connectorConfig interface{}, natsWorker *node.NatsWorker, rateLimiter *utils.RateLimiter) *Worker {
	bucketCount := 32
	bucketMask := uint32(bucketCount - 1)
	workerCount := runtime.NumCPU() * 2

	// 类型断言获取配置
	cfg, ok := connectorConfig.(interface{ GetID() string })
	if !ok || cfg == nil || cfg.GetID() == "" {
		log.Fatal("connector 配置类型错误或 ID 为空")
		return nil
	}

	w := &Worker{
		nodeID:             cfg.GetID(),
		ClientReadChan:     make(chan *ConnectionPack, 2048),
		clientHandlers:     make(map[protocal.PackageType]PacketTypeHandler),
		MiddleWorker:       natsWorker,
		data:               make(map[string]any),
		LocalHandlers:      make(LogicHandler),
		bucketMask:         bucketMask,
		clientWorkerCount:  workerCount,
		maxConnectionCount: 100000,
		connSemaphore:      make(chan struct{}, 100000),
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
	userRouteCache, err := NewUserRouteCache()
	if err != nil {
		log.Fatal("创建用户路由缓存失败: %v", err)
		return nil
	}
	w.UserRouteCache = userRouteCache

	return w
}

func NewClientBucket() *ClientBucket {
	return &ClientBucket{
		clients: make(map[string]Connection),
	}
}

// Run 启动 Worker，合并了原 Connector.Run() 和 Manager.Run() 的功能
func (w *Worker) Run(topic string, maxConn int, addr string) error {
	if w.isRunning {
		return nil
	}

	log.Info("connector worker 组件正在配置")
	w.isRunning = true

	// 启动 NATS Worker
	err := w.MiddleWorker.Run(config.InjectedConfig.Nats.URL, topic)
	if err != nil {
		log.Fatal("nats 启动失败")
		return err
	}

	// 启动 WebSocket 服务
	log.Info("websocket worker 正在启动服务")
	for i := range w.clientWorkerCount {
		go w.clientWorkerRoutine(i)
	}

	go w.clientReadRoutine()
	go w.monitorPerformance()
	w.injectDefaultHandlers()

	http.HandleFunc("/ws/", w.upgradeFunc) // 注意匹配子路径
	log.Info("websocket worker 启动了 %d 个 worker 协程和 %d 个连接分片桶", w.clientWorkerCount, len(w.clientBuckets))
	log.Info("http 监听地址 %s", addr)
	return http.ListenAndServe(addr, nil)
}

func (w *Worker) injectDefaultHandlers() {
	w.clientHandlers[protocal.Handshake] = w.handshakeHandler
	w.clientHandlers[protocal.HandshakeAck] = w.handshakeAckHandler
	w.clientHandlers[protocal.Heartbeat] = w.heartbeatHandler
	w.clientHandlers[protocal.Data] = w.messageHandler
	w.clientHandlers[protocal.Kick] = w.kickHandler

	w.LocalHandlers["connector.joinqueue"] = joinQueueHandler
}

func (w *Worker) upgradeFunc(writer http.ResponseWriter, r *http.Request) {
	userID, authMethod, err := w.identifyUser(r)
	if err != nil {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		log.Warn("连接鉴权失败 remote=%s err=%v", r.RemoteAddr, err)
		return
	}
	if !connectionRateLimiter.Allow() {
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
	if w.websocketUpgrade == nil {
		upgrade = &websocketUpgrade
	} else {
		upgrade = w.websocketUpgrade
	}
	header := writer.Header()
	header.Add("Server", "go-mahjong-soul")
	log.Debug("WebSocket connection attempt from %s, User-Agent: %s", r.RemoteAddr, r.UserAgent())

	conn, err := upgrade.Upgrade(writer, r, nil)
	if err != nil {
		log.Error("websocket 升级失败, err:%#v", err)
	}

	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	client := takeLongConnection(conn, w)
	client.TakeSession().SetUserID(userID)
	w.BindUser(userID, client)
	w.addClient(client)
	client.Run()
	log.Debug("WebSocket connection established: userID=%s method=%s cid=%s remote=%s", userID, authMethod, client.ConnID, r.RemoteAddr)
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

func (w *Worker) Close() {
	if w.isRunning {
		if w.MiddleWorker != nil {
			w.MiddleWorker.Close()
		}
		if w.UserRouteCache != nil {
			w.UserRouteCache.Close()
		}
		w.isRunning = false
	}
}

func (w *Worker) clientWorkerRoutine(workerID int) {
	for messagePack := range w.clientWorkers[workerID] {
		startTime := time.Now()

		w.dealPack(messagePack)

		processingTime := time.Since(startTime).Milliseconds()
		atomic.AddInt64(&w.stats.messageProcessed, 1)
		oldAvg := atomic.LoadInt64(&w.stats.avgProcessingTime)
		newAvg := (oldAvg*9 + processingTime) / 10
		atomic.StoreInt64(&w.stats.avgProcessingTime, newAvg)
	}
}

func (w *Worker) clientReadRoutine() {
	for messagePack := range w.ClientReadChan {
		hash := fnv32(messagePack.ConnID)
		workerID := hash % uint32(w.clientWorkerCount)
		select {
		case w.clientWorkers[workerID] <- messagePack:
			// 分发到工作池
		default:
			atomic.AddInt64(&w.stats.messageErrors, 1)
			log.Warn("工作池满了，直接处理:\n workerID:%#v\n messagePack:%#v", workerID, messagePack)
			w.dealPack(messagePack)
		}
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

func (w *Worker) Push(message *stream.ServicePacket) {
	buf, err := protocal.MessageEncode(message.Body)
	if err != nil {
		log.Error("pushMessage 推送编码错误, err:%#v", err)
		return
	}

	res, err := protocal.Wrap(protocal.Data, buf)
	if err != nil {
		log.Error("pushMessage 打包类型错误, err:%#v", err)
		return
	}

	if message.Body.Type == protocal.Push {
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
			if err := w.doPush(&res, &conn); err != nil {
				log.Error("pushMessage 发送失败 userID=%s err=%v", userID, err)
			}
		}
	}
}

func (w *Worker) doPush(bye *[]byte, conn *Connection) error {
	return (*conn).SendMessage(*bye) // 接口不能自动 (*). ?
}

func (w *Worker) dealPack(messagePack *ConnectionPack) {
	packet, err := protocal.Decode(messagePack.Body)
	if err != nil {
		atomic.AddInt64(&w.stats.messageErrors, 1)
		log.Error("解码错误, pack: %#v, err: %#v", packet, err)
		return
	}
	if err := w.doEvent(packet, messagePack.ConnID); err != nil {
		atomic.AddInt64(&w.stats.messageErrors, 1)
		log.Error("事件处理错误, pack: %#v, err: %#v", packet, err)
		return
	}
}

// doEvent 处理协议层的时间
func (w *Worker) doEvent(packet *protocal.Packet, connID string) error {
	bucket := w.getBucket(connID)

	bucket.RLock()
	conn, ok := bucket.clients[connID]
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

	secret := ""
	if config.Conf != nil {
		secret = config.Conf.JwtConf.Secret
	}
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

// ws/test={userID}
func (w *Worker) extractUserIDFromTestPath(path string) (string, bool) {
	if config.Conf == nil || !config.Conf.JwtConf.AllowTestPath {
		return "", false
	}
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "", false
	}

	segments := strings.Split(trimmed, "/")
	for _, segment := range segments {
		if strings.HasPrefix(segment, "test=") {
			userID := strings.TrimPrefix(segment, "test=")
			if userID != "" {
				return userID, true
			}
		}
	}
	return "", false
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
}

func (w *Worker) handshakeHandler(packet *protocal.Packet, conn Connection) error {
	log.Debug("握手事件发生: %#v", packet.ParseBody())
	res := protocal.HandshakeResponse{
		Code: 200,
		Sys: protocal.Sys{
			Heartbeat: 3,
		},
	}
	data, _ := json.Marshal(res)
	buf, err := protocal.Wrap(packet.Type, data)
	if err != nil {
		log.Error("handshakeHandler 打包错误 err:%v", err)
		return err
	}
	return conn.SendMessage(buf)
}

func (w *Worker) handshakeAckHandler(packet *protocal.Packet, c Connection) error {
	log.Debug("握手确认事件发生: %#v", packet.ParseBody())
	return nil
}

func (w *Worker) heartbeatHandler(packet *protocal.Packet, conn Connection) error {
	log.Debug("心跳事件发生: %#v", packet.ParseBody())
	var res []byte
	data, _ := json.Marshal(res)
	buf, err := protocal.Wrap(packet.Type, data)
	if err != nil {
		log.Error("heartbeatHandler 打包错误 err:%v", err)
		return err
	}
	return conn.SendMessage(buf)
}

func (w *Worker) messageHandler(packet *protocal.Packet, conn Connection) error {
	parse := packet.ParseBody()
	routes := parse.Route // 如 hall.marchRequest
	routeList := strings.Split(routes, ".")
	if len(routeList) < 2 {
		return errors.New(fmt.Sprintf("route 格式错误, %v", parse))
	}
	if routeList[0] != "connector" {
		routes = routeList[0] // 转发到下一个链路
	}

	handler, exi := w.LocalHandlers[routes]
	if exi {
		data, err := handler(conn.TakeSession(), parse.Data)
		if err != nil {
			return err
		}
		if data != nil {
			marshal, _ := json.Marshal(data)
			parse.Type = protocal.Response
			parse.Data = marshal
			encode, err := protocal.MessageEncode(parse)
			if err != nil {
				log.Warn("messageHandler 编码错误, %#v", data)
				return err
			}
			res, err := protocal.Wrap(packet.Type, encode)
			if err != nil {
				log.Warn("messageHandler 打包错误, %#v", data)
				return err
			}
			return w.doPush(&res, &conn)
		}
	} else {
		log.Warn("messageHandler 发现不支持的路由, %#v", parse)
	}

	return nil
}

func (w *Worker) kickHandler(packet *protocal.Packet, conn Connection) error {
	log.Debug("踢人事件发生: %#v", packet.ParseBody())

	return nil
}
