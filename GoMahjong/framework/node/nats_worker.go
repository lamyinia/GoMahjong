package node

import (
	"common/log"
	"encoding/json"
	"framework/stream"
)

type NatsWorker struct {
	NatsCli   Client
	readChan  chan []byte
	writeChan chan *stream.Message
	handlers  LogicHandler
}

func newWorker() *NatsWorker {
	return &NatsWorker{
		readChan:  make(chan []byte, 1024),
		writeChan: make(chan *stream.Message, 1024),
		handlers:  make(LogicHandler),
	}
}

func (worker *NatsWorker) Run(topic string) error {
	worker.NatsCli = NewNatsClient(topic, worker.readChan)
	worker.NatsCli.Run("nats://localhost:4222")

	go worker.readChanMessage()
	go worker.writeChanMessage()
	return nil
}

func (worker *NatsWorker) readChanMessage() {
	for {
		select {
		case rawMessage := <-worker.readChan:
			var message stream.Message
			json.Unmarshal(rawMessage, message)
			route := message.Route
			if handler := worker.handlers[route]; handler != nil {
				go func() {
					body := message.Body
					result := handler(body.Data)
					if result != nil {
						dataResp, _ := json.Marshal(result)
						body.Data = dataResp
						messageResp := &stream.Message{
							Source:      message.Destination,
							Destination: message.Source,
							Body:        body,
							UserID:      message.UserID,
							ConnID:      message.ConnID,
						}
						worker.writeChan <- messageResp
					}
				}()
			}
		}
	}
}

func (worker *NatsWorker) writeChanMessage() {
	for {
		select {
		case message, ok := <-worker.writeChan:
			if ok {
				marshal, _ := json.Marshal(message)
				err := worker.NatsCli.SendMessage(message.Destination, marshal)
				if err != nil {
					log.Error("nats 发送错误, message: %#v", message)
				}
			}
		}
	}
}

func (worker *NatsWorker) Close() {
	if worker.NatsCli != nil {
		worker.NatsCli.Close()
	}
}

func (worker *NatsWorker) RegisterHandlers(handlers LogicHandler) {
	worker.handlers = handlers
}
