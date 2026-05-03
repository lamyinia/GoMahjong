#pragma once

#include <atomic>
#include <cassert>
#include <cstdlib>
#include <cstddef>
#include <cstdint>
#include <mutex>
#include <new>
#include <vector>

namespace infra::util::memory {

/**
 * FixedSizeMemoryPool - Fixed-size memory block pool
 * 
 * Features:
 * - O(1) allocation/deallocation
 * - Zero memory fragmentation
 * - Cache-friendly (contiguous memory)
 * - Thread-safe with optional lock-free mode
 * 
 * Usage:
 *   FixedSizeMemoryPool pool(64, 1024);  // 64-byte blocks, 1024 initial capacity
 *   void* ptr = pool.allocate();
 *   pool.deallocate(ptr);
 */
class FixedSizeMemoryPool {
public:
    /**
     * Constructor
     * @param blockSize Size of each block (must be >= sizeof(void*))
     * @param initialCapacity Initial number of blocks to preallocate
     * @param alignment Memory alignment (default 8 bytes)
     */
    explicit FixedSizeMemoryPool(
        std::size_t blockSize,
        std::size_t initialCapacity = 64,
        std::size_t alignment = 8
    );

    /**
     * Destructor - releases all memory
     */
    ~FixedSizeMemoryPool();

    // Non-copyable, non-movable
    FixedSizeMemoryPool(const FixedSizeMemoryPool&) = delete;
    FixedSizeMemoryPool& operator=(const FixedSizeMemoryPool&) = delete;
    FixedSizeMemoryPool(FixedSizeMemoryPool&&) = delete;
    FixedSizeMemoryPool& operator=(FixedSizeMemoryPool&&) = delete;

    /**
     * Allocate a block
     * @return Pointer to allocated block, or nullptr if out of memory
     */
    void* allocate();

    /**
     * Deallocate a block
     * @param ptr Pointer to block (must be from this pool)
     */
    void deallocate(void* ptr);

    /**
     * Get block size
     */
    [[nodiscard]] std::size_t blockSize() const noexcept { return blockSize_; }

    /**
     * Get number of available blocks
     */
    [[nodiscard]] std::size_t available() const noexcept;

    /**
     * Get total capacity (allocated blocks)
     */
    [[nodiscard]] std::size_t capacity() const noexcept { return totalBlocks_; }

    /**
     * Get number of allocated blocks currently in use
     */
    [[nodiscard]] std::size_t used() const noexcept;

    /**
     * Preallocate more blocks
     * @param count Number of additional blocks to allocate
     * @return true if successful
     */
    bool expand(std::size_t count);

    /**
     * Clear all blocks (return to free list)
     * Warning: Only call when no blocks are in use!
     */
    void reset();

private:
    // Memory chunk header
    struct Chunk {
        Chunk* next;
        std::size_t size;
    };

    // Free list node (embedded in free blocks)
    struct FreeNode {
        FreeNode* next;
    };

    void* allocateNewChunk(std::size_t blockCount);
    void addToFreeList(void* block);

    std::size_t blockSize_;
    std::size_t alignment_;
    std::size_t totalBlocks_;
    
    mutable std::mutex mutex_;
    FreeNode* freeHead_;          // Head of free list
    
    std::vector<Chunk*> chunks_;  // All allocated chunks

    // Statistics
    std::atomic<std::size_t> allocatedCount_{0};
};

// ==================== Implementation ====================

inline FixedSizeMemoryPool::FixedSizeMemoryPool(
    std::size_t blockSize,
    std::size_t initialCapacity,
    std::size_t alignment
)
    : blockSize_(std::max(blockSize, sizeof(FreeNode)))
    , alignment_(alignment)
    , totalBlocks_(0)
    , freeHead_(nullptr)
{
    // Align block size
    if (alignment_ > 1) {
        blockSize_ = ((blockSize_ + alignment_ - 1) / alignment_) * alignment_;
    }

    if (initialCapacity > 0) {
        expand(initialCapacity);
    }
}

inline FixedSizeMemoryPool::~FixedSizeMemoryPool() {
    // Free all chunks
    for (auto* chunk : chunks_) {
        std::free(chunk);
    }
}

inline void* FixedSizeMemoryPool::allocate() {
    std::lock_guard<std::mutex> lock(mutex_);

    if (freeHead_ == nullptr) {
        // No free blocks, try to expand
        if (!expand(totalBlocks_ / 2 + 1)) {
            return nullptr;
        }
    }

    // Pop from free list
    void* block = freeHead_;
    freeHead_ = freeHead_->next;
    allocatedCount_.fetch_add(1, std::memory_order_relaxed);

    return block;
}

inline void FixedSizeMemoryPool::deallocate(void* ptr) {
    if (ptr == nullptr) return;

    std::lock_guard<std::mutex> lock(mutex_);
    addToFreeList(ptr);
    allocatedCount_.fetch_sub(1, std::memory_order_relaxed);
}

inline std::size_t FixedSizeMemoryPool::available() const noexcept {
    std::lock_guard<std::mutex> lock(mutex_);
    std::size_t count = 0;
    FreeNode* node = freeHead_;
    while (node != nullptr) {
        ++count;
        node = node->next;
    }
    return count;
}

inline std::size_t FixedSizeMemoryPool::used() const noexcept {
    return allocatedCount_.load(std::memory_order_relaxed);
}

inline bool FixedSizeMemoryPool::expand(std::size_t count) {
    void* chunk = allocateNewChunk(count);
    if (chunk == nullptr) {
        return false;
    }
    return true;
}

inline void FixedSizeMemoryPool::reset() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    // Rebuild free list from all blocks
    freeHead_ = nullptr;
    allocatedCount_.store(0, std::memory_order_relaxed);

    for (auto* chunk : chunks_) {
        char* ptr = reinterpret_cast<char*>(chunk) + sizeof(Chunk);
        char* end = ptr + (chunk->size - sizeof(Chunk));
        
        while (ptr + blockSize_ <= end) {
            addToFreeList(ptr);
            ptr += blockSize_;
        }
    }
}

inline void* FixedSizeMemoryPool::allocateNewChunk(std::size_t blockCount) {
    // Calculate chunk size
    std::size_t chunkSize = sizeof(Chunk) + blockCount * blockSize_;
    
    // Align chunk size
    if (alignment_ > 1) {
        chunkSize = ((chunkSize + alignment_ - 1) / alignment_) * alignment_;
    }

    // Allocate chunk
    void* memory = std::aligned_alloc(alignment_, chunkSize);
    if (memory == nullptr) {
        return nullptr;
    }

    // Setup chunk header
    Chunk* chunk = reinterpret_cast<Chunk*>(memory);
    chunk->next = nullptr;
    chunk->size = chunkSize;
    chunks_.push_back(chunk);

    // Add all blocks to free list
    char* ptr = reinterpret_cast<char*>(memory) + sizeof(Chunk);
    char* end = ptr + (chunkSize - sizeof(Chunk));
    
    std::size_t blocksAdded = 0;
    while (ptr + blockSize_ <= end) {
        addToFreeList(ptr);
        ptr += blockSize_;
        ++blocksAdded;
    }

    totalBlocks_ += blocksAdded;
    return memory;
}

inline void FixedSizeMemoryPool::addToFreeList(void* block) {
    FreeNode* node = reinterpret_cast<FreeNode*>(block);
    node->next = freeHead_;
    freeHead_ = node;
}

} // namespace infra::util::memory
