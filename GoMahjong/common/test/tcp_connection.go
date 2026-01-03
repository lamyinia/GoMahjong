package test

import (
	"core/infrastructure/message/protocol"
	"errors"
	"io"
	"net"
	"sync"
)

// TCPConnection TCP 连接封装，处理拆包粘包
// 与 WebSocket 不同，TCP 是流式协议，需要自己处理消息边界
type TCPConnection struct {
	conn   net.Conn
	buffer []byte     // 缓冲区，用于处理粘包
	mu     sync.Mutex // 保护 buffer 的并发访问
	closed bool
}

// NewTCPConnection 创建 TCP 连接
func NewTCPConnection(conn net.Conn) *TCPConnection {
	return &TCPConnection{
		conn:   conn,
		buffer: make([]byte, 0, 4096), // 初始容量 4KB
		closed: false,
	}
}

// ReadPacket 读取一个完整的 pomelo 协议包
// 处理拆包粘包的核心逻辑
func (t *TCPConnection) ReadPacket() (*protocol.Packet, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil, errors.New("connection closed")
	}

	// 步骤 1: 确保缓冲区有至少 4 字节（头部：1字节类型 + 3字节长度）
	for len(t.buffer) < protocol.HeaderLen {
		temp := make([]byte, 4096)
		n, err := t.conn.Read(temp)
		if err != nil {
			if err == io.EOF {
				t.closed = true
			}
			return nil, err
		}
		if n == 0 {
			continue
		}
		t.buffer = append(t.buffer, temp[:n]...)
	}

	// 步骤 2: 解析头部，获取消息体长度（关键：使用 Len 字段）
	bodyLen := protocol.BytesToInt(t.buffer[1:protocol.HeaderLen]) // 这里需要 Len 字段！

	// 验证消息长度
	if bodyLen > protocol.MaxPacketSize {
		return nil, errors.New("packet body size too large")
	}

	totalLen := protocol.HeaderLen + bodyLen

	// 步骤 3: 确保缓冲区有完整消息（处理拆包）
	for len(t.buffer) < totalLen {
		temp := make([]byte, 4096)
		n, err := t.conn.Read(temp)
		if err != nil {
			if err == io.EOF {
				t.closed = true
			}
			return nil, err
		}
		if n == 0 {
			continue
		}
		t.buffer = append(t.buffer, temp[:n]...)
	}

	// 步骤 4: 提取完整消息
	packet := make([]byte, totalLen)
	copy(packet, t.buffer[:totalLen])
	t.buffer = t.buffer[totalLen:] // 移除已处理的数据（处理粘包）

	// 步骤 5: 解析 pomelo 协议包
	return protocol.Decode(packet)
}

// WritePacket 写入一个完整的 pomelo 协议包
func (t *TCPConnection) WritePacket(packageType protocol.PackageType, body []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return errors.New("connection closed")
	}

	// 使用 protocol.Wrap 编码（包含 Len 字段）
	packet, err := protocol.Wrap(packageType, body)
	if err != nil {
		return err
	}

	// TCP 需要完整写入，可能分多次写入
	_, err = t.conn.Write(packet)
	return err
}

// Close 关闭连接
func (t *TCPConnection) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}

	t.closed = true
	return t.conn.Close()
}

// IsClosed 检查连接是否已关闭
func (t *TCPConnection) IsClosed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closed
}

// RemoteAddr 获取远程地址
func (t *TCPConnection) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

// LocalAddr 获取本地地址
func (t *TCPConnection) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}
