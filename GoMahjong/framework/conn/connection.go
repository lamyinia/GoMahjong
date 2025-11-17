package conn

import (
	"common/log"
	"github.com/gorilla/websocket"
	"sync"
	"time"
)

type Connection interface {
	GetSession() *Session
	SendMessage(buf []byte) error
	Close()
}

type MessagePack struct {
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
	manager       *Manager
	ReadChan      chan *MessagePack
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

func (con *LongConnection) writeMessage() {
	con.pingTicker = time.NewTicker(pingInterval)
	defer func() {
		if con.WriteChan != nil {
			con.writeChanOnce.Do(func() {
				close(con.WriteChan)
			})
		}
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
			log.Warn("%v", string(message))
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

func (con *LongConnection) readMessage() {
	defer func() {
		log.Info("客户端[%s] 读时间停止", con.ConnID)
		con.manager.removeClient(con)
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
			log.Info("[%s] 收到二进制消息 %+v", con.ConnID, string(message))
			if messageType == websocket.BinaryMessage {
				select {
				case con.ReadChan <- &MessagePack{ConnID: con.ConnID, Body: message}:
				case <-con.closeChan:
					log.Info("客户端[%s] 异常 while sending to channel", con.ConnID)
					return
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

func (con *LongConnection) GetSession() *Session {
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
	con.manager = nil
	con.ReadChan = nil
	con.WriteChan = nil
	con.Session = nil
	con.pingTicker = nil
	con.closeChan = nil
}

func NewLongConnection(conn *websocket.Conn, manager *Manager) *LongConnection {
	return GetLongConnectionPool().Get(conn, manager)
}
