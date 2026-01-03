package test

import (
	"common/log"
	"core/infrastructure/message/protocol"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// TCPClientExample TCP 客户端示例
// 演示如何使用 TCPConnection 发送和接收消息
func TCPClientExample(serverAddr string) error {
	// 连接到服务器
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	defer conn.Close()

	log.Info("已连接到服务器: %s", serverAddr)

	// 创建 TCP 连接封装
	tcpConn := NewTCPConnection(conn)

	// 1. 发送握手消息
	if err := sendHandshake(tcpConn); err != nil {
		return fmt.Errorf("握手失败: %v", err)
	}

	// 2. 读取握手响应
	packet, err := tcpConn.ReadPacket()
	if err != nil {
		return fmt.Errorf("读取握手响应失败: %v", err)
	}

	if packet.Type == protocol.Handshake {
		log.Info("握手成功")
		if body, ok := packet.Body.(protocol.HandshakeBody); ok {
			log.Info("服务器信息: Type=%s, Version=%s",
				body.Sys.Type, body.Sys.Version)
		}
	}

	// 3. 启动心跳
	go sendHeartbeat(tcpConn)

	// 4. 发送测试消息
	for i := 0; i < 5; i++ {
		if err := sendTestMessage(tcpConn, i); err != nil {
			log.Error("发送测试消息失败: %v", err)
			continue
		}

		// 读取响应
		packet, err := tcpConn.ReadPacket()
		if err != nil {
			log.Error("读取响应失败: %v", err)
			break
		}

		if packet.Type == protocol.Data {
			message := packet.ParseBody()
			if message != nil {
				log.Info("收到响应: Route=%s", message.Route)
			}
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}

// sendHandshake 发送握手消息
func sendHandshake(conn *TCPConnection) error {
	body := protocol.HandshakeBody{
		Sys: protocol.Sys{
			Type:         "go-client",
			Version:      "1.0",
			ProtoVersion: 1,
			Heartbeat:    3,
			Serializer:   "json",
		},
	}

	data, _ := json.Marshal(body)
	return conn.WritePacket(protocol.Handshake, data)
}

// sendHeartbeat 定期发送心跳
func sendHeartbeat(conn *TCPConnection) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if conn.IsClosed() {
				return
			}
			if err := conn.WritePacket(protocol.Heartbeat, []byte{}); err != nil {
				log.Error("发送心跳失败: %v", err)
				return
			}
		}
	}
}

// sendTestMessage 发送测试消息
func sendTestMessage(conn *TCPConnection, seq int) error {
	message := &protocol.Message{
		Type:  protocol.Request,
		ID:    uint(seq),
		Route: "test.echo",
		Data:  []byte(fmt.Sprintf(`{"message": "test %d"}`, seq)),
	}

	encoded, err := protocol.MessageEncode(message)
	if err != nil {
		return err
	}

	return conn.WritePacket(protocol.Data, encoded)
}
