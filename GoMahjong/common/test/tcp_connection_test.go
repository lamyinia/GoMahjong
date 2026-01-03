package test

import (
	"core/infrastructure/message/protocol"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

// TestTCPConnection_ReadPacket 测试 TCP 连接的拆包粘包处理
func TestTCPConnection_ReadPacket(t *testing.T) {
	// 启动测试服务器
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("启动服务器失败: %v", err)
	}
	defer listener.Close()

	serverAddr := listener.Addr().String()

	// 服务器 goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := listener.Accept()
		if err != nil {
			t.Errorf("接受连接失败: %v", err)
			return
		}
		defer conn.Close()

		tcpConn := NewTCPConnection(conn)

		// 读取多个消息（测试拆包粘包）
		for i := 0; i < 3; i++ {
			packet, err := tcpConn.ReadPacket()
			if err != nil {
				t.Errorf("读取消息失败: %v", err)
				return
			}

			if packet.Type != protocol.Data {
				t.Errorf("期望消息类型 Data，得到 %d", packet.Type)
			}
		}
	}()

	// 客户端连接
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	tcpConn := NewTCPConnection(conn)

	// 测试场景 1: 正常发送多个消息（可能粘包）
	for i := 0; i < 3; i++ {
		message := &protocol.Message{
			Type:  protocol.Request,
			ID:    uint(i),
			Route: "test.route",
			Data:  []byte(fmt.Sprintf(`{"seq": %d}`, i)),
		}

		encoded, _ := protocol.MessageEncode(message)
		if err := tcpConn.WritePacket(protocol.Data, encoded); err != nil {
			t.Fatalf("发送消息失败: %v", err)
		}
	}

	// 等待服务器处理
	time.Sleep(100 * time.Millisecond)
	wg.Wait()
}

// TestTCPConnection_StickyPacket 测试粘包场景
// 模拟 TCP 将多个消息合并在一起发送
func TestTCPConnection_StickyPacket(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("启动服务器失败: %v", err)
	}
	defer listener.Close()

	serverAddr := listener.Addr().String()

	var receivedCount int
	var mu sync.Mutex

	// 服务器
	go func() {
		conn, _ := listener.Accept()
		defer conn.Close()

		tcpConn := NewTCPConnection(conn)

		for {
			_, err := tcpConn.ReadPacket()
			if err != nil {
				break
			}

			mu.Lock()
			receivedCount++
			mu.Unlock()

			if receivedCount >= 3 {
				break
			}
		}
	}()

	// 客户端：手动构造粘包（多个消息合并发送）
	conn, _ := net.Dial("tcp", serverAddr)
	defer conn.Close()

	// 构造 3 个消息并合并发送（模拟粘包）
	var allData []byte
	for i := 0; i < 3; i++ {
		message := &protocol.Message{
			Type:  protocol.Request,
			ID:    uint(i),
			Route: "test.route",
			Data:  []byte(fmt.Sprintf(`{"seq": %d}`, i)),
		}

		encoded, _ := protocol.MessageEncode(message)
		packet, _ := protocol.Wrap(protocol.Data, encoded)
		allData = append(allData, packet...)
	}

	// 一次性发送所有数据（粘包）
	conn.Write(allData)

	// 等待处理
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	count := receivedCount
	mu.Unlock()

	if count != 3 {
		t.Errorf("期望收到 3 个消息，实际收到 %d 个", count)
	}
}

// TestTCPConnection_SplitPacket 测试拆包场景
// 模拟 TCP 将一个消息拆分成多次发送
func TestTCPConnection_SplitPacket(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("启动服务器失败: %v", err)
	}
	defer listener.Close()

	serverAddr := listener.Addr().String()

	var receivedPacket *protocol.Packet
	var mu sync.Mutex

	// 服务器
	go func() {
		conn, _ := listener.Accept()
		defer conn.Close()

		tcpConn := NewTCPConnection(conn)

		packet, err := tcpConn.ReadPacket()
		if err != nil {
			t.Errorf("读取消息失败: %v", err)
			return
		}

		mu.Lock()
		receivedPacket = packet
		mu.Unlock()
	}()

	// 客户端：分多次发送一个消息（模拟拆包）
	conn, _ := net.Dial("tcp", serverAddr)
	defer conn.Close()

	message := &protocol.Message{
		Type:  protocol.Request,
		ID:    1,
		Route: "test.route",
		Data:  []byte(`{"message": "split packet test"}`),
	}

	encoded, _ := protocol.MessageEncode(message)
	packet, _ := protocol.Wrap(protocol.Data, encoded)

	// 分 3 次发送（模拟拆包）
	part1 := packet[:len(packet)/3]
	part2 := packet[len(packet)/3 : len(packet)*2/3]
	part3 := packet[len(packet)*2/3:]

	conn.Write(part1)
	time.Sleep(10 * time.Millisecond)
	conn.Write(part2)
	time.Sleep(10 * time.Millisecond)
	conn.Write(part3)

	// 等待处理
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	pkt := receivedPacket
	mu.Unlock()

	if pkt == nil {
		t.Fatal("未收到消息")
	}

	if pkt.Type != protocol.Data {
		t.Errorf("期望消息类型 Data，得到 %d", pkt.Type)
	}

	msg := pkt.ParseBody()
	if msg == nil {
		t.Fatal("解析消息体失败")
	}

	if msg.Route != "test.route" {
		t.Errorf("期望路由 test.route，得到 %s", msg.Route)
	}
}

// TestTCPConnection_Handshake 测试握手流程
func TestTCPConnection_Handshake(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("启动服务器失败: %v", err)
	}
	defer listener.Close()

	serverAddr := listener.Addr().String()

	// 服务器
	go func() {
		conn, _ := listener.Accept()
		defer conn.Close()

		tcpConn := NewTCPConnection(conn)

		// 读取握手
		packet, err := tcpConn.ReadPacket()
		if err != nil {
			t.Errorf("读取握手失败: %v", err)
			return
		}

		if packet.Type != protocol.Handshake {
			t.Errorf("期望 Handshake，得到 %d", packet.Type)
			return
		}

		// 发送握手响应
		response := protocol.HandshakeResponse{
			Code: 200,
			Sys: protocol.Sys{
				Type:         "go-server",
				Version:      "1.0",
				ProtoVersion: 1,
				Heartbeat:    3,
			},
		}

		data, _ := json.Marshal(response)
		tcpConn.WritePacket(protocol.Handshake, data)
	}()

	// 客户端
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	tcpConn := NewTCPConnection(conn)

	// 发送握手
	handshake := protocol.HandshakeBody{
		Sys: protocol.Sys{
			Type:         "go-client",
			Version:      "1.0",
			ProtoVersion: 1,
			Heartbeat:    3,
		},
	}

	data, _ := json.Marshal(handshake)
	if err := tcpConn.WritePacket(protocol.Handshake, data); err != nil {
		t.Fatalf("发送握手失败: %v", err)
	}

	// 读取握手响应
	packet, err := tcpConn.ReadPacket()
	if err != nil {
		t.Fatalf("读取握手响应失败: %v", err)
	}

	if packet.Type != protocol.Handshake {
		t.Errorf("期望 Handshake 响应，得到 %d", packet.Type)
	}

	if body, ok := packet.Body.(protocol.HandshakeBody); ok {
		if body.Sys.Type != "go-server" {
			t.Errorf("期望服务器类型 go-server，得到 %s", body.Sys.Type)
		}
	}
}
