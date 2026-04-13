// Main entry point

import { wsClient } from './ws';
import { gameBoard } from './ui/board';
import { tileToName } from './ui/tile';
import type { WSMessage, LogMessage, PlayerInfo, Tile, GameState } from './types';

// State
let currentPlayerId: string | null = null;
let selectedTile: Tile | null = null;
let showPayload = true;

// DOM Elements
const statusEl = document.getElementById('status')!;
const connectBtn = document.getElementById('connectBtn') as HTMLButtonElement;
const disconnectBtn = document.getElementById('disconnectBtn') as HTMLButtonElement;
const playerIdInput = document.getElementById('playerId') as HTMLInputElement;
const tokenInput = document.getElementById('token') as HTMLInputElement;
const playersListEl = document.getElementById('playersList')!;
const logContainerEl = document.getElementById('logContainer')!;
const showPayloadCheckbox = document.getElementById('showPayload') as HTMLInputElement;

// Action buttons
const actionButtons = {
  draw: document.getElementById('actionDraw') as HTMLButtonElement,
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
  actionButtons.draw.addEventListener('click', () => sendAction('game.draw'));
  actionButtons.chi.addEventListener('click', () => sendAction('game.chi'));
  actionButtons.pon.addEventListener('click', () => sendAction('game.pon'));
  actionButtons.kan.addEventListener('click', () => sendAction('game.kan'));
  actionButtons.ron.addEventListener('click', () => sendAction('game.ron'));
  actionButtons.tsumo.addEventListener('click', () => sendAction('game.tsumo'));
  actionButtons.riichi.addEventListener('click', () => sendAction('game.riichi'));
  actionButtons.pass.addEventListener('click', () => sendAction('game.pass'));
  
  // Custom action
  sendCustomBtn.addEventListener('click', sendCustomAction);
  
  // Log controls
  document.getElementById('clearLogBtn')?.addEventListener('click', clearLog);
  showPayloadCheckbox.addEventListener('change', () => {
    showPayload = showPayloadCheckbox.checked;
  });
  
  // Add player
  document.getElementById('addPlayerBtn')?.addEventListener('click', addPlayer);
}

async function connect() {
  const playerId = playerIdInput.value.trim();
  const token = tokenInput.value.trim();
  
  if (!playerId) {
    alert('Please enter a Player ID');
    return;
  }
  
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
    refreshPlayers();
    
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
    refreshPlayers();
  });
}

function updateConnectionUI(connected: boolean) {
  connectBtn.disabled = connected;
  disconnectBtn.disabled = !connected;
  sendCustomBtn.disabled = !connected;
  
  Object.values(actionButtons).forEach(btn => {
    btn.disabled = !connected;
  });
  
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
    case 'game.state':
      const state = msg.payload as GameState;
      gameBoard.update(state);
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

function sendAction(route: string, extraPayload?: Record<string, unknown>) {
  if (!wsClient.isConnected()) {
    addLog('ERROR', 'Not connected');
    return;
  }
  
  let payload: Record<string, unknown> = extraPayload || {};
  
  // Add selected tile for play action
  if (route === 'game.playTile' && selectedTile) {
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

function updateActionButtons() {
  // Enable play button only when a tile is selected
  // This is just an example - real logic depends on game state
}

async function refreshPlayers() {
  try {
    const response = await fetch('/api/players');
    const players: PlayerInfo[] = await response.json();
    
    if (players.length === 0) {
      playersListEl.innerHTML = '<p>No players connected</p>';
      return;
    }
    
    playersListEl.innerHTML = players.map(p => `
      <div class="player-item">
        <span class="player-id">${p.playerId}</span>
        <span class="player-status">
          ${p.tcpConnected ? 'TCP' : ''} ${p.wsConnected ? 'WS' : ''}
        </span>
      </div>
    `).join('');
    
  } catch (err) {
    console.error('Failed to refresh players:', err);
  }
}

async function addPlayer() {
  const newPlayerId = (document.getElementById('newPlayerId') as HTMLInputElement).value.trim();
  const newToken = (document.getElementById('newPlayerToken') as HTMLInputElement).value.trim();
  
  if (!newPlayerId) {
    alert('Please enter a Player ID');
    return;
  }
  
  try {
    const response = await fetch('/api/connect', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ playerId: newPlayerId, token: newToken || undefined }),
    });
    
    const data = await response.json();
    
    if (data.error) {
      addLog('ERROR', data.error);
    } else {
      addLog('INFO', `Player added: ${newPlayerId}`);
      refreshPlayers();
    }
  } catch (err) {
    addLog('ERROR', `Add player failed: ${err}`);
  }
}

// Start
init();
