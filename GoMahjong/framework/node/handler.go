package node

type LogicFunc func(message []byte) any

type LogicHandler map[string]LogicFunc
