package conn

import (
	"common/log"
	"errors"
	"framework/node"
	"framework/protocal"
	"framework/stream"
	"github.com/gorilla/websocket"
	"hash/fnv"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
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

	// connectionRateLimiter todo 限流
)

type CheckOriginHandler func(r *http.Request) bool

type PacketTypeHandler func(packet *protocal.Packet, c Connection) error

type ClientBucket struct {
	sync.RWMutex
	clients map[string]Connection
}

type Manager struct {
	dataLock           sync.RWMutex // 仅用于保护data字段
	websocketUpgrade   *websocket.Upgrader
	topic              string
	CheckOriginHandler CheckOriginHandler
	data               map[string]any

	clientBuckets  []*ClientBucket
	ClientReadChan chan *MessagePack
	clientWorkers  []chan *MessagePack
	clientHandlers map[protocal.PackageType]PacketTypeHandler
	bucketMask     uint32
	workerCount    int

	MiddleReadChan chan []byte
	MiddleWorker   *node.NatsClient
	MiddlePushChan chan *stream.Message

	ConnectorHandlers LogicHandler

	maxConnectionCount int
	connSemaphore      chan struct{}
	stats              struct {
		messageProcessed   int64
		messageErrors      int64
		avgProcessingTime  int64
		currentConnections int32
	}
}

func NewManager() *Manager {
	bucketCount := 32
	bucketMask := uint32(bucketCount - 1)
	workerCount := runtime.NumCPU() * 2

	m := &Manager{
		ClientReadChan:     make(chan *MessagePack, 2048),
		clientHandlers:     make(map[protocal.PackageType]PacketTypeHandler),
		MiddleReadChan:     make(chan []byte, 2048),
		MiddlePushChan:     make(chan *stream.Message, 2048),
		data:               make(map[string]any),
		bucketMask:         bucketMask,
		workerCount:        workerCount,
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
	for i := range m.workerCount {
		go m.clientWorkerRoutine(i)
	}

	go m.clientReadRoutine()
	go m.middleWorkerReadRoutine()
	go m.middleWorkerPushRoutine()
	go m.monitorPerformance()

	http.HandleFunc("/ws", m.upgradeFunc)
	log.Info("websocket manager 启动了 %d 个 worker 协程和 %d 个连接分片桶", m.workerCount, len(m.clientBuckets))
	log.Fatal("connector listen serve err:%v", http.ListenAndServe(addr, nil))
}

func (m *Manager) upgradeFunc(w http.ResponseWriter, r *http.Request) {

}

func (m *Manager) removeClient(con *LongConnection) {

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

}

func (m *Manager) middleWorkerReadRoutine() {

}

func (m *Manager) middleWorkerPushRoutine() {

}

func (m *Manager) monitorPerformance() {

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

func (m *Manager) getBucket(cid string) *ClientBucket {
	hash := fnv32(cid)
	index := hash & m.bucketMask
	return m.clientBuckets[index]
}

func fnv32(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}
