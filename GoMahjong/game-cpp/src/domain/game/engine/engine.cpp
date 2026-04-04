#include "domain/game/engine/engine.h"
#include "domain/game/engine/mahjong/riichi_mahjong4p_engine.h"

namespace domain::game::engine {

std::unique_ptr<Engine> Engine::create(EngineType type) {
    switch (type) {
        case EngineType::RiichiMahjong4P:
            return std::make_unique<RiichiMahjong4PEngine>();
        // TODO: 实现其他引擎类型
        // case EngineType::RiichiMahjong3P:
        //     return std::make_unique<RiichiMahjong3PEngine>();
        // case EngineType::TexasHoldem:
        //     return std::make_unique<TexasHoldemEngine>();
        // case EngineType::SanZhang:
        //     return std::make_unique<SanZhangEngine>();
        default:
            return nullptr;
    }
}

} // namespace domain::game::engine
