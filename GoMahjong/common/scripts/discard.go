package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/charmbracelet/log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

// InteractiveClientMode 交互式客户端模式
// 用户启动后先输入 userID，然后根据命令交互
func InteractiveClientMode() {
	// 第 1 步：获取用户 ID
	userID := getUserID()
	if userID == "" {
		log.Fatal("userID cannot be empty")
	}

	// 第 2 步：创建客户端并连接
	client := NewTestClient(userID)
	if err := client.Connect(); err != nil {
		log.Fatalf("connect failed: %v", err)
	}
	defer client.Close()

	fmt.Printf("\n[%s] Connected to Connector!\n", userID)
	fmt.Printf("[%s] Type 'help' for available commands\n\n", userID)

	// 第 3 步：启动后台循环处理消息
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// 后台消息处理循环
	wg.Add(1)
	go func() {
		defer wg.Done()
		messageLoop(ctx, client)
	}()

	// 后台信号处理（Ctrl+C）
	wg.Add(1)
	go func() {
		defer wg.Done()
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		<-sigCh
		fmt.Printf("\n[%s] Received interrupt signal, closing...\n", userID)
		cancel()
	}()

	// 第 4 步：主线程处理用户输入
	inputLoop(ctx, client)

	// 等待所有 goroutine 结束
	wg.Wait()
	fmt.Printf("[%s] Goodbye!\n", userID)
}

// getUserID 从标准输入获取用户 ID
func getUserID() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your userID: ")

	userID, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	return strings.TrimSpace(userID)
}

// inputLoop 处理用户输入的命令
func inputLoop(ctx context.Context, client *TestClient) {
	reader := bufio.NewReader(os.Stdin)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		// 显示提示符
		fmt.Printf("[%s] > ", client.userID)

		// 读取一行输入
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("[%s] input error: %v", client.userID, err)
			return
		}

		// 处理命令
		cmd := strings.TrimSpace(line)
		if cmd == "" {
			continue
		}
		if cmd == "quit" {
			return
		}

		// 执行命令
		if err := client.HandleCommand(cmd); err != nil {
			log.Printf("[%s] command error: %v", client.userID, err)
		}
	}
}

// messageLoop：后台处理服务器推送的消息
func messageLoop(ctx context.Context, client *TestClient) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-client.messageQueue:
			if msg != nil {
				fmt.Printf("\n[%s] ← %s\n", client.userID, msg.Route)
				if len(msg.Payload) > 0 {
					for k, v := range msg.Payload {
						fmt.Printf("    %s: %v\n", k, v)
					}
				}
				fmt.Printf("[%s] > ", client.userID) // 重新显示提示符
			}
		}
	}
}
