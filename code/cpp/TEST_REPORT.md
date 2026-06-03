# TEST_REPORT.md

Spec-SHA256: aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
Spec-SHA256 (host): aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
Included-Specs: None

LLM-Name: gemini-3-5-flash
Mode: test-author
Deployment-Template: cli-tool.template.md v0.3.29

Preset-Resolution:
- System: none (system defaults only)
- User: none
- Project: none

Hints-Files-Read:
- cli-tool.cpp.milestones.hints.md (c6e80c18bbc4a726d99104a68456971d29d59a3d82eda00495b85acd7899ea9d)
- zypper-declarative.cpp.decisions.hints.md (fd815bece1004a16ecde42bf85aaee5d139a3c0504b5706eedd8aeccc79315b2)

Test-Compile-Gate: pass

Binary-Discovery-Path: ../../zypper-declarative
- Expected post-translation entry point source path: ../../src/main.cpp

Target Language Resolved: C++ (C++17)

## Tests Produced

| Test Function Name | File | Covered Spec Example / Behavior |
|---|---|---|
| `test_bare_invocation_shows_help` | `test_global.cpp` | `bare_invocation_shows_help` |
| `test_version_verb_bare_word` | `test_global.cpp` | `version_verb_bare_word` |
| `test_version_flag_alias` | `test_global.cpp` | `version_flag_alias` |
| `test_help_verb_bare_word` | `test_global.cpp` | `help_verb_bare_word` |
| `test_unknown_verb_rejected` | `test_global.cpp` | `unknown_verb_rejected` |
| `test_apply_no_op_when_converged` | `test_apply.cpp` | `apply_no_op_when_converged` |
| `test_apply_writes_and_deletes_etc_file` | `test_apply.cpp` | `apply_writes_and_deletes_etc_file` |
| `test_apply_absent_scope_unmanaged` | `test_apply.cpp` | `apply_absent_scope_unmanaged` |
| `test_apply_manifest_invalid` | `test_apply.cpp` | `apply_manifest_invalid` |
| `test_apply_manifest_unreadable` | `test_apply.cpp` | `apply_manifest_unreadable` |
| `test_apply_transaction_unavailable` | `test_apply.cpp` | `apply_transaction_unavailable` |
| `test_apply_package_failure_rolls_back` | `test_apply.cpp` | `apply_package_failure_rolls_back` |
| `test_apply_rejects_full_describe_dump` | `test_apply.cpp` | `apply_rejects_full_describe_dump` |
| `test_idempotent_second_apply` | `test_apply.cpp` | `idempotent_second_apply` |
| `test_diff_prints_plan` | `test_diff.cpp` | `diff_prints_plan` |
| `test_diff_manifest_unreadable` | `test_diff.cpp` | `diff_manifest_unreadable` |
| `test_diff_unchanged_machine_no_drift` | `test_diff.cpp` | `diff_unchanged_machine_no_drift` |
| `test_diff_offline_two_files` | `test_diff.cpp` | `diff_offline_two_files` |
| `test_verify_clean` | `test_verify.cpp` | `verify_clean` |
| `test_verify_against_external_state_dump` | `test_verify.cpp` | `verify_against_external_state_dump` |
| `test_verify_malformed_state_dump` | `test_verify.cpp` | `verify_malformed_state_dump` |
| `test_verify_detects_drift` | `test_verify.cpp` | `verify_detects_drift` |
| `test_verify_no_applied_record` | `test_verify.cpp` | `verify_no_applied_record` |
| `test_verify_default_scope_ignores_usr` | `test_verify.cpp` | `verify_default_scope_ignores_usr` |
| `test_verify_scope_full_detects_unmanaged_addition` | `test_verify.cpp` | `verify_scope_full_detects_unmanaged_addition` |
| `test_verify_scope_full_detects_modified_package_file` | `test_verify.cpp` | `verify_scope_full_detects_modified_package_file` |
| `test_verify_offline_manifest_and_state` | `test_verify.cpp` | `verify_offline_manifest_and_state` |
| `test_verify_offline_no_applied_record_ok` | `test_verify.cpp` | `verify_offline_no_applied_record_ok` |
| `test_verify_state_path_extension_yaml` | `test_verify.cpp` | `verify_state_path_extension_yaml` |
| `test_status_reports_generation` | `test_status.cpp` | `status_reports_generation` |
| `test_status_no_declaration` | `test_status.cpp` | `status_no_declaration` |
| `test_status_unknown_argument` | `test_status.cpp` | `status_unknown_argument` |
| `test_describe_emits_manifest` | `test_describe.cpp` | `describe_emits_manifest` |
| `test_describe_output_unwritable` | `test_describe.cpp` | `describe_output_unwritable` |
| `test_describe_bootstraps_desired_manifest` | `test_describe.cpp` | `describe_bootstraps_desired_manifest` |
| `test_describe_traverses_etc_subdirectories` | `test_describe.cpp` | `describe_traverses_etc_subdirectories` |
| `test_describe_records_symlink_verbatim` | `test_describe.cpp` | `describe_records_symlink_verbatim` |
| `test_describe_skips_special_file` | `test_describe.cpp` | `describe_skips_special_file` |
| `test_describe_config_files_bounded_to_etc` | `test_describe.cpp` | `describe_config_files_bounded_to_etc` |
| `test_describe_populates_content_store` | `test_describe.cpp` | `describe_populates_content_store` |
| `test_describe_without_content_store_is_readonly` | `test_describe.cpp` | `describe_without_content_store_is_readonly` |
| `test_scope_attributes_always_object` | `test_describe.cpp` | `scope_attributes_always_object` |
| `test_describe_scope_full_emits_observational_scopes` | `test_describe.cpp` | `describe_scope_full_emits_observational_scopes` |
| `test_describe_format_yaml` | `test_describe.cpp` | `describe_format_yaml` |
| `test_describe_unknown_format` | `test_describe.cpp` | `describe_unknown_format` |
| `test_describe_out_extension_yaml` | `test_describe.cpp` | `describe_out_extension_yaml` |
| `test_describe_out_extension_json` | `test_describe.cpp` | `describe_out_extension_json` |
| `test_describe_format_overrides_extension` | `test_describe.cpp` | `describe_format_overrides_extension` |
| `test_describe_repositories_from_reposd` | `test_describe.cpp` | `describe_repositories_from_reposd` |
| `test_describe_unreadable_scope_strict` | `test_describe.cpp` | `describe_unreadable_scope_strict` |
| `test_describe_unreadable_scope_warn` | `test_describe.cpp` | `describe_unreadable_scope_warn` |
| `test_describe_omits_genuinely_empty_scope` | `test_describe.cpp` | `describe_omits_genuinely_empty_scope` |
| `test_host_self_checks` | `test_describe.cpp` | `describe_suppresses_package_pristine_etc_file`, `describe_actual_state_omits_pristine`, `describe_symlink_and_target_judged_independently` |
| `test_intent_diff_yields_deletion` | `test_model.cpp` | `intent_diff_yields_deletion` |
| `test_drift_ignores_unmanaged_packaged_file` | `test_model.cpp` | `drift_ignores_unmanaged_packaged_file` |
| `test_drift_type_transition_is_modified` | `test_model.cpp` | `drift_type_transition_is_modified` |
| `test_yaml_manifest_accepted` | `test_model.cpp` | `yaml_manifest_accepted` |
| `test_yaml_format_identity_stable` | `test_model.cpp` | `yaml_format_identity_stable` |
| `test_yaml_unsafe_rejected` | `test_model.cpp` | `yaml_unsafe_rejected` |
| `test_describe_pristine_distro_symlink_suppressed` | `test_model.cpp` | `describe_pristine_distro_symlink_suppressed` |
| `test_describe_default_alternative_symlink_suppressed` | `test_model.cpp` | `describe_default_alternative_symlink_suppressed` |
| `test_describe_crypto_policies_symlinks_not_alternatives` | `test_model.cpp` | `describe_crypto_policies_symlinks_not_alternatives` |

## Ambiguities & Decisions

1. **Transaction Simulation**: Snapshot creation and live commit execution are environment-dependent. We structured these tests using unprivileged mock roots, and used honest standard skips (`TEST_SKIP`) on parts requiring actual live system privilege.
2. **Alternatives Classification**: As specified in decisions hints, alternatives symlinks are classified exclusively under `/etc/alternatives` or those in `/var/lib/alternatives`. This is covered through synthetic root tests that correctly classification non-alternative symlinks as pristine and suppresses them.
3. **YAML Strict Type Safety**: Unsafe expansions or arbitrary non-standard tags (`!!sh/test`) are tested to confirm they are explicitly rejected under the YAML parser's safe-loading profile.
