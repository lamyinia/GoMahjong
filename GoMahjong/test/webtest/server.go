package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
)

// WebServer manages WebSocket connections and TCP clients
type WebServer struct {
	app      *http.Server
	tcpHost  string
	tcpPort  int
	players  map[string]*PlayerSession
	mu       sync.RWMutex
	upgrader websocket.Upgrader
	done     chan struct{}
}

// PlayerSession represents a player's WebSocket + TCP connection
type PlayerSession struct {
	PlayerID  string
	WSConn    *websocket.Conn
	WSConnMu  sync.Mutex
	TCPClient *TCPClient
	LogQueue  chan LogMessage
	Done      chan struct{}
}

// LogMessage for logging to frontend
type LogMessage struct {
	PlayerID  string `json:"playerId"`
	Level     string `json:"level"` // RECV, SEND, ERROR, INFO
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// WSCommand message from frontend
type WSCommand struct {
	Route   string          `json:"route"`
	Payload json.RawMessage `json:"payload"`
}

// WSMessage message to frontend
type WSMessage struct {
	Route   string      `json:"route"`
	Payload interface{} `json:"payload"`
}

// NewWebServer creates a new web server
func NewWebServer(webPort int, tcpHost string, tcpPort int) *WebServer {
	s := &WebServer{
		tcpHost: tcpHost,
		tcpPort: tcpPort,
		players: make(map[string]*PlayerSession),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for testing
			},
		},
		done: make(chan struct{}),
	}

	mux := http.NewServeMux()

	// Serve webui/dist/ static files
	distDir := s.findDistDir()
	if distDir == "" {
		log.Error("webui/dist/ not found. Run 'cd webui && npm run build' first.")
		log.Error("Looking for dist in: webui/dist, ./webui/dist, ../webui/dist")
	}
	log.Info("Serving static files from", "dir", distDir)
	fs := http.FileServer(http.Dir(distDir))
	mux.Handle("/", fs)

	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/api/players", s.handleGetPlayers)
	mux.HandleFunc("/api/connect", s.handleConnectPlayer)
	mux.HandleFunc("/api/disconnect", s.handleDisconnectPlayer)

	s.app = &http.Server{
		Addr:    fmt.Sprintf(":%d", webPort),
		Handler: mux,
	}

	return s
}

// Start begins the web server
func (s *WebServer) Start() error {
	log.Info("Web server started", "addr", s.app.Addr)
	return s.app.ListenAndServe()
}

// Stop shuts down the server
func (s *WebServer) Stop() {
	close(s.done)
	s.app.Close()

	s.mu.Lock()
	for _, player := range s.players {
		player.TCPClient.Close()
		close(player.Done)
	}
	s.players = make(map[string]*PlayerSession)
	s.mu.Unlock()

	log.Info("Web server stopped")
}

// findDistDir finds the webui/dist directory
func (s *WebServer) findDistDir() string {
	// Possible locations for dist directory
	candidates := []string{
		"webui/dist",
		"./webui/dist",
		"../webui/dist",
	}

	// Try to find absolute path based on executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "webui/dist"),
			filepath.Join(exeDir, "../webui/dist"),
		)
	}

	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "index.html")); err == nil {
			if abs, err := filepath.Abs(dir); err == nil {
				return abs
			}
		}
	}

	return ""
}

