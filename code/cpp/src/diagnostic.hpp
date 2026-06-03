// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#pragma once

#include <string>
#include <vector>
#include <optional>

namespace zd {

// Diagnostic: severity, domain, message. domain is one of:
// packages | repositories | services | files | units | manifest |
// transaction | invocation.
enum class Severity { Error, Warning };

struct Diagnostic {
    Severity severity = Severity::Error;
    std::string domain;
    std::string message;

    std::string format() const;  // "[error] domain=<d>: <message>"
};

// A simple Result type: a value of type T or a Diagnostic error. Internal
// behaviours return errors to their caller; only the verb layer maps them to
// exit codes.
template <class T>
struct Result {
    std::optional<T> value;
    std::optional<Diagnostic> error;

    bool ok() const { return !error.has_value(); }
    static Result<T> success(T v) { Result<T> r; r.value = std::move(v); return r; }
    static Result<T> fail(Diagnostic d) { Result<T> r; r.error = std::move(d); return r; }
};

// Void-like result for behaviours that produce no value but can fail.
struct VoidResult {
    std::optional<Diagnostic> error;
    bool ok() const { return !error.has_value(); }
    static VoidResult success() { return VoidResult{}; }
    static VoidResult fail(Diagnostic d) { VoidResult r; r.error = std::move(d); return r; }
};

inline Diagnostic make_error(const std::string& domain, const std::string& msg) {
    return Diagnostic{Severity::Error, domain, msg};
}
inline Diagnostic make_warning(const std::string& domain, const std::string& msg) {
    return Diagnostic{Severity::Warning, domain, msg};
}

}  // namespace zd
