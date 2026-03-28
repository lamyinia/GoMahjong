#include "app.hpp"

#include <boost/asio.hpp>

#include <google/protobuf/stubs/common.h>

#include <csignal>
#include <cstdlib>
#include <iostream>
#include <thread>
#include <vector>

#include "infrastructure/config/config.hpp"
#include "infrastructure/log/logger.hpp"
#include "infrastructure/net/tcp_server.hpp"

namespace gomahjong::bootstrap {

    int run() {
        GOOGLE_PROTOBUF_VERIFY_VERSION;

        infra::log::init();

        try {
            boost::asio::io_context ioc;

            const auto cfg = infra::config::Config::load_from_file_or_default("config/dev/application.json");
            const auto& s = cfg.server();

            const std::size_t max_accumulated_bytes =
                std::max<std::size_t>(256 * 1024, static_cast<std::size_t>(s.max_frame_bytes) * 4);


            boost::asio::signal_set signals(ioc, SIGINT, SIGTERM);
            signals.async_wait([&](const boost::system::error_code &, int) {
                std::cout << "[app] signal received, stopping\n";
                ioc.stop();
            });
            
            // 根据硬件并发能力计算工作线程数，至少2个。
            unsigned int threads = std::max(2u, std::thread::hardware_concurrency() > 0 ? std::thread::hardware_concurrency() / 2 : 2u);
            std::cerr << "threads: " << threads << "\n";
            std::vector<std::thread> workers;

            // 预分配工作线程容器。
            workers.reserve(threads);

            // 创建并启动工作线程。
            for (unsigned int i = 0; i < threads; ++i) {
                workers.emplace_back([&] { ioc.run(); });
            }

            // 主线程等待所有工作线程退出。
            for (auto &t: workers) {
                t.join();
            }

            google::protobuf::ShutdownProtobufLibrary();
            return EXIT_SUCCESS;
        } catch (const std::exception &e) {
            std::cerr << "[app] fatal: " << e.what() << "\n";
            google::protobuf::ShutdownProtobufLibrary();
            return EXIT_FAILURE;
        }
    }

} // namespace gomahjong::bootstrap
