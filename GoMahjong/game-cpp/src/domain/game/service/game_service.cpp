#include "domain/game/service/game_service.hpp"

#include "domain/game/room/room_manager.h"
#include "infrastructure/log/logger.hpp"

#include <grpcpp/grpcpp.h>

// proto 生成的头文件
#include "game_service.grpc.pb.h"

namespace domain::game::service {

    // grpc 服务实现（内部类）
    class GameServiceImpl final : public gomahjong::rpc::GameService::Service {
    public:
        explicit GameServiceImpl(domain::game::room::RoomManager &room_manager) : room_manager_(room_manager) {
        }

        grpc::Status CreateRoom(
                grpc::ServerContext *context,
                const gomahjong::rpc::CreateRoomRequest *request,
                gomahjong::rpc::CreateRoomResponse *response) override {
            (void) context;

            // 验证请求
            if (request->players().empty()) {
                response->set_success(false);
                response->set_message("玩家列表不能为空");
                return grpc::Status::OK;
            }

            // 转换 players
            std::map<std::string, std::string> players;
            for (const auto &[user_id, connector_topic]: request->players()) {
                players[user_id] = connector_topic;
            }

            // 调用 RoomManager 创建房间
            const auto *room = room_manager_.create_room(players, request->engine_type());
            if (!room) {
                response->set_success(false);
                response->set_message("创建房间失败：玩家数量不正确或玩家已在其他房间");
                return grpc::Status::OK;
            }

            LOG_INFO("[GameService] 创建房间成功: {}, 玩家数: {}", room->id, players.size());

            response->set_success(true);
            response->set_room_id(room->id);
            response->set_message("房间创建成功");
            return grpc::Status::OK;
        }

    private:
        domain::game::room::RoomManager &room_manager_;
    };

    // GameService::Impl
    struct GameService::Impl {
        domain::game::room::RoomManager &room_manager;
        std::shared_ptr<GameServiceImpl> grpc_service;

        explicit Impl(domain::game::room::RoomManager &rm) : room_manager(rm) {
            grpc_service = std::make_shared<GameServiceImpl>(room_manager);
        }
    };

    GameService::GameService(domain::game::room::RoomManager &room_manager)
        : impl_(std::make_unique<Impl>(room_manager)) {
    }

    GameService::~GameService() = default;

    CreateRoomResponse GameService::create_room(const CreateRoomRequest &request) {
        const auto *room = impl_->room_manager.create_room(request.players, request.engine_type);

        CreateRoomResponse response;
        if (room) {
            response.success = true;
            response.room_id = room->id;
            response.message = "房间创建成功";
        } else {
            response.success = false;
            response.message = "创建房间失败";
        }
        return response;
    }

    std::shared_ptr<grpc::Service> GameService::get_grpc_service() const {
        return std::static_pointer_cast<grpc::Service>(impl_->grpc_service);
    }

} // namespace domain::game::service
