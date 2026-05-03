cmake_minimum_required(VERSION 3.20)

project(gomahjong_cpp LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

find_package(Boost REQUIRED)
set(THREADS_PREFER_PTHREAD_FLAG ON)
find_package(Threads REQUIRED)
find_package(Protobuf CONFIG REQUIRED)
# Conan's protobuf-conan-protoc-target.cmake uses find_program() which
# picks up /usr/bin/protoc instead of Conan's protoc. Fix it by using
# protobuf_PACKAGE_FOLDER_RELEASE to locate the correct protoc.
if(TARGET protobuf::protoc AND DEFINED protobuf_PACKAGE_FOLDER_RELEASE)
    find_program(_conan_protoc
        NAMES protoc
        PATHS "${protobuf_PACKAGE_FOLDER_RELEASE}/bin"
        NO_DEFAULT_PATH
    )
    if(_conan_protoc)
        set_property(TARGET protobuf::protoc PROPERTY IMPORTED_LOCATION "${_conan_protoc}")
        set(Protobuf_PROTOC_EXECUTABLE "${_conan_protoc}" CACHE FILEPATH "Conan protoc" FORCE)
        message(STATUS "Using Conan protoc: ${_conan_protoc}")
    endif()
endif()
find_package(gRPC CONFIG REQUIRED)
find_package(mongocxx REQUIRED)
find_package(OpenSSL REQUIRED)
find_package(spdlog REQUIRED)
find_package(nlohmann_json REQUIRED)



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

# Generate C++ protobuf sources using Conan's protoc explicitly
set(PROTO_PB_SRCS)
set(PROTO_PB_HDRS)
foreach(PROTO_FILE ${PROTO_FILES})
    get_filename_component(PROTO_NAME "${PROTO_FILE}" NAME_WE)
    list(APPEND PROTO_PB_SRCS "${GENERATED_DIR}/${PROTO_NAME}.pb.cc")
    list(APPEND PROTO_PB_HDRS "${GENERATED_DIR}/${PROTO_NAME}.pb.h")
endforeach()

add_custom_command(
        OUTPUT ${PROTO_PB_SRCS} ${PROTO_PB_HDRS}
        COMMAND $<TARGET_FILE:protobuf::protoc>
        ARGS --cpp_out=${GENERATED_DIR} -I ${PROTO_DIR} ${PROTO_FILES}
        DEPENDS ${PROTO_FILES} protobuf::protoc
        COMMENT "Generating C++ protobuf sources with Conan protoc"
        VERBATIM
)

target_sources(proto_generated PRIVATE ${PROTO_PB_SRCS} ${PROTO_PB_HDRS})

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

# ── etcd v3 gRPC API codegen ──
set(ETCD_PROTO_DIR ${PROTO_DIR}/etcd)
set(ETCD_GENERATED_DIR ${GENERATED_DIR}/etcd)
file(MAKE_DIRECTORY ${ETCD_GENERATED_DIR}/mvccpb)
file(MAKE_DIRECTORY ${ETCD_GENERATED_DIR}/etcdserverpb)
file(MAKE_DIRECTORY ${ETCD_GENERATED_DIR}/authpb)

set(ETCD_PROTO_FILES
        ${ETCD_PROTO_DIR}/authpb/auth.proto
        ${ETCD_PROTO_DIR}/mvccpb/kv.proto
        ${ETCD_PROTO_DIR}/etcdserverpb/rpc.proto
)

# Generate protobuf C++ sources for etcd protos
set(ETCD_PB_SRCS
        ${ETCD_GENERATED_DIR}/authpb/auth.pb.cc
        ${ETCD_GENERATED_DIR}/mvccpb/kv.pb.cc
        ${ETCD_GENERATED_DIR}/etcdserverpb/rpc.pb.cc
)
set(ETCD_PB_HDRS
        ${ETCD_GENERATED_DIR}/authpb/auth.pb.h
        ${ETCD_GENERATED_DIR}/mvccpb/kv.pb.h
        ${ETCD_GENERATED_DIR}/etcdserverpb/rpc.pb.h
)

add_custom_command(
        OUTPUT ${ETCD_PB_SRCS} ${ETCD_PB_HDRS}
        COMMAND $<TARGET_FILE:protobuf::protoc>
        ARGS --cpp_out=${ETCD_GENERATED_DIR} -I ${ETCD_PROTO_DIR} ${ETCD_PROTO_FILES}
        DEPENDS ${ETCD_PROTO_FILES} protobuf::protoc
        COMMENT "Generating etcd protobuf sources"
        VERBATIM
)

# Generate gRPC stubs for etcd protos
set(ETCD_GRPC_SRCS
        ${ETCD_GENERATED_DIR}/etcdserverpb/rpc.grpc.pb.cc
)
set(ETCD_GRPC_HDRS
        ${ETCD_GENERATED_DIR}/etcdserverpb/rpc.grpc.pb.h
)

add_custom_command(
        OUTPUT ${ETCD_GRPC_SRCS} ${ETCD_GRPC_HDRS}
        COMMAND $<TARGET_FILE:protobuf::protoc>
        ARGS
        -I ${ETCD_PROTO_DIR}
        --grpc_out=${ETCD_GENERATED_DIR}
        --plugin=protoc-gen-grpc=$<TARGET_FILE:gRPC::grpc_cpp_plugin>
        ${ETCD_PROTO_DIR}/etcdserverpb/rpc.proto
        DEPENDS ${ETCD_PROTO_DIR}/etcdserverpb/rpc.proto
        COMMENT "Generating etcd gRPC stubs"
        VERBATIM
)

add_custom_target(etcd_grpc_codegen
        DEPENDS ${ETCD_PB_SRCS} ${ETCD_PB_HDRS} ${ETCD_GRPC_SRCS} ${ETCD_GRPC_HDRS}
)

target_sources(proto_generated PRIVATE
        ${ETCD_PB_SRCS} ${ETCD_PB_HDRS}
        ${ETCD_GRPC_SRCS} ${ETCD_GRPC_HDRS}
)
target_include_directories(proto_generated PUBLIC ${ETCD_GENERATED_DIR})
add_dependencies(proto_generated etcd_grpc_codegen)

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
        gRPC::grpc++
        mongo::mongocxx_shared
        OpenSSL::SSL
        OpenSSL::Crypto
)


