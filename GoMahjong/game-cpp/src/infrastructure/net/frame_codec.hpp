#pragma once

#include <array>
#include <cstdint>
#include <optional>
#include <span>
#include <vector>

namespace gomahjong::net {

inline std::array<std::uint8_t, 4> encode_length_be(std::uint32_t len) {
    return {
        static_cast<std::uint8_t>((len >> 24) & 0xFF),
        static_cast<std::uint8_t>((len >> 16) & 0xFF),
        static_cast<std::uint8_t>((len >> 8) & 0xFF),
        static_cast<std::uint8_t>(len & 0xFF),
    };
}

inline std::uint32_t decode_length_be(std::span<const std::uint8_t, 4> buf) {
    return (static_cast<std::uint32_t>(buf[0]) << 24) |
           (static_cast<std::uint32_t>(buf[1]) << 16) |
           (static_cast<std::uint32_t>(buf[2]) << 8) |
           static_cast<std::uint32_t>(buf[3]);
}

struct Frame {
    std::vector<std::uint8_t> body;
};

class FrameDecoder {
public:
    explicit FrameDecoder(std::uint32_t max_frame_size, std::size_t max_accumulated_bytes)
        : max_frame_size_(max_frame_size),
          max_accumulated_bytes_(max_accumulated_bytes) {}

    // Append incoming bytes.
    // Returns false if accumulated bytes exceed configured maximum.
    bool append(std::span<const std::uint8_t> bytes) {
        if (bytes.empty()) {
            return true;
        }
        if (buffer_.size() + bytes.size() > max_accumulated_bytes_) {
            return false;
        }
        buffer_.insert(buffer_.end(), bytes.begin(), bytes.end());
        return true;
    }

    // Try to pop a complete frame.
    // Returns std::nullopt if no complete frame is available.
    // Throws nothing; caller should treat invalid length as protocol error.
    std::optional<Frame> try_pop() {
        if (buffer_.size() < 4) {
            return std::nullopt;
        }

        std::array<std::uint8_t, 4> len_buf{buffer_[0], buffer_[1], buffer_[2], buffer_[3]};
        std::uint32_t len = decode_length_be(std::span<const std::uint8_t, 4>(len_buf));
        if (len == 0 || len > max_frame_size_) {
            invalid_length_ = true;
            return std::nullopt;
        }

        if (buffer_.size() < 4ULL + len) {
            return std::nullopt;
        }

        Frame f;
        f.body.assign(buffer_.begin() + 4, buffer_.begin() + 4 + len);
        buffer_.erase(buffer_.begin(), buffer_.begin() + 4 + len);
        return f;
    }

    bool invalid_length() const noexcept { return invalid_length_; }

private:
    std::vector<std::uint8_t> buffer_;
    std::uint32_t max_frame_size_{0};
    std::size_t max_accumulated_bytes_{0};
    bool invalid_length_{false};
};

} // namespace gomahjong::net
