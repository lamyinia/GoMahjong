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

// ==================== 基础类型 (matching protobuf) ====================

export interface Tile {
  type: number;  // TileType 连续编码 (0-33)
  id: number;    // 同类型编号 (0-3)，数牌5的id=0为赤宝牌
}

export interface Situation {
  dealerIndex: number;    // 庄家座位 (0-3)
  roundWind: string;      // 场风: "East","South","West","North"
  roundNumber: number;    // 局数 (1-4)
  honba: number;          // 本场数
  riichiSticks: number;   // 供托（立直棒）
}

export interface Meld {
  actionType: string;     // "CHI","PENG","GANG","ANKAN","KAKAN"
  seatIndex: number;      // 鸣牌玩家座位
  fromSeat: number;       // 来源玩家座位（暗杠=-1）
  tiles: Tile[];          // 副露的牌
}

export interface HuClaim {
  winnerSeat: number;     // 和牌玩家座位
  loserSeat: number;      // 放铳玩家座位（自摸时=-1）
  winTile: Tile;          // 和牌
  han: number;            // 番数
  fu: number;             // 符数
  yaku: number[];         // 役列表
  points: number;         // 点数
}

export interface PlayerRanking {
  seatIndex: number;
  playerId: string;
  points: number;
  rank: number;           // 排名 (1-4)
}

export interface PlayerSeat {
  seatIndex: number;
  playerId: string;
}

export interface PlayerOperation {
  type: string;           // "HU","GANG","PENG","CHI"
  tiles: Tile[];          // 操作涉及的牌
}

export interface PlayerPublicState {
  seatIndex: number;
  discardTiles: Tile[];   // 弃牌区
  melds: Meld[];          // 副露
  riichi: boolean;        // 是否立直
  points: number;         // 当前点数
}

// ==================== 客户端请求 (C→S) ====================

export interface PlayTileRequest {
  tile: Tile;
}

export interface MeldRequest {
  actionType: string;     // "CHI","PENG","GANG"
  tiles: Tile[];          // 选择的牌
}

export interface AnkanRequest {
  tiles: Tile[];          // 暗杠的牌
}

export interface KakanRequest {
  tile: Tile;             // 加杠的牌
}

export interface RiichiRequest {
  tile: Tile;             // 立直时打出的牌
}

export interface SkipRequest {}

export interface KyuushuKyuukaiRequest {}

// ==================== 服务端推送 (S→C) ====================

export interface RoundStartPush {
  seats: PlayerSeat[];          // 座位映射
  doraIndicators: Tile[];       // 已翻开的宝牌指示牌
  situation: Situation;         // 场况
  handTiles: Tile[];            // 自己的手牌（仅自己可见）
  currentTurn: number;          // 当前出牌玩家座位
}

export interface DrawTilePush {
  tile: Tile;
  isKanDraw: boolean;           // 是否为岭上摸牌（杠后摸牌）
}

export interface DiscardTilePush {
  seatIndex: number;
  tile: Tile;
}

export interface RiichiPush {
  seatIndex: number;
}

export interface MeldActionPush {
  actionType: string;           // "CHI","PENG","GANG"
  seatIndex: number;
  fromSeat: number;
  tiles: Tile[];
  doraIndicator?: Tile;         // 杠后新翻的宝牌指示牌（仅GANG时有值）
}

export interface AnkanPush {
  seatIndex: number;
  tiles: Tile[];
  doraIndicator?: Tile;         // 杠后新翻的宝牌指示牌
}

export interface KakanPush {
  seatIndex: number;
  fromSeat: number;             // 原碰来源
  tiles: Tile[];
  doraIndicator?: Tile;         // 杠后新翻的宝牌指示牌
}

export interface RonPush {
  winnerSeat: number;
  loserSeat: number;
  winTile: Tile;
}

export interface TsumoPush {
  winnerSeat: number;
  winTile: Tile;
}

export interface RoundEndPush {
  endType: string;              // "RON","TSUMO","DRAW_EXHAUSTIVE","DRAW_3RON","DRAW_4KAN"
  claims: HuClaim[];
  uraDoraIndicators: Tile[];    // 里宝牌指示牌（立直和牌时翻开）
  delta: number[];              // 点数变化 [4]
  points: number[];             // 当前点数 [4]
  reason: string;               // 流局原因
  nextDealer: number;           // 下局庄家座位（-1=游戏结束）
}

export interface GameEndPush {
  rankings: PlayerRanking[];
}

export interface PlayerDisconnectPush {
  seatIndex: number;
}

export interface PlayerReconnectPush {
  seatIndex: number;
}

export interface OperationsPush {
  operations: PlayerOperation[];
  availableSecs: number;         // 操作限时秒数
}

export interface GameStatePush {
  seats: PlayerSeat[];           // 座位映射
  situation: Situation;          // 场况
  doraIndicators: Tile[];        // 宝牌指示牌
  handTiles: Tile[];             // 自己的手牌（仅自己可见）
  players: PlayerPublicState[];  // 各玩家公开信息
  currentTurn: number;           // 当前出牌玩家座位
  remainingTiles: number;        // 剩余牌数
  operations: PlayerOperation[]; // 当前可选操作
  availableSecs: number;         // 操作限时秒数
}

// ==================== 路由常量 ====================

export const Route = {
  // 请求 (C→S)
  SNAPSHOOT: 'rmj4p.snapshoot',
  PLAY_TILE: 'rmj4p.playTile',
  MELD: 'rmj4p.meld',
  ANKAN: 'rmj4p.ankan',
  KAKAN: 'rmj4p.kakan',
  RIICHI: 'rmj4p.riichi',
  SKIP: 'rmj4p.skip',
  KYUUSHU_KYUUKAI: 'rmj4p.kyuushuKyuukai',

  // 推送 (S→C)
  ROUND_START: 'rmj4p.roundStart',
  DRAW_TILE: 'rmj4p.drawTile',
  DISCARD_TILE: 'rmj4p.discardTile',
  RIICHI_PUSH: 'rmj4p.riichi',
  MELD_ACTION: 'rmj4p.meldAction',
  ANKAN_PUSH: 'rmj4p.ankan',
  KAKAN_PUSH: 'rmj4p.kakan',
  RON: 'rmj4p.ron',
  TSUMO: 'rmj4p.tsumo',
  ROUND_END: 'rmj4p.roundEnd',
  GAME_END: 'rmj4p.gameEnd',
  OPERATIONS: 'rmj4p.operations',
  GAME_STATE: 'rmj4p.gameState',
  PLAYER_DISCONNECT: 'rmj4p.playerDisconnect',
  PLAYER_RECONNECT: 'rmj4p.playerReconnect',

  // Debug
  DEBUG_CREATE_ROOM: 'rmj4p.debug.createRoom',
} as const;

// ==================== UI 状态 ====================

export interface UIState {
  connected: boolean;
  playerId: string | null;
  wsConnected: boolean;
  gameState: GameStatePush | null;
  selectedTile: Tile | null;
  players: PlayerInfo[];
}
