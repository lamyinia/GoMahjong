package stream

type PushUser struct {
	UserID      string `json:"userID"`
	ConnectorID string `json:"connectorID"`
}

type PushData struct {
	Data   []byte `json:"data"`
	Router string `json:"router"`
}

type PushMessage struct {
	PushData PushData       `json:"pushData"`
	Users    []PushUser     `json:"users"`
	Message  *ServicePacket `json:"message"`
}
