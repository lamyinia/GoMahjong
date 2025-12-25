# TrueSkill 评分系统详解

TrueSkill 是微软开发的**现代竞技评分系统**，专为**多人、团队、动态匹配**的游戏设计。让我详细解释其原理、优势，以及如何在麻将游戏中应用。

## 一、基本概念

### 1. TrueSkill 是什么？

- 

  微软 Xbox Live 的匹配评级系统

- 

  Elo 系统的概率扩展版本

- 

  适用于**不确定性较高**的竞技游戏

- 

  能处理**团队比赛**和**多人排名赛**

### 2. 核心思想

传统 Elo 用**单个数值**表示技能，TrueSkill 用**两个参数**：

- 

  **μ (mu)**：技能水平估计（均值）

- 

  **σ (sigma)**：技能不确定性（标准差）

```
玩家技能 = 概率分布 N(μ, σ²)
而不是固定值
```

## 二、TrueSkill vs Elo 对比

| 特性       | Elo 系统      | TrueSkill 系统  |
| ---------- | ------------- | --------------- |
| 技能表示   | 单一数值 R    | 概率分布 (μ, σ) |
| 不确定性   | 隐含在 K 值中 | 显式建模 σ      |
| 团队支持   | 需要扩展      | 原生支持        |
| 多人排名   | 需要特殊处理  | 原生支持        |
| 收敛速度   | 固定 K 值     | 动态调整        |
| 实现复杂度 | 简单          | 中等            |
| 典型应用   | 国际象棋      | Xbox Live 游戏  |

## 三、TrueSkill 数学原理

### 1. 玩家技能建模

```
每个玩家的技能是一个正态分布：
技能 ~ N(μ, σ²)

比赛表现 = 技能 + 随机波动
表现 ~ N(μ, β²) 其中 β 是表现方差
```

### 2. 比赛预测

```
玩家A vs 玩家B：

A获胜概率 = P(表现_A > 表现_B)
          = Φ((μ_A - μ_B) / √(σ_A² + σ_B² + 2β²))
          
其中 Φ 是标准正态分布CDF
```

### 3. 贝叶斯更新

比赛后，根据结果更新分布：

```
后验分布 ∝ 先验分布 × 似然函数
```

## 四、TrueSkill 更新算法

### 1. 因子图算法

TrueSkill 使用**消息传递算法**在因子图上计算：

```
步骤：
1. 建立比赛因子图
2. 传播消息（期望传播）
3. 计算后验分布
```

### 2. 更新公式（简化版）

对于 1v1 比赛，A 击败 B：

```
# 计算所需中间变量
c = √(σ_A² + σ_B² + 2β²)
t = (μ_A - μ_B) / c

# 计算 v 和 w
v = v(t, ε)  # ε 是平局边际
w = w(t, ε)

# 更新 μ 和 σ
μ_A' = μ_A + (σ_A² / c) * v
μ_B' = μ_B - (σ_B² / c) * v

σ_A' = σ_A * √(1 - (σ_A² / c²) * w)
σ_B' = σ_B * √(1 - (σ_B² / c²) * w)
```

## 五、TrueSkill 在多人游戏中的应用

### 1. 麻将的适配

麻将需要处理**四人排名**，TrueSkill 有原生支持：

```
// 麻将排名更新示例
type Player struct {
    Mu    float64
    Sigma float64
}

func UpdateTrueSkillForMahjong(players []Player, ranks []int) {
    // ranks: 1,2,3,4 表示顺位
    
    // 使用 TrueSkill 的排名更新算法
    // 将四人排名视为一系列成对比较
    for i := 0; i < 4; i++ {
        for j := 0; j < 4; j++ {
            if i == j { continue }
            
            if ranks[i] < ranks[j] {
                // i 排名高于 j
                UpdatePairwise(players[i], players[j], true)
            } else if ranks[i] > ranks[j] {
                // i 排名低于 j
                UpdatePairwise(players[i], players[j], false)
            } else {
                // 平局（麻将中罕见但可能）
                UpdatePairwise(players[i], players[j], "draw")
            }
        }
    }
}
```

## 六、TrueSkill 的优势

### 1. 动态不确定性

```
新玩家：σ 大 → 快速调整 μ
老玩家：σ 小 → 稳定，变化慢
长期不玩：σ 增大 → 回归均值
```

