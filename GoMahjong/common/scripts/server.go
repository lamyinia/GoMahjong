package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
)

// WebServer 管理 Web UI 和多个玩家客户端
type WebServer struct {
	app      *http.Server
	players  map[string]*PlayerClient
	mu       sync.RWMutex
	upgrader websocket.Upgrader
}

// PlayerClient 代表一个玩家的客户端
type PlayerClient struct {
	userID   string
	client   *TestClient
	conn     *websocket.Conn
	connMu   sync.Mutex
	logQueue chan LogMessage
	done     chan struct{}
}

// LogMessage 日志消息
type LogMessage struct {
	PlayerID  string `json:"playerId"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// CommandMessage 命令消息
type CommandMessage struct {
	PlayerID string `json:"playerId"`
	Command  string `json:"command"`
}

// NewWebServer 创建新的 Web 服务器
func NewWebServer(port int) *WebServer {
	ws := &WebServer{
		players: make(map[string]*PlayerClient),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ws.handleIndex)
	mux.HandleFunc("/ws", ws.handleWebSocket)
	mux.HandleFunc("/api/players", ws.handleGetPlayers)

	ws.app = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return ws
}

// Start 启动 Web 服务器
func (ws *WebServer) Start() error {
	log.Info("Starting Web UI server", "addr", ws.app.Addr)
	return ws.app.ListenAndServe()
}

// AddPlayer 添加玩家
func (ws *WebServer) AddPlayer(userID string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if _, exists := ws.players[userID]; exists {
		return fmt.Errorf("player %s already exists", userID)
	}

	// 创建客户端
	client := NewTestClient(userID)
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect player %s: %v", userID, err)
	}

	pc := &PlayerClient{
		userID:   userID,
		client:   client,
		logQueue: make(chan LogMessage, 100),
		done:     make(chan struct{}),
	}

	ws.players[userID] = pc

	// 启动消息处理循环
	go ws.playerMessageLoop(pc)

	log.Info("Player added", "userID", userID)
	return nil
}

// playerMessageLoop 处理玩家的消息
func (ws *WebServer) playerMessageLoop(pc *PlayerClient) {
	defer func() {
		ws.mu.Lock()
		delete(ws.players, pc.userID)
		ws.mu.Unlock()
		pc.client.Close()
		close(pc.done)
	}()

	for {
		select {
		case <-pc.done:
			return
		case msg := <-pc.client.messageQueue:
			if msg != nil {
				logMsg := LogMessage{
					PlayerID:  pc.userID,
					Level:     "RECV",
					Message:   fmt.Sprintf("%v", msg),
					Timestamp: getCurrentTime(),
				}
				ws.sendLog(logMsg)
			}
		}
	}
}

func (ws *WebServer) sendLog(msg LogMessage) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	// 只发送给该玩家自己
	if pc, exists := ws.players[msg.PlayerID]; exists {
		select {
		case pc.logQueue <- msg:
		default:
			// 队列满，跳过
		}
	}
}

// handleIndex 处理主页请求
func (ws *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, htmlContent)
}

// handleWebSocket 处理 WebSocket 连接
func (ws *WebServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("WebSocket upgrade failed", "err", err)
		return
	}
	defer conn.Close()

	// 读取初始消息获取 playerID
	var msg map[string]string
	if err := conn.ReadJSON(&msg); err != nil {
		log.Error("Failed to read initial message", "err", err)
		return
	}

	playerID, ok := msg["playerId"]
	if !ok {
		log.Error("Missing playerId in initial message")
		return
	}

	ws.mu.RLock()
	pc, exists := ws.players[playerID]
	ws.mu.RUnlock()

	if !exists {
		log.Error("Player not found", "playerID", playerID)
		conn.WriteJSON(map[string]string{"error": "player not found"})
		return
	}

	pc.connMu.Lock()
	pc.conn = conn
	pc.connMu.Unlock()

	log.Info("WebSocket connected", "playerID", playerID)

	// 启动日志转发 goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-pc.done:
				return
			case logMsg := <-pc.logQueue:
				pc.connMu.Lock()
				if pc.conn != nil {
					if err := pc.conn.WriteJSON(logMsg); err != nil {
						pc.connMu.Unlock()
						return
					}
				}
				pc.connMu.Unlock()
			}
		}
	}()

	// 处理来自客户端的消息
	for {
		var cmdMsg CommandMessage
		if err := conn.ReadJSON(&cmdMsg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("WebSocket error", "err", err)
			}
			break
		}

		// 执行命令
		if err := pc.client.HandleCommand(cmdMsg.Command); err != nil {
			logMsg := LogMessage{
				PlayerID:  playerID,
				Level:     "ERROR",
				Message:   fmt.Sprintf("Command error: %v", err),
				Timestamp: getCurrentTime(),
			}
			ws.sendLog(logMsg)
		} else {
			logMsg := LogMessage{
				PlayerID:  playerID,
				Level:     "SEND",
				Message:   cmdMsg.Command,
				Timestamp: getCurrentTime(),
			}
			ws.sendLog(logMsg)
		}
	}

	<-done
	pc.connMu.Lock()
	pc.conn = nil
	pc.connMu.Unlock()
}

// handleGetPlayers 获取所有玩家信息
func (ws *WebServer) handleGetPlayers(w http.ResponseWriter, r *http.Request) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	players := make([]map[string]string, 0)
	for userID := range ws.players {
		players = append(players, map[string]string{
			"userID": userID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(players)
}

// getCurrentTime 获取当前时间字符串
func getCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
