package game

// LoadInfo 负载信息
// 用于计算 game 节点的综合负载评分
type LoadInfo struct {
	GameCount   int     // 当前对局数（房间数）
	PlayerCount int     // 当前玩家数
	CPUUsage    float64 // CPU 使用率（0-100）
	MemUsage    float64 // 内存使用率（0-100）
}

// CalculateLoad 计算综合负载评分
// 权重：CPU 30%、内存 20%、对局数 25%、玩家数 25%
// 返回值越小表示负载越低
func (li *LoadInfo) CalculateLoad() float64 {
	// 归一化处理：假设最大值为 100
	// CPU 和内存已经是百分比，直接使用
	// 对局数和玩家数需要归一化（这里假设最大值为 100，实际可以根据配置调整）
	normalizedGameCount := float64(li.GameCount) / 100.0
	if normalizedGameCount > 1.0 {
		normalizedGameCount = 1.0
	}

	normalizedPlayerCount := float64(li.PlayerCount) / 100.0
	if normalizedPlayerCount > 1.0 {
		normalizedPlayerCount = 1.0
	}

	// 计算加权平均
	load := li.CPUUsage*0.3 + li.MemUsage*0.2 + normalizedGameCount*100*0.25 + normalizedPlayerCount*100*0.25

	return load
}
