package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"
)

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