### 2. 团队比赛处理

```
// 团队技能 = 成员技能之和
teamSkill := N(Σμ_i, Σσ_i²)
```

### 3. 排名预测准确

可以计算任意排名的概率，不只是胜负。

## 七、TrueSkill 在麻将中的实现

### 1. 数据结构

```
type TrueSkillRating struct {
    Mu    float64   // 技能均值
    Sigma float64   // 技能不确定性
    Beta  float64   // 表现方差 (通常固定)
    Tau   float64   // 动态因子
    DrawProbability float64 // 平局概率
}

type PlayerTrueSkill struct {
    ID       string
    Rating   TrueSkillRating
    Games    int
    LastPlay time.Time
}
```

### 2. 麻将专用参数设置

```
// 麻将推荐参数
func NewMahjongTrueSkill() TrueSkillRating {
    return TrueSkillRating{
        Mu:    25.0,      // 初始均值
        Sigma: 25.0 / 3,  // 初始不确定性
        Beta:  25.0 / 6,  // 表现方差
        Tau:   0.1,       // 动态因子
        DrawProbability: 0.01, // 麻将几乎不平局
    }
}
```

### 3. 四人排名更新算法

```
func UpdateMultiplayerTrueSkill(players []*PlayerTrueSkill, ranks []int) {
    n := len(players)
    
    // 转换为排名列表，1为最高
    // ranks[i] = 1,2,3,4
    
    // 计算每个玩家的表现后验
    for i := 0; i < n; i++ {
        // 收集所有对手信息
        opponents := make([]TrueSkillRating, 0, n-1)
        for j := 0; j < n; j++ {
            if i != j {
                opponents = append(opponents, players[j].Rating)
            }
        }
        
        // 计算后验分布
        newRating := computePosterior(players[i].Rating, opponents, ranks, i)
        
        // 应用动态因子（长期不玩增加不确定性）
        newRating = applyDynamics(players[i], newRating)
        
        players[i].Rating = newRating
    }
}
```

## 八、TrueSkill 2.0 增强

TrueSkill 2 增加了更多特性：

### 1. 个人表现因子

```
// 考虑和牌点数、役满等
performanceFactor := calculatePerformanceFactor(
    playerPoints,
    handValue,     // 番数
    isDealer,
    isTsumo,
)
```

### 2. 游戏质量评估

可以计算比赛的**平衡性预测**，用于匹配。

## 九、在麻将服务器中的实现方案

### 1. 完整实现示例

