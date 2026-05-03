// Game board rendering

import type { GameStatePush, Tile, Meld } from '../types';
import { renderTilesTo, createTileElement } from './tile';

export class GameBoard {
  private handEl: HTMLElement;
  private discardEls: Map<number, HTMLElement>;
  private meldEls: Map<number, HTMLElement>;
  private doraEl: HTMLElement;
  private remainingEl: HTMLElement;
  private turnEl: HTMLElement;
  
  private selfSeatIndex: number = -1;
  private seatToElMap: Map<number, { discard: HTMLElement; meld: HTMLElement }> = new Map();

  private selectedTile: Tile | null = null;
  private onTileSelect: ((tile: Tile | null) => void) | null = null;
  private drawnTile: Tile | null = null;  // 最新摸到的牌
  private handTiles: Tile[] = [];  // 手牌（不含摸牌）

  constructor() {
    this.handEl = document.getElementById('hand')!;
    this.doraEl = document.getElementById('dora')!;
    this.remainingEl = document.getElementById('remaining')!;
    this.turnEl = document.getElementById('turn')!;
    this.discardEls = new Map(); // Will be initialized by setSelfSeat
    this.meldEls = new Map();
  }

  setSelfSeat(seatIndex: number) {
    this.selfSeatIndex = seatIndex;
    
    // 清除旧映射
    this.discardEls.clear();
    this.meldEls.clear();

    // 映射逻辑 (Mahjong standard: 0=East, 1=South, 2=West, 3=North)
    // 我们固定让 self 在 bottom (ID=4)
    const positions = [
      { offset: 0, id: 4 }, // Self -> Bottom
      { offset: 1, id: 3 }, // Next -> Right
      { offset: 2, id: 2 }, // Opposite -> Top
      { offset: 3, id: 1 }, // Previous -> Left
    ];

    for (const pos of positions) {
      const realSeat = (this.selfSeatIndex + pos.offset) % 4;
      const discardEl = document.getElementById(`discard-${pos.id}`);
      const meldEl = document.getElementById(`meld-${pos.id}`);
      if (discardEl) this.discardEls.set(realSeat, discardEl);
      if (meldEl) this.meldEls.set(realSeat, meldEl);
    }
    console.log('[GameBoard] Seat mapping updated for selfSeat:', seatIndex, 'Map size:', this.discardEls.size);
  }

  setOnTileSelect(callback: (tile: Tile | null) => void) {
    this.onTileSelect = callback;
  }

  update(state: GameStatePush) {
    console.log('[GameBoard] update state:', state);
    // Update hand tiles
    this.renderHand(state.handTiles);
    
    // Update remaining tiles
    this.remainingEl.textContent = String(state.remainingTiles);
    
    // Update current turn
    this.turnEl.textContent = `Player ${state.currentTurn + 1}`;
    
    // Update dora indicators
    this.renderDora(state.doraIndicators);
    
    // Update melds
    if (state.players && Array.isArray(state.players)) {
      for (const p of state.players) {
        if (p && p.melds) {
          this.renderMelds(p.seatIndex + 1, p.melds);
        }
      }
    }
  }

  private tileSortKey(tile: Tile): string {
    return String(tile.type).padStart(2, '0') + String(tile.id).padStart(2, '0');
  }

  private renderHand(tiles: Tile[]) {
    this.handTiles = [...tiles];
    this.drawnTile = null;
    this.renderHandDOM();
  }

  private renderHandDOM() {
    this.handEl.innerHTML = '';
    const sorted = [...this.handTiles].sort((a, b) => this.tileSortKey(a).localeCompare(this.tileSortKey(b)));
    
    sorted.forEach(tile => {
      this.handEl.appendChild(this.createHandTileEl(tile));
    });
    
    // 摸到的牌用间隔隔开，放在最右边
    if (this.drawnTile) {
      const spacer = document.createElement('span');
      spacer.className = 'hand-spacer';
      this.handEl.appendChild(spacer);
      this.handEl.appendChild(this.createHandTileEl(this.drawnTile));
    }
  }

  private createHandTileEl(tile: Tile): HTMLElement {
    const el = createTileElement(tile);
    el.addEventListener('click', () => {
      if (this.selectedTile &&
          this.selectedTile.type === tile.type &&
          this.selectedTile.id === tile.id) {
        el.classList.remove('selected');
        this.selectedTile = null;
      } else {
        this.handEl.querySelectorAll('.tile.selected').forEach(t => {
          t.classList.remove('selected');
        });
        el.classList.add('selected');
        this.selectedTile = tile;
      }
      this.onTileSelect?.(this.selectedTile);
    });
    return el;
  }

  updateDiscards(playerId: number, tiles: Tile[]) {
    const el = this.discardEls.get(playerId);
    if (el) {
      renderTilesTo(el, tiles, false);
    }
  }

