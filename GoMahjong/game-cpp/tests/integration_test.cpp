#include <boost/asio.hpp>
#include <iostream>
#include <vector>
#include <thread>
#include <atomic>
#include <chrono>
#include <cassert>
#include <memory>
#include <mutex>
#include <optional>
#include <condition_variable>
#include <queue>
#include <cstdlib>

// Protobuf
#include "envelope.pb.h"
#include "auth.pb.h"
#include "game_mahjong.pb.h"

// Infrastructure
#include "infrastructure/net/channel/codec/length_field_encoder.h"
#include "infrastructure/net/channel/i_channel.h"
#include "infrastructure/net/session/session_manager.h"
#include "infrastructure/log/logger.hpp"
#include "infrastructure/config/config.hpp"

// Domain
#include "domain/game/room/room_manager.h"
#include "domain/game/room/room_actor.h"
#include "domain/game/event/game_event.h"

using namespace infra::net::channel;
using namespace infra::net::session;
using namespace domain::game::room;
namespace event = domain::game::event;
using boost::asio::ip::tcp;

static std::uint16_t getTestServerPort() {
    if (const char *v = std::getenv("GOMAHJONG_TCP_PORT")) {
        try {
            const int p = std::stoi(v);
            if (p > 0 && p <= 65535) {
                return static_cast<std::uint16_t>(p);
            }
        } catch (...) {
        }
    }
    return 8010;
}

// === Protobuf Test Client ===

class ProtobufTestClient {
public:
    ProtobufTestClient(boost::asio::io_context &ioc, uint16_t port)
            : socket_(ioc), port_(port), seq_(0) {}

    bool connect() {
        try {
            tcp::endpoint ep(tcp::v4(), port_);
            socket_.connect(ep);
            connected_ = true;
            LOG_INFO("[test_client] connected to port {}", port_);
            return true;
        } catch (const std::exception &e) {
            LOG_ERROR("[test_client] connect failed: {}", e.what());
            return false;
        }
    }

    // Send Protobuf message with Envelope wrapper
    bool sendMessage(const std::string &route, const google::protobuf::Message &msg) {
        if (!connected_) return false;

        try {
            // 1. Serialize payload
            std::string payload = msg.SerializeAsString();

            // 2. Create Envelope
            gomahjong::net::Envelope envelope;
            envelope.set_route(route);
            envelope.set_payload(payload);
            envelope.set_client_seq(++seq_);

            // 3. Serialize Envelope
            std::string envelopeData = envelope.SerializeAsString();

            // 4. Add length header (4 bytes, big endian)
            uint32_t len = htonl(static_cast<uint32_t>(envelopeData.size()));
            std::vector<uint8_t> buffer;
            buffer.reserve(4 + envelopeData.size());

            const uint8_t *lenBytes = reinterpret_cast<const uint8_t *>(&len);
            buffer.insert(buffer.end(), lenBytes, lenBytes + 4);
            buffer.insert(buffer.end(), envelopeData.begin(), envelopeData.end());

            // 5. Send
            boost::asio::write(socket_, boost::asio::buffer(buffer));

            // 6. Check if connection is still valid (peer may have closed)
            boost::system::error_code ec;
            char peek_buf[1];
            size_t peeked = socket_.receive(boost::asio::buffer(peek_buf),
                                            boost::asio::socket_base::message_peek, ec);
            if (ec == boost::asio::error::eof) {
                // Server closed connection
                connected_ = false;
                LOG_INFO("[test_client] server closed connection");
                return false;
            }

            LOG_DEBUG("[test_client] sent message, route={}, size={}", route, buffer.size());
            return true;
        } catch (const std::exception &e) {
            LOG_ERROR("[test_client] send failed: {}", e.what());
            return false;
        }
    }

