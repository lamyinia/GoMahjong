#pragma once

#include <cstdint>
#include <memory>
#include <vector>

#include <grpcpp/server.h>

namespace infra::rpc {

    // grpc 服务端封装
    // 提供服务注册、启动、停止
    class GrpcServer {
    public:
        explicit GrpcServer(std::uint16_t port);

        ~GrpcServer();

        GrpcServer(const GrpcServer &) = delete;

        GrpcServer &operator=(const GrpcServer &) = delete;

        // 注册服务
        // service: grpc 服务实现
        void register_service(std::shared_ptr<grpc::Service> service);

        // 启动服务
        // blocking: 是否阻塞运行（默认非阻塞，后台线程运行）
        // 返回是否启动成功
        bool start(bool blocking = false);

        // 停止服务
        void stop();

        // 是否正在运行
        [[nodiscard]] bool is_running() const noexcept;

        // 等待服务结束（阻塞）
        void wait();

    private:
        std::uint16_t port_;
        std::vector<std::shared_ptr<grpc::Service>> services_;
        std::unique_ptr<grpc::Server> server_;
        bool running_{false};
    };

} // namespace infra::rpc
