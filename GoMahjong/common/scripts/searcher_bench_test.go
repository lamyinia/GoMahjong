package main

import (
	"runtime/game/engines/mahjong"
	"testing"
)

func BenchmarkCandidates_NoCache(b *testing.B) {
	_, _, _, hand14 := makeSearcherAndHands()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := mahjong.NewSearcher()
		_ = s.SeekCandidates(hand14, 0, nil)
	}
}

func makeSearcherAndHands() (*mahjong.Searcher, mahjong.Hand34, mahjong.Hand34, []mahjong.Tile) {
	s := mahjong.NewSearcher()

	// Kokushi tenpai (13 uniques)
	hKokushi13, _ := mahjong.Hand34FromTiles(tiles(
		mahjong.Man1, mahjong.Man9,
		mahjong.Pin1, mahjong.Pin9,
		mahjong.So1, mahjong.So9,
		mahjong.East, mahjong.South, mahjong.West, mahjong.North,
		mahjong.White, mahjong.Green, mahjong.Red,
	))

	// Normal agari: 123m 123p 123s 789m EE
	hNormal14, _ := mahjong.Hand34FromTiles(tiles(
		mahjong.Man1, mahjong.Man2, mahjong.Man3,
		mahjong.Pin1, mahjong.Pin2, mahjong.Pin3,
		mahjong.So1, mahjong.So2, mahjong.So3,
		mahjong.Man7, mahjong.Man8, mahjong.Man9,
		mahjong.East, mahjong.East,
	))

	// Hand14 for riichi candidates
	hand14 := tiles(
		mahjong.Man1, mahjong.Man2, mahjong.Man3,
		mahjong.Pin1, mahjong.Pin2, mahjong.Pin3,
		mahjong.So1, mahjong.So2, mahjong.So3,
		mahjong.Man7, mahjong.Man8,
		mahjong.East, mahjong.East,
		mahjong.So1,
	)

	return s, hKokushi13, hNormal14, hand14
}

func BenchmarkShantenAll_Cached(b *testing.B) {
	s, hKokushi13, _, _ := makeSearcherAndHands()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.ShantenAll(hKokushi13, 0)
	}
}

func BenchmarkShantenAll_NoCache(b *testing.B) {
	_, hKokushi13, _, _ := makeSearcherAndHands()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := mahjong.NewSearcher()
		_ = s.ShantenAll(hKokushi13, 0)
	}
}

func BenchmarkIsAgariAll_Normal_Cached(b *testing.B) {
	s, _, hNormal14, _ := makeSearcherAndHands()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.IsAgariAll(hNormal14, 0)
	}
}

func BenchmarkWaitsAndUkeire_ChiitoiTenpai_Cached(b *testing.B) {
	s := mahjong.NewSearcher()
	h13, _ := mahjong.Hand34FromTiles(tiles(
		mahjong.Man1, mahjong.Man1,
		mahjong.Man2, mahjong.Man2,
		mahjong.Man3, mahjong.Man3,
		mahjong.Pin1, mahjong.Pin1,
		mahjong.Pin2, mahjong.Pin2,
		mahjong.So1, mahjong.So1,
		mahjong.East,
	))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.WaitsAndUkeire(h13, 0, nil)
	}
}

func BenchmarkRiichiCandidates_Cached(b *testing.B) {
	s, _, _, hand14 := makeSearcherAndHands()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.SeekCandidates(hand14, 0, nil)
	}
}
