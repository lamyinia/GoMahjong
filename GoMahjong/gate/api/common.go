package api

import (
	"common/http"
	"time"
)

// PingHandler ping 检查
func PingHandler(c *http.Context) error {
	c.Success(map[string]interface{}{
		"message":   "pong",
		"timestamp": time.Now().Unix(),
		"service":   "gate",
	})
	return nil
}

// HealthHandler 健康检查
func HealthHandler(c *http.Context) error {
	// TODO: 检查依赖服务状态
	status := checkServiceHealth()

	if status["healthy"].(bool) {
		c.Success(status)
	} else {
		c.ErrorWithCode(50001, "服务不健康")
	}
	return nil
}

func checkServiceHealth() map[string]interface{} {
	// TODO: 实际的健康检查逻辑
	return map[string]interface{}{
		"healthy": true,
		"services": map[string]string{
			"database":     "ok",
			"redis":        "ok",
			"user_service": "ok",
		},
		"timestamp": time.Now().Unix(),
	}
}
