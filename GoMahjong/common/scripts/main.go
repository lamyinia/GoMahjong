package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"

	"framework/protocal"
)

const connectorWS = "ws://127.0.0.1:8082/ws"

type TestClient struct {
	userID string
	conn   *websocket.Conn
	done   chan struct{}
}

func NewTestClient(userID string) *TestClient {
	return &TestClient{
		userID: userID,
		done:   make(chan struct{}),
	}
}

func (tc *TestClient) Connect() error {
	url := connectorWS + "/test=" + tc.userID
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	tc.conn = conn

	if err := tc.sendHandshake(); err != nil {
		return err
	}
	if err := tc.sendHandshakeAck(); err != nil {
		return err
	}

	go tc.listenLoop()
	go tc.heartbeatLoop()
	log.Printf("[%s] connected", tc.userID)
	return nil
}

func (tc *TestClient) Close() {
	select {
	case <-tc.done:
	default:
		close(tc.done)
	}
	if tc.conn != nil {
		_ = tc.conn.Close()
	}
}

func (tc *TestClient) listenLoop() {
	for {
		if tc.conn == nil {
			return
		}
		mt, msg, err := tc.conn.ReadMessage()
		if err != nil {
			log.Printf("[%s] read error: %v", tc.userID, err)
			return
		}
		if mt == websocket.BinaryMessage {
			packet, err := protocal.Decode(msg)
			if err != nil {
				log.Printf("[%s] decode error: %v", tc.userID, err)
				continue
			}
			if packet.Type == protocal.Data {
				body := packet.ParseBody()
				log.Printf("[%s] recv route=%s payload=%s", tc.userID, body.Route, string(body.Data))
			} else {
				log.Printf("[%s] recv packet type=%d, detail=%#v", tc.userID, packet.Type, packet.ParseBody())
			}
		} else {
			log.Printf("[%s] recv text: %s", tc.userID, string(msg))
		}
	}
}

func (tc *TestClient) heartbeatLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-tc.done:
			return
		case <-ticker.C:
			buf, err := protocal.Wrap(protocal.Heartbeat, []byte{})
			if err != nil {
				log.Printf("[%s] heartbeat wrap err: %v", tc.userID, err)
				continue
			}
			if tc.conn == nil {
				return
			}
			if err := tc.conn.WriteMessage(websocket.BinaryMessage, buf); err != nil {
				log.Printf("[%s] heartbeat send err: %v", tc.userID, err)
				return
			}
		}
	}
}

func (tc *TestClient) sendHandshake() error {
	body := protocal.HandshakeBody{
		Sys: protocal.Sys{
			Type:         "go-client",
			Version:      "1.0",
			ProtoVersion: 1,
			Heartbeat:    3,
			Serializer:   "json",
		},
	}
	data, _ := json.Marshal(body)
	buf, err := protocal.Wrap(protocal.Handshake, data)
	if err != nil {
		return err
	}
	return tc.conn.WriteMessage(websocket.BinaryMessage, buf)
}

func (tc *TestClient) sendHandshakeAck() error {
	buf, err := protocal.Wrap(protocal.HandshakeAck, []byte{})
	if err != nil {
		return err
	}
	return tc.conn.WriteMessage(websocket.BinaryMessage, buf)
}

func (tc *TestClient) SendRequest(route string, body any) error {
	if tc.conn == nil {
		return ErrNotConnected
	}
	payload, _ := json.Marshal(body)
	msg := &protocal.Message{
		Type:  protocal.Request,
		ID:    1,
		Route: route,
		Data:  payload,
	}
	return tc.sendMessage(msg)
}

func (tc *TestClient) sendMessage(msg *protocal.Message) error {
	body, err := protocal.MessageEncode(msg)
	if err != nil {
		return err
	}
	buf, err := protocal.Wrap(protocal.Data, body)
	if err != nil {
		return err
	}
	return tc.conn.WriteMessage(websocket.BinaryMessage, buf)
}

var ErrNotConnected = fmt.Errorf("client not connected")

func main() {
	buf, _ := protocal.Wrap(protocal.Handshake, []byte{255})
	fmt.Println(buf)
}

func test1() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		<-sigCh
		cancel()
	}()

	userIDs := []string{"user-001"}
	var wg sync.WaitGroup

	for _, uid := range userIDs {
		tc := NewTestClient(uid)
		wg.Add(1)
		go func(client *TestClient) {
			defer wg.Done()

			if err := client.Connect(); err != nil {
				log.Printf("[%s] connect failed: %v", client.userID, err)
				return
			}
			defer client.Close()
			<-ctx.Done()
		}(tc)
	}

	wg.Wait()
	log.Println("demo finished")
}
