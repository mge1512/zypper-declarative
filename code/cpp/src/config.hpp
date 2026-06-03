// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#pragma once

#include <string>
#include <map>
#include <optional>
#include <vector>

#include "types.hpp"

namespace zd {

// Effective configuration for one invocation: CONFIG defaults overlaid by
// command-line key=value options. Behaviour is never controlled via
// environment variables (forbidden by template).
struct Config {
    TransactionMode transaction_mode = TransactionMode::Auto;  // transaction-mode / mode
    std::optional<std::string> manifest_path;                  // manifest-path
    ManifestFormat manifest_format = ManifestFormat::Json;     // manifest-format
    std::optional<ManifestFormat> explicit_format;             // format=
    OnUnreadable on_unreadable = OnUnreadable::Error;           // on-unreadable
    ScanScope scope = ScanScope::Etc;                          // scope
    std::optional<std::string> state_path;                     // state-path
    std::string root = "/";                                    // root
    std::optional<std::string> out;                            // out
    std::optional<std::string> repo_lock;                      // repo-lock
    std::optional<std::string> content_store;                  // content-store
    std::optional<std::string> keep_list;                      // keep-list
    bool signature_verification = true;                        // signature-verification
    std::optional<std::string> keyring;                        // keyring
    std::string activation_policy = "reboot";                  // activation-policy
    std::string applied_root = "/";                            // applied-root
};

// Parse result for the global CLI surface.
struct ParsedArgs {
    std::string verb;                                  // bare-word verb (may be empty)
    std::map<std::string, std::string> options;        // key=value options
    std::vector<std::string> bad_args;                 // unrecognised bare words / flags
    bool help = false;                                 // help / --help / -h
    bool version = false;                              // version / --version
    bool ok = true;                                    // false => invocation error
    std::string error;                                 // diagnostic text when !ok
};

// Recognised option keys (CONFIG knobs and per-verb options).
bool is_known_option_key(const std::string& key);

// Parse argv (after the program name) into ParsedArgs. Accepts key=value
// options in any position. version/help are bare-word verbs; --version,
// --help, -h are tolerated aliases. Any other --flag or unknown bare word is
// an invocation error.
ParsedArgs parse_args(const std::vector<std::string>& args);

// Build a Config from parsed options. Returns nullopt and sets `err` when an
// option value is invalid (unknown enum value, etc.).
std::optional<Config> build_config(const ParsedArgs& parsed, std::string& err);

}  // namespace zd
