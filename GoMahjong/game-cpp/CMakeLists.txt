cmake_minimum_required(VERSION 3.20)

project(gomahjong_cpp LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

find_package(Boost REQUIRED)
find_package(Protobuf REQUIRED)

include(FetchContent)

FetchContent_Declare(
        spdlog
        GIT_REPOSITORY https://github.com/gabime/spdlog.git
        GIT_TAG v1.14.1
)
FetchContent_Declare(
        nlohmann_json
        GIT_REPOSITORY https://github.com/nlohmann/json.git
        GIT_TAG v3.11.3
)

set(SPDLOG_BUILD_SHARED OFF CACHE BOOL "" FORCE)
set(SPDLOG_BUILD_EXAMPLE OFF CACHE BOOL "" FORCE)
set(SPDLOG_BUILD_TESTS OFF CACHE BOOL "" FORCE)
set(SPDLOG_BUILD_BENCH OFF CACHE BOOL "" FORCE)

FetchContent_MakeAvailable(spdlog nlohmann_json)

# Protobuf codegen
set(PROTO_DIR ${CMAKE_CURRENT_SOURCE_DIR}/proto)
set(GENERATED_DIR ${CMAKE_CURRENT_BINARY_DIR}/generated)
file(MAKE_DIRECTORY ${GENERATED_DIR})

set(PROTO_FILES ${PROTO_DIR}/envelope.proto)

protobuf_generate_cpp(PROTO_SRCS PROTO_HDRS ${PROTO_FILES})
## 自动调用 protoc envelope.proto --cpp_out=./build

add_executable(gomahjong_server
        src/main.cpp
        src/bootstrap/app.cpp
        src/infrastructure/log/logger.cpp
        src/infrastructure/config/config.cpp
        src/infrastructure/net/transport/tcp_transport.cpp
        ${PROTO_SRCS}
        ${PROTO_HDRS}
        include/infrastructure/net/channel/i_channel.h
        src/infrastructure/net/session/session.cpp
        include/infrastructure/net/session/session.h
        src/infrastructure/net/session/session_manager.cpp
        include/infrastructure/net/session/session_manager.h
        src/bootstrap/server_hub.cpp
        include/bootstrap/server_hub.h
        src/domain/game/room/room_manager.cpp
        include/domain/game/room/room_manager.h
        src/domain/game/engine/engine.cpp
        include/domain/game/engine/engine.h
        include/infrastructure/net/listener/i_listener.h
        src/infrastructure/net/listener/tcp_listener.cpp
        include/infrastructure/net/listener/tcp_listener.h
        include/infrastructure/net/transport/tcp_transport.h
)

target_include_directories(gomahjong_server PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/src
        ${CMAKE_CURRENT_SOURCE_DIR}/include
        ${CMAKE_CURRENT_BINARY_DIR}
)

target_link_libraries(gomahjong_server PRIVATE
        Boost::boost
        protobuf::libprotobuf
        spdlog::spdlog
        nlohmann_json::nlohmann_json
)

if (CMAKE_CXX_COMPILER_ID MATCHES "Clang|GNU")
    target_compile_definitions(gomahjong_server PRIVATE SPDLOG_FUNCTION=__PRETTY_FUNCTION__)
elseif (MSVC)
    target_compile_definitions(gomahjong_server PRIVATE SPDLOG_FUNCTION=__FUNCSIG__)
endif ()

set(THREADS_PREFER_PTHREAD_FLAG ON)  # 指示在可用时优先使用pthread库。
find_package(Threads REQUIRED)
target_link_libraries(gomahjong_server PRIVATE Threads::Threads)

# Test executable
add_executable(tcp_server_test
        tests/tcp_server_test.cpp
        src/infrastructure/log/logger.cpp
        src/infrastructure/net/transport/tcp_transport.cpp
        src/infrastructure/net/listener/tcp_listener.cpp
)

target_include_directories(tcp_server_test PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/src
        ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_link_libraries(tcp_server_test PRIVATE
        Boost::boost
        spdlog::spdlog
        Threads::Threads
)

if (CMAKE_CXX_COMPILER_ID MATCHES "Clang|GNU")
    target_compile_definitions(tcp_server_test PRIVATE SPDLOG_FUNCTION=__PRETTY_FUNCTION__)
elseif (MSVC)
    target_compile_definitions(tcp_server_test PRIVATE SPDLOG_FUNCTION=__FUNCSIG__)
endif ()

# ServerHub test executable (tests connecting to configured port)
add_executable(server_hub_test
        tests/server_hub_test.cpp
        src/infrastructure/config/config.cpp
)

target_include_directories(server_hub_test PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/src
        ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_link_libraries(server_hub_test PRIVATE
        Boost::boost
        nlohmann_json::nlohmann_json
        Threads::Threads
)
