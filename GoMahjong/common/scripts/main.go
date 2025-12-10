package main

import (
	"github.com/charmbracelet/log"
)

func main() {
	// 启动 Web UI 模式
	WebUIMode()
}

// WebUIMode Web UI 模式 - 单个页面管理多个玩家
func WebUIMode() {
	// 创建 Web 服务器
	server := NewWebServer(8080)

	// 添加玩家
	players := []string{
		"67473d339411b23c61f5a001",
		"69398f053498e7e7190b23a7",
		"69398f0c3498e7e7190b23a8",
		"69398f1c3498e7e7190b23a9",
	}

	for _, playerID := range players {
		if err := server.AddPlayer(playerID); err != nil {
			log.Printf("Failed to add player %s: %v", playerID, err)
		}
	}

	// 启动 Web 服务器
	log.Info("Web UI available at http://localhost:8080")
	if err := server.Start(); err != nil {
		log.Fatal("Server error", "err", err)
	}
}
