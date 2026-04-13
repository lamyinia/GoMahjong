cmake_minimum_required(VERSION 3.20)

project(gomahjong_cpp LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

find_package(Boost REQUIRED)
set(THREADS_PREFER_PTHREAD_FLAG ON)
find_package(Threads REQUIRED)
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
        ${PROTO_DIR}/game_mahjong.proto
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

# Collect source files automatically
file(GLOB_RECURSE SERVER_SOURCES CONFIGURE_DEPENDS "src/*.cpp")
file(GLOB_RECURSE SERVER_HEADERS CONFIGURE_DEPENDS "include/*.h" "include/*.hpp")

add_executable(gomahjong_server
        ${SERVER_SOURCES}
        ${SERVER_HEADERS}
        ${GAME_SERVICE_GRPC_SRCS}
        ${GAME_SERVICE_GRPC_HDRS}
)

# Helper function for common target setup
function(setup_target target_name)
    target_include_directories(${target_name} PRIVATE
            ${CMAKE_CURRENT_SOURCE_DIR}/src
            ${CMAKE_CURRENT_SOURCE_DIR}/include
            ${CMAKE_CURRENT_BINARY_DIR}
            ${GENERATED_DIR}
    )
    if (CMAKE_CXX_COMPILER_ID MATCHES "Clang|GNU")
        target_compile_definitions(${target_name} PRIVATE SPDLOG_FUNCTION=__PRETTY_FUNCTION__)
    elseif (MSVC)
        target_compile_definitions(${target_name} PRIVATE SPDLOG_FUNCTION=__FUNCSIG__)
    endif ()
endfunction()

setup_target(gomahjong_server)

target_link_libraries(gomahjong_server PRIVATE
        Boost::boost
        proto_generated
        spdlog::spdlog
        nlohmann_json::nlohmann_json
        Threads::Threads
        etcd-cpp-api
        gRPC::grpc++
        mongo::mongocxx_shared
)

add_executable(mongo_pool_test
        tests/mongo_pool_test.cpp
        src/infrastructure/persistence/mongo_pool.cpp
        src/infrastructure/log/logger.cpp
)
setup_target(mongo_pool_test)

target_link_libraries(mongo_pool_test PRIVATE
        Boost::boost
        spdlog::spdlog
        mongo::mongocxx_shared
        Threads::Threads
)

# Room test
file(GLOB_RECURSE ROOM_SOURCES CONFIGURE_DEPENDS "src/domain/game/room/*.cpp")
file(GLOB_RECURSE ENGINE_SOURCES CONFIGURE_DEPENDS "src/domain/game/engine/*.cpp")
file(GLOB_RECURSE OUTBOUND_SOURCES CONFIGURE_DEPENDS "src/domain/game/outbound/*.cpp")
file(GLOB_RECURSE SESSION_SOURCES CONFIGURE_DEPENDS "src/infrastructure/net/session/*.cpp")
file(GLOB_RECURSE CHANNEL_SOURCES CONFIGURE_DEPENDS "src/infrastructure/net/channel/*.cpp")

add_executable(room_test
        tests/room_test.cpp
        ${ROOM_SOURCES}
        ${ENGINE_SOURCES}
        ${OUTBOUND_SOURCES}
        ${SESSION_SOURCES}
        ${CHANNEL_SOURCES}
        src/infrastructure/util/timing_wheel.cpp
        src/infrastructure/log/logger.cpp
)
setup_target(room_test)

target_link_libraries(room_test PRIVATE
        Boost::boost
        proto_generated
        spdlog::spdlog
        nlohmann_json::nlohmann_json
        Threads::Threads
)

# Integration test
add_executable(integration_test
        tests/integration_test.cpp
        ${ROOM_SOURCES}
        ${ENGINE_SOURCES}
        ${OUTBOUND_SOURCES}
        ${SESSION_SOURCES}
        ${CHANNEL_SOURCES}
        src/infrastructure/util/timing_wheel.cpp
        src/infrastructure/log/logger.cpp
)
setup_target(integration_test)

target_link_libraries(integration_test PRIVATE
        Boost::boost
        proto_generated
        spdlog::spdlog
        nlohmann_json::nlohmann_json
        Threads::Threads
)

# Game timer test
file(GLOB_RECURSE TIMER_SOURCES CONFIGURE_DEPENDS "src/domain/game/engine/mahjong/timer/*.cpp")
file(GLOB_RECURSE UTIL_TIMER_SOURCES CONFIGURE_DEPENDS "src/infrastructure/util/timing_wheel.cpp" "src/infrastructure/util/timer_thread.cpp")

add_executable(game_timer_test
        tests/game_timer_test.cpp
        ${TIMER_SOURCES}
        ${UTIL_TIMER_SOURCES}
        src/infrastructure/log/logger.cpp
)
setup_target(game_timer_test)

target_link_libraries(game_timer_test PRIVATE
        Boost::boost
        spdlog::spdlog
        Threads::Threads
)
