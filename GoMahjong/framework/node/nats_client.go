package node

import (
	"common/log"
	"github.com/nats-io/nats.go"
)

type Client interface {
	Run(string) error
	SendMessage(string, []byte) error
	Close() error
}

// NatsClient 不能及时发现 nats 服务关闭
type NatsClient struct {
	topic    string
	conn     *nats.Conn
	readChan chan []byte
}

func NewNatsClient(topic string, readChan chan []byte) *NatsClient {
	return &NatsClient{
		topic:    topic,
		readChan: readChan,
	}
}

func (nc *NatsClient) IsConnected() bool {
	return nc.conn != nil && nc.conn.IsConnected()
}

func (nc *NatsClient) Run(url string) error {
	log.Info("nats 服务正在启动, url:%s", url)
	var err error
	nc.conn, err = nats.Connect(url)
	if err != nil {
		log.Error("nats 连接错误,err:%v", err)
		return err
	}
	go nc.Subscribe()

	log.Info("nats 服务启动成功, url:%s", url)
	return nil
}

func (nc *NatsClient) Subscribe() {
	_, err := nc.conn.Subscribe(nc.topic, func(message *nats.Msg) {
		nc.readChan <- message.Data
	})
	if err != nil {
		log.Error("nats sub err:%v", err)
	}
}

func (nc *NatsClient) Close() error {
	if nc.conn == nil {
		return nil
	}

	nc.conn.Close()
	log.Info("NATS 连接已关闭")

	return nil
}

func (nc *NatsClient) SendMessage(subject string, data []byte) error {
	if !nc.IsConnected() {
		return ErrNotConnected
	}

	return nc.conn.Publish(subject, data)
}
