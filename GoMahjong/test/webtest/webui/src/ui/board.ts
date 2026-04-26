// Game board rendering

import type { GameStatePush, Tile } from '../types';
import { renderTilesTo, createTileElement } from './tile';

export class GameBoard {
  private handEl: HTMLElement;
  private discardEls: Map<number, HTMLElement>;
  private doraEl: HTMLElement;
  private remainingEl: HTMLElement;
  private turnEl: HTMLElement;
  
  private selectedTile: Tile | null = null;
  private onTileSelect: ((tile: Tile | null) => void) | null = null;

  constructor() {
    this.handEl = document.getElementById('hand')!;
    this.discardEls = new Map([
      [1, document.getElementById('discard-1')!],
      [2, document.getElementById('discard-2')!],
      [3, document.getElementById('discard-3')!],
      [4, document.getElementById('discard-4')!],
    ]);
    this.doraEl = document.getElementById('dora')!;
    this.remainingEl = document.getElementById('remaining')!;
    this.turnEl = document.getElementById('turn')!;
  }

  setOnTileSelect(callback: (tile: Tile | null) => void) {
    this.onTileSelect = callback;
  }

  update(state: GameStatePush) {
    // Update hand tiles
    this.renderHand(state.handTiles);
    
    // Update remaining tiles
    this.remainingEl.textContent = String(state.remainingTiles);
    
    // Update current turn
    this.turnEl.textContent = `Player ${state.currentTurn + 1}`;
    
    // Update dora (would come from server)
    // this.doraEl.textContent = tileToChar(state.dora);
  }

  private renderHand(tiles: Tile[]) {
    this.handEl.innerHTML = '';
    
    tiles.forEach(tile => {
      const el = createTileElement(tile);
      
      el.addEventListener('click', () => {
        // Toggle selection
        if (this.selectedTile && 
            this.selectedTile.type === tile.type && 
            this.selectedTile.id === tile.id) {
          // Deselect
          el.classList.remove('selected');
          this.selectedTile = null;
        } else {
          // Deselect previous
          this.handEl.querySelectorAll('.tile.selected').forEach(t => {
            t.classList.remove('selected');
          });
          // Select new
          el.classList.add('selected');
          this.selectedTile = tile;
        }
        
        this.onTileSelect?.(this.selectedTile);
      });
      
      this.handEl.appendChild(el);
    });
  }

  updateDiscards(playerId: number, tiles: Tile[]) {
    const el = this.discardEls.get(playerId);
    if (el) {
      renderTilesTo(el, tiles, false);
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
