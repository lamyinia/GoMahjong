package vo

type RankingType int

const (
	RankingNovice RankingType = iota
	RankingGuard
	RankingHero
	RankingSaint
	RankingSky
)

const (
	RankingNoviceMax = 299
	RankingGuardMin  = 300
	RankingGuardMax  = 599
	RankingHeroMin   = 600
	RankingHeroMax   = 1199
	RankingSaintMin  = 1200
	RankingSaintMax  = 1799
	RankingSkyMin    = 1800
)

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
	return RankingSky
}

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

func GetAllRankings() []RankingType {
	return []RankingType{
		RankingNovice,
		RankingGuard,
		RankingHero,
		RankingSaint,
		RankingSky,
	}
}
