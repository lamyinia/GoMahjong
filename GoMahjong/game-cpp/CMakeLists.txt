cmake_minimum_required(VERSION 3.20)

project(gomahjong_cpp LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

find_package(Boost REQUIRED)
find_package(Protobuf REQUIRED)
find_package(gRPC CONFIG REQUIRED)
find_package(etcd-cpp-api REQUIRED)
find_package(mongocxx REQUIRED)

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

# Protobuf 文件
set(PROTO_FILES
        ${PROTO_DIR}/envelope.proto
        ${PROTO_DIR}/game_service.proto
        ${PROTO_DIR}/auth.proto
        ${PROTO_DIR}/heartbeat.proto
)

# Generate protobuf sources into ${GENERATED_DIR} and attach them to a CMake target.
add_library(proto_generated STATIC)
target_link_libraries(proto_generated PUBLIC protobuf::libprotobuf gRPC::grpc++)
target_include_directories(proto_generated PUBLIC ${GENERATED_DIR})

protobuf_generate(
        TARGET proto_generated
        LANGUAGE cpp
        PROTOC_OUT_DIR ${GENERATED_DIR}
        APPEND_PATH OFF
        IMPORT_DIRS ${PROTO_DIR}
        PROTOS ${PROTO_FILES}
)

# Generate gRPC sources for game_service.proto
get_filename_component(GAME_SERVICE_PROTO "${PROTO_DIR}/game_service.proto" ABSOLUTE)
get_filename_component(GAME_SERVICE_BASE "${GAME_SERVICE_PROTO}" NAME_WE)
set(GAME_SERVICE_GRPC_SRCS "${GENERATED_DIR}/${GAME_SERVICE_BASE}.grpc.pb.cc")
set(GAME_SERVICE_GRPC_HDRS "${GENERATED_DIR}/${GAME_SERVICE_BASE}.grpc.pb.h")

add_custom_command(
        OUTPUT ${GAME_SERVICE_GRPC_SRCS} ${GAME_SERVICE_GRPC_HDRS}
        COMMAND $<TARGET_FILE:protobuf::protoc>
        ARGS
        -I ${PROTO_DIR}
        --grpc_out=${GENERATED_DIR}
        --plugin=protoc-gen-grpc=$<TARGET_FILE:gRPC::grpc_cpp_plugin>
        ${GAME_SERVICE_PROTO}
        DEPENDS ${GAME_SERVICE_PROTO}
        COMMENT "Generating gRPC code for game_service.proto"
        VERBATIM
)

add_custom_target(game_service_grpc_codegen
        DEPENDS ${GAME_SERVICE_GRPC_SRCS} ${GAME_SERVICE_GRPC_HDRS}
)

target_sources(proto_generated PRIVATE
        ${GAME_SERVICE_GRPC_SRCS}
        ${GAME_SERVICE_GRPC_HDRS}
)

add_dependencies(proto_generated game_service_grpc_codegen)

add_executable(gomahjong_server
        src/main.cpp
        src/bootstrap/app.cpp
        src/infrastructure/log/logger.cpp
        src/infrastructure/config/config.cpp
        src/infrastructure/net/transport/tcp_transport.cpp
        ${GAME_SERVICE_GRPC_SRCS}
        ${GAME_SERVICE_GRPC_HDRS}
        # Channel 层
        include/infrastructure/net/channel/i_channel.h
        include/infrastructure/net/channel/channel_handler_context.h
        include/infrastructure/net/channel/i_channel_handler.h
        include/infrastructure/net/channel/channel_pipeline.h
        src/infrastructure/net/channel/channel_pipeline.cpp
        include/infrastructure/net/channel/tcp_channel.h
        src/infrastructure/net/channel/tcp_channel.cpp
        # Codec 层
        include/infrastructure/net/channel/codec/length_field_decoder.h
        src/infrastructure/net/channel/codec/length_field_decoder.cpp
        include/infrastructure/net/channel/codec/length_field_encoder.h
        src/infrastructure/net/channel/codec/length_field_encoder.cpp
        include/infrastructure/net/channel/codec/protobuf_decoder.h
        src/infrastructure/net/channel/codec/protobuf_decoder.cpp
        include/infrastructure/net/channel/codec/protobuf_encoder.h
        src/infrastructure/net/channel/codec/protobuf_encoder.cpp
        # Session 层
        src/infrastructure/net/session/session.cpp
        include/infrastructure/net/session/session.h
        src/infrastructure/net/session/session_manager.cpp
        include/infrastructure/net/session/session_manager.h
        # Bootstrap
        src/bootstrap/server_hub.cpp
        include/bootstrap/server_hub.h
        # Domain
        src/domain/game/room/room_manager.cpp
        include/domain/game/room/room_manager.h
        src/domain/game/engine/engine.cpp
        include/domain/game/engine/engine.h
        # Listener
        include/infrastructure/net/listener/i_listener.h
        src/infrastructure/net/listener/tcp_listener.cpp
        include/infrastructure/net/listener/tcp_listener.h
        # Transport
        include/infrastructure/net/transport/tcp_transport.h
        src/infrastructure/discovery/etcd_client.cpp
        include/infrastructure/discovery/etcd_client.hpp
        src/infrastructure/discovery/service_registry.cpp
        include/infrastructure/discovery/service_registry.hpp
        src/infrastructure/rpc/grpc_server.cpp
        include/infrastructure/rpc/grpc_server.hpp
        src/domain/game/service/game_service.cpp
        include/domain/game/service/game_service.hpp
        src/infrastructure/persistence/mongo_client.cpp
        include/infrastructure/persistence/mongo_client.hpp
        # Reliability
        src/infrastructure/net/reliability/wild_endpoint.cpp
        include/infrastructure/net/reliability/wild_endpoint.h
        src/infrastructure/net/reliability/wild_endpoint_manager.cpp
        include/infrastructure/net/reliability/wild_endpoint_manager.h
        src/infrastructure/net/reliability/auth_handler.cpp
        include/infrastructure/net/reliability/auth_handler.h
)

target_include_directories(gomahjong_server PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/src
        ${CMAKE_CURRENT_SOURCE_DIR}/include
        ${CMAKE_CURRENT_BINARY_DIR}
        ${GENERATED_DIR}
)

target_link_libraries(gomahjong_server PRIVATE
        Boost::boost
        proto_generated
        spdlog::spdlog
        nlohmann_json::nlohmann_json
        etcd-cpp-api
        gRPC::grpc++
        mongo::mongocxx_shared
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
        src/infrastructure/net/channel/channel_pipeline.cpp
        src/infrastructure/net/channel/tcp_channel.cpp
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
        Threads::Threads)
