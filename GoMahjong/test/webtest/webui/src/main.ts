// Main entry point

import { wsClient } from './ws';
import { gameBoard } from './ui/board';
import { tileToChar, tileToName } from './ui/tile';
import type { WSMessage, LogMessage, Tile, GameStatePush, RoundStartPush, DrawTilePush, DiscardTilePush, OperationsPush, PlayerOperation, MeldActionPush, AnkanPush, KakanPush } from './types';
import { Route } from './types';

// State
let currentPlayerId: string | null = null;
let selectedTile: Tile | null = null;
let showPayload = true;
let currentOperations: PlayerOperation[] = [];  // 当前可选操作
let isMyTurn = false;  // 是否轮到自己出牌

// DOM Elements
const statusEl = document.getElementById('status')!;
const connectBtn = document.getElementById('connectBtn') as HTMLButtonElement;
const disconnectBtn = document.getElementById('disconnectBtn') as HTMLButtonElement;
const playerIdInput = document.getElementById('playerId') as HTMLInputElement;
const tokenInput = document.getElementById('token') as HTMLInputElement;
const logContainerEl = document.getElementById('logContainer')!;
const showPayloadCheckbox = document.getElementById('showPayload') as HTMLInputElement;

// Action buttons
const actionButtons = {
  chi: document.getElementById('actionChi') as HTMLButtonElement,
  pon: document.getElementById('actionPon') as HTMLButtonElement,
  kan: document.getElementById('actionKan') as HTMLButtonElement,
  ron: document.getElementById('actionRon') as HTMLButtonElement,
  tsumo: document.getElementById('actionTsumo') as HTMLButtonElement,
  riichi: document.getElementById('actionRiichi') as HTMLButtonElement,
  pass: document.getElementById('actionPass') as HTMLButtonElement,
};

// Custom action
const customRouteInput = document.getElementById('customRoute') as HTMLInputElement;
const customPayloadInput = document.getElementById('customPayload') as HTMLTextAreaElement;
const sendCustomBtn = document.getElementById('sendCustomBtn') as HTMLButtonElement;

// Initialize
function init() {
  // Set up WebSocket handlers
  wsClient.setHandlers(
    handleMessage,
    handleLog,
    handleConnectionChange
  );

  // Set up board tile selection
  gameBoard.setOnTileSelect((tile) => {
    selectedTile = tile;
    addLog('INFO', tile ? `SelectTile: ${tileToName(tile)} (type=${tile.type}, id=${tile.id}, char=${tileToChar(tile)})` : 'SelectTile: -');
    updateActionButtons();
  });

  // Bind event listeners
  bindEvents();
  
  // Update UI
  updateConnectionUI(false);
}

function bindEvents() {
  // Connection
  connectBtn.addEventListener('click', connect);
  disconnectBtn.addEventListener('click', disconnect);
  
  // Actions
  const actionDraw = document.getElementById('actionDraw') as HTMLButtonElement;
  actionDraw.addEventListener('click', () => sendAction(Route.PLAY_TILE));
  actionButtons.chi.addEventListener('click', () => sendMeldAction('CHI'));
  actionButtons.pon.addEventListener('click', () => sendMeldAction('PENG'));
  actionButtons.kan.addEventListener('click', () => sendKanAction());
  actionButtons.ron.addEventListener('click', () => sendHuAction('ron'));
  actionButtons.tsumo.addEventListener('click', () => sendHuAction('tsumo'));
  actionButtons.riichi.addEventListener('click', () => sendRiichi());
  actionButtons.pass.addEventListener('click', () => sendAction(Route.SKIP));
  
  // Custom action
  sendCustomBtn.addEventListener('click', sendCustomAction);
  
  // Log controls
  document.getElementById('clearLogBtn')?.addEventListener('click', clearLog);
  showPayloadCheckbox.addEventListener('change', () => {
    showPayload = showPayloadCheckbox.checked;
  });
}

async function connect() {
  const playerId = playerIdInput.value.trim();
  const token = tokenInput.value.trim();
  
  if (!playerId) {
    addLog('ERROR', 'Player ID is required');
    return;
  }

  // 立即保存到全局变量
  currentPlayerId = playerId;
  console.log('[Debug] currentPlayerId set to:', currentPlayerId);

  try {
    // First, connect via HTTP API
    const response = await fetch('/api/connect', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ playerId, token: token || undefined }),
    });
    
    const data = await response.json();
    
    if (data.error) {
      addLog('ERROR', data.error);
      return;
    }
    
    currentPlayerId = playerId;
    addLog('INFO', `TCP connected: ${playerId}`);
    
    // Then connect WebSocket
    await wsClient.connect(playerId);
    
    updateConnectionUI(true);
    
  } catch (err) {
    addLog('ERROR', `Connect failed: ${err}`);
  }
}

