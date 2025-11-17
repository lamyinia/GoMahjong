package node

import "framework/stream"

type HandlerFunc func(message *stream.Message) any
type LogicHandler map[string]HandlerFunc
