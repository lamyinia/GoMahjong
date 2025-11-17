package conn

import (
	"common/log"
	"framework/node"
	"framework/stream"
	"github.com/gorilla/websocket"
	"hash/fnv"
	"net/http"
	"runtime"
	"sync"
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

type ClientBucket struct {
	sync.RWMutex
	clients map[string]Connection
}

type Manager struct {
	topic string

	clientBuckets  []*ClientBucket
	ClientReadChan chan *MessagePack

	MiddleReadChan chan []byte
	MiddleWorker   *node.NatsWorker

	MiddlePushChan chan *stream.Message

	bucketMask  uint32
	workerCount int

	ConnectorHandlers node.LogicHandler
}

func NewManager() *Manager {
	bucketCount := 32
	bucketMask := uint32(bucketCount - 1)
	workerCount := runtime.NumCPU() * 2

	m := &Manager{
		ClientReadChan: make(chan *MessagePack, 2048),
		MiddleReadChan: make(chan []byte, 2048),
		MiddlePushChan: make(chan *stream.Message, 2048),
		bucketMask:     bucketMask,
		workerCount:    workerCount,
	}

	m.clientBuckets = make([]*ClientBucket, bucketCount)
	for i := range bucketCount {
		m.clientBuckets[i] = NewClientBucket()
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

}

func (m *Manager) clientReadRoutine() {

}

func (m *Manager) middleWorkerReadRoutine() {

}

func (m *Manager) middleWorkerPushRoutine() {

}

func (m *Manager) monitorPerformance() {

}

func fnv32(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}
