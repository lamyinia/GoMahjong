package game

import (
	"common/discovery"
	"common/log"
	"context"
	"fmt"
	"runtime"
	"time"
)

// Monitor 监控器
// 负责收集负载信息并上报给 etcd
type Monitor struct {
	roomManager    *RoomManager
	registry       *discovery.Registry
	updateInterval time.Duration
	stopCh         chan struct{}
}

// NewMonitor 创建监控器
// roomManager: 房间管理器，用于获取房间数和玩家数
// registry: 服务注册器，用于上报负载信息
// updateInterval: 更新间隔（建议 5-10 秒）
func NewMonitor(roomManager *RoomManager, registry *discovery.Registry, updateInterval time.Duration) *Monitor {
	return &Monitor{
		roomManager:    roomManager,
		registry:       registry,
		updateInterval: updateInterval,
		stopCh:         make(chan struct{}),
	}
}

// Start 启动监控器
// 在独立的 goroutine 中定期收集负载信息并上报
func (m *Monitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.updateInterval)
	defer ticker.Stop()

	// 立即执行一次
	m.reportLoad()

	for {
		select {
		case <-ctx.Done():
			log.Info("Monitor 收到停止信号，退出监控")
			return
		case <-m.stopCh:
			log.Info("Monitor 收到停止信号，退出监控")
			return
		case <-ticker.C:
			m.reportLoad()
		}
	}
}

// Stop 停止监控器
func (m *Monitor) Stop() {
	close(m.stopCh)
}

// reportLoad 收集负载信息并上报
func (m *Monitor) reportLoad() {
	loadInfo := m.collectLoadInfo()
	load := loadInfo.CalculateLoad()

	err := m.registry.UpdateLoad(load)
	if err != nil {
		log.Error(fmt.Sprintf("Monitor 上报负载信息失败: %v", err))
	} else {
		log.Info(fmt.Sprintf("Monitor 上报负载信息成功: Load=%.2f, Games=%d, Players=%d, CPU=%.2f%%, Mem=%.2f%%",
			load, loadInfo.GameCount, loadInfo.PlayerCount, loadInfo.CPUUsage, loadInfo.MemUsage))
	}
}

// collectLoadInfo 收集负载信息
func (m *Monitor) collectLoadInfo() *LoadInfo {
	// 从 RoomManager 获取房间数和玩家数
	gameCount, playerCount := m.roomManager.GetStats()

	// 获取 CPU 使用率（简化实现，后续可以使用 gopsutil 库）
	cpuUsage := m.getCPUUsage()

	// 获取内存使用率
	memUsage := m.getMemoryUsage()

	return &LoadInfo{
		GameCount:   gameCount,
		PlayerCount: playerCount,
		CPUUsage:    cpuUsage,
		MemUsage:    memUsage,
	}
}

// getCPUUsage 获取 CPU 使用率（简化实现）
// 注意：这是一个简化版本，实际应该使用 gopsutil 库获取准确的 CPU 使用率
// 这里使用一个占位实现，返回 0，后续可以扩展
func (m *Monitor) getCPUUsage() float64 {
	// TODO: 使用 gopsutil 库获取准确的 CPU 使用率
	// 当前返回 0，表示暂不统计 CPU（不影响负载计算的其他部分）
	// 后续可以添加: github.com/shirou/gopsutil/v3/cpu
	return 0.0
}

// getMemoryUsage 获取内存使用率
func (m *Monitor) getMemoryUsage() float64 {
	var mStats runtime.MemStats
	runtime.ReadMemStats(&mStats)

	// 计算当前进程的内存使用率
	// 注意：这里计算的是 Go 进程的内存使用，不是系统总内存
	// 如果需要系统总内存使用率，需要使用 gopsutil 库

	// 获取系统总内存（简化处理，假设为 8GB）
	// 实际应该使用 gopsutil 获取系统总内存
	totalMemory := uint64(8 * 1024 * 1024 * 1024) // 8GB

	if totalMemory == 0 {
		return 0.0
	}

	// 计算内存使用率（当前进程使用的内存 / 系统总内存）
	memUsage := float64(mStats.Sys) / float64(totalMemory) * 100.0

	// 限制在 0-100 之间
	if memUsage > 100.0 {
		memUsage = 100.0
	}
	if memUsage < 0.0 {
		memUsage = 0.0
	}

	return memUsage
}
