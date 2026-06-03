// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#include "cli.hpp"

#include <csignal>
#include <iostream>
#include <string>
#include <vector>

namespace {
volatile std::sig_atomic_t g_stop = 0;
extern "C" void on_signal(int) { g_stop = 1; }
}  // namespace

int main(int argc, char** argv) {
    // Clean exit on SIGTERM/SIGINT; an interrupted apply discards its
    // transaction (the transaction is sealed only after all convergence
    // succeeds, so any signal before sealing leaves nothing as the boot
    // default).
    std::signal(SIGTERM, on_signal);
    std::signal(SIGINT, on_signal);

    std::vector<std::string> args;
    for (int i = 1; i < argc; ++i) {
        std::string a = argv[i];
        // When surfaced as `zypper declarative ...`, zypper invokes the command
        // with the subcommand name stripped; when invoked directly the first
        // arg is already the verb. Tolerate a leading "declarative" token.
        if (i == 1 && a == "declarative") continue;
        args.push_back(a);
    }

    return zd::dispatch(args, std::cout, std::cerr);
}
