#pragma once

// Convenience header for memory utilities

#include "infrastructure/util/memory/fixed_size_pool.hpp"
#include "infrastructure/util/memory/typed_object_pool.hpp"

namespace infra::util {

// Re-export memory namespace
namespace memory = ::infra::util::memory;

} // namespace infra::util
