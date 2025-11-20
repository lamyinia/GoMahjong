package node

import (
	"common/log"
	"encoding/json"
	"framework/stream"
)

type NatsWorker struct {
	NatsCli   Client
	readChan  chan []byte
	writeChan chan *stream.ServicePacket
	handlers  LogicHandler
}

func NewNatsWorker() *NatsWorker {
	return &NatsWorker{
		readChan:  make(chan []byte, 1024),
		writeChan: make(chan *stream.ServicePacket, 1024),
		handlers:  make(LogicHandler),
	}
}

func (worker *NatsWorker) Run(url string, topic string) error {
	worker.NatsCli = NewNatsClient(topic, worker.readChan)
	worker.NatsCli.Run(url)

	go worker.readChanMessage()
	go worker.writeChanMessage()
	return nil
}

func (worker *NatsWorker) readChanMessage() {
	for {
		select {
		case rawMessage := <-worker.readChan:
			var packet stream.ServicePacket
			json.Unmarshal(rawMessage, packet)
			route := packet.Route
			if handler := worker.handlers[route]; handler != nil {
				go func() {
					body := packet.Body
					result := handler(body.Data)
					if result != nil {
						dataResp, _ := json.Marshal(result)
						body.Data = dataResp
						messageResp := &stream.ServicePacket{
							Source:      packet.Destination,
							Destination: packet.Source,
							Body:        body,
							UserID:      packet.UserID,
							ConnID:      packet.ConnID,
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
