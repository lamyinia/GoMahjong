#include <memory>

namespace infra::net::transport {
    class ITransport : public std::enable_shared_from_this<ITransport> {

    };

}