function disconnect() {
  if (!currentPlayerId) return;
  
  wsClient.disconnect();
  
  fetch('/api/disconnect', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ playerId: currentPlayerId }),
  }).then(() => {
    addLog('INFO', 'Disconnected');
    currentPlayerId = null;
    updateConnectionUI(false);
  });
}

function updateConnectionUI(connected: boolean) {
  connectBtn.disabled = connected;
  disconnectBtn.disabled = !connected;
  sendCustomBtn.disabled = !connected;
  
  // Action buttons are only enabled by OperationsPush, not by connection
  if (connected) {
    Object.values(actionButtons).forEach(btn => btn.disabled = true);
    const actionDraw = document.getElementById('actionDraw') as HTMLButtonElement;
    if (actionDraw) actionDraw.disabled = true;
  } else {
    Object.values(actionButtons).forEach(btn => btn.disabled = true);
    const actionDraw = document.getElementById('actionDraw') as HTMLButtonElement;
    if (actionDraw) actionDraw.disabled = true;
  }
  
  statusEl.textContent = connected ? `Connected: ${currentPlayerId}` : 'Not connected';
  statusEl.className = 'status' + (connected ? ' connected' : '');
}

function handleConnectionChange(connected: boolean) {
  if (!connected && currentPlayerId) {
    addLog('INFO', 'WebSocket disconnected');
  }
}

function handleMessage(msg: WSMessage) {
  addLog('RECV', msg.route, msg.payload);
  
  // Handle specific routes
  switch (msg.route) {
    case Route.ROUND_START: {
      const push = msg.payload as RoundStartPush;
      addLog('INFO', `Round Start: seat=${getSelfSeat()}, handCount=${push.handTiles?.length || 0}`);
      setSelfSeat(push.seats);
      isMyTurn = (push.currentTurn === getSelfSeat());
      
      const updateData = {
        handTiles: push.handTiles || [],
        doraIndicators: push.doraIndicators || [],
        currentTurn: push.currentTurn,
        situation: push.situation,
        seats: push.seats,
        players: [],
        remainingTiles: 0,
        operations: [],
        availableSecs: 0,
      };
      gameBoard.update(updateData);
      addLog('INFO', `Round start: ${push.situation.roundWind}${push.situation.roundNumber} honba=${push.situation.honba}`);
      updateActionButtons();
      break;
    }

    case Route.DRAW_TILE: {
      const push = msg.payload as DrawTilePush;
      gameBoard.addHandTile(push.tile);
      isMyTurn = true;  // 摸牌 = 轮到自己
      updateActionButtons();
      addLog('INFO', `Draw: ${tileToName(push.tile)}${push.isKanDraw ? ' (kan)' : ''}`);
      break;
    }

    case Route.DISCARD_TILE: {
      const push = msg.payload as DiscardTilePush;
      gameBoard.addDiscardTile(push.seatIndex, push.tile);
      
      const mySeat = getSelfSeat();
      addLog('INFO', `Discard Debug: msgSeat=${push.seatIndex}, selfSeat=${mySeat}`);

      // 只要是自己的座位，就根据消息里的 tile 信息移除手牌
      if (Number(push.seatIndex) === Number(mySeat)) {
        gameBoard.removeHandTile(push.tile);
        isMyTurn = false;
        selectedTile = null; // 确保选中状态也清理
      }
      
      clearOperationsUI();
      updateActionButtons();
      addLog('INFO', `Discard: seat ${push.seatIndex} → ${tileToName(push.tile)}`);
      break;
    }

    case Route.OPERATIONS: {
      const push = msg.payload as OperationsPush;
      currentOperations = push.operations;
      applyOperations(push.operations, push.availableSecs);
      addLog('INFO', `Operations: ${push.operations.map(o => o.type).join(',')} (${push.availableSecs}s)`);
      break;
    }

    case Route.MELD_ACTION: {
      const push = msg.payload as MeldActionPush;
      gameBoard.addMeld(push.seatIndex, { actionType: push.actionType, seatIndex: push.seatIndex, fromSeat: push.fromSeat, tiles: push.tiles });
      
      if (push.seatIndex === getSelfSeat()) {
        // 副露移除手牌逻辑：从 tiles 中去掉那张被吃/碰的牌（来自别人的弃牌）
        // 简单处理：如果是吃/碰/明杠，tiles 里的最后一张通常是拿到的那张，或者是根据协议确定
        // 这里我们先移除 tiles 中匹配的所有牌（除了被拿走的那张，如果能确定的话）
        // 稳妥起见，我们假设 push.tiles 包含了所有展示出来的牌，我们需要移除其中原本在手里的部分
        gameBoard.removeHandTiles(push.tiles); 
        isMyTurn = true;
      }
      clearOperationsUI();
      updateActionButtons();
      addLog('INFO', `Meld: seat ${push.seatIndex} ${push.actionType}`);
      break;
    }

    case Route.ANKAN_PUSH: {
      const push = msg.payload as AnkanPush;
      gameBoard.addMeld(push.seatIndex, { actionType: 'ANKAN', seatIndex: push.seatIndex, fromSeat: -1, tiles: push.tiles });
      if (push.seatIndex === getSelfSeat()) {
        gameBoard.removeHandTiles(push.tiles);
      }
      clearOperationsUI();
      addLog('INFO', `Ankan: seat ${push.seatIndex}`);
      break;
    }

    case Route.KAKAN_PUSH: {
      const push = msg.payload as KakanPush;
      gameBoard.addMeld(push.seatIndex, { actionType: 'KAKAN', seatIndex: push.seatIndex, fromSeat: push.fromSeat, tiles: push.tiles });
      if (push.seatIndex === getSelfSeat()) {
        gameBoard.removeHandTile(push.tiles[0]);
      }
      clearOperationsUI();
      addLog('INFO', `Kakan: seat ${push.seatIndex}`);
      break;
    }

    case Route.GAME_STATE:
      const state = msg.payload as GameStatePush;
      gameBoard.update(state);
      isMyTurn = (state.currentTurn === getSelfSeat());
      updateActionButtons();
      break;

    case 'auth.login.response':
      const authResp = msg.payload as { success: boolean; pid?: number; message?: string };
      if (authResp.success) {
        addLog('INFO', `Auth success, PID: ${authResp.pid}`);
      } else {
        addLog('ERROR', `Auth failed: ${authResp.message}`);
      }
      break;
  }
}

