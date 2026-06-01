// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// cli.cpp -- dispatch, key=value parsing, global commands, option validation.
#include "cli.hpp"
#include "config.hpp"
#include "commands.hpp"
#include "command_runner.hpp"
#include "system_reader.hpp"
#include "meta.hpp"

#include <iostream>
#include <sstream>
#include <map>
#include <set>
#include <cstdlib>

namespace zd {

void debug_log(const std::string& msg) {
    const char* d = std::getenv("ZYPPER_DECLARATIVE_DEBUG");
    if (d && std::string(d) == "1")
        std::cerr << "DEBUG: " << msg << "\n";
}

std::string usage_text() {
    std::ostringstream os;
    os << "usage: zypper-declarative <verb> [key=value ...]\n"
       << "       zypper declarative <verb> [key=value ...]\n\n"
       << "verbs:\n"
       << "  apply       converge the system to the desired manifest\n"
       << "  diff        print what apply would change (no modification)\n"
       << "  verify      check the system against a reference declaration\n"
       << "  status      print the current declarative state\n"
       << "  describe    emit the actual state as a manifest\n\n"
       << "global commands:\n"
       << "  version     print program name, version, and spec hash (exit 0)\n"
       << "  help        print this usage (exit 0)\n\n"
       << "options (key=value; accepted in any position):\n"
       << "  mode=auto|external|internal      transaction binding (default auto)\n"
       << "  manifest-path=<path>             desired/reference manifest\n"
       << "  format=json|yaml                 serialisation for this invocation\n"
       << "  state-path=<path>                captured actual state (offline)\n"
       << "  root=<path>                      root to describe (default /)\n"
       << "  out=<path>                       describe output file (default stdout)\n"
       << "  on-unreadable=error|warn         describe unreadable-scope handling\n"
       << "  scope=etc|full                   describe/verify read scope\n"
       << "  manifest-format=json|yaml        default serialisation\n"
       << "  repo-lock=<repo>                 fallback pinned repository\n"
       << "  content-store=<path>             base path for content_ref\n"
       << "  keep-list=<path>                 allowlist of persistent paths\n"
       << "  signature-verification=on|off    manifest signature checking\n"
       << "  keyring=<path>                   keyring for signature verification\n"
       << "  activation-policy=reboot|soft-reboot|none\n"
       << "  applied-root=<path>              generation root for the applied record\n";
    return os.str();
}

static void print_version() {
    std::cout << kProgramName << " " << kVersion
              << " spec:" << kSpecSha256 << "\n";
}

// Parse a format value; returns false on unknown.
static bool parse_format(const std::string& v, ManifestFormat& out) {
    if (v == "json") { out = ManifestFormat::Json; return true; }
    if (v == "yaml") { out = ManifestFormat::Yaml; return true; }
    return false;
}

// usage error helper (stderr, exit 2)
static int usage_error(const std::string& why) {
    std::cerr << "error: invocation: " << why << "\n";
    std::cerr << usage_text();
    return 2;
}

// The recognised verbs.
static const std::set<std::string>& verbs() {
    static const std::set<std::string> v = {"apply", "diff", "verify", "status", "describe"};
    return v;
}

// The recognised option keys.
static const std::set<std::string>& option_keys() {
    static const std::set<std::string> k = {
        "mode", "manifest-path", "format", "state-path", "root", "out",
        "on-unreadable", "scope", "manifest-format", "repo-lock", "content-store",
        "keep-list", "signature-verification", "keyring", "activation-policy",
        "applied-root"
    };
    return k;
}

int run(const std::vector<std::string>& args) {
    // Separate the verb (first bare word) from key=value options. Options may
    // appear in any position (before or after the verb).
    std::string verb;
    std::vector<std::pair<std::string, std::string>> opts;

    // First pass: detect global commands and their flag aliases anywhere.
    for (const auto& a : args) {
        if (a == "version" || a == "--version") { print_version(); return 0; }
        if (a == "help" || a == "--help" || a == "-h") {
            std::cout << usage_text();
            return 0;
        }
    }

    // Second pass: collect verb and options.
    bool verb_seen = false;
    for (const auto& a : args) {
        auto eq = a.find('=');
        if (eq != std::string::npos) {
            std::string key = a.substr(0, eq);
            std::string val = a.substr(eq + 1);
            if (option_keys().find(key) == option_keys().end())
                return usage_error("unknown option '" + key + "'");
            if (val.empty())
                return usage_error("missing value for option '" + key + "'");
            // Validate the value eagerly so a bad value is an invocation error
            // even before (or without) a verb, per the spec CLI contract.
            if (key == "format" || key == "manifest-format") {
                ManifestFormat f;
                if (!parse_format(val, f))
                    return usage_error("unknown " + key + " value '" + val + "'");
            } else if (key == "mode") {
                if (val != "auto" && val != "external" && val != "internal")
                    return usage_error("unknown value '" + val + "' for mode");
            } else if (key == "on-unreadable") {
                if (val != "error" && val != "warn")
                    return usage_error("unknown value '" + val + "' for on-unreadable");
            } else if (key == "scope") {
                if (val != "etc" && val != "full")
                    return usage_error("unknown value '" + val + "' for scope");
            } else if (key == "signature-verification") {
                if (val != "on" && val != "off")
                    return usage_error("unknown value '" + val + "' for signature-verification");
            } else if (key == "activation-policy") {
                if (val != "reboot" && val != "soft-reboot" && val != "none")
                    return usage_error("unknown value '" + val + "' for activation-policy");
            }
            opts.emplace_back(key, val);
        } else if (a.rfind("--", 0) == 0 || (a.size() >= 1 && a[0] == '-')) {
            // a POSIX-style flag that is not a tolerated global alias
            return usage_error("unknown option '" + a + "' (options are key=value)");
        } else {
            if (!verb_seen) { verb = a; verb_seen = true; }
            else return usage_error("unexpected argument '" + a + "'");
        }
    }

    // Bare invocation (no verb, no options consumed) -> usage to stdout, exit 0.
    if (!verb_seen) {
        std::cout << usage_text();
        return 0;
    }

    if (verbs().find(verb) == verbs().end())
        return usage_error("unknown verb '" + verb + "'");

    // Build the resolved Config from the options.
    Config cfg;
    for (const auto& [key, val] : opts) {
        if (key == "mode") {
            if (val == "auto") cfg.transaction_mode = TransactionMode::Auto;
            else if (val == "external") cfg.transaction_mode = TransactionMode::External;
            else if (val == "internal") cfg.transaction_mode = TransactionMode::Internal;
            else return usage_error("unknown value '" + val + "' for mode");
        } else if (key == "manifest-path") {
            cfg.manifest_path = val;
        } else if (key == "format") {
            ManifestFormat f;
            if (!parse_format(val, f)) return usage_error("unknown format value '" + val + "'");
            cfg.explicit_format = f;
        } else if (key == "manifest-format") {
            ManifestFormat f;
            if (!parse_format(val, f)) return usage_error("unknown manifest-format value '" + val + "'");
            cfg.manifest_format = f;
        } else if (key == "state-path") {
            cfg.state_path = val;
        } else if (key == "root") {
            cfg.root = val;
        } else if (key == "out") {
            cfg.out = val;
        } else if (key == "on-unreadable") {
            if (val == "error") cfg.on_unreadable = OnUnreadable::Error;
            else if (val == "warn") cfg.on_unreadable = OnUnreadable::Warn;
            else return usage_error("unknown value '" + val + "' for on-unreadable");
        } else if (key == "scope") {
            if (val == "etc") cfg.scope = ScanScope::Etc;
            else if (val == "full") cfg.scope = ScanScope::Full;
            else return usage_error("unknown value '" + val + "' for scope");
        } else if (key == "repo-lock") {
            cfg.repo_lock = val;
        } else if (key == "content-store") {
            cfg.content_store = val;
        } else if (key == "keep-list") {
            cfg.keep_list = val;
        } else if (key == "signature-verification") {
            if (val == "on") cfg.signature_verification = true;
            else if (val == "off") cfg.signature_verification = false;
            else return usage_error("unknown value '" + val + "' for signature-verification");
        } else if (key == "keyring") {
            cfg.keyring = val;
        } else if (key == "activation-policy") {
            if (val != "reboot" && val != "soft-reboot" && val != "none")
                return usage_error("unknown value '" + val + "' for activation-policy");
            cfg.activation_policy = val;
        } else if (key == "applied-root") {
            cfg.applied_root = val;
        }
    }

    // scope is accepted only on describe and verify.
    bool scope_given = false;
    for (const auto& [k, v] : opts) { (void)v; if (k == "scope") scope_given = true; }
    if (scope_given && verb != "describe" && verb != "verify")
        return usage_error("scope is accepted only on describe and verify");

    OSCommandRunner runner;
    ZyppSystemReader zr;
    // Use the live SystemReader for the real root "/"; for a synthetic test root
    // pass nullptr so package ownership is treated as unpackaged (offline).
    bool live_root = (cfg.root == "/" || cfg.root.empty());
    const SystemReader* reader = live_root ? static_cast<const SystemReader*>(&zr) : nullptr;

    if (verb == "apply")    return cmd_apply(cfg, runner, reader);
    if (verb == "diff")     return cmd_diff(cfg, runner, reader);
    if (verb == "verify")   return cmd_verify(cfg, runner, reader);
    if (verb == "status")   return cmd_status(cfg, runner, reader);
    if (verb == "describe") return cmd_describe(cfg, runner, reader);
    return usage_error("unknown verb '" + verb + "'");
}

} // namespace zd
