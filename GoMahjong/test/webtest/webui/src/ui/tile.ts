// Tile rendering utilities

import type { Tile } from '../types';

// Text-based tile labels (unambiguous across all fonts)
const TILE_LABELS = {
  // Man (Characters/Wan) - 1-9
  man: ['1m', '2m', '3m', '4m', '5m', '6m', '7m', '8m', '9m'],
  // Pin (Dots/Tong) - 1-9
  pin: ['1p', '2p', '3p', '4p', '5p', '6p', '7p', '8p', '9p'],
  // So (Bamboo/Tiao) - 1-9
  so: ['1s', '2s', '3s', '4s', '5s', '6s', '7s', '8s', '9s'],
  // Feng (Winds) - East, South, West, North
  feng: ['東', '南', '西', '北'],
  // Dragon - White, Green, Red
  dragon: ['白', '發', '中'],
};

// Category names for CSS class
const CATEGORY_NAMES = ['man', 'pin', 'so', 'feng', 'dragon'] as const;

// Tile.type is continuous encoding (0-33):
//   0-8 = Man1-9, 9-17 = Pin1-9, 18-26 = So1-9, 27-30 = East..North, 31-33 = White..Red
// Convert to category index and offset within category
function tileCategoryInfo(tile: Tile): { category: string; offset: number } {
  const t = tile.type;
  if (t >= 0 && t <= 8) return { category: 'man', offset: t };
  if (t >= 9 && t <= 17) return { category: 'pin', offset: t - 9 };
  if (t >= 18 && t <= 26) return { category: 'so', offset: t - 18 };
  if (t >= 27 && t <= 30) return { category: 'feng', offset: t - 27 };
  if (t >= 31 && t <= 33) return { category: 'dragon', offset: t - 31 };
  return { category: 'unknown', offset: 0 };
}

export function tileToChar(tile: Tile): string {
  const { category, offset } = tileCategoryInfo(tile);
  const labels = TILE_LABELS[category as keyof typeof TILE_LABELS];
  if (!labels) return '?';
  return labels[offset] || '?';
}

export function createTileElement(tile: Tile, clickable = false): HTMLElement {
  const el = document.createElement('span');
  const { category } = tileCategoryInfo(tile);
  el.className = `tile ${category}`;
  el.textContent = tileToChar(tile);
  el.title = tileToName(tile);
  el.dataset.type = String(tile.type);
  el.dataset.id = String(tile.id);

  if (clickable) {
    el.addEventListener('click', () => {
      el.classList.toggle('selected');
    });
  }

  return el;
}

export function createTileElements(tiles: Tile[], clickable = false): HTMLElement[] {
  return tiles.map(t => createTileElement(t, clickable));
}

export function renderTilesTo(container: HTMLElement, tiles: Tile[], clickable = false): void {
  container.innerHTML = '';
  const elements = createTileElements(tiles, clickable);
  elements.forEach(el => container.appendChild(el));
}

// Tile name for display
export function tileToName(tile: Tile): string {
  const typeNames = ['Man', 'Pin', 'So', 'Feng', 'Dragon'];
  const idNames: Record<string, string[]> = {
    man: ['1', '2', '3', '4', '5', '6', '7', '8', '9'],
    pin: ['1', '2', '3', '4', '5', '6', '7', '8', '9'],
    so: ['1', '2', '3', '4', '5', '6', '7', '8', '9'],
    feng: ['East', 'South', 'West', 'North'],
    dragon: ['White', 'Green', 'Red'],
  };

  const { category, offset } = tileCategoryInfo(tile);
  const names = idNames[category];
  if (!names) return `Unknown(${tile.type})`;
  const catIdx = CATEGORY_NAMES.indexOf(category as typeof CATEGORY_NAMES[number]);
  return `${names[offset]} ${typeNames[catIdx]}`;
}
