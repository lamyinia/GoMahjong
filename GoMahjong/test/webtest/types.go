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
	"auth.login":      func() proto.Message { return &bizpb.AuthRequest{} },
	"heartbeat.ping":  func() proto.Message { return &bizpb.HeartbeatPing{} },
	"game.playTile":   func() proto.Message { return &bizpb.PlayTileRequest{} },
	"game.createRoom": func() proto.Message { return &bizpb.DebugCreateRoomRequest{} },

	// Response/Push routes (C++ → frontend)
	"auth.login.response":      func() proto.Message { return &bizpb.AuthResponse{} },
	"heartbeat.pong":           func() proto.Message { return &bizpb.HeartbeatPong{} },
	"game.state":               func() proto.Message { return &bizpb.GameStatePush{} },
	"game.createRoom.response": func() proto.Message { return &bizpb.DebugCreateRoomResponse{} },

	// Game push routes (C++ → frontend)
	"game.round.start":  func() proto.Message { return &bizpb.RoundStartPush{} },
	"game.draw.tile":    func() proto.Message { return &bizpb.DrawTilePush{} },
	"game.discard.tile": func() proto.Message { return &bizpb.DiscardTilePush{} },
	"game.riichi.push":  func() proto.Message { return &bizpb.RiichiPush{} },
	"game.meld.action":  func() proto.Message { return &bizpb.MeldActionPush{} },
	"game.ankan.push":   func() proto.Message { return &bizpb.AnkanPush{} },
	"game.kakan.push":   func() proto.Message { return &bizpb.KakanPush{} },
	"game.ron":          func() proto.Message { return &bizpb.RonPush{} },
	"game.tsumo":        func() proto.Message { return &bizpb.TsumoPush{} },
	"game.round.end":    func() proto.Message { return &bizpb.RoundEndPush{} },
	"game.end":          func() proto.Message { return &bizpb.GameEndPush{} },
	"game.operations":   func() proto.Message { return &bizpb.OperationsPush{} },

	// Game request routes (frontend → C++)
	"game.meld":     func() proto.Message { return &bizpb.MeldRequest{} },
	"game.ankan":    func() proto.Message { return &bizpb.AnkanRequest{} },
	"game.kakan":    func() proto.Message { return &bizpb.KakanRequest{} },
	"game.riichi":   func() proto.Message { return &bizpb.RiichiRequest{} },
	"game.skip":     func() proto.Message { return &bizpb.SkipRequest{} },
	"game.snapshot": func() proto.Message { return &bizpb.SnapShootRequest{} },
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
