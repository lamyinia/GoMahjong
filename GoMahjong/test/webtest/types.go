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

// requestRouteRegistry maps frontend→C++ routes to proto message constructors for JSON→Protobuf conversion.
// Note: Some routes are reused for both request and push (e.g. rmj4p.riichi/ankan/kakan),
// so request/push must use different registries.
var requestRouteRegistry = map[string]func() proto.Message{
	// Auth/Heartbeat (frontend → C++)
	"auth.login":     func() proto.Message { return &bizpb.AuthRequest{} },
	"heartbeat.ping": func() proto.Message { return &bizpb.HeartbeatPing{} },

	// rmj4p C→S 请求
	"rmj4p.snapshoot":          func() proto.Message { return &bizpb.SnapShootRequest{} },
	"rmj4p.playTile":           func() proto.Message { return &bizpb.PlayTileRequest{} },
	"rmj4p.meld":               func() proto.Message { return &bizpb.MeldRequest{} },
	"rmj4p.ankan":              func() proto.Message { return &bizpb.AnkanRequest{} },
	"rmj4p.kakan":              func() proto.Message { return &bizpb.KakanRequest{} },
	"rmj4p.riichi":             func() proto.Message { return &bizpb.RiichiRequest{} },
	"rmj4p.skip":               func() proto.Message { return &bizpb.SkipRequest{} },
	"rmj4p.kyuushuKyuukai":     func() proto.Message { return &bizpb.KyuushuKyuukaiRequest{} },
	"rmj4p.debug.createRoom":   func() proto.Message { return &bizpb.DebugCreateRoomRequest{} },
}

// pushRouteRegistry maps C++→frontend routes to proto message constructors for Protobuf→JSON conversion.
var pushRouteRegistry = map[string]func() proto.Message{
	// Auth/Heartbeat (C++ → frontend)
	"auth.login.response": func() proto.Message { return &bizpb.AuthResponse{} },
	"heartbeat.pong":      func() proto.Message { return &bizpb.HeartbeatPong{} },

	// rmj4p 推送
	"rmj4p.roundStart":       func() proto.Message { return &bizpb.RoundStartPush{} },
	"rmj4p.drawTile":         func() proto.Message { return &bizpb.DrawTilePush{} },
	"rmj4p.discardTile":      func() proto.Message { return &bizpb.DiscardTilePush{} },
	"rmj4p.riichi":           func() proto.Message { return &bizpb.RiichiPush{} },
	"rmj4p.meldAction":       func() proto.Message { return &bizpb.MeldActionPush{} },
	"rmj4p.ankan":            func() proto.Message { return &bizpb.AnkanPush{} },
	"rmj4p.kakan":            func() proto.Message { return &bizpb.KakanPush{} },
	"rmj4p.ron":              func() proto.Message { return &bizpb.RonPush{} },
	"rmj4p.tsumo":            func() proto.Message { return &bizpb.TsumoPush{} },
	"rmj4p.roundEnd":         func() proto.Message { return &bizpb.RoundEndPush{} },
	"rmj4p.gameEnd":          func() proto.Message { return &bizpb.GameEndPush{} },
	"rmj4p.playerDisconnect": func() proto.Message { return &bizpb.PlayerDisconnectPush{} },
	"rmj4p.playerReconnect":  func() proto.Message { return &bizpb.PlayerReconnectPush{} },
	"rmj4p.operations":       func() proto.Message { return &bizpb.OperationsPush{} },
	"rmj4p.gameState":        func() proto.Message { return &bizpb.GameStatePush{} },

	// Debug response
	"rmj4p.debug.createRoom.response": func() proto.Message { return &bizpb.DebugCreateRoomResponse{} },
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
