// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
// tests by: claude-opus-4-8
//
// Test runner entry point. Runs all registered black-box tests against the
// built binary at ../../zypper-declarative (the cli-tool template's canonical
// BINARY-LOCATION relative to this test directory).
#include "test_harness.hpp"
#include <cstdlib>
#include <ctime>

int main() {
    std::srand(static_cast<unsigned>(std::time(nullptr)) ^ static_cast<unsigned>(::getpid()));
    return zdtest::run_all();
}
