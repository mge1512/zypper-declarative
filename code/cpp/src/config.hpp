// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Resolved CONFIG knobs and the small Result type the internal behaviours use
// to return errors to their caller (only the verb layer maps to an exit code).
#ifndef ZD_CONFIG_HPP
#define ZD_CONFIG_HPP

#include <optional>
#include <string>

#include "types.hpp"

namespace zd {

// CONFIG knobs (CONFIG section of the spec). All are key=value options;
// command-line overrides preset. Defaults match the spec.
struct Config {
    TransactionMode transaction_mode = TransactionMode::Auto;
    std::string manifest_path;                 // default supplied by delivery
    ManifestFormat manifest_format = ManifestFormat::Json;  // fallback default
    bool on_unreadable_error = true;           // on-unreadable: error (default)
    ScanScope scope = ScanScope::Etc;          // etc (default) or full
    std::string repo_lock;
    std::string content_store;                 // "" = read-only describe
    std::string keep_list;                     // path to allowlist
    bool signature_verification = true;        // default on
    std::string keyring;
    std::string activation_policy = "reboot";
    std::string applied_root = "/";            // load-applied-record root
    // describe-specific
    std::string root = "/";                    // describe root
    std::string out;                           // describe output ("" = stdout)
    std::optional<ManifestFormat> explicit_format;  // format= if given
    std::optional<std::string> state_path;     // verify/diff captured state
    bool state_path_given = false;
    bool manifest_path_given = false;
};

// Result<T>: an internal behaviour returns either a value or a Diagnostic.
template <class T>
struct Result {
    bool ok = false;
    T value{};
    Diagnostic error;
    static Result success(T v) { return Result{true, std::move(v), {}}; }
    static Result fail(Diagnostic d) { return Result{false, {}, std::move(d)}; }
};

// Result<void> equivalent.
struct Status {
    bool ok = false;
    Diagnostic error;
    static Status success() { return Status{true, {}}; }
    static Status fail(Diagnostic d) { return Status{false, std::move(d)}; }
};

}  // namespace zd

#endif  // ZD_CONFIG_HPP