# ═══════════════════════════════════════════════════════════════
#  Test infrastructure (Google Test)
# ═══════════════════════════════════════════════════════════════

find_package(GTest REQUIRED)
enable_testing()

# Helper: add a gtest-based test target with minimal boilerplate
#   add_gtest(target_name
#       SOURCES   test_file.cpp [extra sources...]
#       LIBS     [extra link libs...]
#   )
function(add_gtest target_name)
    cmake_parse_arguments(ARG "" "" "SOURCES;LIBS" ${ARGN})

    add_executable(${target_name} ${ARG_SOURCES})
    setup_target(${target_name})
    target_link_libraries(${target_name} PRIVATE
            GTest::gtest
            GTest::gtest_main
            ${ARG_LIBS}
    )
    add_test(NAME ${target_name} COMMAND ${target_name})
endfunction()

# ── Pre-collected source groups (shared by multiple tests) ──

file(GLOB_RECURSE ROOM_SOURCES CONFIGURE_DEPENDS "src/domain/game/room/*.cpp")
file(GLOB_RECURSE ENGINE_SOURCES CONFIGURE_DEPENDS "src/domain/game/engine/*.cpp")
file(GLOB_RECURSE OUTBOUND_SOURCES CONFIGURE_DEPENDS "src/domain/game/outbound/*.cpp")
file(GLOB_RECURSE SESSION_SOURCES CONFIGURE_DEPENDS "src/infrastructure/net/session/*.cpp")
file(GLOB_RECURSE CHANNEL_SOURCES CONFIGURE_DEPENDS "src/infrastructure/net/channel/*.cpp")
file(GLOB_RECURSE TIMER_SOURCES CONFIGURE_DEPENDS "src/domain/game/engine/mahjong/timer/*.cpp")

# ── Test targets ──

# Mongo pool
add_gtest(mongo_pool_test
    SOURCES
        tests/mongo_pool_test.cpp
        src/infrastructure/persistence/mongo_pool.cpp
        src/infrastructure/log/logger.cpp
    LIBS
        Boost::boost spdlog::spdlog mongo::mongocxx_shared Threads::Threads
)

# Room
add_gtest(room_test
    SOURCES
        tests/room_test.cpp
        ${ROOM_SOURCES} ${ENGINE_SOURCES} ${OUTBOUND_SOURCES}
        ${SESSION_SOURCES} ${CHANNEL_SOURCES}
        src/infrastructure/util/timing_wheel.cpp
        src/infrastructure/log/logger.cpp
    LIBS
        Boost::boost proto_generated spdlog::spdlog
        nlohmann_json::nlohmann_json Threads::Threads
)

# Integration
add_gtest(integration_test
    SOURCES
        tests/integration_test.cpp
        ${ROOM_SOURCES} ${ENGINE_SOURCES} ${OUTBOUND_SOURCES}
        ${SESSION_SOURCES} ${CHANNEL_SOURCES}
        src/infrastructure/util/timing_wheel.cpp
        src/infrastructure/log/logger.cpp
    LIBS
        Boost::boost proto_generated spdlog::spdlog
        nlohmann_json::nlohmann_json Threads::Threads
)

# Game timer
add_gtest(game_timer_test
    SOURCES
        tests/game_timer_test.cpp
        ${TIMER_SOURCES}
        src/infrastructure/util/timing_wheel.cpp
        src/infrastructure/util/timer_thread.cpp
        src/infrastructure/log/logger.cpp
    LIBS
        Boost::boost spdlog::spdlog Threads::Threads
)

# Coroutine
add_gtest(coroutine_test
    SOURCES
        tests/coroutine_test.cpp
        src/infrastructure/util/thread_pool.cpp
        src/infrastructure/util/timing_wheel.cpp
        src/infrastructure/util/timer_thread.cpp
        src/infrastructure/log/logger.cpp
    LIBS
        Boost::boost spdlog::spdlog Threads::Threads
)

# Memory pool
add_gtest(memory_pool_test
    SOURCES
        tests/memory_pool_test.cpp
        src/infrastructure/log/logger.cpp
    LIBS
        Boost::boost spdlog::spdlog Threads::Threads
)

# GameEvent pool
add_gtest(game_event_pool_test
    SOURCES
        tests/game_event_pool_test.cpp
        src/infrastructure/log/logger.cpp
    LIBS
        Boost::boost spdlog::spdlog Threads::Threads
)