function handleLog(log: LogMessage) {
  addLog(log.level, log.message);
}

function addLog(level: LogMessage['level'], message: string, payload?: unknown) {
  const entry = document.createElement('div');
  entry.className = 'log-entry';
  
  const time = document.createElement('span');
  time.className = 'time';
  time.textContent = new Date().toLocaleTimeString();
  
  const levelSpan = document.createElement('span');
  levelSpan.className = `level level-${level}`;
  levelSpan.textContent = level;
  
  entry.appendChild(time);
  entry.appendChild(levelSpan);
  
  // Check if message is a route
  if (message.includes('.')) {
    const routeSpan = document.createElement('span');
    routeSpan.className = 'route';
    routeSpan.textContent = message;
    entry.appendChild(routeSpan);
    
    if (showPayload && payload) {
      const payloadSpan = document.createElement('span');
      payloadSpan.className = 'payload';
      payloadSpan.textContent = JSON.stringify(payload);
      entry.appendChild(payloadSpan);
    }
  } else {
    const msgSpan = document.createElement('span');
    msgSpan.textContent = message;
    entry.appendChild(msgSpan);
  }
  
  logContainerEl.appendChild(entry);
  logContainerEl.scrollTop = logContainerEl.scrollHeight;
}

function clearLog() {
  logContainerEl.innerHTML = '';
}

function sendMeldAction(actionType: string) {
  if (!wsClient.isConnected()) {
    addLog('ERROR', 'Not connected');
    return;
  }
  if (!selectedTile) {
    addLog('ERROR', 'Select a tile first');
    return;
  }
  const payload = { actionType, tiles: [selectedTile] };
  if (wsClient.send(Route.MELD, payload)) {
    addLog('SEND', Route.MELD, payload);
  }
}

function sendRiichi() {
  if (!wsClient.isConnected()) {
    addLog('ERROR', 'Not connected');
    return;
  }
  if (!selectedTile) {
    addLog('ERROR', 'Select a tile to discard for riichi');
    return;
  }
  const payload = { tile: selectedTile };
  if (wsClient.send(Route.RIICHI, payload)) {
    addLog('SEND', Route.RIICHI, payload);
    gameBoard.clearSelection();
    selectedTile = null;
  }
}

function sendAction(route: string, extraPayload?: Record<string, unknown>) {
  if (!wsClient.isConnected()) {
    addLog('ERROR', 'Not connected');
    return;
  }
  
  let payload: Record<string, unknown> = extraPayload || {};
  
  // Add selected tile for play action
  if (route === Route.PLAY_TILE && selectedTile) {
    payload.tile = selectedTile;
    gameBoard.clearSelection();
    selectedTile = null;
  }
  
  if (wsClient.send(route, payload)) {
    addLog('SEND', route, payload);
  }
}

