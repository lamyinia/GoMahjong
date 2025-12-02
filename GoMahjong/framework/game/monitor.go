package game

import (
	"common/discovery"
	"common/log"
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
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

// Report 启动监控器
// 在独立的 goroutine 中定期收集负载信息并上报
func (m *Monitor) Report(ctx context.Context) {
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
		log.Debug(fmt.Sprintf("Monitor 上报负载信息成功: Load=%.2f, Games=%d, Players=%d, CPU=%.2f%%, Mem=%.2f%%",
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

// getCPUUsage 获取 CPU 使用率
// 使用 gopsutil 库获取系统整体 CPU 使用率（所有核心的平均值）
// 采样间隔为 200ms，平衡准确性和性能
// 注意：cpu.Percent() 第一次调用会立即返回，后续调用会等待采样间隔
func (m *Monitor) getCPUUsage() float64 {
	// 使用 200ms 采样间隔获取 CPU 使用率
	// false 表示获取所有 CPU 核心的平均使用率，而不是每个核心的单独值
	// 对于负载均衡，我们关心的是系统整体 CPU 使用率
	percentages, err := cpu.Percent(200*time.Millisecond, false)
	if err != nil {
		log.Error(fmt.Sprintf("Monitor 获取 CPU 使用率失败: %v", err))
		return 0.0
	}

	// percentages 是一个切片，包含所有 CPU 核心的使用率
	// 由于传入 false，切片中只有一个元素（所有核心的平均值）
	if len(percentages) == 0 {
		log.Warn("Monitor 获取 CPU 使用率返回空结果")
		return 0.0
	}

	cpuUsage := percentages[0]

	// 限制在 0-100 之间（理论上不会超出，但做防御性检查）
	if cpuUsage > 100.0 {
		cpuUsage = 100.0
	}
	if cpuUsage < 0.0 {
		cpuUsage = 0.0
	}

	return cpuUsage
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
