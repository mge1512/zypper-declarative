// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#include "config.hpp"

#include <set>

namespace zd {

static const std::set<std::string>& known_keys() {
    static const std::set<std::string> keys = {
        "transaction-mode", "mode", "manifest-path", "manifest-format", "format",
        "state-path", "root", "out", "on-unreadable", "scope", "repo-lock",
        "content-store", "keep-list", "signature-verification", "keyring",
        "activation-policy", "applied-root"};
    return keys;
}

bool is_known_option_key(const std::string& key) {
    return known_keys().count(key) > 0;
}

ParsedArgs parse_args(const std::vector<std::string>& args) {
    ParsedArgs p;
    for (const auto& a : args) {
        if (a == "--version") { p.version = true; continue; }
        if (a == "--help" || a == "-h") { p.help = true; continue; }

        auto eq = a.find('=');
        if (eq != std::string::npos && eq > 0 && a[0] != '-') {
            std::string key = a.substr(0, eq);
            std::string val = a.substr(eq + 1);
            if (!is_known_option_key(key)) {
                p.ok = false;
                p.error = "unknown option '" + key + "'";
                p.bad_args.push_back(a);
                continue;
            }
            p.options[key] = val;
            continue;
        }

        // Bare word: a verb, or an unrecognised token.
        if (!a.empty() && a[0] == '-') {
            // POSIX --flag style is forbidden for options.
            p.ok = false;
            p.error = "unknown argument '" + a + "'";
            p.bad_args.push_back(a);
            continue;
        }

        if (a == "version") { p.version = true; continue; }
        if (a == "help") { p.help = true; continue; }

        if (p.verb.empty()) {
            p.verb = a;
        } else {
            // A second bare word is not expected.
            p.ok = false;
            p.error = "unexpected argument '" + a + "'";
            p.bad_args.push_back(a);
        }
    }
    return p;
}

static std::optional<ManifestFormat> parse_format(const std::string& v) {
    if (v == "json") return ManifestFormat::Json;
    if (v == "yaml") return ManifestFormat::Yaml;
    return std::nullopt;
}

std::optional<Config> build_config(const ParsedArgs& parsed, std::string& err) {
    Config c;
    const auto& o = parsed.options;

    auto get = [&](const std::string& k) -> std::optional<std::string> {
        auto it = o.find(k);
        if (it == o.end()) return std::nullopt;
        return it->second;
    };

    // transaction mode: `mode` is an alias for `transaction-mode`.
    if (auto m = get("transaction-mode")) {
        if (*m == "auto") c.transaction_mode = TransactionMode::Auto;
        else if (*m == "external") c.transaction_mode = TransactionMode::External;
        else if (*m == "internal") c.transaction_mode = TransactionMode::Internal;
        else { err = "unknown transaction-mode value '" + *m + "'"; return std::nullopt; }
    }
    if (auto m = get("mode")) {
        if (*m == "auto") c.transaction_mode = TransactionMode::Auto;
        else if (*m == "external") c.transaction_mode = TransactionMode::External;
        else if (*m == "internal") c.transaction_mode = TransactionMode::Internal;
        else { err = "unknown mode value '" + *m + "'"; return std::nullopt; }
    }

    if (auto v = get("manifest-path")) c.manifest_path = *v;
    if (auto v = get("state-path")) c.state_path = *v;
    if (auto v = get("root")) c.root = *v;
    if (auto v = get("out")) c.out = *v;
    if (auto v = get("repo-lock")) c.repo_lock = *v;
    if (auto v = get("content-store")) c.content_store = *v;
    if (auto v = get("keep-list")) c.keep_list = *v;
    if (auto v = get("keyring")) c.keyring = *v;
    if (auto v = get("applied-root")) c.applied_root = *v;
    if (auto v = get("activation-policy")) c.activation_policy = *v;

    if (auto v = get("manifest-format")) {
        auto f = parse_format(*v);
        if (!f) { err = "unknown manifest-format value '" + *v + "'"; return std::nullopt; }
        c.manifest_format = *f;
    }
    if (auto v = get("format")) {
        auto f = parse_format(*v);
        if (!f) { err = "unknown format value '" + *v + "'"; return std::nullopt; }
        c.explicit_format = *f;
    }
    if (auto v = get("on-unreadable")) {
        if (*v == "error") c.on_unreadable = OnUnreadable::Error;
        else if (*v == "warn") c.on_unreadable = OnUnreadable::Warn;
        else { err = "unknown on-unreadable value '" + *v + "'"; return std::nullopt; }
    }
    if (auto v = get("scope")) {
        if (*v == "etc") c.scope = ScanScope::Etc;
        else if (*v == "full") c.scope = ScanScope::Full;
        else { err = "unknown scope value '" + *v + "'"; return std::nullopt; }
    }
    if (auto v = get("signature-verification")) {
        if (*v == "on") c.signature_verification = true;
        else if (*v == "off") c.signature_verification = false;
        else { err = "unknown signature-verification value '" + *v + "'"; return std::nullopt; }
    }

    return c;
}

}  // namespace zd