```
package trueskill

import (
    "math"
)

const (
    initialMu    = 25.0
    initialSigma = initialMu / 3.0
    beta         = initialMu / 6.0
    tau          = 0.1
    drawMargin   = 0.0 // 麻将基本不平局
)

type Rating struct {
    Mu    float64
    Sigma float64
}

type TrueSkill struct {
    Beta  float64
    Tau   float64
    DrawProbability float64
}

func NewTrueSkill() *TrueSkill {
    return &TrueSkill{
        Beta:  beta,
        Tau:   tau,
        DrawProbability: 0.0,
    }
}

// 1v1 更新
func (ts *TrueSkill) Update1v1(r1, r2 Rating, score float64) (Rating, Rating) {
    c := math.Sqrt(r1.Sigma*r1.Sigma + r2.Sigma*r2.Sigma + 2*ts.Beta*ts.Beta)
    
    // 计算胜负
    t := (r1.Mu - r2.Mu) / c
    
    // 计算v和w函数
    v := ts.v(t, score)
    w := ts.w(t, score)
    
    // 更新均值
    mu1 := r1.Mu + (r1.Sigma*r1.Sigma/c) * v
    mu2 := r2.Mu - (r2.Sigma*r2.Sigma/c) * v
    
    // 更新标准差
    sigma1 := r1.Sigma * math.Sqrt(1 - (r1.Sigma*r1.Sigma/(c*c)) * w)
    sigma2 := r2.Sigma * math.Sqrt(1 - (r2.Sigma*r2.Sigma/(c*c)) * w)
    
    // 应用动态因子
    sigma1 = math.Sqrt(sigma1*sigma1 + ts.Tau*ts.Tau)
    sigma2 = math.Sqrt(sigma2*sigma2 + ts.Tau*ts.Tau)
    
    return Rating{Mu: mu1, Sigma: sigma1}, Rating{Mu: mu2, Sigma: sigma2}
}

// 四人麻将排名更新
func (ts *TrueSkill) UpdateMahjong(ratings []Rating, ranks []int) []Rating {
    n := len(ratings)
    if n != 4 {
        panic("Mahjong requires 4 players")
    }
    
    // 转换排名：1位最好，4位最差
    // 计算每个玩家的表现后验
    
    // 使用近似算法：对每个玩家，与所有对手进行成对更新
    newRatings := make([]Rating, n)
    copy(newRatings, ratings)
    
    // 多次迭代以获得稳定解
    for iter := 0; iter < 10; iter++ {
        for i := 0; i < n; i++ {
            for j := 0; j < n; j++ {
                if i == j { continue }
                
                // 根据排名决定胜负
                var score float64
                if ranks[i] < ranks[j] {
                    score = 1.0  // i 胜
                } else if ranks[i] > ranks[j] {
                    score = 0.0  // i 负
                } else {
                    score = 0.5  // 平局
                }
                
                // 成对更新
                newI, newJ := ts.Update1v1(newRatings[i], newRatings[j], score)
                newRatings[i] = newI
                newRatings[j] = newJ
            }
        }
    }
    
    return newRatings
}

// v 函数
func (ts *TrueSkill) v(t, score float64) float64 {
    if score == 0.5 {
        // 平局
        x := t
        return ts.vDraw(x)
    }
    
    // 胜负
    x := t
    if score == 1.0 {
        return ts.vWin(x)
    } else {
        return ts.vWin(-x)
    }
}

// w 函数
func (ts *TrueSkill) w(t, score float64) float64 {
    if score == 0.5 {
        x := t
        return ts.wDraw(x)
    }
    
    x := t
    if score == 1.0 {
        return ts.wWin(x)
    } else {
        return ts.wWin(-x)
    }
}

// 辅助函数
func (ts *TrueSkill) vWin(x float64) float64 {
    denom := cumulative(x)
    if denom < 1e-5 {
        return -x
    }
    return pdf(x) / denom
}

func (ts *TrueSkill) wWin(x float64) float64 {
    denom := cumulative(x)
    if denom < 1e-5 {
        return 1.0
    }
    v := ts.vWin(x)
    return v * (v + x)
}

func (ts *TrueSkill) vDraw(x float64) float64 {
    sqrt2 := math.Sqrt(2)
    a := math.Exp((sqrt2*ts.Beta)*(sqrt2*ts.Beta)/2)
    denom := cumulative(x) - cumulative(x - sqrt2*ts.Beta)
    if denom < 1e-5 {
        return -x
    }
    return (pdf(x) - pdf(x - sqrt2*ts.Beta)) / denom
}

func (ts *TrueSkill) wDraw(x float64) float64 {
    sqrt2 := math.Sqrt(2)
    denom := cumulative(x) - cumulative(x - sqrt2*ts.Beta)
    if denom < 1e-5 {
        return 1.0
    }
    v := ts.vDraw(x)
    w := v*v + (x*pdf(x) - (x - sqrt2*ts.Beta)*pdf(x - sqrt2*ts.Beta))/denom
    return w
}

// 正态分布PDF
func pdf(x float64) float64 {
    return math.Exp(-x*x/2) / math.Sqrt(2*math.Pi)
}

// 正态分布CDF
func cumulative(x float64) float64 {
    return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}
```

### 2. 与麻将系统的集成

```
type MahjongTrueSkillSystem struct {
    ts        *TrueSkill
    players   map[string]*PlayerTrueSkill
    minSigma  float64 // 最小不确定性
    activityDecay float64 // 活跃度衰减
}

func (m *MahjongTrueSkillSystem) ProcessGame(game *GameResult) {
    // 获取玩家评分
    ratings := make([]Rating, 4)
    playerIDs := make([]string, 4)
    
    for i, player := range game.Players {
        ratings[i] = m.GetRating(player.ID)
        playerIDs[i] = player.ID
    }
    
    // 计算排名
    ranks := m.calculateRanks(game.FinalPoints)
    
    // 更新 TrueSkill
    newRatings := m.ts.UpdateMahjong(ratings, ranks)
    
    // 保存更新
    for i, id := range playerIDs {
        m.players[id].Rating = newRatings[i]
        m.players[id].Games++
        m.players[id].LastPlay = time.Now()
    }
}

// 计算技能值（暴露给用户的分数）
func (m *MahjongTrueSkillSystem) GetDisplayRating(playerID string) float64 {
    player := m.players[playerID]
    
    // TrueSkill 暴露分 = μ - 3σ
    // 表示"保守估计的最低技能"
    return player.Rating.Mu - 3*player.Rating.Sigma
}
```

