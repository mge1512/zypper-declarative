// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
// tests by: gemini-3-5-flash
#include "test_utils.hpp"

// ### EXAMPLE: bare_invocation_shows_help
TEST_CASE(test_bare_invocation_shows_help) {
    auto res = run_command({"../../zypper-declarative"});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "usage:");
}

// ### EXAMPLE: version_verb_bare_word
TEST_CASE(test_version_verb_bare_word) {
    auto res = run_command({"../../zypper-declarative", "version"});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "zypper-declarative");
    ASSERT_CONTAINS(res.stdout_data, "spec:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3");
}

// ### EXAMPLE: version_flag_alias
TEST_CASE(test_version_flag_alias) {
    auto res = run_command({"../../zypper-declarative", "--version"});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "zypper-declarative");
    ASSERT_CONTAINS(res.stdout_data, "spec:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3");
}

// ### EXAMPLE: help_verb_bare_word
TEST_CASE(test_help_verb_bare_word) {
    auto res = run_command({"../../zypper-declarative", "help"});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "usage:");
}

// ### EXAMPLE: unknown_verb_rejected
TEST_CASE(test_unknown_verb_rejected) {
    auto res = run_command({"../../zypper-declarative", "frobnicate"});
    ASSERT_EQ(res.exit_code, 2);
    ASSERT_CONTAINS(res.stderr_data, "usage:");
}