  addHandTile(tile: Tile) {
    // 摸到的牌作为 drawnTile，单独放在最右边
    this.drawnTile = tile;
    this.renderHandDOM();
  }

  addDiscardTile(seatIndex: number, tile: Tile) {
    const el = this.discardEls.get(Number(seatIndex));
    if (el) {
      const tileEl = createTileElement(tile);
      tileEl.style.cursor = 'default';
      el.appendChild(tileEl);
    } else {
      console.warn('[GameBoard] No discard element found for seatIndex:', seatIndex, 'Current map:', Array.from(this.discardEls.keys()));
    }
  }

  removeHandTile(tile: Tile) {
    // 如果是摸到的牌被打出
    if (this.drawnTile &&
        this.drawnTile.type === tile.type &&
        this.drawnTile.id === tile.id) {
      this.drawnTile = null;
    } else {
      // 从手牌中移除 - 必须手动比较 type 和 id
      console.log(`[GameBoard] Trying to remove: type=${tile.type}(${typeof tile.type}), id=${tile.id}(${typeof tile.id})`);
      const idx = this.handTiles.findIndex(t => {
        const match = Number(t.type) === Number(tile.type) && Number(t.id) === Number(tile.id);
        return match;
      });
      if (idx !== -1) {
        console.log(`[GameBoard] Found tile at index ${idx}, removing it.`);
        this.handTiles.splice(idx, 1);
      } else {
        console.error('[GameBoard] Tile not found in hand:', tile, 'Current hand:', this.handTiles);
      }
    }
    // 如果摸牌还在，把它合并回手牌再重新渲染
    if (this.drawnTile) {
      this.handTiles.push(this.drawnTile);
      this.drawnTile = null;
    }
    this.renderHandDOM();
    // Clear selection
    if (this.selectedTile &&
        this.selectedTile.type === tile.type &&
        this.selectedTile.id === tile.id) {
      this.selectedTile = null;
      this.onTileSelect?.(null);
    }
  }

  removeHandTiles(tiles: Tile[]) {
    for (const t of tiles) {
      if (this.drawnTile &&
          this.drawnTile.type === t.type &&
          this.drawnTile.id === t.id) {
        this.drawnTile = null;
      } else {
        const idx = this.handTiles.findIndex(ht => ht.type === t.type && ht.id === t.id);
        if (idx !== -1) this.handTiles.splice(idx, 1);
      }
    }
    if (this.drawnTile) {
      this.handTiles.push(this.drawnTile);
      this.drawnTile = null;
    }
    this.renderHandDOM();
    this.selectedTile = null;
    this.onTileSelect?.(null);
  }

  addMeld(seatIndex: number, meld: Meld) {
    const el = this.meldEls.get(Number(seatIndex));
    if (!el) {
      console.warn('[GameBoard] No meld element found for seatIndex:', seatIndex);
      return;
    }
    const meldGroup = document.createElement('span');
    meldGroup.className = 'meld-group';
    for (const t of meld.tiles) {
      const tileEl = createTileElement(t);
      tileEl.style.cursor = 'default';
      meldGroup.appendChild(tileEl);
    }
    el.appendChild(meldGroup);
  }

  clearOperations() {
    // Disable all action buttons
    const actionDraw = document.getElementById('actionDraw') as HTMLButtonElement;
    if (actionDraw) actionDraw.disabled = true;
    document.querySelectorAll('.action-buttons button').forEach(btn => {
      (btn as HTMLButtonElement).disabled = true;
    });
  }

  private renderDora(doraIndicators: Tile[]) {
    this.doraEl.innerHTML = '';
    if (!doraIndicators || !Array.isArray(doraIndicators) || doraIndicators.length === 0) {
      this.doraEl.textContent = '?';
      return;
    }
    for (const t of doraIndicators) {
      const tileEl = createTileElement(t);
      tileEl.style.cursor = 'default';
      this.doraEl.appendChild(tileEl);
    }
  }

  private renderMelds(seatIndex: number, melds: Meld[]) {
    const el = this.meldEls.get(Number(seatIndex));
    if (!el) return;
    el.innerHTML = '';
    for (const meld of melds) {
      const meldGroup = document.createElement('span');
      meldGroup.className = 'meld-group';
      for (const t of meld.tiles) {
        const tileEl = createTileElement(t);
        tileEl.style.cursor = 'default';
        meldGroup.appendChild(tileEl);
      }
      el.appendChild(meldGroup);
    }
  }

  getSelectedTile(): Tile | null {
    return this.selectedTile;
  }

  clearSelection() {
    this.selectedTile = null;
    this.handEl.querySelectorAll('.tile.selected').forEach(t => {
      t.classList.remove('selected');
    });
  }
}

// Singleton
export const gameBoard = new GameBoard();
