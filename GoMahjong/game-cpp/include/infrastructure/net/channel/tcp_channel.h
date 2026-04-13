#pragma once

#include "i_channel.h"
#include "channel_pipeline.h"
#include "infrastructure/net/transport/i_transport.h"

#include <atomic>
#include <memory>
#include <string>
#include <boost/asio.hpp>
#include <boost/asio/strand.hpp>

namespace infra::net::channel {
    namespace transport = infra::net::transport;

    /**
     * @brief TCP Channel 实现
     * 
     * 封装 TcpTransport，提供：
     * - Handler 链管理（Pipeline）
     * - 进站/出站事件传播
     * - 统一的读写接口
     */
    class TcpChannel : public IChannel {
    public:
        using Strand = boost::asio::strand<boost::asio::any_io_executor>;

        explicit TcpChannel(
            std::shared_ptr<transport::ITransport> transport,
            const Strand& strand
        );

        ~TcpChannel() override;

        // === IChannel 接口实现 ===

        ChannelType type() const noexcept override { return ChannelType::Tcp; }
        std::string id() const override { return id_; }
        bool is_active() const override { return active_; }

        // Handler 管理
        void add_inbound(std::shared_ptr<ChannelInboundHandler> handler) override;
        void add_outbound(std::shared_ptr<ChannelOutboundHandler> handler) override;
        void add_duplex(std::shared_ptr<ChannelDuplexHandler> handler) override;
        ChannelPipeline& pipeline() override { return pipeline_; }

        // 出站事件入口
        void send(Bytes&& data) override;
        void flush() override;
        void close() override;

        // 进站事件入口
        void start_read() override;

        // Transport 操作（由 Pipeline 调用）
        void transport_write(Bytes&& data) override;
        void transport_flush() override;
        void transport_close() override;

        // 回调
        void set_on_error(OnError on_error) override;
        void set_on_inactive(OnInactive on_inactive) override;

    private:
        void on_transport_bytes(Bytes&& data);
        void on_transport_closed();
        void on_transport_error(const std::error_code& ec);

        static std::string generate_id();

    private:
        std::shared_ptr<transport::ITransport> transport_;
        ChannelPipeline pipeline_;
        Strand strand_;
        std::string id_;
        std::atomic<bool> active_{false};
        std::atomic<bool> reading_{false};
        OnError on_error_;
        OnInactive on_inactive_;
    };

} // namespace infra::net::channel
