package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
)

// NewWebServer creates a new web server
func NewWebServer(webPort int, tcpHost string, tcpPort int) *WebServer {
	s := &WebServer{
		tcpHost: tcpHost,
		tcpPort: tcpPort,
		players: make(map[string]*PlayerSession),
		upgrader: websocketUpgrader(),
		done:     make(chan struct{}),
	}

	mux := http.NewServeMux()

	// 找到前端构建产物目录，然后作为静态文件服务挂载到 HTTP 服务器，webui/dist/ static files
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
	mux.HandleFunc("/api/createRoom", s.handleCreateRoom)

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
	candidates := []string{
		"webui/dist",
		"./webui/dist",
		"../webui/dist",
	}

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
