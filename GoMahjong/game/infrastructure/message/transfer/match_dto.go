package transfer

type MatchSuccessDTO struct {
	GameNodeID string            `json:"gameNodeID"`
	Players    map[string]string `json:"players"`
}