// handleGetPlayers returns list of connected players
func (s *WebServer) handleGetPlayers(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	players := make([]map[string]interface{}, 0)
	for id, p := range s.players {
		players = append(players, map[string]interface{}{
			"playerId":     id,
			"tcpConnected": p.TCPClient.IsConnected(),
			"wsConnected":  p.WSConn != nil,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(players)
}

// handleConnectPlayer creates a new player session with TCP connection
func (s *WebServer) handleConnectPlayer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID string `json:"playerId"`
		Token    string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.PlayerID == "" {
		http.Error(w, "playerId required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.players[req.PlayerID]; exists {
		http.Error(w, "player already connected", http.StatusBadRequest)
		return
	}

	// Create TCP client
	client := NewTCPClient(s.tcpHost, s.tcpPort, req.PlayerID)
	if err := client.Connect(); err != nil {
		http.Error(w, fmt.Sprintf("TCP connect failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Create session
	session := &PlayerSession{
		PlayerID:  req.PlayerID,
		TCPClient: client,
		LogQueue:  make(chan LogMessage, 100),
		Done:      make(chan struct{}),
	}
	s.players[req.PlayerID] = session

	// Start TCP message forwarder
	go s.forwardTCPMessages(session)

	// Send auth if token provided
	if req.Token != "" {
		if err := client.SendAuth(req.Token); err != nil {
			s.sendLog(session, "ERROR", fmt.Sprintf("Auth failed: %v", err))
		} else {
			s.sendLog(session, "SEND", "auth.login")
		}
	}

	log.Info("Player connected", "playerId", req.PlayerID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "connected",
		"playerId": req.PlayerID,
	})
}

// handleDisconnectPlayer removes a player session
func (s *WebServer) handleDisconnectPlayer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID string `json:"playerId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	session, exists := s.players[req.PlayerID]
	if exists {
		session.TCPClient.Close()
		close(session.Done)
		delete(s.players, req.PlayerID)
	}
	s.mu.Unlock()

	if !exists {
		http.Error(w, "player not found", http.StatusNotFound)
		return
	}

	log.Info("Player disconnected", "playerId", req.PlayerID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "disconnected"})
}

// handleWebSocket handles WebSocket connections from frontend
func (s *WebServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("WebSocket upgrade failed", "err", err)
		return
	}
	defer conn.Close()

	// Read initial message to get playerId
	var initMsg struct {
		PlayerID string `json:"playerId"`
	}
	if err := conn.ReadJSON(&initMsg); err != nil {
		log.Error("Failed to read init message", "err", err)
		return
	}

	s.mu.RLock()
	session, exists := s.players[initMsg.PlayerID]
	s.mu.RUnlock()

	if !exists {
		conn.WriteJSON(map[string]string{"error": "player not connected"})
		return
	}

	// Attach WebSocket to session
	session.WSConnMu.Lock()
	session.WSConn = conn
	session.WSConnMu.Unlock()

	log.Info("WebSocket connected", "playerId", initMsg.PlayerID)
	s.sendLog(session, "INFO", "WebSocket connected")

	// Start log forwarder
	logDone := make(chan struct{})
	go s.forwardLogs(session, conn, logDone)

	// Handle incoming commands
	for {
		var cmd WSCommand
		if err := conn.ReadJSON(&cmd); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("WebSocket error", "err", err)
			}
			break
		}

		// Forward to TCP server
		if err := session.TCPClient.SendMessage(cmd.Route, cmd.Payload); err != nil {
			s.sendLog(session, "ERROR", fmt.Sprintf("Send failed: %v", err))
		} else {
			s.sendLog(session, "SEND", cmd.Route)
		}
	}

	// Cleanup
	<-logDone
	session.WSConnMu.Lock()
	session.WSConn = nil
	session.WSConnMu.Unlock()

	log.Info("WebSocket disconnected", "playerId", initMsg.PlayerID)
}

// forwardTCPMessages forwards TCP messages to WebSocket
func (s *WebServer) forwardTCPMessages(session *PlayerSession) {
	for {
		select {
		case <-session.Done:
			return
		case <-s.done:
			return
		case msg, ok := <-session.TCPClient.RecvChan:
			if !ok {
				return
			}
			s.sendLog(session, "RECV", msg.Route)

			// Forward to WebSocket if connected
			session.WSConnMu.Lock()
			if session.WSConn != nil {
				session.WSConn.WriteJSON(WSMessage{
					Route:   msg.Route,
					Payload: msg.Payload,
				})
			}
			session.WSConnMu.Unlock()
		}
	}
}

// forwardLogs forwards log messages to WebSocket
func (s *WebServer) forwardLogs(session *PlayerSession, conn *websocket.Conn, done chan struct{}) {
	defer close(done)

	for {
		select {
		case <-session.Done:
			return
		case <-s.done:
			return
		case logMsg, ok := <-session.LogQueue:
			if !ok {
				return
			}
			if err := conn.WriteJSON(logMsg); err != nil {
				return
			}
		}
	}
}

// sendLog sends a log message to the session
func (s *WebServer) sendLog(session *PlayerSession, level, message string) {
	select {
	case session.LogQueue <- LogMessage{
		PlayerID:  session.PlayerID,
		Level:     level,
		Message:   message,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}:
	default:
		// Queue full, skip
	}
}
