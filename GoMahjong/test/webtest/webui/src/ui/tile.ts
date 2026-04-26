// Tile rendering utilities

import type { Tile } from '../types';

// Unicode mahjong characters
const TILE_CHARS = {
  // Man (Characters/Wan) - 1-9
  man: ['\u{1F019}', '\u{1F01A}', '\u{1F01B}', '\u{1F01C}', '\u{1F01D}',
        '\u{1F01E}', '\u{1F01F}', '\u{1F020}', '\u{1F021}'],
  // Pin (Dots/Tong) - 1-9
  pin: ['\u{1F007}', '\u{1F008}', '\u{1F009}', '\u{1F00A}', '\u{1F00B}',
        '\u{1F00C}', '\u{1F00D}', '\u{1F00E}', '\u{1F00F}'],
  // So (Bamboo/Tiao) - 1-9
  so: ['\u{1F010}', '\u{1F011}', '\u{1F012}', '\u{1F013}', '\u{1F014}',
       '\u{1F015}', '\u{1F016}', '\u{1F017}', '\u{1F018}'],
  // Feng (Winds) - East, South, West, North
  feng: ['\u{1F000}', '\u{1F001}', '\u{1F002}', '\u{1F003}'],
  // Dragon - White, Green, Red
  dragon: ['\u{1F004}', '\u{1F005}', '\u{1F006}'],
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
  const chars = TILE_CHARS[category as keyof typeof TILE_CHARS];
  if (!chars) return '?';
  return chars[offset] || '?';
}

export function createTileElement(tile: Tile, clickable = false): HTMLElement {
  const el = document.createElement('span');
  const { category } = tileCategoryInfo(tile);
  el.className = `tile ${category}`;
  el.textContent = tileToChar(tile);
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