    // Receive Protobuf message
    template<typename T>
    std::optional<T> receiveMessage(const std::string &expectedRoute) {
        if (!connected_) return std::nullopt;

        try {
            // 1. Read length header (4 bytes)
            uint32_t len = 0;
            boost::asio::read(socket_, boost::asio::buffer(&len, 4));
            len = ntohl(len);

            if (len > 65536) {  // Max 64KB
                LOG_ERROR("[test_client] message too large: {}", len);
                return std::nullopt;
            }

            // 2. Read payload
            std::vector<uint8_t> buffer(len);
            boost::asio::read(socket_, boost::asio::buffer(buffer));

            // 3. Parse Envelope
            gomahjong::net::Envelope envelope;
            if (!envelope.ParseFromArray(buffer.data(), static_cast<int>(buffer.size()))) {
                LOG_ERROR("[test_client] failed to parse Envelope");
                return std::nullopt;
            }

            LOG_DEBUG("[test_client] received message, route={}", envelope.route());

            if (envelope.route() != expectedRoute) {
                LOG_WARN("[test_client] unexpected route: {} (expected {})",
                         envelope.route(), expectedRoute);
                return std::nullopt;
            }

            // 4. Parse payload
            T msg;
            if (!msg.ParseFromString(envelope.payload())) {
                LOG_ERROR("[test_client] failed to parse payload");
                return std::nullopt;
            }

            return msg;
        } catch (const std::exception &e) {
            LOG_ERROR("[test_client] receive failed: {}", e.what());
            return std::nullopt;
        }
    }

    void close() {
        if (connected_) {
            boost::system::error_code ec;
            socket_.close(ec);
            connected_ = false;
            LOG_INFO("[test_client] closed");
        }
    }

    bool isConnected() const { return connected_; }

private:
    tcp::socket socket_;
    uint16_t port_;
    bool connected_ = false;
    uint64_t seq_;
};

// === Mock Channel for Testing ===

class MockChannel : public IChannel {
public:
    explicit MockChannel(const std::string &id = "mock_001", ChannelType type = ChannelType::Tcp)
            : id_(id), type_(type), active_(true) {}

    // === IChannel 接口 ===

    ChannelType type() const noexcept override { return type_; }

    std::string id() const override { return id_; }

    bool is_active() const override { return active_; }

    void add_inbound(std::shared_ptr<ChannelInboundHandler> handler) override {
        inbound_handlers_.push_back(std::move(handler));
    }

    void add_outbound(std::shared_ptr<ChannelOutboundHandler> handler) override {
        outbound_handlers_.push_back(std::move(handler));
    }

    void add_duplex(std::shared_ptr<ChannelDuplexHandler> handler) override {
        duplex_handlers_.push_back(std::move(handler));
    }

    ChannelPipeline &pipeline() override {
        throw std::runtime_error("MockChannel does not support pipeline");
    }

    void send(Bytes &&data) override {
        std::lock_guard<std::mutex> lock(mutex_);
        write_buffer_ = std::move(data);
    }

    void flush() override {
        std::lock_guard<std::mutex> lock(mutex_);
        written_data_.push(write_buffer_);
        write_buffer_.clear();
        cv_.notify_one();
    }

    void close() override {
        active_ = false;
    }

    void start_read() override {
        // Mock: do nothing
    }

    void transport_write(Bytes &&data) override {
        std::lock_guard<std::mutex> lock(mutex_);
        written_data_.push(std::move(data));
        cv_.notify_one();
    }

    void transport_flush() override {
        // Mock: do nothing
    }

    void transport_close() override {
        active_ = false;
    }

    void set_on_error(OnError on_error) override {
        on_error_ = std::move(on_error);
    }

    // === Mock 辅助方法 ===

    void setId(const std::string &id) { id_ = id; }

    void setActive(bool active) { active_ = active; }

    // 模拟接收数据
    void simulateRead(Bytes &&data) {
        // 触发 inbound handlers
        for (auto &handler: inbound_handlers_) {
            // handler->channel_read(...);
        }
    }

    // 获取写入的数据
    std::optional<Bytes> getWrittenData(std::chrono::milliseconds timeout = std::chrono::milliseconds(100)) {
        std::unique_lock<std::mutex> lock(mutex_);
        if (cv_.wait_for(lock, timeout, [this] { return !written_data_.empty(); })) {
            auto data = std::move(written_data_.front());
            written_data_.pop();
            return data;
        }
        return std::nullopt;
    }

