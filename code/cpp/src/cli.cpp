// generated from spec: zypper-declarative.spec.md sha256:1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e
#include "cli.hpp"

#include <iostream>
#include <set>
#include <sstream>

#include "commands.hpp"
#include "meta.hpp"

namespace zd {

std::string version_text() {
    std::ostringstream o;
    o << kProgramName << " " << kVersion << " spec:" << kSpecSha256 << "\n";
    return o.str();
}

std::string usage_text() {
    std::ostringstream o;
    o << "usage: zypper-declarative <verb> [key=value ...]\n"
      << "\n"
      << "verbs:\n"
      << "  apply      converge the system to the desired manifest\n"
      << "  diff       print what apply would change (dry run)\n"
      << "  verify     check the actual state against a reference\n"
      << "  status     print the current declarative state\n"
      << "  describe   emit the actual state as a manifest\n"
      << "  init       adopt the current state as the managed baseline\n"
      << "  version    print version and spec hash\n"
      << "  help       print this usage\n"
      << "\n"
      << "options (key=value; precede or follow the verb):\n"
      << "  mode=auto|external|internal     transaction binding (default auto)\n"
      << "  manifest-path=<path>            desired/reference manifest\n"
      << "  state-path=<path>               captured actual state (verify, diff)\n"
      << "  format=json|yaml                serialisation for this invocation\n"
      << "  root=<path>                     root to describe (default /)\n"
      << "  out=<path>                      describe output file (default stdout)\n"
      << "  on-unreadable=error|warn        describe unreadable-source handling\n"
      << "  scope=etc|full                  describe/verify read scope\n";
    return o.str();
}

int run_cli(const std::vector<std::string>& args, const CommandRunner& runner) {
    // Separate bare words (verbs / global commands) from key=value options.
    Invocation inv;
    std::vector<std::string> bare;
    bool unknown_option_form = false;
    std::string offending;

    for (const auto& a : args) {
        auto eq = a.find('=');
        if (eq != std::string::npos && eq > 0 && a[0] != '-') {
            inv.options[a.substr(0, eq)] = a.substr(eq + 1);
        } else if (a == "--version" || a == "--help" || a == "-h") {
            bare.push_back(a);  // tolerated global aliases
        } else if (!a.empty() && a[0] == '-') {
            // POSIX --flag style is not used for options.
            unknown_option_form = true;
            offending = a;
        } else {
            bare.push_back(a);
        }
    }

    // Global commands first (handled by the dispatcher, not behaviours).
    for (const auto& b : bare) {
        if (b == "version" || b == "--version") {
            std::cout << version_text();
            return 0;
        }
        if (b == "help" || b == "--help" || b == "-h") {
            std::cout << usage_text();
            return 0;
        }
    }

    // Bare invocation (no verb at all): print usage to stdout, exit 0.
    if (bare.empty() && inv.options.empty()) {
        std::cout << usage_text();
        return 0;
    }

    // An unknown option-form flag (POSIX --flag style for a non-global option)
    // is an invocation error.
    if (unknown_option_form && bare.empty()) {
        std::cerr << "error: invocation: unrecognised argument " << offending << "\n";
        std::cerr << usage_text();
        return 2;
    }

    // The first bare word is the verb.
    if (bare.empty()) {
        // Only options given but no verb -> invocation error unless it's a help
        // discovery. A bad format value (e.g. `format=bad`) reaches a verb only
        // when a verb is present; with no verb we validate the option globally.
        // Validate any format= value here so `format=bad_value` -> exit 2.
        auto it = inv.options.find("format");
        if (it != inv.options.end() && it->second != "json" && it->second != "yaml") {
            std::cerr << usage_text();
            return 2;
        }
        // No verb: treat as discovery, print usage to stdout, exit 0.
        std::cout << usage_text();
        return 0;
    }

    inv.verb = bare.front();
    if (bare.size() > 1) {
        // Extra bare words are not expected for any verb (verbs take key=value).
        std::cerr << usage_text();
        return 2;
    }

    if (unknown_option_form) {
        std::cerr << usage_text();
        return 2;
    }

    if (inv.verb == "apply")    return cmd_apply(inv, runner);
    if (inv.verb == "diff")     return cmd_diff(inv, runner);
    if (inv.verb == "verify")   return cmd_verify(inv, runner);
    if (inv.verb == "status")   return cmd_status(inv, runner);
    if (inv.verb == "describe") return cmd_describe(inv, runner);
    if (inv.verb == "init")     return cmd_init(inv, runner);

    // Unknown verb.
    std::cerr << usage_text();
    return 2;
}

}  // namespace zd
