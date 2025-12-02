package mahjong

import (
	"framework/game"
	"framework/game/engines"
)

type RiichiMahjong4p struct {
	State engines.GameState
}

func (eg *RiichiMahjong4p) Initialize(players []*game.PlayerInfo) error {
	return nil
}

func (eg *RiichiMahjong4p) CalculateScore() map[string]int {
	return nil
}
