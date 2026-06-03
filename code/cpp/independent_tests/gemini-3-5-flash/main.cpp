// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
// tests by: gemini-3-5-flash
#include "test_utils.hpp"
#include <iomanip>

int main() {
    std::cout << "==================================================" << std::endl;
    std::cout << "  zypper-declarative C++ Independent Test Suite   " << std::endl;
    std::cout << "  LLM Name: gemini-3-5-flash                      " << std::endl;
    std::cout << "  Spec-SHA256: aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3" << std::endl;
    std::cout << "==================================================" << std::endl;

    int passed = 0;
    int failed = 0;
    int skipped = 0;

    // Run each registered test case
    for (const auto& tc : g_test_cases) {
        size_t initial_results = g_test_results.size();
        
        try {
            tc.fn();
        } catch (const std::exception& e) {
            register_test_result(tc.name, "unknown", 0, true, false, std::string("Caught exception: ") + e.what());
        } catch (...) {
            register_test_result(tc.name, "unknown", 0, true, false, "Caught unknown exception");
        }

        size_t final_results = g_test_results.size();
        if (final_results == initial_results) {
            // No result was registered, meaning the test passed!
            g_test_results.push_back({tc.name, "", 0, false, false, "Pass"});
            passed++;
            std::cout << std::left << std::setw(60) << ("[ PASS ] " + tc.name) << "OK" << std::endl;
        } else {
            // Some result was registered
            bool has_fail = false;
            bool has_skip = false;
            std::string msg;
            for (size_t i = initial_results; i < final_results; ++i) {
                if (g_test_results[i].failed) has_fail = true;
                if (g_test_results[i].skipped) has_skip = true;
                if (!g_test_results[i].message.empty()) {
                    if (!msg.empty()) msg += "; ";
                    msg += g_test_results[i].message;
                }
            }
            if (has_fail) {
                failed++;
                std::cout << std::left << std::setw(60) << ("[ FAIL ] " + tc.name) << "FAILED: " << msg << std::endl;
            } else if (has_skip) {
                skipped++;
                std::cout << std::left << std::setw(60) << ("[ SKIP ] " + tc.name) << "SKIPPED: " << msg << std::endl;
            } else {
                passed++;
                std::cout << std::left << std::setw(60) << ("[ PASS ] " + tc.name) << "OK" << std::endl;
            }
        }
    }

    std::cout << "==================================================" << std::endl;
    std::cout << "  Test Suite Run Complete." << std::endl;
    std::cout << "  Passed:  " << passed << std::endl;
    std::cout << "  Failed:  " << failed << std::endl;
    std::cout << "  Skipped: " << skipped << std::endl;
    std::cout << "==================================================" << std::endl;

    return failed > 0 ? 1 : 0;
}
