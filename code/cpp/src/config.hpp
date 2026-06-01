// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// config.hpp -- the resolved CONFIG knobs and the key=value option model.
// Control via environment variables is forbidden (CONFIG-ENV-VARS: forbidden);
// the only env var consulted anywhere is ZYPPER_DECLARATIVE_DEBUG, a trace gate.
#ifndef ZD_CONFIG_HPP
#define ZD_CONFIG_HPP

#include "types.hpp"
#include <string>
#include <map>
#include <optional>

namespace zd {

struct Config {
    TransactionMode transaction_mode = TransactionMode::Auto;
    std::optional<std::string> manifest_path;
    ManifestFormat manifest_format = ManifestFormat::Json; // default json
    OnUnreadable on_unreadable = OnUnreadable::Error;       // default error
    ScanScope scope = ScanScope::Etc;                        // default etc
    std::optional<std::string> repo_lock;
    std::optional<std::string> content_store;
    std::optional<std::string> keep_list;
    bool signature_verification = true;                      // default on
    std::optional<std::string> keyring;
    std::string activation_policy = "reboot";
    std::string applied_root = "/";
    // describe-specific
    std::string root = "/";
    std::optional<std::string> out;
    std::optional<ManifestFormat> explicit_format;           // format= for this invocation
    std::optional<std::string> state_path;
};

// One parsed key=value option.
struct Option { std::string key; std::string value; };

// Debug trace gate (NOT behaviour control).
void debug_log(const std::string& msg);

} // namespace zd

#endif // ZD_CONFIG_HPP
