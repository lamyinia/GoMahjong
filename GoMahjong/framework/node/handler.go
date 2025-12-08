package node

type LogicFunc func(message []byte) any

type SubscriberHandler map[string]LogicFunc
