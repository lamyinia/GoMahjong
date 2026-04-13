// WebSocket message types

export interface WSMessage {
  route: string;
  payload: unknown;
}

export interface LogMessage {
  playerId: string;
  level: 'RECV' | 'SEND' | 'ERROR' | 'INFO';
  message: string;
  timestamp: string;
}

// Connection types

export interface ConnectRequest {
  playerId: string;
  token?: string;
}

export interface ConnectResponse {
  status: string;
  playerId?: string;
  error?: string;
}

export interface PlayerInfo {
  playerId: string;
  tcpConnected: boolean;
  wsConnected: boolean;
}

// Game types (matching protobuf)

export interface Tile {
  type: number;  // 0=wan, 1=tiao, 2=tong, 3=feng, 4=dragon
  id: number;    // 1-9 for wan/tiao/tong, 1-4 for feng, 1-3 for dragon
}

export interface GameState {
  roomId: string;
  handTiles: Tile[];
  discardTiles: Tile[];
  currentTurn: number;
  remainingTiles: number;
}

export interface PlayTileRequest {
  tile: Tile;
}

// UI state

export interface UIState {
  connected: boolean;
  playerId: string | null;
  wsConnected: boolean;
  gameState: GameState | null;
  selectedTile: Tile | null;
  players: PlayerInfo[];
}