## 十、TrueSkill 参数调优

### 1. 麻将专用参数建议

```
type MahjongConfig struct {
    // 初始值
    InitialMu    float64  // 25.0
    InitialSigma float64  // 8.333
    
    // 游戏特性
    Beta         float64  // 4.1667 (表现方差)
    Tau          float64  // 0.0833 (动态因子)
    
    // 收敛控制
    MinSigma     float64  // 2.0 (最小不确定性)
    DrawMargin   float64  // 0.0 (麻将几乎不平局)
}
```

### 2. 参数影响

- 

  **Beta 增大**：单局影响变小，收敛变慢

- 

  **Tau 增大**：长期不玩衰减更快

- 

  **MinSigma 增大**：系统更"健忘"

## 十一、TrueSkill 的优势与局限

### 优势

1. 

   **概率建模**：天然处理不确定性

2. 

   **快速收敛**：新手快速定级

3. 

   **团队支持**：原生多人游戏

4. 

   **动态调整**：适应玩家水平变化

5. 

   **匹配质量**：可计算预期比赛质量

### 局限

1. 

   **计算复杂**：比 Elo 复杂

2. 

   **参数敏感**：需要精细调优

3. 

   **理解成本**：用户难理解 (μ, σ)

4. 

   **实现难度**：需要数学库支持

## 十二、麻将场景推荐方案

### 方案1：纯 TrueSkill

```
// 完全使用 TrueSkill
// 优点：理论完善，处理不确定性
// 缺点：实现复杂，用户难懂
```

### 方案2：混合系统

```
// 前端显示 Elo 风格分
// 后端用 TrueSkill 计算
DisplayRating = μ - 3σ
```

### 方案3：TrueSkill 只用于匹配

```
// 段位用传统 Elo
// 匹配用 TrueSkill 计算平衡性
```

## 十三、与其他系统的对比

| 系统          | 麻将适用性 | 实现难度 | 用户体验     |
| ------------- | ---------- | -------- | ------------ |
| 天凤R值       | 最佳       | 中等     | 熟悉         |
| Elo 变体      | 良好       | 简单     | 易理解       |
| Glicko-2      | 良好       | 中等     | 中等         |
| **TrueSkill** | **优秀**   | **复杂** | **需要解释** |

## 十四、推荐选择

对于您的麻将服务器：

### 阶段1：启动阶段

```
使用天凤R值变体
- 实现简单
- 社区熟悉
- 已验证有效
```

### 阶段2：增长阶段

```
考虑 Glicko-2
- 处理不确定性
- 比 TrueSkill 简单
- 仍是单参数
```

### 阶段3：成熟阶段

```
可考虑 TrueSkill
- 需要团队功能时
- 用户量大时
- 有足够开发资源时
```

## 十五、TrueSkill 开源库

Go 语言实现：

1. 

   **github.com/maopen/trueskill-go**

2. 

   **github.com/smhg/true-skill**

Python 参考实现：

```
# 最成熟的实现
import trueskill
```

## 总结

**TrueSkill 是强大的现代评分系统**，特别适合：

1. 

   多人排名游戏（如麻将）

2. 

   需要处理不确定性的场景

3. 

   团队竞技

4. 

   高质量匹配

**但对于麻将服务器**，考虑到：

1. 

   社区习惯天凤R值

2. 

   实现复杂性

3. 

   用户理解成本

**建议**：从**天凤R值系统**开始，当需要更高级功能时再考虑 TrueSkill。如果要使用 TrueSkill，建议采用**混合系统**：后端 TrueSkill 计算，前端显示简化分。

TrueSkill 最大的价值在于其**概率建模思想**，即使不完整实现，也可以借鉴其理念来改进传统评分系统。