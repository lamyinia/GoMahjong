package conn

import (
	"common/log"
	"encoding/json"
	"errors"
	"fmt"
	"framework/protocol"
	"strings"
)

func (w *Worker) handshakeHandler(packet *protocol.Packet, conn Connection) error {
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
	return conn.SendMessage(buf)
}

func (w *Worker) handshakeAckHandler(packet *protocol.Packet, c Connection) error {
	log.Debug("握手确认事件发生: %#v", packet.ParseBody())
	return nil
}

func (w *Worker) heartbeatHandler(packet *protocol.Packet, conn Connection) error {
	log.Debug("心跳事件发生: %#v", packet.ParseBody())
	var res []byte
	data, _ := json.Marshal(res)
	buf, err := protocol.Wrap(packet.Type, data)
	if err != nil {
		log.Error("heartbeatHandler 打包错误 err:%v", err)
		return err
	}
	return conn.SendMessage(buf)
}

func (w *Worker) messageHandler(packet *protocol.Packet, conn Connection) error {
	parse := packet.ParseBody()
	routes := parse.Route // 如 hall.marchRequest
	routeList := strings.Split(routes, ".")
	if len(routeList) < 2 {
		return errors.New(fmt.Sprintf("route 格式错误, %v", parse))
	}
	if routeList[0] != "connector" {
		routes = routeList[0] // 转发到下一个链路
	}

	handler, exi := w.MessageTypeHandlers[routes]
	if exi {
		data, err := handler(conn.TakeSession(), parse.Data)
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
			return w.doPush(&res, &conn)
		}
	} else {
		log.Warn("messageHandler 发现不支持的路由, %#v", parse)
	}

	return nil
}

func (w *Worker) kickHandler(packet *protocol.Packet, conn Connection) error {
	log.Debug("踢人事件发生: %#v", packet.ParseBody())

	return nil
}
