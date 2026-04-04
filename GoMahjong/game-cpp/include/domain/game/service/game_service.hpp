#pragma once

#include <cstdint>
#include <map>
#include <memory>
#include <string>

// 前向声明
namespace grpc {
    class Service;
}

namespace domain::game::room {
    class RoomManager;
}

namespace domain::game::engine {
    class GameFrame;
}

namespace domain::game::service {

    // 创建房间请求
    struct CreateRoomRequest {
        std::map<std::string, std::string> players; // userID -> connectorTopic
        std::int32_t engine_type{};
    };

    // 创建房间响应
    struct CreateRoomResponse {
        bool success{};
        std::string room_id;
        std::string message;
    };

    // 游戏服务接口
    class IGameService {
    public:
        virtual ~IGameService() = default;

        virtual CreateRoomResponse create_room(const CreateRoomRequest &request) = 0;
    };

    // 游戏服务实现（单例模式）
    class GameService : public IGameService {
    public:
        // 获取单例实例
        static GameService& instance();

        ~GameService() override;

        GameService(const GameService &) = delete;
        GameService &operator=(const GameService &) = delete;

        CreateRoomResponse create_room(const CreateRoomRequest &request) override;

        // 获取 grpc 服务实现（用于注册到 GrpcServer）
        [[nodiscard]] std::shared_ptr<grpc::Service> get_grpc_service() const;

        // 获取 RoomManager（供 Handler 使用）
        domain::game::room::RoomManager& room_manager();

    private:
        GameService();  // 私有构造函数
        class Impl;
        std::unique_ptr<Impl> impl_;
    };

} // namespace domain::game::service