    // 清空写入缓冲区
    void clearWrittenData() {
        std::lock_guard<std::mutex> lock(mutex_);
        while (!written_data_.empty()) {
            written_data_.pop();
        }
    }

private:
    std::string id_;
    ChannelType type_;
    std::atomic<bool> active_;
    OnError on_error_;

    std::vector<std::shared_ptr<ChannelInboundHandler>> inbound_handlers_;
    std::vector<std::shared_ptr<ChannelOutboundHandler>> outbound_handlers_;
    std::vector<std::shared_ptr<ChannelDuplexHandler>> duplex_handlers_;

    std::mutex mutex_;
    std::condition_variable cv_;
    Bytes write_buffer_;
    std::queue<Bytes> written_data_;
};

// === Test Utilities ===

std::string generateTestToken(const std::string &playerId) {
    // Format: "playerId:timestamp"
    auto now = std::chrono::system_clock::now();
    auto timestamp = std::chrono::duration_cast<std::chrono::seconds>(
            now.time_since_epoch()).count();
    return playerId + ":" + std::to_string(timestamp);
}

// 生成唯一 ID
std::string generateUniqueId() {
    static std::atomic<uint64_t> counter{0};
    return "id_" + std::to_string(++counter);
}

// === Integration Tests ===

// Test 1: Authentication flow
void test_authentication_flow() {
    LOG_INFO("=== Test 1: Authentication Flow ===");

    boost::asio::io_context ioc;
    ProtobufTestClient client(ioc, getTestServerPort());

    // 1. Connect
    assert(client.connect());

    // 2. Send auth request
    gomahjong::net::AuthRequest authReq;
    authReq.set_token(generateTestToken("player_001"));
    authReq.set_device_id("test_device");
    authReq.set_version("1.0.0");

    assert(client.sendMessage("auth.login", authReq));

    // 3. Receive auth response
    auto resp = client.receiveMessage<gomahjong::net::AuthResponse>("auth.login.response");
    assert(resp.has_value());
    assert(resp->success());
    assert(resp->pid() == 1);  // player_001 parsed as uint64

    LOG_INFO("[test] authentication success, player_id={}", resp->pid());

    client.close();
    LOG_INFO("[test] Test 1 PASSED\n");
}

// Test 2: Authentication timeout
void test_auth_timeout() {
    LOG_INFO("=== Test 2: Authentication Timeout ===");

    boost::asio::io_context ioc;
    ProtobufTestClient client(ioc, getTestServerPort());

    // 1. Connect but don't send auth
    assert(client.connect());

    // 2. Wait for timeout (should be disconnected)
    LOG_INFO("[test] waiting for auth timeout...");
    std::this_thread::sleep_for(std::chrono::seconds(6));

    // 3. Try to send - should fail
    gomahjong::net::AuthRequest authReq;
    authReq.set_token(generateTestToken("player_002"));

    bool sendResult = client.sendMessage("auth.login", authReq);
    assert(!sendResult || !client.isConnected());  // Either send fails or disconnected

    LOG_INFO("[test] Test 2 PASSED\n");
}

// Test 3: Multiple clients authentication
void test_multiple_clients_auth() {
    LOG_INFO("=== Test 3: Multiple Clients Authentication ===");

    boost::asio::io_context ioc;
    std::vector<std::unique_ptr<ProtobufTestClient>> clients;

    // 1. Create 3000 clients
    for (int i = 0; i < 3000; i++) {
        auto client = std::make_unique<ProtobufTestClient>(ioc, getTestServerPort());
        assert(client->connect());
        clients.push_back(std::move(client));
    }

    // 2. All clients authenticate
    for (int i = 0; i < 3000; i++) {
        std::string playerId = "player_" + std::to_string(100 + i);

        gomahjong::net::AuthRequest authReq;
        authReq.set_token(generateTestToken(playerId));

        assert(clients[i]->sendMessage("auth.login", authReq));

        auto resp = clients[i]->receiveMessage<gomahjong::net::AuthResponse>("auth.login.response");
        assert(resp.has_value());
        assert(resp->success());

        LOG_INFO("[test] client {} authenticated as {}", i, playerId);
    }

    // 3. Close all
    for (auto &client: clients) {
        client->close();
    }

    LOG_INFO("[test] Test 3 PASSED\n");
}

