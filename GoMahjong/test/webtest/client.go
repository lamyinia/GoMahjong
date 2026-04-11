package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/charmbracelet/log"
)

// TCPClient connects to the C++ game server
type TCPClient struct {
	host      string
	port      int
	playerID  string
	conn      net.Conn
	connMu    sync.Mutex
	sendCh    chan []byte
	RecvChan  chan *TCPMessage
	done      chan struct{}
	connected bool
}

// TCPMessage represents a received message
type TCPMessage struct {
	Route   string
	Payload json.RawMessage
}

// NewTCPClient creates a new TCP client
func NewTCPClient(host string, port int, playerID string) *TCPClient {
	return &TCPClient{
		host:     host,
		port:     port,
		playerID: playerID,
		sendCh:   make(chan []byte, 100),
		RecvChan: make(chan *TCPMessage, 100),
		done:     make(chan struct{}),
	}
}

// Connect establishes TCP connection to game server
func (c *TCPClient) Connect() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	c.conn = conn
	c.connected = true

	// Start read/write loops
	go c.readLoop()
	go c.writeLoop()

	log.Debug("TCP connected", "addr", addr, "playerId", c.playerID)
	return nil
}

// Close disconnects from server
func (c *TCPClient) Close() {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if !c.connected {
		return
	}

	c.connected = false
	close(c.done)
	if c.conn != nil {
		c.conn.Close()
	}
}

// IsConnected returns connection status
func (c *TCPClient) IsConnected() bool {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	return c.connected
}

// SendAuth sends authentication request
func (c *TCPClient) SendAuth(token string) error {
	// AuthRequest proto: token, device_id, version
	payload := map[string]string{
		"token":    token,
		"deviceId": "webtest",
		"version":  "1.0.0",
	}
	return c.SendMessage("auth.login", payload)
}

// SendMessage sends a message with envelope wrapper
func (c *TCPClient) SendMessage(route string, payload interface{}) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}

	var payloadBytes []byte
	var err error

	switch p := payload.(type) {
	case []byte:
		payloadBytes = p
	case json.RawMessage:
		payloadBytes = p
	default:
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
	}

	// Create envelope: route + payload
	envelope := map[string]interface{}{
		"route":   route,
		"payload": payloadBytes,
	}

	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	// Add length prefix (4 bytes, big endian)
	buf := make([]byte, 4+len(envelopeBytes))
	binary.BigEndian.PutUint32(buf[:4], uint32(len(envelopeBytes)))
	copy(buf[4:], envelopeBytes)

	// Send to write channel
	select {
	case c.sendCh <- buf:
		return nil
	default:
		return fmt.Errorf("send buffer full")
	}
}

// writeLoop sends messages to server
func (c *TCPClient) writeLoop() {
	for {
		select {
		case <-c.done:
			return
		case data := <-c.sendCh:
			c.connMu.Lock()
			if c.conn != nil {
				if _, err := c.conn.Write(data); err != nil {
					log.Error("TCP write error", "err", err)
					c.connMu.Unlock()
					c.Close()
					return
				}
			}
			c.connMu.Unlock()
		}
	}
}

// readLoop receives messages from server
func (c *TCPClient) readLoop() {
	defer c.Close()

	for {
		select {
		case <-c.done:
			return
		default:
			// Read length prefix (4 bytes)
			lenBuf := make([]byte, 4)
			if _, err := io.ReadFull(c.conn, lenBuf); err != nil {
				if err != io.EOF {
					log.Error("TCP read length error", "err", err)
				}
				return
			}

			length := binary.BigEndian.Uint32(lenBuf)
			if length > 65536 { // Max 64KB
				log.Error("Message too large", "length", length)
				return
			}

			// Read payload
			payloadBuf := make([]byte, length)
			if _, err := io.ReadFull(c.conn, payloadBuf); err != nil {
				log.Error("TCP read payload error", "err", err)
				return
			}

			// Parse envelope
			var envelope struct {
				Route   string          `json:"route"`
				Payload json.RawMessage `json:"payload"`
			}
			if err := json.Unmarshal(payloadBuf, &envelope); err != nil {
				log.Error("Parse envelope error", "err", err)
				continue
			}

			// Forward to recv channel
			msg := &TCPMessage{
				Route:   envelope.Route,
				Payload: envelope.Payload,
			}

			select {
			case c.RecvChan <- msg:
			default:
				log.Warn("Recv buffer full, dropping message", "route", envelope.Route)
			}
		}
	}
}
