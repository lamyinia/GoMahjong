package conn

import (
	"sync"
)

type Session struct {
	sync.RWMutex
	ConnID  string                 // 连接 ID
	UserID  string                 // 用户 ID
	data    map[string]interface{} // 单连接数据（仅当前连接可见）
	all     map[string]interface{} // 全局共享数据（所有连接可见）
	manager *Manager
}

func NewSession(connID string, manager *Manager) *Session {
	return &Session{
		ConnID:  connID,
		data:    make(map[string]any),
		all:     make(map[string]any),
		manager: manager,
	}
}

func (s *Session) SetData(connID string, data map[string]any) {
	s.Lock()
	defer s.Unlock()
	if s.ConnID == connID {
		for k, v := range data {
			s.data[k] = v
		}
	}
}

func (s *Session) SetAll(data map[string]any) {
	s.Lock()
	defer s.Unlock()
	for k, v := range data {
		s.all[k] = v
	}
}

func (s *Session) Close() {
	s.Lock()
	defer s.Unlock()
	s.data = make(map[string]any)
	s.all = make(map[string]any)
}
