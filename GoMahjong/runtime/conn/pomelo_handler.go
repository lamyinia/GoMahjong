package conn

import (
	"common/log"
	"core/infrastructure/message/protocol"
	"core/infrastructure/message/stream"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func (w *Worker) handshakeHandler(packet *protocol.Packet, conn *Connection) error {
	log.Debug("握手事件发生: %#v", packet.ParseBody())
	res := protocol.HandshakeResponse{
		Code: 200,
		Sys: protocol.Sys{
			Heartbeat: 3,
		},
	}
	data, _ := json.Marshal(res)
	buf, err := protocol.Wrap(packet.Type, data)
	if err != nil {
		log.Error("handshakeHandler 打包错误 err:%v", err)
		return err
	}
	return (*conn).SendMessage(buf)
}

func (w *Worker) handshakeAckHandler(packet *protocol.Packet, c *Connection) error {
	log.Debug("握手确认事件发生: %#v", packet.ParseBody())
	return nil
}

func (w *Worker) heartbeatHandler(packet *protocol.Packet, conn *Connection) error {
	log.Debug("心跳事件发生: %#v", packet.ParseBody())
	var res []byte
	data, _ := json.Marshal(res)
	buf, err := protocol.Wrap(packet.Type, data)
	if err != nil {
		log.Error("heartbeatHandler 打包错误 err:%v", err)
		return err
	}
	return (*conn).SendMessage(buf)
}

// 游戏客户端和 game 通信的枢纽
// see: stream.ServicePacket
func (w *Worker) messageHandler(packet *protocol.Packet, conn *Connection) error {
	parse := packet.ParseBody()
	routes := parse.Route
	routeList := strings.Split(routes, ".")
	if len(routeList) < 2 {
		return errors.New(fmt.Sprintf("route 格式错误, %v", parse))
	}

	// 需要做路由转发
	if routeList[0] != "connector" {
		if routeList[0] != "game" {
			// 暂时只支持转发到 game 节点的路由转发
			return fmt.Errorf("不支持的路由转发: %s", routeList[0])
		}
		return w.dispatchNext(parse, conn)
	}

	handler, exi := w.MessageTypeHandlers[routes]
	if exi {
		data, err := handler((*conn).TakeSession(), parse.Data)
		if err != nil {
			return err
		}
		if data != nil {
			marshal, _ := json.Marshal(data)
			parse.Type = protocol.Response
			parse.Data = marshal
			encode, err := protocol.MessageEncode(parse)
			if err != nil {
				log.Warn("messageHandler 编码错误, %#v", data)
				return err
			}
			res, err := protocol.Wrap(packet.Type, encode)
			if err != nil {
				log.Warn("messageHandler 打包错误, %#v", data)
				return err
			}
			return w.doPush(&res, conn)
		}
	} else {
		log.Warn("messageHandler 发现不支持的路由, %#v", parse)
	}

	return nil
}

func (w *Worker) kickHandler(packet *protocol.Packet, conn *Connection) error {
	log.Debug("踢人事件发生: %#v", packet.ParseBody())

	return nil
}

func (w *Worker) dispatchNext(message *protocol.Message, con *Connection) error {
	userID := (*con).TakeSession().UserID
	next, exi := w.UserRouteCache.Get(userID)
	if !exi {
		return fmt.Errorf("%s 的 key 不存在", userID)
	}

	servicePacket := &stream.ServicePacket{
		Body:        message,
		Source:      w.nodeID,
		Destination: next,
		Route:       message.Route, // 将 pomelo 的 route 提取到 nats 通信层面
		SessionData: nil,
		PushUser:    nil,
	}
	if err := w.MiddleWorker.PushMessage(servicePacket); err != nil {
		return err
	}
	return nil
}
