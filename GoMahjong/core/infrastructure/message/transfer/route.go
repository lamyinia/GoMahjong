package transfer

const MatchingSuccess = "matching.success"
const JoinQueue = "connector.joinqueue"

const GamePush = "game.push"
const DispatchWaitMain = "gameplay.operations.main"
const DispatchWaitReaction = "gameplay.operations.reaction"

// 游戏推送路由
const GameplayRoundStart = "gameplay.round.start"   // 回合开始
const GameplayDraw = "gameplay.draw"                // 摸牌
const GameplayDiscard = "gameplay.discard"          // 出牌
const GameplayRiichi = "gameplay.riichi"            // 立直
const GameplayChi = "gameplay.chi"                  // 吃牌
const GameplayPeng = "gameplay.peng"                // 碰牌
const GameplayGang = "gameplay.gang"                // 明杠
const GameplayAnkan = "gameplay.ankan"              // 暗杠
const GameplayKakan = "gameplay.kakan"              // 加杠
const GameplayRon = "gameplay.ron"                  // 荣和
const GameplayTsumo = "gameplay.tsumo"              // 自摸
const GameplayRoundEnd = "gameplay.round.end"       // 回合结束
const GameplayGameEnd = "gameplay.game.end"         // 游戏结束
const GameplayStateUpdate = "gameplay.state.update" // 状态更新
