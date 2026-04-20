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
        explicit GameServiceImpl(domain::game::room::RoomManager &room_manager) : room_manager_(room_manager) {}

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

            // 转换 players（只取 userId）
            std::vector<std::string> players;
            for (const auto &[player_id, _]: request->players()) {
                players.push_back(player_id);
            }

            // 调用 RoomManager 创建房间
            auto roomId = room_manager_.create_room(players, request->engine_type());
            if (roomId.empty()) {
                response->set_success(false);
                response->set_message("创建房间失败：玩家数量不正确或玩家已在其他房间");
                return grpc::Status::OK;
            }

            LOG_INFO("创建房间成功: {}, 玩家数: {}", roomId, players.size());

            response->set_success(true);
            response->set_room_id(roomId);
            response->set_message("房间创建成功");
            return grpc::Status::OK;
        }

    private:
        domain::game::room::RoomManager &room_manager_;
    };

    class GameService::Impl {
    public:
        domain::game::room::RoomManager* room_manager{nullptr};
        std::shared_ptr<GameServiceImpl> grpc_service;

        void init(domain::game::room::RoomManager& rm) {
            room_manager = &rm;
            grpc_service = std::make_shared<GameServiceImpl>(rm);
        }
    };

    GameService& GameService::instance() {
        static GameService instance;
        return instance;
    }

    GameService::GameService(): impl_(std::make_unique<Impl>()) {
        LOG_INFO("单例实例已创建");
    }

    GameService::~GameService() = default;

    void GameService::init(domain::game::room::RoomManager& room_manager) {
        impl_->init(room_manager);
        LOG_INFO("initialized with external RoomManager");
    }

    CreateRoomResponse GameService::create_room(const CreateRoomRequest &request) {
        if (!impl_->room_manager) {
            CreateRoomResponse response;
            response.success = false;
            response.message = "RoomManager not initialized";
            return response;
        }

        std::vector<std::string> players;
        players.reserve(request.players.size());
        for (const auto &[user_id, _] : request.players) {
            players.push_back(user_id);
        }

        auto roomId = impl_->room_manager->create_room(players, request.engine_type);

        CreateRoomResponse response;
        if (!roomId.empty()) {
            response.success = true;
            response.room_id = roomId;
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

    domain::game::room::RoomManager& GameService::room_manager() {
        if (!impl_->room_manager) {
            static domain::game::room::RoomManager dummy;
            LOG_ERROR("room_manager() called before init()");
            return dummy;
        }
        return *impl_->room_manager;
    }

} // namespace domain::game::service
