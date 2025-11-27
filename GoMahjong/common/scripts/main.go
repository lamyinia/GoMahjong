package main

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	connectorWS = "ws://127.0.0.1:8082/ws"
	playerCount = 4
)

type TestPlayer struct {
	userID string
	conn   *websocket.Conn
}

func main() {
	players := []string{"user-001"}
	var wg sync.WaitGroup

	for _, uid := range players {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			runPlayer(id)
		}(uid)
	}

	wg.Wait()
	log.Println("demo finished")
}

func runPlayer(userID string) {
	url := connectorWS + "/test=" + userID
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Printf("[%s] dial failed: %v\n", userID, err)
		return
	}
	defer conn.Close()

	// 监听推送
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Printf("[%s] read error: %v\n", userID, err)
				return
			}
			log.Printf("[%s] recv: %s\n", userID, string(msg))
		}
	}()

	// 例如发送 “加入匹配” 消息
	//joinReq := `{"route":"march.match.join","body":{}}`
	//if err := conn.WriteMessage(websocket.TextMessage, []byte(joinReq)); err != nil {
	//	log.Printf("[%s] send join failed: %v\n", userID, err)
	//	return
	//}

	time.Sleep(30 * time.Second) // 演示期间保持连接
}
