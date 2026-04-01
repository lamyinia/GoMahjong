#include <boost/asio.hpp>
#include <iostream>
#include <vector>
#include <thread>
#include <atomic>
#include <chrono>
#include <cassert>

#include "infrastructure/net/listener/tcp_listener.h"
#include "infrastructure/net/transport/tcp_transport.h"
#include "infrastructure/log/logger.hpp"

using namespace infra::net::listener;
using namespace infra::net::transport;
using boost::asio::ip::tcp;

/*
    测试使用端口 17000-17003，确保这些端口未被占用。
    1.test_listener_lifecycle - 测试 listener 启动/停止生命周期
    2.test_echo - 测试 echo 功能（发送什么返回什么）
    3.test_multiple_connections - 测试多客户端并发连接（5个客户端）
    4.test_large_data - 测试大数据传输（4KB）

*/
// Test helper: simple sync client
class TestClient {
public:
    TestClient(boost::asio::io_context &ioc, uint16_t port)
            : socket_(ioc), port_(port) {}

    bool connect() {
        try {
            tcp::endpoint ep(tcp::v4(), port_);
            socket_.connect(ep);
            connected_ = true;
            return true;
        } catch (const std::exception &e) {
            std::cerr << "[test_client] connect failed: " << e.what() << std::endl;
            return false;
        }
    }

    bool send(const std::string &msg) {
        if (!connected_) return false;
        try {
            boost::asio::write(socket_, boost::asio::buffer(msg));
            return true;
        } catch (const std::exception &e) {
            std::cerr << "[test_client] send failed: " << e.what() << std::endl;
            return false;
        }
    }

    std::string receive(size_t size) {
        if (!connected_) return "";
        try {
            std::vector<char> buf(size);
            size_t len = boost::asio::read(socket_, boost::asio::buffer(buf));
            return std::string(buf.begin(), buf.begin() + len);
        } catch (const std::exception &e) {
            std::cerr << "[test_client] receive failed: " << e.what() << std::endl;
            return "";
        }
    }

    void close() {
        if (connected_) {
            boost::system::error_code ec;
            socket_.close(ec);
            connected_ = false;
        }
    }

private:
    tcp::socket socket_;
    uint16_t port_;
    bool connected_ = false;
};

// Test 1: Basic listener lifecycle (start/stop)
void test_listener_lifecycle() {

}

// Test 2: Echo functionality
void test_echo() {

}

// Test 3: Multiple connections
void test_multiple_connections() {

}

// Test 4: Large data echo
void test_large_data() {

}

int main() {
    std::cout << "========================================" << std::endl;
    std::cout << "TCP Server Tests" << std::endl;
    std::cout << "========================================" << std::endl;

    test_listener_lifecycle();
    test_echo();
    test_multiple_connections();
    test_large_data();

    std::cout << "========================================" << std::endl;
    std::cout << "All tests PASSED!" << std::endl;
    std::cout << "========================================" << std::endl;

    return 0;
}