// Test 4: Room creation and game event
void test_room_and_game_event() {
    LOG_INFO("=== Test 4: Room Creation and Game Event ===");

    // This test verifies the RoomManager and ActorPool integration
    // without requiring full server stack

    // 1. Create RoomManager
    RoomManager roomManager(4, 1000);
    roomManager.start();

    // 3. Create a room with 4 players
    std::vector<std::string> players = {"player_100", "player_101", "player_102", "player_103"};
    Room *room = roomManager.create_room(players, 1);  // Engine type 1 = Riichi Mahjong

    assert(room != nullptr);
    assert(!room->getId().empty());
    LOG_INFO("[test] room created, id={}", room->getId());

    // 4. Submit a game event
    auto event = event::GameEvent::playTile("player_100",
                                            event::Tile{event::TileType::Wan1, 0});

    roomManager.submitEvent(room->getId(), std::move(event));

    // 5. Wait for event processing
    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    // 6. Cleanup
    roomManager.stop();

    LOG_INFO("[test] Test 4 PASSED\n");
}

// Test 5: Session management
void test_session_management() {

}

// Test 6: Room lifecycle with sessions
void test_room_lifecycle_with_sessions() {
    LOG_INFO("=== Test 6: Room Lifecycle with Sessions ===");

    // 1. 创建 RoomManager
    RoomManager roomManager(4, 1000);
    roomManager.start();

    // 2. 创建 SessionManager
    SessionManager sessionMgr;

    // 3. 创建 4 个玩家会话
    std::vector<std::string> playerIds = {"player_200", "player_201", "player_202", "player_203"};
    std::vector<std::shared_ptr<Session>> sessions;

    for (const auto &pid: playerIds) {
        auto channel = std::make_shared<MockChannel>("channel_" + pid);
        auto session = sessionMgr.create_or_get_session(pid, channel);
        sessions.push_back(session);
    }

    assert(sessionMgr.size() == 4);
    LOG_INFO("[test] 4 player sessions created");

    // 4. 创建房间
    Room *room = roomManager.create_room(playerIds, 1);
    assert(room != nullptr);
    assert(!room->getId().empty());

    LOG_INFO("[test] room created, id={}", room->getId());

    // 5. 验证玩家-房间映射
    for (const auto &pid: playerIds) {
        auto *foundRoom = roomManager.get_player_room(pid);
        assert(foundRoom == room);
    }

    LOG_INFO("[test] player-room mapping verified");

    // 6. 发送游戏事件
    auto event = event::GameEvent::gameStart(room->getId());
    roomManager.submitEvent(room->getId(), std::move(event));

    std::this_thread::sleep_for(std::chrono::milliseconds(100));

    LOG_INFO("[test] game start event submitted");

    // 7. 清理
    roomManager.stop();

    LOG_INFO("[test] Test 6 PASSED\n");
}

// Test 7: Concurrent room operations
void test_concurrent_room_operations() {
    LOG_INFO("=== Test 7: Concurrent Room Operations ===");

    // 1. 创建 RoomManager
    RoomManager roomManager(8, 2000);
    roomManager.start();

    // 2. 并发创建多个房间
    const int numRooms = 10;
    std::vector<std::thread> threads;
    std::atomic<int> successCount{0};
    std::mutex roomsMutex;
    std::vector<std::string> roomIds;

    for (int i = 0; i < numRooms; i++) {
        threads.emplace_back([&roomManager, &successCount, &roomsMutex, &roomIds, i]() {
            std::vector<std::string> players = {
                    "player_" + std::to_string(i * 4 + 0),
                    "player_" + std::to_string(i * 4 + 1),
                    "player_" + std::to_string(i * 4 + 2),
                    "player_" + std::to_string(i * 4 + 3)
            };

            Room *room = roomManager.create_room(players, 1);
            if (room != nullptr) {
                successCount++;
                std::lock_guard<std::mutex> lock(roomsMutex);
                roomIds.push_back(room->getId());
            }
        });
    }

    // 3. 等待所有线程完成
    for (auto &t: threads) {
        t.join();
    }

    assert(successCount == numRooms);
    LOG_INFO("[test] {} rooms created concurrently", successCount.load());

    // 4. 并发提交事件
    threads.clear();
    for (const auto &roomId: roomIds) {
        threads.emplace_back([&roomManager, roomId]() {
            auto event = event::GameEvent::roundStart(1, "player_0");
            roomManager.submitEvent(roomId, std::move(event));
        });
    }

    for (auto &t: threads) {
        t.join();
    }

    LOG_INFO("[test] events submitted to all rooms");

    // 5. 清理
    roomManager.stop();

    LOG_INFO("[test] Test 7 PASSED\n");
}

