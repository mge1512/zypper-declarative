// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
//
// Diagnostic and a small Result type. Internal behaviours return errors to
// their caller (a Diagnostic); only the verb layer maps to an exit code.
#ifndef ZD_DIAGNOSTIC_HPP
#define ZD_DIAGNOSTIC_HPP

#include <optional>
#include <string>
#include <utility>
#include <vector>

namespace zd {

enum class Severity { Error, Warning };

// domain in {packages, repositories, services, files, units, manifest,
// transaction, invocation}
struct Diagnostic {
    Severity severity = Severity::Error;
    std::string domain;
    std::string message;

    std::string format() const {
        std::string sev = (severity == Severity::Error) ? "error" : "warning";
        return sev + ": " + domain + ": " + message;
    }
};

inline Diagnostic err(std::string domain, std::string message) {
    return Diagnostic{Severity::Error, std::move(domain), std::move(message)};
}
inline Diagnostic warn(std::string domain, std::string message) {
    return Diagnostic{Severity::Warning, std::move(domain), std::move(message)};
}

// Result<T>: either a value or a Diagnostic error.
template <class T>
class Result {
public:
    Result(T value) : value_(std::move(value)) {}
    Result(Diagnostic e) : error_(std::move(e)) {}

    bool ok() const { return !error_.has_value(); }
    const Diagnostic& error() const { return *error_; }
    T& value() { return *value_; }
    const T& value() const { return *value_; }

private:
    std::optional<T> value_;
    std::optional<Diagnostic> error_;
};

}  // namespace zd

#endif  // ZD_DIAGNOSTIC_HPP
