package node

import (
	"common/log"
	"core/infrastructure/message/protocol"
	"core/infrastructure/message/transfer"
	"encoding/json"
	"fmt"
)

// PushHandler 处理 Push 类型消息的回调
type PushHandler func(users []string, body *protocol.Message, route string)
type LogicFunc func(message []byte) any
type SubscriberHandler map[string]LogicFunc

type NatsWorker struct {
	NatsCli           Client
	readChan          chan []byte
	writeChan         chan *transfer.ServicePacket
	subscriberHandler SubscriberHandler
	pushHandler       PushHandler
}

func NewNatsWorker() *NatsWorker {
	return &NatsWorker{
		readChan:          make(chan []byte, 1024),
		writeChan:         make(chan *transfer.ServicePacket, 1024),
		subscriberHandler: make(SubscriberHandler),
	}
}

// Run
// url nats 服务的地址
// nodeID 本地订阅的 nats 服务的频道
func (worker *NatsWorker) Run(url string, nodeID string) error {
	worker.NatsCli = NewNatsClient(nodeID, worker.readChan)
	if err := worker.NatsCli.Run(url); err != nil {
		return err
	}

	go worker.readChanMessage()
	go worker.writeChanMessage()
	return nil
}

func (worker *NatsWorker) readChanMessage() {
	for {
		select {
		case rawMessage := <-worker.readChan:
			var packet transfer.ServicePacket
			err := json.Unmarshal(rawMessage, &packet)
			if err != nil {
				log.Warn("NatsWorker-节点通信 packet 解析错误: %#v", packet)
				continue
			}
			route := packet.Route
			body := packet.Body
			if handler := worker.subscriberHandler[route]; handler != nil || body.Type == protocol.Push {
				go func() {
					// 如果 body.Type == protocol.Push，可能执行 handler，也可能不执行 handler，但是一定要推送
					// 处理器可能涉及 IO 操作，新开一个协程去处理
					var result any
					if handler != nil {
						result = handler(body.Data)
					}
					switch body.Type {
					case protocol.Request:
						if result != nil {
							dataResp, _ := json.Marshal(&result)
							body.Data = dataResp
							body.Type = protocol.Response
							messageResp := &transfer.ServicePacket{
								Source:      packet.Destination,
								Destination: packet.Source,
								Body:        body,
							}
							worker.writeChan <- messageResp
						}
					case protocol.Push:
						if worker.pushHandler != nil {
							worker.pushHandler(packet.PushUser, body, route)
						}
					}
				}()
			} else {
				log.Warn("NatsWorker-不支持的路由类型: %#v")
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

func (worker *NatsWorker) RegisterHandlers(handlers SubscriberHandler) {
	worker.subscriberHandler = handlers
}

// RegisterPushHandler 注册 Push 处理器
func (worker *NatsWorker) RegisterPushHandler(handler PushHandler) {
	worker.pushHandler = handler
}

// PushMessage 主动推送消息
// 将消息写入 writeChan，由 writeChanMessage goroutine 自动发送
func (worker *NatsWorker) PushMessage(packet *transfer.ServicePacket) error {
	select {
	case worker.writeChan <- packet:
		return nil
	default:
		return fmt.Errorf("推送消息失败：writeChan 已满")
	}
}
