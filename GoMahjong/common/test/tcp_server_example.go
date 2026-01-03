package test

import (
	"common/log"
	"core/infrastructure/message/protocol"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// TCPServerExample TCP 服务器示例
// 演示如何使用 TCPConnection 处理拆包粘包
func TCPServerExample(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("监听失败: %v", err)
	}
	defer listener.Close()

	log.Info("TCP 服务器启动，监听地址: %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error("接受连接失败: %v", err)
			continue
		}

		// 每个连接启动一个 goroutine 处理
		go handleTCPConnection(conn)
	}
}

// handleTCPConnection 处理单个 TCP 连接
func handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	log.Info("新连接: %s -> %s", conn.RemoteAddr(), conn.LocalAddr())

	// 创建 TCP 连接封装（自动处理拆包粘包）
	tcpConn := NewTCPConnection(conn)

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	for {
		// 读取完整消息（自动处理拆包粘包）
		packet, err := tcpConn.ReadPacket()
		if err != nil {
			if err.Error() == "connection closed" {
				log.Info("连接已关闭: %s", conn.RemoteAddr())
			} else {
				log.Error("读取消息失败: %v", err)
			}
			break
		}

		// 重置读取超时
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		// 处理不同类型的消息
		switch packet.Type {
		case protocol.Handshake:
			handleHandshake(tcpConn, packet)
		case protocol.Heartbeat:
			handleHeartbeat(tcpConn)
		case protocol.Data:
			handleData(tcpConn, packet)
		default:
			log.Warn("未知消息类型: %d", packet.Type)
		}
	}
}

// handleHandshake 处理握手消息
func handleHandshake(conn *TCPConnection, packet *protocol.Packet) {
	log.Info("收到握手消息")

	// 解析握手数据
	if body, ok := packet.Body.(protocol.HandshakeBody); ok {
		log.Info("客户端信息: Type=%s, Version=%s, Heartbeat=%d",
			body.Sys.Type, body.Sys.Version, body.Sys.Heartbeat)
	}

	// 发送握手响应
	response := protocol.HandshakeResponse{
		Code: 200,
		Sys: protocol.Sys{
			Type:         "go-server",
			Version:      "1.0",
			ProtoVersion: 1,
			Heartbeat:    3,
			Serializer:   "json",
		},
	}

	data, _ := json.Marshal(response)
	if err := conn.WritePacket(protocol.Handshake, data); err != nil {
		log.Error("发送握手响应失败: %v", err)
	}
}

// handleHeartbeat 处理心跳消息
func handleHeartbeat(conn *TCPConnection) {
	log.Debug("收到心跳消息")
	// 发送心跳响应
	if err := conn.WritePacket(protocol.Heartbeat, []byte{}); err != nil {
		log.Error("发送心跳响应失败: %v", err)
	}
}

// handleData 处理数据消息
func handleData(conn *TCPConnection, packet *protocol.Packet) {
	message := packet.ParseBody()
	if message == nil {
		log.Warn("解析消息体失败")
		return
	}

	log.Info("收到数据消息: Route=%s, Type=%d", message.Route, message.Type)

	// 示例：回显消息
	if message.Type == protocol.Request {
		response := &protocol.Message{
			Type:  protocol.Response,
			ID:    message.ID,
			Route: message.Route,
			Data:  message.Data, // 回显原数据
		}

		encoded, _ := protocol.MessageEncode(response)
		if err := conn.WritePacket(protocol.Data, encoded); err != nil {
			log.Error("发送响应失败: %v", err)
		}
	}
}
