package node

import "framework/stream"

type ServerNode struct {
	remoteCli Client
	readChan  chan []byte
	writeChan chan *stream.Message
}
