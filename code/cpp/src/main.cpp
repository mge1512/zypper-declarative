// generated from spec: zypper-declarative.spec.md sha256:1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e
//
// Entry point: signal handling and dispatch only. All behaviour lives in the
// implementation modules; this file performs no spec behaviour itself.
#include <csignal>
#include <string>
#include <vector>

#include "cli.hpp"
#include "command_runner.hpp"

namespace {
// Clean exit on SIGTERM / SIGINT: no partial output. For the read-only verbs a
// default termination is clean; the apply path discards its transaction on
// interruption (handled inside the transaction module on a live host).
extern "C" void on_signal(int) {
    // async-signal-safe: terminate without flushing partial buffered output.
    _exit(130);
}
}  // namespace

int main(int argc, char** argv) {
    std::signal(SIGTERM, on_signal);
    std::signal(SIGINT, on_signal);

    std::vector<std::string> args;
    for (int i = 1; i < argc; ++i) args.emplace_back(argv[i]);

    zd::OSCommandRunner runner;
    return zd::run_cli(args, runner);
}
