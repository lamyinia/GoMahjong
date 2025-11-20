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
	数据流向 LongConnection.ReadChan=Manager.ClientReadChan -> clientBuckets[connID]
*/

type Manager struct {
	dataLock           sync.RWMutex // 仅用于保护data字段
	websocketUpgrade   *websocket.Upgrader
	topic              string
	CheckOriginHandler CheckOriginHandler
	data               map[string]any

	clientBuckets     []*ClientBucket
	ClientReadChan    chan *MessagePack
	clientWorkers     []chan *MessagePack
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

	connMap sync.Map
}

func NewManager() *Manager {
	bucketCount := 32
	bucketMask := uint32(bucketCount - 1)
	workerCount := runtime.NumCPU() * 2

	m := &Manager{
		ClientReadChan:     make(chan *MessagePack, 2048),
		clientHandlers:     make(map[protocal.PackageType]PacketTypeHandler),
		MiddleWorker:       node.NewNatsWorker(),
		data:               make(map[string]any),
		bucketMask:         bucketMask,
		clientWorkerCount:  workerCount,
		maxConnectionCount: 100000,
		connSemaphore:      make(chan struct{}, 100000),
	}

	// 初始化客户端分片
	m.clientBuckets = make([]*ClientBucket, bucketCount)
	for i := range bucketCount {
		m.clientBuckets[i] = NewClientBucket()
	}

	// 初始化客户端工作池
	m.clientWorkers = make([]chan *MessagePack, workerCount)
	for i := range workerCount {
		m.clientWorkers[i] = make(chan *MessagePack, 256)
	}

	m.CheckOriginHandler = func(r *http.Request) bool {
		return true
	}

	return m
}

func NewClientBucket() *ClientBucket {
	return &ClientBucket{
		clients: make(map[string]Connection),
	}
}

func (m *Manager) Run(addr string) {
	log.Info("websocket manager 正在启动服务")
	for i := range m.clientWorkerCount {
		go m.clientWorkerRoutine(i)
	}

	go m.clientReadRoutine()
	go m.monitorPerformance()
	m.injectDefaultHandlers()

	http.HandleFunc("/ws", m.upgradeFunc)
	log.Info("websocket manager 启动了 %d 个 worker 协程和 %d 个连接分片桶", m.clientWorkerCount, len(m.clientBuckets))
	log.Fatal("connector listen serve err:%v", http.ListenAndServe(addr, nil))
}

func (m *Manager) injectDefaultHandlers() {
	m.clientHandlers[protocal.Handshake] = m.handshakeHandler
	m.clientHandlers[protocal.HandshakeAck] = m.handshakeAckHandler
	m.clientHandlers[protocal.Heartbeat] = m.heartbeatHandler
	m.clientHandlers[protocal.Data] = m.messageHandler
	m.clientHandlers[protocal.Kick] = m.kickHandler
}

func (m *Manager) upgradeFunc(w http.ResponseWriter, r *http.Request) {
	userID, authMethod, err := m.identifyUser(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		log.Warn("连接鉴权失败 remote=%s err=%v", r.RemoteAddr, err)
		return
	}
	if !connectionRateLimiter.Allow() {
		http.Error(w, "Too many connections", http.StatusTooManyRequests)
		log.Warn("连接速率限流 exceeded from %s", r.RemoteAddr)
		return
	}
	if atomic.LoadInt32(&m.stats.currentConnections) >= int32(m.maxConnectionCount) {
		http.Error(w, "Server is at capacity", http.StatusServiceUnavailable)
		log.Warn("连接达到阈值 %s", r.RemoteAddr)
		return
	}

	var upgrader *websocket.Upgrader
	if m.websocketUpgrade == nil {
		upgrader = &websocketUpgrade
	} else {
		upgrader = m.websocketUpgrade
	}
	header := w.Header()
	header.Add("Server", "go-mahjong-soul")
	log.Debug("WebSocket connection attempt from %s, User-Agent: %s", r.RemoteAddr, r.UserAgent())

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("websocket 升级失败, err:%#v", err)
	}

	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	client := takeLongConnection(conn, m)
	client.GetSession().SetUserID(userID)
	m.BindUser(userID, client)
	m.addClient(client)
	client.Run()
	log.Debug("WebSocket connection established: userID=%s method=%s cid=%s remote=%s", userID, authMethod, client.ConnID, r.RemoteAddr)
}

func (m *Manager) addClient(client *LongConnection) {
	bucket := m.getBucket(client.ConnID)

	select {
	case m.connSemaphore <- struct{}{}:
		bucket.RUnlock()
		bucket.clients[client.ConnID] = client
		bucket.Unlock()

		m.dataLock.RLock()
		client.GetSession().SetAll(m.data)
		m.dataLock.RUnlock()

		atomic.AddInt32(&m.stats.currentConnections, 1)
	default:
		log.Warn("addClient: 连接数达到上限")
		client.Close()
	}
}

func (m *Manager) removeClient(con *LongConnection) {
	bucket := m.getBucket(con.ConnID)
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

	if session := con.GetSession(); session != nil {
		m.UnbindUser(session.GetUserID(), con)
	}

	con.Close()

	if m.connSemaphore != nil {
		select {
		case <-m.connSemaphore:
		default:
		}
	}

	atomic.AddInt32(&m.stats.currentConnections, -1)
}

func (m *Manager) Close() {

}

