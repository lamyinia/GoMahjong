#include "infrastructure/rpc/grpc_server.hpp"

#include <grpcpp/server_builder.h>

#include "infrastructure/log/logger.hpp"

namespace infra::rpc {

    GrpcServer::GrpcServer(std::uint16_t port) : port_(port) {
    }

    GrpcServer::~GrpcServer() {
        stop();
    }

    void GrpcServer::register_service(std::shared_ptr<grpc::Service> service) {
        services_.push_back(std::move(service));
    }

    bool GrpcServer::start(bool blocking) {
        if (running_) {
            return true;
        }

        grpc::ServerBuilder builder;

        // 绑定端口
        std::string server_address = std::string("0.0.0.0:") + std::to_string(port_);
        builder.AddListeningPort(server_address, grpc::InsecureServerCredentials());

        // 注册所有服务
        for (const auto &service: services_) {
            builder.RegisterService(service.get());
        }

        // 构建服务器
        server_ = builder.BuildAndStart();
        if (!server_) {
            LOG_ERROR("[GrpcServer] 启动失败，端口: {}", port_);
            return false;
        }

        running_ = true;
        LOG_INFO("[GrpcServer] 启动成功，监听端口: {}", port_);

        if (blocking) {
            server_->Wait();
        }

        return true;
    }

    void GrpcServer::stop() {
        if (!running_ || !server_) {
            return;
        }

        LOG_INFO("[GrpcServer] 正在停止...");
        server_->Shutdown();
        server_.reset();
        running_ = false;
        LOG_INFO("[GrpcServer] 已停止");
    }

    bool GrpcServer::is_running() const noexcept {
        return running_;
    }

    void GrpcServer::wait() {
        if (server_) {
            server_->Wait();
        }
    }

} // namespace infra::rpc
