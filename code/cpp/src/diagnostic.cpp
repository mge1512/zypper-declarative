// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#include "diagnostic.hpp"

namespace zd {

std::string Diagnostic::format() const {
    std::string sev = (severity == Severity::Error) ? "error" : "warning";
    return "[" + sev + "] domain=" + domain + ": " + message;
}

}  // namespace zd