// Test 8: Error handling - duplicate player in room
void test_error_duplicate_player() {
    LOG_INFO("=== Test 8: Error Handling - Duplicate Player ===");

    RoomManager roomManager(4, 1000);
    roomManager.start();

    // 1. 创建第一个房间
    std::vector<std::string> players1 = {"p1", "p2", "p3", "p4"};
    Room *room1 = roomManager.create_room(players1, 1);
    assert(room1 != nullptr);

    LOG_INFO("[test] first room created");

    // 2. 尝试用重复玩家创建房间
    std::vector<std::string> players2 = {"p1", "p5", "p6", "p7"};  // p1 已在 room1
    Room *room2 = roomManager.create_room(players2, 1);
    assert(room2 == nullptr);  // 应该失败

    LOG_INFO("[test] duplicate player rejected correctly");

    // 3. 验证原房间不受影响
    assert(roomManager.get_player_room("p1") == room1);

    LOG_INFO("[test] original room unaffected");

    roomManager.stop();

    LOG_INFO("[test] Test 8 PASSED\n");
}

// Test 9: Actor pool stress test
void test_actor_pool_stress() {
    LOG_INFO("=== Test 9: Actor Pool Stress Test ===");

    const int numActors = 4;
    const int numEvents = 1000;

    RoomManager roomManager(numActors, 5000);
    roomManager.start();

    // 1. 创建一个房间
    std::vector<std::string> players = {"s1", "s2", "s3", "s4"};
    Room *room = roomManager.create_room(players, 1);
    assert(room != nullptr);

    LOG_INFO("[test] room created for stress test");

    // 2. 提交大量事件
    auto start = std::chrono::high_resolution_clock::now();

    for (int i = 0; i < numEvents; i++) {
        auto event = event::GameEvent::turnStart("s1", 30);
        roomManager.submitEvent(room->getId(), std::move(event));
    }

    auto end = std::chrono::high_resolution_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::microseconds>(end - start);

    LOG_INFO("[test] {} events submitted in {} us", numEvents, duration.count());
    LOG_INFO("[test] throughput: {} events/sec",
             (numEvents * 1000000.0) / duration.count());

    // 3. 等待处理完成
    std::this_thread::sleep_for(std::chrono::milliseconds(500));

    roomManager.stop();

    LOG_INFO("[test] Test 9 PASSED\n");
}

// === Main ===

int main() {
    // Initialize logger
    infra::log::init(infra::config::LogConfig{"debug"});

    std::cout << "========================================" << std::endl;
    std::cout << "Integration Tests" << std::endl;
    std::cout << "========================================" << std::endl;

    // Note: Tests 1-3 require a running server

    auto start = std::chrono::steady_clock::now();

    test_authentication_flow();      // Requires server
//    test_auth_timeout();              // Requires server
    test_multiple_clients_auth();     // Requires server

    auto end = std::chrono::steady_clock::now();
    auto elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);
    std::cout << "一共用时 " << elapsed.count() << " 毫秒\n";
    /*
     * 有递归锁+3000client 5925ms
     */

//    test_room_and_game_event();         // Standalone
//    test_session_management();          // Standalone
//    test_room_lifecycle_with_sessions();// Standalone
//    test_concurrent_room_operations();  // Standalone
//    test_error_duplicate_player();      // Standalone
//    test_actor_pool_stress();           // Standalone
//    std::cout << "========================================" << std::endl;
//    std::cout << "Standalone tests PASSED!" << std::endl;
//    std::cout << "========================================" << std::endl;

    return 0;
}
