#include <boost/asio.hpp>
#include <iostream>
#include <vector>
#include <thread>
#include <chrono>
#include <cassert>

#include "infrastructure/config/config.hpp"

using boost::asio::ip::tcp;

/*
    测试 ServerHub TcpListener 端口连接
    从 config/dev/application.json 加载端口配置（默认 8010）
    假设服务端已启动，测试客户端连接和 echo 功能
    
    1.test_connect - 测试连接到服务端端口
    2.test_echo - 测试 echo 功能
    3.test_multiple_connections - 测试多客户端并发连接
    4.test_large_data - 测试大数据传输
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

// Test 1: Connect to server port
void test_connect(uint16_t port) {
    std::cout << "[TEST] test_connect..." << std::endl;

    boost::asio::io_context ioc;
    TestClient client(ioc, port);
    assert(client.connect());
    std::cout << "[TEST] Successfully connected to port " << port << std::endl;

    client.close();

    std::cout << "[TEST] test_connect PASSED" << std::endl;
}

// Test 2: Echo functionality
void test_echo(uint16_t port) {
    std::cout << "[TEST] test_echo..." << std::endl;

    boost::asio::io_context ioc;
    TestClient client(ioc, port);
    assert(client.connect());

    std::string test_msg = "hello tcp server";
    assert(client.send(test_msg));

    std::string response = client.receive(test_msg.size());
    assert(response == test_msg);
    std::cout << "[TEST] Echo response: " << response << std::endl;

    client.close();

    std::cout << "[TEST] test_echo PASSED" << std::endl;
}

// Test 3: Multiple connections
void test_multiple_connections(uint16_t port) {
    std::cout << "[TEST] test_multiple_connections..." << std::endl;

    boost::asio::io_context ioc;

    // Create 5 clients
    std::vector<std::unique_ptr<TestClient>> clients;
    for (int i = 0; i < 5; i++) {
        auto client = std::make_unique<TestClient>(ioc, port);
        assert(client->connect());
        clients.push_back(std::move(client));
    }

    std::this_thread::sleep_for(std::chrono::milliseconds(200));

    // Each client sends a message and receives echo
    for (int i = 0; i < 5; i++) {
        std::string msg = "msg_" + std::to_string(i);
        assert(clients[i]->send(msg));
        std::string resp = clients[i]->receive(msg.size());
        assert(resp == msg);
        std::cout << "[TEST] Client " << i << " echo: " << resp << std::endl;
    }

    // Close all clients
    for (auto &client: clients) {
        client->close();
    }

    std::cout << "[TEST] test_multiple_connections PASSED" << std::endl;
}

// Test 4: Large data echo
void test_large_data(uint16_t port) {
    std::cout << "[TEST] test_large_data..." << std::endl;

    boost::asio::io_context ioc;
    TestClient client(ioc, port);
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
    std::cout << "[TEST] Received " << received.size() << " bytes, matches sent data" << std::endl;

    client.close();

    std::cout << "[TEST] test_large_data PASSED" << std::endl;
}

int main() {
    std::cout << "========================================" << std::endl;
    std::cout << "ServerHub TcpListener Port Tests" << std::endl;
    std::cout << "========================================" << std::endl;

    // Load config from application.json
    const auto cfg = infra::config::Config::load_from_file("../config/dev/application.json");
    uint16_t port = cfg.server().net.tcp.port;
    std::cout << "[TEST] Testing port: " << port << std::endl;

    test_connect(port);
    test_echo(port);
    test_multiple_connections(port);
    test_large_data(port);

    std::cout << "========================================" << std::endl;
    std::cout << "All tests PASSED!" << std::endl;
    std::cout << "========================================" << std::endl;

    return 0;
}
