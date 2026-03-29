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
    std::cout << "[TEST] test_listener_lifecycle..." << std::endl;

    boost::asio::io_context ioc;
    auto work = boost::asio::make_work_guard(ioc);

    std::thread io_thread([&ioc] { ioc.run(); });

    TcpListener listener(ioc, tcp::endpoint(tcp::v4(), 17000));

    std::atomic<int> accept_count{0};
    listener.start(
            [&accept_count](std::shared_ptr<ITransport> transport) {
                accept_count++;
                transport->close();
            },
            [](const std::error_code &ec) {
                std::cerr << "[TEST] accept error: " << ec.message() << std::endl;
            });

    // Give listener time to start
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    // Verify listener is started
    TestClient client(ioc, 17000);
    assert(client.connect());
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    assert(accept_count == 1);

    // Stop listener
    listener.stop();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    // Verify no more accepts after stop
    TestClient client2(ioc, 17000);
    bool connected = client2.connect(); // May fail or succeed but not accepted
    std::this_thread::sleep_for(std::chrono::milliseconds(100));
    assert(accept_count == 1); // Should still be 1

    work.reset();
    ioc.stop();
    io_thread.join();

    std::cout << "[TEST] test_listener_lifecycle PASSED" << std::endl;
}

// Test 2: Echo functionality
void test_echo() {
    std::cout << "[TEST] test_echo..." << std::endl;

    boost::asio::io_context ioc;
    auto work = boost::asio::make_work_guard(ioc);

    std::thread io_thread([&ioc] { ioc.run(); });

    TcpListener listener(ioc, tcp::endpoint(tcp::v4(), 17001));

    std::atomic<bool> received{false};
    std::atomic<bool> closed{false};

    listener.start(
            [&received, &closed](std::shared_ptr<ITransport> transport) {
                transport->start(
                        [&received, transport](ITransport::Bytes &&data) {
                            transport->send(std::move(data));
                            received = true;
                        },
                        [&closed] {
                            closed = true;
                        },
                        [](const std::error_code &ec) {
                            std::cerr << "[TEST] transport error: " << ec.message() << std::endl;
                        });
            },
            [](const std::error_code &ec) {
                std::cerr << "[TEST] accept error: " << ec.message() << std::endl;
            });

    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    TestClient client(ioc, 17001);
    assert(client.connect());

    std::string test_msg = "hello tcp server";
    assert(client.send(test_msg));

    std::string response = client.receive(test_msg.size());
    assert(response == test_msg);

    client.close();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    listener.stop();

    work.reset();
    ioc.stop();
    io_thread.join();

    std::cout << "[TEST] test_echo PASSED" << std::endl;
}

// Test 3: Multiple connections
void test_multiple_connections() {
    std::cout << "[TEST] test_multiple_connections..." << std::endl;

    boost::asio::io_context ioc;
    auto work = boost::asio::make_work_guard(ioc);

    std::thread io_thread([&ioc] { ioc.run(); });

    TcpListener listener(ioc, tcp::endpoint(tcp::v4(), 17002));

    std::atomic<int> connection_count{0};
    std::atomic<int> close_count{0};

    listener.start(
            [&connection_count, &close_count](std::shared_ptr<ITransport> transport) {
                connection_count++;
                transport->start(
                        [transport](ITransport::Bytes &&data) {
                            transport->send(std::move(data));
                        },
                        [&close_count] {
                            close_count++;
                        },
                        [](const std::error_code &ec) {
                            std::cerr << "[TEST] transport error: " << ec.message() << std::endl;
                        });
            },
            [](const std::error_code &ec) {
                std::cerr << "[TEST] accept error: " << ec.message() << std::endl;
            });

    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    // Create 5 clients
    std::vector<std::unique_ptr<TestClient>> clients;
    for (int i = 0; i < 5; i++) {
        auto client = std::make_unique<TestClient>(ioc, 17002);
        assert(client->connect());
        clients.push_back(std::move(client));
    }

    std::this_thread::sleep_for(std::chrono::milliseconds(200));
    assert(connection_count == 5);

    // Each client sends a message
    for (int i = 0; i < 5; i++) {
        std::string msg = "msg_" + std::to_string(i);
        assert(clients[i]->send(msg));
        std::string resp = clients[i]->receive(msg.size());
        assert(resp == msg);
    }

    // Close all clients
    for (auto &client: clients) {
        client->close();
    }

    std::this_thread::sleep_for(std::chrono::milliseconds(200));
    assert(close_count == 5);

    listener.stop();

    work.reset();
    ioc.stop();
    io_thread.join();

    std::cout << "[TEST] test_multiple_connections PASSED" << std::endl;
}

// Test 4: Large data echo
void test_large_data() {
    std::cout << "[TEST] test_large_data..." << std::endl;

    boost::asio::io_context ioc;
    auto work = boost::asio::make_work_guard(ioc);

    std::thread io_thread([&ioc] { ioc.run(); });

    TcpListener listener(ioc, tcp::endpoint(tcp::v4(), 17003));

    listener.start(
            [](std::shared_ptr<ITransport> transport) {
                transport->start(
                        [transport](ITransport::Bytes &&data) {
                            transport->send(std::move(data));
                        },
                        [] {},
                        [](const std::error_code &ec) {
                            std::cerr << "[TEST] transport error: " << ec.message() << std::endl;
                        });
            },
            [](const std::error_code &ec) {
                std::cerr << "[TEST] accept error: " << ec.message() << std::endl;
            });

    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    TestClient client(ioc, 17003);
    assert(client.connect());

    // Send 4KB data
    std::string large_data(4096, 'A');
    for (size_t i = 0; i < large_data.size(); i++) {
        large_data[i] = static_cast<char>('A' + (i % 26));
    }

    assert(client.send(large_data));

    // Receive in chunks
    std::string received;
    while (received.size() < large_data.size()) {
        size_t remaining = large_data.size() - received.size();
        size_t chunk_size = std::min(remaining, static_cast<size_t>(1024));
        std::string chunk = client.receive(chunk_size);
        if (chunk.empty()) break;
        received += chunk;
    }

    assert(received == large_data);

    client.close();
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    listener.stop();

    work.reset();
    ioc.stop();
    io_thread.join();

    std::cout << "[TEST] test_large_data PASSED" << std::endl;
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
