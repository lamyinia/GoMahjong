#pragma once

// Convenience header for coroutine utilities

#include "infrastructure/util/coroutine/task.hpp"
#include "infrastructure/util/coroutine/awaiters.hpp"
#include "infrastructure/util/coroutine/generator.hpp"

namespace infra::util {

// Re-export coro namespace for convenience
namespace coroutine = coro;

} // namespace infra::util
