// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#include "cli.hpp"
#include "config.hpp"
#include "verbs.hpp"
#include "command_runner.hpp"
#include "meta.hpp"

#include <set>

namespace zd {

void print_version(std::ostream& os) {
    os << "zypper-declarative " << ZD_VERSION << " spec:" << ZD_SPEC_SHA256 << "\n";
}

void print_usage(std::ostream& os) {
    os << "usage: zypper-declarative <verb> [key=value ...]\n"
       << "       (or: zypper declarative <verb> [key=value ...])\n"
       << "\n"
       << "verbs:\n"
       << "  apply      converge the system to the desired manifest in a snapshot\n"
       << "  diff       dry run: print what apply would change\n"
       << "  verify     check the actual state against a reference declaration\n"
       << "  status     print the current declarative state\n"
       << "  describe   read the actual state and emit it as a manifest\n"
       << "  init       adopt the current state as the managed baseline\n"
       << "  version    print version and embedded spec hash\n"
       << "  help       print this usage\n"
       << "\n"
       << "options (key=value, any position):\n"
       << "  mode=auto|external|internal      transaction binding\n"
       << "  manifest-path=<path>             desired/reference manifest\n"
       << "  state-path=<path>                captured actual state (offline)\n"
       << "  format=json|yaml                 serialisation for this invocation\n"
       << "  root=<path>                      describe root (default /)\n"
       << "  out=<path>                       describe/init output file\n"
       << "  on-unreadable=error|warn         unreadable-source handling\n"
       << "  scope=etc|full                   describe/verify read scope\n"
       << "  content-store=<path>             content store base path\n"
       << "  keep-list=<path>                 allowlist of undeclared paths\n"
       << "  applied-root=<path>              generation root for the applied record\n"
       << "\n"
       << "exit codes: 0 success  1 logical failure  2 invocation error\n";
}

int dispatch(const std::vector<std::string>& argv, std::ostream& out, std::ostream& err) {
    ParsedArgs parsed = parse_args(argv);

    // version/help global commands win (handled by the dispatcher).
    if (parsed.version) { print_version(out); return 0; }
    if (parsed.help) { print_usage(out); return 0; }

    if (!parsed.ok) {
        print_usage(err);
        err << "error: domain=invocation: " << parsed.error << "\n";
        return 2;
    }

    // Bare invocation (no verb): print usage to stdout, exit 0. But an invalid
    // option value (e.g. format=bad_value) is still an invocation error even
    // without a verb, so option values are validated first.
    {
        std::string cfgerr0;
        auto cfg0 = build_config(parsed, cfgerr0);
        if (!cfg0) {
            print_usage(err);
            err << "error: domain=invocation: " << cfgerr0 << "\n";
            return 2;
        }
    }

    if (parsed.verb.empty()) {
        print_usage(out);
        return 0;
    }

    const std::set<std::string> verbs = {"apply", "diff", "verify", "status", "describe", "init"};
    if (verbs.find(parsed.verb) == verbs.end()) {
        print_usage(err);
        err << "error: domain=invocation: unknown verb '" << parsed.verb << "'\n";
        return 2;
    }

    // scope is accepted only on describe and verify.
    if (parsed.options.count("scope") && parsed.verb != "describe" && parsed.verb != "verify") {
        print_usage(err);
        err << "error: domain=invocation: option 'scope' is not accepted for verb '"
            << parsed.verb << "'\n";
        return 2;
    }

    std::string cfgerr;
    auto cfg = build_config(parsed, cfgerr);
    if (!cfg) {
        print_usage(err);
        err << "error: domain=invocation: " << cfgerr << "\n";
        return 2;
    }

    OSCommandRunner runner;
    if (parsed.verb == "apply") return verb_apply(*cfg, runner, out, err);
    if (parsed.verb == "diff") return verb_diff(*cfg, runner, out, err);
    if (parsed.verb == "verify") return verb_verify(*cfg, runner, out, err);
    if (parsed.verb == "status") return verb_status(*cfg, runner, out, err);
    if (parsed.verb == "describe") return verb_describe(*cfg, runner, out, err);
    if (parsed.verb == "init") return verb_init(*cfg, runner, out, err);

    print_usage(err);
    return 2;
}

}  // namespace zd
