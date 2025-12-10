package main

import (
	"encoding/json"
	"fmt"
	"framework/protocol"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
)

const connectorWS = "ws://127.0.0.1:8082/ws"

type ReceivedMessage struct {
	Route     string
	Payload   map[string]any
	Timestamp time.Time
}

type TestClient struct {
	userID       string
	conn         *websocket.Conn
	done         chan struct{}
	messageQueue chan *ReceivedMessage
}

func NewTestClient(userID string) *TestClient {
	return &TestClient{
		userID:       userID,
		done:         make(chan struct{}),
		messageQueue: make(chan *ReceivedMessage, 100),
	}
}

func (tc *TestClient) Connect() error {
	url := connectorWS + "/test=" + tc.userID
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	tc.conn = conn

	if err := tc.sendHandshake(); err != nil {
		return err
	}
	if err := tc.sendHandshakeAck(); err != nil {
		return err
	}

	go tc.listenLoop()
	go tc.heartbeatLoop()
	log.Printf("[%s] connected", tc.userID)
	return nil
}

func (tc *TestClient) Close() {
	select {
	case <-tc.done:
	default:
		close(tc.done)
	}
	if tc.conn != nil {
		_ = tc.conn.Close()
	}
}

func (tc *TestClient) listenLoop() {
	for {
		if tc.conn == nil {
			return
		}
		mt, msg, err := tc.conn.ReadMessage()
		if err != nil {
			log.Printf("[%s] read error: %v", tc.userID, err)
			return
		}
		if mt == websocket.BinaryMessage {
			packet, err := protocol.Decode(msg)
			if err != nil {
				log.Printf("[%s] decode error: %v", tc.userID, err)
				continue
			}
			if packet.Type == protocol.Data {
				body := packet.ParseBody()

				// 解析 payload
				var payload map[string]any
				if err := json.Unmarshal(body.Data, &payload); err != nil {
					payload = map[string]any{"raw": string(body.Data)}
				}

				msg := &ReceivedMessage{
					Route:     body.Route,
					Payload:   payload,
					Timestamp: time.Now(),
				}

				// 放入消息队列
				select {
				case tc.messageQueue <- msg:
				case <-tc.done:
					return
				}
			} else {
				log.Debug("[%s] recv packet type=%d", tc.userID, packet.Type)
			}
		}
	}
}

func (tc *TestClient) heartbeatLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-tc.done:
			return
		case <-ticker.C:
			buf, err := protocol.Wrap(protocol.Heartbeat, []byte{})
			if err != nil {
				log.Printf("[%s] heartbeat wrap err: %v", tc.userID, err)
				continue
			}
			if tc.conn == nil {
				return
			}
			if err := tc.conn.WriteMessage(websocket.BinaryMessage, buf); err != nil {
				log.Printf("[%s] heartbeat send err: %v", tc.userID, err)
				return
			}
		}
	}
}

func (tc *TestClient) sendHandshake() error {
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
	buf, err := protocol.Wrap(protocol.Handshake, data)
	if err != nil {
		return err
	}
	return tc.conn.WriteMessage(websocket.BinaryMessage, buf)
}

func (tc *TestClient) sendHandshakeAck() error {
	buf, err := protocol.Wrap(protocol.HandshakeAck, []byte{})
	if err != nil {
		return err
	}
	return tc.conn.WriteMessage(websocket.BinaryMessage, buf)
}

func (tc *TestClient) SendRequest(route string, body any) error {
	if tc.conn == nil {
		return ErrNotConnected
	}
	payload, _ := json.Marshal(body)
	msg := &protocol.Message{
		Type:  protocol.Request,
		ID:    1,
		Route: route,
		Data:  payload,
	}
	return tc.sendMessage(msg)
}

func (tc *TestClient) sendMessage(msg *protocol.Message) error {
	body, err := protocol.MessageEncode(msg)
	if err != nil {
		return err
	}
	buf, err := protocol.Wrap(protocol.Data, body)
	if err != nil {
		return err
	}
	return tc.conn.WriteMessage(websocket.BinaryMessage, buf)
}

var ErrNotConnected = fmt.Errorf("client not connected")

// WaitForMessage 等待接收服务器消息
func (tc *TestClient) WaitForMessage(timeout time.Duration) *ReceivedMessage {
	select {
	case msg := <-tc.messageQueue:
		return msg
	case <-time.After(timeout):
		return nil
	case <-tc.done:
		return nil
	}
}

// HandleCommand 处理用户输入的命令
func (tc *TestClient) HandleCommand(cmd string) error {
	parts := strings.Fields(strings.TrimSpace(cmd))
	if len(parts) == 0 {
		return nil
	}

	action := parts[0]

	switch action {
	case "join":
		// 加入匹配队列
		return tc.SendRequest("connector.joinqueue", map[string]any{})

	case "play":
		// 出牌：play 5p
		if len(parts) < 2 {
			fmt.Println("usage: play <tile>")
			return nil
		}
		return tc.SendRequest("game.gameHandler.playTile", map[string]any{
			"tile": parts[1],
		})

	case "peng":
		// 碰
		return tc.SendRequest("game.gameHandler.peng", map[string]any{})

	case "gang":
		// 杠
		return tc.SendRequest("game.gameHandler.gang", map[string]any{})

	case "hu":
		// 胡
		return tc.SendRequest("game.gameHandler.hu", map[string]any{})

	case "skip":
		// 跳过
		return tc.SendRequest("game.gameHandler.skip", map[string]any{})

	case "status":
		// 显示当前状态
		tc.printStatus()
		return nil

	case "help":
		// 显示帮助
		tc.printHelp()
		return nil

	case "quit":
		// 退出
		close(tc.done)
		return nil

	default:
		fmt.Printf("unknown command: %s\n", action)
		return nil
	}
}

// printHelp：显示帮助信息
func (tc *TestClient) printHelp() {
	fmt.Println("\n=== Commands ===")
	fmt.Println("join              - Join matching queue")
	fmt.Println("play <tile>       - Play a tile (e.g., play 5p)")
	fmt.Println("peng              - Peng (碰)")
	fmt.Println("gang              - Gang (杠)")
	fmt.Println("hu                - Hu (胡)")
	fmt.Println("skip              - Skip turn")
	fmt.Println("status            - Show current status")
	fmt.Println("help              - Show this help")
	fmt.Println("quit              - Quit the game")
	fmt.Println("================\n")
}

// printStatus：显示当前状态
func (tc *TestClient) printStatus() {
	fmt.Printf("\n[%s] Status:\n", tc.userID)
	fmt.Printf("  Connected: %v\n", tc.conn != nil)
	fmt.Printf("  Messages in queue: %d\n", len(tc.messageQueue))
	fmt.Println()
}
