package conn

import (
	"common/config"
	"strings"
)

// 测试白名单

// ws/test={userID}
func (w *Worker) extractUserIDFromTestPath(path string) (string, bool) {
	if config.ConnectorConfig.JwtConf.AllowTestPath {
		return "", false
	}
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "", false
	}

	segments := strings.Split(trimmed, "/")
	for _, segment := range segments {
		if strings.HasPrefix(segment, "test=") {
			userID := strings.TrimPrefix(segment, "test=")
			if userID != "" {
				return userID, true
			}
		}
	}
	return "", false
}
