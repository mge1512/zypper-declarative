// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Entry point. CLI dispatch wiring only: install signal handlers for a clean
// exit, collect argv, and call into the implementation. No behaviour is
// implemented here (per SOURCE-PARTITIONING: one-entry-one-implementation).

#include <csignal>
#include <cstdlib>
#include <string>
#include <vector>

#include "cli.hpp"
#include "command_runner.hpp"

extern "C" void zd_on_signal(int) {
    // Clean exit on SIGTERM/SIGINT with no partial output. Read-only verbs
    // hold no transaction; apply discards an in-flight transaction in its own
    // handler. _exit is async-signal-safe.
    _exit(130);
}

int main(int argc, char** argv) {
    std::signal(SIGTERM, zd_on_signal);
    std::signal(SIGINT, zd_on_signal);

    std::vector<std::string> args;
    for (int i = 1; i < argc; ++i) args.emplace_back(argv[i]);

    zd::OSCommandRunner runner;
    return zd::dispatch(args, runner);
}
