package main

import (
	"runtime/game/engines/mahjong"
	"testing"
)

func tt(t mahjong.TileType) mahjong.Tile {
	return mahjong.Tile{Type: t, ID: 1}
}

func tiles(types ...mahjong.TileType) []mahjong.Tile {
	out := make([]mahjong.Tile, 0, len(types))
	for _, t := range types {
		out = append(out, tt(t))
	}
	return out
}

func TestRiichiSearcher_KokushiShantenAndAgari(t *testing.T) {
	s := mahjong.NewSearcher()

	// 13-sided kokushi tenpai: all 13 terminals/honors, no pair.
	h13, _ := mahjong.Hand34FromTiles(tiles(
		mahjong.Man1, mahjong.Man9,
		mahjong.Pin1, mahjong.Pin9,
		mahjong.So1, mahjong.So9,
		mahjong.East, mahjong.South, mahjong.West, mahjong.North,
		mahjong.White, mahjong.Green, mahjong.Red,
	))
	if got := s.ShantenAll(h13, 0); got != 0 {
		t.Fatalf("kokushi shanten expected 0, got %d", got)
	}

	// Kokushi agari: add a duplicate terminal/honor.
	h14 := h13
	h14[int(mahjong.Man1)]++
	if !s.IsAgariAll(h14, 0) {
		t.Fatalf("kokushi agari expected true")
	}
}

func TestRiichiSearcher_ChiitoiShantenAndAgari(t *testing.T) {
	s := mahjong.NewSearcher()

	// 6 pairs + 1 single => chiitoi tenpai (shanten 0)
	h13, _ := mahjong.Hand34FromTiles(tiles(
		mahjong.Man1, mahjong.Man1,
		mahjong.Man2, mahjong.Man2,
		mahjong.Man3, mahjong.Man3,
		mahjong.Pin1, mahjong.Pin1,
		mahjong.Pin2, mahjong.Pin2,
		mahjong.So1, mahjong.So1,
		mahjong.East,
	))
	if got := s.ShantenAll(h13, 0); got != 0 {
		t.Fatalf("chiitoi shanten expected 0, got %d", got)
	}

	waits, ukeire := s.WaitsAndUkeire(h13, 0, nil)
	if len(waits) != 1 || waits[0] != mahjong.East {
		t.Fatalf("chiitoi waits expected [East], got %v", waits)
	}
	if ukeire != 3 {
		t.Fatalf("chiitoi ukeire expected 3 (4-1), got %d", ukeire)
	}

	// 7 pairs => agari
	h14 := h13
	h14[int(mahjong.East)]++
	if !s.IsAgariAll(h14, 0) {
		t.Fatalf("chiitoi agari expected true")
	}
}

func TestRiichiSearcher_NormalAgari(t *testing.T) {
	s := mahjong.NewSearcher()

	// 123m 123p 123s 789m + EE
	h14, _ := mahjong.Hand34FromTiles(tiles(
		mahjong.Man1, mahjong.Man2, mahjong.Man3,
		mahjong.Pin1, mahjong.Pin2, mahjong.Pin3,
		mahjong.So1, mahjong.So2, mahjong.So3,
		mahjong.Man7, mahjong.Man8, mahjong.Man9,
		mahjong.East, mahjong.East,
	))
	if !s.IsAgariAll(h14, 0) {
		t.Fatalf("normal agari expected true")
	}
}

func TestRiichiSearcher_NormalAgari_WithFixedMelds(t *testing.T) {
	s := mahjong.NewSearcher()

	// Assume one open meld already fixed (e.g. 123m). Remaining concealed tiles should form 3 melds + a pair.
	// concealed: 123p 123s 789m + EE => 11 tiles
	h11, _ := mahjong.Hand34FromTiles(tiles(
		mahjong.Pin1, mahjong.Pin2, mahjong.Pin3,
		mahjong.So1, mahjong.So2, mahjong.So3,
		mahjong.Man7, mahjong.Man8, mahjong.Man9,
		mahjong.East, mahjong.East,
	))
	if !s.IsAgariAll(h11, 1) {
		t.Fatalf("normal agari with fixedMelds=1 expected true")
	}

	// With fixed melds, chiitoi/kokushi should be excluded.
	// For an obviously kokushi-like 13 tiles, fixedMelds>0 should not return kokushi shanten.
	h13, _ := mahjong.Hand34FromTiles(tiles(
		mahjong.Man1, mahjong.Man9,
		mahjong.Pin1, mahjong.Pin9,
		mahjong.So1, mahjong.So9,
		mahjong.East, mahjong.South, mahjong.West, mahjong.North,
		mahjong.White, mahjong.Green, mahjong.Red,
	))
	if got := s.ShantenAll(h13, 1); got == 0 {
		t.Fatalf("with fixedMelds>0, shanten should not be kokushi tenpai (0); got %d", got)
	}
}

func TestRiichiSearcher_RiichiCandidates(t *testing.T) {
	s := mahjong.NewSearcher()

	// Tenpai shape (after discarding So1): 123m 123p 123s + 78m + EE; waits = 6m or 9m
	hand14 := tiles(
		mahjong.Man1, mahjong.Man2, mahjong.Man3,
		mahjong.Pin1, mahjong.Pin2, mahjong.Pin3,
		mahjong.So1, mahjong.So2, mahjong.So3,
		mahjong.Man7, mahjong.Man8,
		mahjong.East, mahjong.East,
		mahjong.So1, // extra tile to discard
	)

	cands := s.SeekCandidates(hand14, 0, nil)
	found := false
	for _, c := range cands {
		if c.DiscardType != mahjong.So1 {
			continue
		}
		found = true
		// waits should contain Man6 and Man9 (order not guaranteed)
		m := map[mahjong.TileType]bool{}
		for _, w := range c.Waits {
			m[w] = true
		}
		if !m[mahjong.Man6] || !m[mahjong.Man9] {
			t.Fatalf("expected waits to include Man6 and Man9, got %v", c.Waits)
		}
		if c.Ukeire != 8 {
			t.Fatalf("expected ukeire=8 (4+4), got %d", c.Ukeire)
		}
		if len(c.DiscardOptions) != 2 {
			t.Fatalf("expected 2 discard options for So1 (since two So1 tiles), got %d", len(c.DiscardOptions))
		}
	}
	if !found {
		t.Fatalf("expected to find riichi candidate discarding So1")
	}
}
