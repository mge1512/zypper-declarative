// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03

#include "cli.hpp"

#include <iostream>
#include <map>
#include <string>

#include "commands.hpp"
#include "config.hpp"
#include "meta.hpp"

namespace zd {

namespace {

void print_usage(std::ostream& os) {
    os << "usage: zypper-declarative <verb> [key=value ...]\n"
       << "\n"
       << "verbs:\n"
       << "  apply      converge the system to the desired manifest\n"
       << "  diff       print what apply would change (dry run)\n"
       << "  verify     check the actual state against a reference\n"
       << "  status     print the current declarative state\n"
       << "  describe   emit the actual state as a manifest\n"
       << "\n"
       << "global commands:\n"
       << "  version    print program name, version, and spec hash\n"
       << "  help       print this usage\n"
       << "\n"
       << "key=value options (precede any bare-word argument):\n"
       << "  mode=auto|external|internal\n"
       << "  manifest-path=<path>   format=json|yaml   state-path=<path>\n"
       << "  root=<path>            out=<path>         scope=etc|full\n"
       << "  on-unreadable=error|warn\n"
       << "  manifest-format=json|yaml   repo-lock=<repo>\n"
       << "  content-store=<path>   keep-list=<path>\n"
       << "  signature-verification=on|off   keyring=<path>\n"
       << "  activation-policy=reboot|soft-reboot|none   applied-root=<path>\n";
}

void print_version(std::ostream& os) {
    os << kProgramName << " " << kVersion << " spec:" << kSpecSha256 << "\n";
}

bool parse_format(const std::string& v, ManifestFormat& out) {
    if (v == "json") { out = ManifestFormat::Json; return true; }
    if (v == "yaml") { out = ManifestFormat::Yaml; return true; }
    return false;
}

// Apply a single key=value option to cfg. Returns false on an unknown key or
// an invalid value (invocation error).
bool apply_option(const std::string& key, const std::string& val, Config& cfg) {
    if (key == "mode" || key == "transaction-mode") {
        if (val == "auto") cfg.transaction_mode = TransactionMode::Auto;
        else if (val == "external") cfg.transaction_mode = TransactionMode::External;
        else if (val == "internal") cfg.transaction_mode = TransactionMode::Internal;
        else return false;
    } else if (key == "manifest-path") {
        cfg.manifest_path = val;
        cfg.manifest_path_given = true;
    } else if (key == "format") {
        ManifestFormat f;
        if (!parse_format(val, f)) return false;
        cfg.explicit_format = f;
    } else if (key == "manifest-format") {
        if (!parse_format(val, cfg.manifest_format)) return false;
    } else if (key == "state-path") {
        cfg.state_path = val;
        cfg.state_path_given = true;
    } else if (key == "root") {
        cfg.root = val;
    } else if (key == "out") {
        cfg.out = val;
    } else if (key == "on-unreadable") {
        if (val == "error") cfg.on_unreadable_error = true;
        else if (val == "warn") cfg.on_unreadable_error = false;
        else return false;
    } else if (key == "scope") {
        if (val == "etc") cfg.scope = ScanScope::Etc;
        else if (val == "full") cfg.scope = ScanScope::Full;
        else return false;
    } else if (key == "repo-lock") {
        cfg.repo_lock = val;
    } else if (key == "content-store") {
        cfg.content_store = val;
    } else if (key == "keep-list") {
        cfg.keep_list = val;
    } else if (key == "signature-verification") {
        if (val == "on") cfg.signature_verification = true;
        else if (val == "off") cfg.signature_verification = false;
        else return false;
    } else if (key == "keyring") {
        cfg.keyring = val;
    } else if (key == "activation-policy") {
        if (val != "reboot" && val != "soft-reboot" && val != "none")
            return false;
        cfg.activation_policy = val;
    } else if (key == "applied-root") {
        cfg.applied_root = val;
    } else {
        return false;  // unknown option key
    }
    return true;
}

}  // namespace

int dispatch(const std::vector<std::string>& argv,
             const CommandRunner& runner) {
    // Scan for the verb (first bare word) and tolerated flag aliases. Options
    // may appear in any position. Unknown option/value/missing value -> exit 2.
    std::string verb;
    Config cfg;
    bool scope_given = false;

    // First pass: handle tolerated global flag aliases, regardless of position.
    for (const auto& a : argv) {
        if (a == "--version") { print_version(std::cout); return 0; }
        if (a == "--help" || a == "-h") { print_usage(std::cout); return 0; }
    }

    for (const auto& a : argv) {
        auto eq = a.find('=');
        if (eq != std::string::npos && eq > 0 && a[0] != '-') {
            std::string key = a.substr(0, eq);
            std::string val = a.substr(eq + 1);
            if (!apply_option(key, val, cfg)) {
                print_usage(std::cerr);
                std::cerr << "error: [invocation] unknown or invalid option: "
                          << a << "\n";
                return 2;
            }
            if (key == "scope") scope_given = true;
            continue;
        }
        // a bare word
        if (a.rfind("--", 0) == 0 || (a.size() > 1 && a[0] == '-')) {
            // a flag form that is not a tolerated alias and not key=value
            print_usage(std::cerr);
            std::cerr << "error: [invocation] unknown argument: " << a << "\n";
            return 2;
        }
        if (a == "version") { print_version(std::cout); return 0; }
        if (a == "help") { print_usage(std::cout); return 0; }
        if (verb.empty()) {
            verb = a;
        } else {
            print_usage(std::cerr);
            std::cerr << "error: [invocation] unexpected argument: " << a
                      << "\n";
            return 2;
        }
    }

    if (verb.empty()) {
        // bare invocation: discovery action, usage to stdout, exit 0.
        print_usage(std::cout);
        return 0;
    }

    // scope is accepted only on describe and verify.
    if (scope_given && verb != "describe" && verb != "verify") {
        print_usage(std::cerr);
        std::cerr << "error: [invocation] scope= is accepted only on describe "
                     "and verify\n";
        return 2;
    }

    if (verb == "apply") return cmd_apply(cfg, runner);
    if (verb == "diff") return cmd_diff(cfg, runner);
    if (verb == "verify") return cmd_verify(cfg, runner);
    if (verb == "status") return cmd_status(cfg, runner);
    if (verb == "describe") return cmd_describe(cfg, runner);

    print_usage(std::cerr);
    std::cerr << "error: [invocation] unknown verb: " << verb << "\n";
    return 2;
}

}  // namespace zd
