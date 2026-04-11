package main

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	bizpb "webtest/proto"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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

// routeRegistry maps routes to proto message constructors for JSON↔Protobuf conversion
var routeRegistry = map[string]func() proto.Message{
	// Request routes (frontend → C++)
	"auth.login":     func() proto.Message { return &bizpb.AuthRequest{} },
	"heartbeat.ping": func() proto.Message { return &bizpb.HeartbeatPing{} },
	"game.play":      func() proto.Message { return &bizpb.PlayTileRequest{} },

	// Response/Push routes (C++ → frontend)
	"auth.login.response": func() proto.Message { return &bizpb.AuthResponse{} },
	"heartbeat.pong":      func() proto.Message { return &bizpb.HeartbeatPong{} },
	"game.state":          func() proto.Message { return &bizpb.GameStatePush{} },
	"game.play.response":  func() proto.Message { return &bizpb.PlayTileResponse{} },
}

// protojsonMarshaler uses camelCase JSON names matching proto json tags
var protojsonMarshaler = protojson.MarshalOptions{EmitUnpopulated: true}
var protojsonUnmarshaler = protojson.UnmarshalOptions{DiscardUnknown: true}

// websocketUpgrader creates a WebSocket upgrader that allows all origins (for testing)
func websocketUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for testing
		},
	}
}
