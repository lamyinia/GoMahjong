package conn

import (
	"common/log"
	"github.com/gorilla/websocket"
	"sync"
	"sync/atomic"
	"time"
)

/*
	处理单个 gorilla/websocket 的连接生命周期、读写事件、心跳
*/

type Connection interface {
	TakeSession() *Session
	SendMessage(buf []byte) error
	Close()
}

type ConnectionPack struct {
	ConnID string
	Body   []byte
}

var connIDBase uint64 = 10000

var (
	pongWait             = 10 * time.Second
	writeWait            = 10 * time.Second
	pingInterval         = (pongWait * 9) / 10
	maxMessageSize int64 = 1024
)

type LongConnection struct {
	ConnID        string
	Conn          *websocket.Conn
	worker        *Worker
	WriteChan     chan []byte
	Session       *Session
	pingTicker    *time.Ticker
	closeChan     chan struct{}
	closeOnce     sync.Once
	readChanOnce  sync.Once
	writeChanOnce sync.Once
}

func (con *LongConnection) Run() {
	go con.readMessage()
	go con.writeMessage()
	con.Conn.SetPongHandler(con.PongHandler)
}

// websocket.Conn.WriteMessage
func (con *LongConnection) writeMessage() {
	con.pingTicker = time.NewTicker(pingInterval)
	defer func() {
		if con.WriteChan != nil {
			con.writeChanOnce.Do(func() {
				close(con.WriteChan)
			})
		}
		con.worker.removeClient(con)
	}()

	for {
		select {
		case message, ok := <-con.WriteChan:
			if !ok {
				if err := con.Conn.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Error("客户端[%s] 连接关闭, %+v", con.ConnID, err)
				}
				con.Close()
				return
			}
			log.Debug("写入消息: %#v", message)
			if err := con.Conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				log.Error("客户端[%s] write stream err :%+v", con.ConnID, err)
			}
		case <-con.pingTicker.C:
			if err := con.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Error("客户端[%s] ping SetWriteDeadline err :%+v", con.ConnID, err)
			}
			if err := con.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Error("客户端[%s] ping  err :%+v", con.ConnID, err)
				con.Close()
			}
		case <-con.closeChan:
			log.Info("客户端[%s] writeMessage stopped", con.ConnID)
			return
		}
	}
}

// 读取客户端消息，打包成 ConnectionPack
func (con *LongConnection) readMessage() {
	defer func() {
		log.Info("客户端[%s] 读事件停止", con.ConnID)
		con.worker.removeClient(con)
	}()

	con.Conn.SetReadLimit(maxMessageSize)
	if err := con.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Error("SetReadDeadline err:%v", err)
		return
	}
	for {
		select {
		case <-con.closeChan:
			log.Info("客户端[%s] 检测到关闭信号", con.ConnID)
			return
		default:
			messageType, message, err := con.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Error("客户端[%s] 异常错误: %v", con.ConnID, err)
				}
				return
			}
			log.Debug("[%s] 收到二进制消息, 大小 %d 字节, 详细: %+v", con.ConnID, len(message), message)
			if messageType == websocket.BinaryMessage {
				pack := &ConnectionPack{ConnID: con.ConnID, Body: message}
				hash := fnv32(con.ConnID)
				workerID := hash % uint32(con.worker.clientWorkerCount)

				select {
				case <-con.closeChan:
					log.Info("客户端[%s] 异常 while sending to channel", con.ConnID)
					return
				case con.worker.clientWorkers[workerID] <- pack:
				default:
					atomic.AddInt64(&con.worker.stats.messageErrors, 1)
					log.Warn("工作池满了，直接处理:\n workerID:%#v\n messagePack:%#v", workerID, pack)
					con.worker.dealPack(pack)
				}
			} else {
				log.Error("不支持的流类型 : %d", messageType)
			}
		}
	}
}

func (con *LongConnection) PongHandler(data string) error {
	if err := con.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return err
	}
	return nil
}

func (con *LongConnection) TakeSession() *Session {
	return con.Session
}

func (con *LongConnection) SendMessage(buf []byte) error {
	con.WriteChan <- buf
	return nil
}

func (con *LongConnection) Close() {
	//确保只执行一次
	con.closeOnce.Do(func() {
		close(con.closeChan)
		if con.Conn != nil {
			_ = con.Conn.Close()
		}
		if con.pingTicker != nil {
			con.pingTicker.Stop()
		}
		if con.Session != nil {
			con.Session.Close()
		}
		log.Info("客户端[%s] 连接关闭", con.ConnID)
		go func(conn *LongConnection) {
			time.Sleep(100 * time.Millisecond)
			GetLongConnectionPool().Put(conn)
		}(con)
	})
}

func (con *LongConnection) reset() {
	con.ConnID = ""
	con.Conn = nil
	con.worker = nil
	con.WriteChan = nil
	con.Session = nil
	con.pingTicker = nil
	con.closeChan = nil
}