func (m *Manager) clientWorkerRoutine(workerID int) {
	for message := range m.clientWorkers[workerID] {
		startTime := time.Now()

		m.dealPack(message)

		processingTime := time.Since(startTime).Milliseconds()
		atomic.AddInt64(&m.stats.messageProcessed, 1)
		oldAvg := atomic.LoadInt64(&m.stats.avgProcessingTime)
		newAvg := (oldAvg*9 + processingTime) / 10
		atomic.StoreInt64(&m.stats.avgProcessingTime, newAvg)
	}
}

func (m *Manager) clientReadRoutine() {
	for messagePack := range m.ClientReadChan {
		hash := fnv32(messagePack.ConnID)
		workerID := hash % uint32(m.clientWorkerCount)
		select {
		case m.clientWorkers[workerID] <- messagePack:
			// 分发到工作池
		default:
			atomic.AddInt64(&m.stats.messageErrors, 1)
			log.Warn("工作池满了，直接处理:\n workerID:%#v\n messagePack:%#v", workerID, messagePack)
			m.dealPack(messagePack)
		}
	}
}

func (m *Manager) monitorPerformance() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		log.Info("性能监控: connections=%d, messages_processed=%d, avg_processing_time=%dμs, errors=%d",
			atomic.LoadInt32(&m.stats.currentConnections),
			atomic.LoadInt64(&m.stats.messageProcessed),
			atomic.LoadInt64(&m.stats.avgProcessingTime),
			atomic.LoadInt64(&m.stats.messageErrors))
	}
}

func (m *Manager) Push(message *stream.ServicePacket) {
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
			connAny, ok := m.connMap.Load(userID)
			if !ok {
				log.Warn("pushMessage 找不到在线用户: %s", userID)
				continue
			}
			conn, ok := connAny.(Connection)
			if !ok {
				log.Warn("pushMessage 索引类型断言失败: %s", userID)
				continue
			}
			if err := m.doPush(&res, &conn); err != nil {
				log.Error("pushMessage 发送失败 userID=%s err=%v", userID, err)
			}
		}
	}
}

func (m *Manager) doPush(bye *[]byte, conn *Connection) error {
	return (*conn).SendMessage(*bye) // 接口不能自动 (*). ?
}

func (m *Manager) dealPack(messagePack *MessagePack) {
	packet, err := protocal.Decode(messagePack.Body)
	if err != nil {
		atomic.AddInt64(&m.stats.messageErrors, 1)
		log.Error("解码错误, pack: %#v, err: %#v", packet, err)
		return
	}
	if err := m.doEvent(packet, messagePack.ConnID); err != nil {
		atomic.AddInt64(&m.stats.messageErrors, 1)
		log.Error("事件处理错误, pack: %#v, err: %#v", packet, err)
		return
	}
}

func (m *Manager) doEvent(packet *protocal.Packet, connID string) error {
	bucket := m.getBucket(connID)

	bucket.RLock()
	conn, ok := bucket.clients[connID]
	bucket.RUnlock()

	if !ok {
		return errors.New("找不到客户端桶")
	}

	handler, ok := m.clientHandlers[packet.Type]
	if !ok {
		return errors.New("找不到处理器")
	}

	return handler(packet, conn)
}

func (m *Manager) getBucket(connID string) *ClientBucket {
	hash := fnv32(connID)
	index := hash & m.bucketMask
	return m.clientBuckets[index]
}

func fnv32(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

// identifyUser 验证请求 URL 的 path
func (m *Manager) identifyUser(r *http.Request) (string, string, error) {
	if userID, ok := m.extractUserIDFromTestPath(r.URL.Path); ok {
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
func (m *Manager) extractUserIDFromTestPath(path string) (string, bool) {
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

func (m *Manager) BindUser(userID string, conn Connection) {
	if userID == "" || conn == nil {
		return
	}

	if oldConn, ok := m.connMap.Load(userID); ok {
		if existing, ok := oldConn.(Connection); ok && existing != conn {
			log.Info("用户 %s 已有连接，踢出旧连接", userID)
			existing.Close()
		}
	}

	m.connMap.Store(userID, conn)
}

func (m *Manager) UnbindUser(userID string, conn Connection) {
	if userID == "" {
		return
	}

	if stored, ok := m.connMap.Load(userID); ok {
		if conn == nil || stored == conn {
			m.connMap.Delete(userID)
		}
	}
}

func (m *Manager) handshakeHandler(packet *protocal.Packet, conn Connection) error {
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

func (m *Manager) handshakeAckHandler(packet *protocal.Packet, c Connection) error {
	return nil
}

func (m *Manager) heartbeatHandler(packet *protocal.Packet, conn Connection) error {
	var res []byte
	data, _ := json.Marshal(res)
	buf, err := protocal.Wrap(packet.Type, data)
	if err != nil {
		log.Error("heartbeatHandler 打包错误 err:%v", err)
		return err
	}
	return conn.SendMessage(buf)
}

func (m *Manager) messageHandler(packet *protocal.Packet, conn Connection) error {
	parse := packet.ParseBody()
	routes := parse.Route // 如 hall.marchRequest
	if routeList := strings.Split(routes, "."); len(routeList) != 2 {
		return errors.New(fmt.Sprintf("route 格式错误, %v", parse))
	}

	handler, exi := m.LocalHandlers[routes]
	if exi {
		data, err := handler(conn.GetSession(), parse.Data)
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
			return m.doPush(&res, &conn)
		}
	} else {
		log.Warn("messageHandler 发现不支持的路由, %#v", parse)
	}

	return nil
}

func (m *Manager) kickHandler(packet *protocal.Packet, conn Connection) error {
	return nil
}
