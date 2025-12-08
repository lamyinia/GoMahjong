package mahjong

import (
	"framework/game/engines"
	"framework/game/share"
)

type RiichiMahjong4p struct {
	State engines.GameState
}

func (eg *RiichiMahjong4p) Initialize(players []*share.PlayerInfo) error {
	return nil
}

func (eg *RiichiMahjong4p) CalculateScore() map[string]int {
	return nil
}
