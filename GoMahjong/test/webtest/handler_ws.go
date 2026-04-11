package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

// 每个 WebSocket 连接由独立的 goroutine 处理
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

		// JSON Protobuf 转换
		if constructor, ok := routeRegistry[cmd.Route]; ok {
			msg := constructor()
			if err := protojsonUnmarshaler.Unmarshal(cmd.Payload, msg); err != nil {
				s.sendLog(session, "ERROR", fmt.Sprintf("JSON→Protobuf decode error for %s: %v", cmd.Route, err))
				continue
			}
			if err := session.TCPClient.SendProtoMessage(cmd.Route, msg); err != nil {
				s.sendLog(session, "ERROR", fmt.Sprintf("Send failed: %v", err))
			} else {
				s.sendLog(session, "SEND", cmd.Route)
			}
		} else {
			// Unknown route: send raw bytes as payload
			if err := session.TCPClient.SendMessage(cmd.Route, []byte(cmd.Payload)); err != nil {
				s.sendLog(session, "ERROR", fmt.Sprintf("Send failed: %v", err))
			} else {
				s.sendLog(session, "SEND", cmd.Route)
			}
		}
	}

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

			// Protobuf 转 JSON
			var payloadForWS interface{}
			if len(msg.Payload) > 0 {
				if constructor, ok := routeRegistry[msg.Route]; ok {
					pbMsg := constructor()
					if err := proto.Unmarshal(msg.Payload, pbMsg); err == nil {
						jsonBytes, err := protojsonMarshaler.Marshal(pbMsg)
						if err == nil {
							var jsonPayload interface{}
							if json.Unmarshal(jsonBytes, &jsonPayload) == nil {
								payloadForWS = jsonPayload
							} else {
								payloadForWS = string(jsonBytes)
							}
						} else {
							payloadForWS = fmt.Sprintf("protojson marshal error: %v", err)
						}
					} else {
						payloadForWS = fmt.Sprintf("proto unmarshal error: %v", err)
					}
				} else {
					payloadForWS = fmt.Sprintf("[binary %d bytes, unknown route]", len(msg.Payload))
				}
			} else {
				payloadForWS = nil
			}

			// Forward to WebSocket if connected
			session.WSConnMu.Lock()
			if session.WSConn != nil {
				session.WSConn.WriteJSON(WSMessage{
					Route:   msg.Route,
					Payload: payloadForWS,
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
