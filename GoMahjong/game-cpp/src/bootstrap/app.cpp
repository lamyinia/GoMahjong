
#include <boost/asio.hpp>

#include <google/protobuf/stubs/common.h>

#include <cstdlib>
#include <iostream>
#include <vector>

#include "bootstrap/app.hpp"
#include "infrastructure/config/config.hpp"
#include "infrastructure/log/logger.hpp"
#include "bootstrap/server_hub.h"

namespace gomahjong::bootstrap {

    int run() {
        GOOGLE_PROTOBUF_VERIFY_VERSION;

        try {
            const auto cfg = infra::config::Config::load_from_file_or_default("../config/dev/application.json");
            infra::log::init();
            ServerHub hub(cfg);

            boost::asio::signal_set signals(hub.ioc(), SIGINT, SIGTERM);
            signals.async_wait([&](const boost::system::error_code &, int) {
                LOG_INFO("[app] signal received, stopping\n");
                std::cout << "[app] signal received, stopping\n";
                // 不要直接 hub.stop()，否则在 ioc 线程里 join 线程可能死锁）
                hub.ioc().stop();
            });

            // 开始组装逻辑
            hub.start();
            hub.ioc().run();
            hub.stop();

            google::protobuf::ShutdownProtobufLibrary();
            return EXIT_SUCCESS;
        } catch (const std::exception &e) {
            std::cerr << "[app] fatal: " << e.what() << "\n";
            google::protobuf::ShutdownProtobufLibrary();
            return EXIT_FAILURE;
        }
    }

} // namespace gomahjong::bootstrap
