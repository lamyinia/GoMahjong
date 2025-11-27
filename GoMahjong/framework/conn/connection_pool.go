package conn

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type LongConnectionPool struct {
	pool      sync.Pool
	count     int32 // 当前池中对象数量
	maxSize   int32 // 最大池大小
	created   int64 // 总共创建的对象数
	reused    int64 // 复用的对象数
	discarded int64 // 丢弃的对象数
}

var (
	globalWsConnectionPool *LongConnectionPool
	poolOnce               sync.Once
)

func NewLongConnectionPool(maxSize int32) *LongConnectionPool {
	p := &LongConnectionPool{
		maxSize: maxSize,
	}
	p.pool = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&p.created, 1)
			atomic.AddInt32(&p.count, 1)
			return &LongConnection{}
		},
	}
	return p
}

func GetLongConnectionPool() *LongConnectionPool {
	poolOnce.Do(func() {
		globalWsConnectionPool = NewLongConnectionPool(10000) // 默认最大池大小为10000
	})
	return globalWsConnectionPool
}

func (po *LongConnectionPool) Get(conn *websocket.Conn, manager *Manager) *LongConnection {
	longConn := po.pool.Get().(*LongConnection)
	atomic.AddInt64(&po.reused, 1)

	connID := fmt.Sprintf("%s-%s-%d", uuid.New().String(), manager.topic, atomic.AddUint64(&connIDBase, 1))

	longConn.Conn = conn
	longConn.manager = manager
	longConn.ConnID = connID
	longConn.WriteChan = make(chan []byte, 1024)
	longConn.ReadChan = manager.ClientReadChan
	longConn.Session = NewSession(connID, manager)
	longConn.closeChan = make(chan struct{})

	longConn.closeOnce = sync.Once{}
	longConn.readChanOnce = sync.Once{}
	longConn.writeChanOnce = sync.Once{}

	return longConn
}

func (po *LongConnectionPool) Put(longConn *LongConnection) {
	if longConn == nil {
		return
	}

	if atomic.LoadInt32(&po.count) > po.maxSize {
		atomic.AddInt64(&po.discarded, 1)
		atomic.AddInt32(&po.count, -1)
		return
	}

	longConn.reset()
	po.pool.Put(longConn)
}

func takeLongConnection(conn *websocket.Conn, manager *Manager) *LongConnection {
	return GetLongConnectionPool().Get(conn, manager)
}
