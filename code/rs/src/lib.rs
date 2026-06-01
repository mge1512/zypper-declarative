// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// Library root: declares the implementation modules so in-crate unit tests can
// reach them. The entry point (src/main.rs) contains only CLI dispatch.

pub mod cli;
pub mod config;
pub mod converge;
pub mod diff;
pub mod error;
pub mod format;
pub mod hash;
pub mod interfaces;
pub mod manifest;
pub mod meta;
pub mod record;
pub mod state;
pub mod txn;
pub mod types;
