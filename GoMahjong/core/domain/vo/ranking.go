package vo

import (
	"fmt"
)

// RankingType 段位枚举
// 数据库只存储数值（score），段位通过 GetRankingByScore 方法计算
type RankingType int

const (
	// RankingNovice 见习：0-299
	RankingNovice RankingType = iota
	// RankingGuard 雀士：300-599
	RankingGuard
	// RankingHero 豪杰：600-1199
	RankingHero
	// RankingSaint 雀圣：1200-1799
	RankingSaint
	// RankingSky 魂天：1800+
	RankingSky
)

// 段位数值范围常量
const (
	// 段位边界值（左闭右闭）
	RankingNoviceMax = 299  // 见习最大值
	RankingGuardMin  = 300  // 雀士最小值
	RankingGuardMax  = 599  // 雀士最大值
	RankingHeroMin   = 600  // 豪杰最小值
	RankingHeroMax   = 1199 // 豪杰最大值
	RankingSaintMin  = 1200 // 雀圣最小值
	RankingSaintMax  = 1799 // 雀圣最大值
	RankingSkyMin    = 1800 // 魂天最小值
)

// GetRankingByScore 根据数值获取段位
// score: 段位数值（从数据库读取）
// 返回：段位枚举
// 规则：
//   - 见习(novice): 0-299
//   - 雀士(guard): 300-599
//   - 豪杰(hero): 600-1199
//   - 雀圣(saint): 1200-1799
//   - 魂天(sky): 1800+
func GetRankingByScore(score int) RankingType {
	if score <= RankingNoviceMax {
		return RankingNovice
	}
	if score >= RankingGuardMin && score <= RankingGuardMax {
		return RankingGuard
	}
	if score >= RankingHeroMin && score <= RankingHeroMax {
		return RankingHero
	}
	if score >= RankingSaintMin && score <= RankingSaintMax {
		return RankingSaint
	}
	// score >= RankingSkyMin
	return RankingSky
}

// GetQueueKey 获取段位对应的 Redis 队列 Key
// 返回：如 "march:queue:rank:novice"
func (r RankingType) GetQueueKey() string {
	return fmt.Sprintf("march:queue:rank:%s", r.String())
}

// GetPlayerInfoKey 获取段位对应的玩家信息 Hash Key
// 返回：如 "march:user:info:rank:novice"
func (r RankingType) GetPlayerInfoKey() string {
	return fmt.Sprintf("march:user:info:rank:%s", r.String())
}

// String 返回段位名称（用于日志和 Redis Key）
func (r RankingType) String() string {
	switch r {
	case RankingNovice:
		return "novice"
	case RankingGuard:
		return "guard"
	case RankingHero:
		return "hero"
	case RankingSaint:
		return "saint"
	case RankingSky:
		return "sky"
	default:
		return "unknown"
	}
}

// GetDisplayName 返回段位显示名称（中文）
func (r RankingType) GetDisplayName() string {
	switch r {
	case RankingNovice:
		return "见习"
	case RankingGuard:
		return "雀士"
	case RankingHero:
		return "豪杰"
	case RankingSaint:
		return "雀圣"
	case RankingSky:
		return "魂天"
	default:
		return "未知"
	}
}

// GetAllRankings 获取所有段位列表（用于遍历匹配）
func GetAllRankings() []RankingType {
	return []RankingType{
		RankingNovice,
		RankingGuard,
		RankingHero,
		RankingSaint,
		RankingSky,
	}
}
