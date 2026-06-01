// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// main.cpp -- entry point. Signal wiring (clean exit on SIGTERM/SIGINT) and a
// call into zd::run(). No behaviour is implemented here (one-entry-one-
// implementation: dispatch lives in cli.cpp, behaviour in the other modules).
#include "cli.hpp"

#include <csignal>
#include <vector>
#include <string>
#include <cstdlib>

namespace {
volatile std::sig_atomic_t g_stop = 0;
extern "C" void on_signal(int) {
    // Async-signal-safe clean exit: no partial output. A long-running apply
    // discards its transaction in the milestone that implements live converge.
    g_stop = 1;
    _exit(130);
}
}

int main(int argc, char** argv) {
    std::signal(SIGTERM, on_signal);
    std::signal(SIGINT, on_signal);

    std::vector<std::string> args;
    args.reserve(argc > 1 ? argc - 1 : 0);
    for (int i = 1; i < argc; ++i) args.emplace_back(argv[i]);

    return zd::run(args);
}
