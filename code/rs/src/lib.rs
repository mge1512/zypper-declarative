// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Library root: module declarations and re-exports. The implementation lives in
// these modules; main.rs is a thin entry point that only parses argv, installs
// signal handlers, and dispatches into cli::run.

pub mod cli;
pub mod config;
pub mod converge;
pub mod diff;
pub mod error;
pub mod interfaces;
pub mod manifest;
pub mod meta;
pub mod record;
pub mod state;
pub mod txn;