function sendCustomAction() {
  const route = customRouteInput.value.trim();
  let payload: unknown = {};
  
  try {
    payload = JSON.parse(customPayloadInput.value);
  } catch {
    addLog('ERROR', 'Invalid JSON payload');
    return;
  }
  
  if (wsClient.send(route, payload)) {
    addLog('SEND', route, payload);
  }
}

function applyOperations(ops: PlayerOperation[], availableSecs: number) {
  // 先禁用所有操作按钮
  clearOperationsUI();

  // 恢复当前操作列表（clearOperationsUI 会清空）
  currentOperations = ops;

  const opTypes = new Set(ops.map(o => o.type));

  // 无操作时直接返回
  if (opTypes.size === 0) return;

  const actionDraw = document.getElementById('actionDraw') as HTMLButtonElement;

  actionDraw.disabled = !(selectedTile && ops.length > 0);
  updateActionButtons();

  if (opTypes.has('HU')) {
    actionButtons.ron.disabled = false;
    actionButtons.tsumo.disabled = false;
  }
  if (opTypes.has('GANG')) {
    actionButtons.kan.disabled = false;
  }
  if (opTypes.has('PENG')) {
    actionButtons.pon.disabled = false;
  }
  if (opTypes.has('CHI')) {
    actionButtons.chi.disabled = false;
  }

  // 有操作时总是可以跳过
  actionButtons.pass.disabled = false;
  actionButtons.riichi.disabled = false;  // TODO: 根据是否可立直判断
}

function clearOperationsUI() {
  Object.values(actionButtons).forEach(btn => btn.disabled = true);
  currentOperations = [];
}

let selfSeat: number = -1;
function getSelfSeat(): number {
  return selfSeat;
}
function setSelfSeat(seats: { seatIndex: number; playerId: string }[]) {
  // 如果当前没有 ID，尝试从输入框重新获取一次作为兜底
  if (!currentPlayerId) {
    currentPlayerId = (document.getElementById('playerId') as HTMLInputElement).value.trim();
  }

  console.log('[Debug] setSelfSeat start:', {
    searchingFor: currentPlayerId,
    availableSeats: seats
  });

  let matched = false;
  for (const s of seats) {
    const sid = String(s.playerId).trim();
    const mid = String(currentPlayerId).trim();
    
    // 提取数字进行比较
    const sidNum = sid.match(/\d+/)?.[0];
    const midNum = mid.match(/\d+/)?.[0];
    const numMatch = sidNum && midNum && (Number(sidNum) === Number(midNum));
    
    if (sid === mid || sid.includes(mid) || mid.includes(sid) || numMatch) {
      selfSeat = Number(s.seatIndex);
      console.log('[Debug] Identity MATCHED! My seatIndex is:', selfSeat);
      gameBoard.setSelfSeat(selfSeat);
      matched = true;
      break;
    }
  }

  if (!matched) {
    console.error('[Debug] Identity CRITICAL FAILURE! Could not find my ID in seats list.');
    addLog('ERROR', `Identity mismatch: MyID=${currentPlayerId}, Seats=${JSON.stringify(seats.map(s => s.playerId))}`);
  }
}

function updateActionButtons() {
  const actionDraw = document.getElementById('actionDraw') as HTMLButtonElement;
  if (!actionDraw) return;
  // 打牌按钮：轮到自己 + 已选牌
  actionDraw.disabled = !(isMyTurn && !!selectedTile);
}

function sendKanAction() {
  if (!wsClient.isConnected()) { addLog('ERROR', 'Not connected'); return; }
  const gangOp = currentOperations.find(o => o.type === 'GANG');
  if (!gangOp) { addLog('ERROR', 'No GANG available'); return; }
  // 区分暗杠和加杠：暗杠用 ankan route，加杠用 kakan route
  // 简化：通过 meld route 发送
  const payload = { actionType: 'GANG', tiles: gangOp.tiles };
  if (wsClient.send(Route.MELD, payload)) {
    addLog('SEND', Route.MELD, payload);
  }
}

function sendHuAction(kind: 'ron' | 'tsumo') {
  if (!wsClient.isConnected()) { addLog('ERROR', 'Not connected'); return; }
  const huOp = currentOperations.find(o => o.type === 'HU');
  if (!huOp) { addLog('ERROR', 'No HU available'); return; }
  const payload = { actionType: 'HU', tiles: huOp.tiles };
  if (wsClient.send(Route.MELD, payload)) {
    addLog('SEND', Route.MELD, payload);
  }
}

// Start
init();
