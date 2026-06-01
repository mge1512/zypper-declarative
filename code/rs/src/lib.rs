// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// Library crate root: re-exports the implementation modules so the entry point
// (and in-tree unit tests) can reach them. The entry point (src/main.rs)
// contains only CLI dispatch; all behaviour lives in these modules
// (SOURCE-PARTITIONING: one-entry-one-implementation).

pub mod clock;
pub mod cli;
pub mod config;
pub mod converge;
pub mod diff;
pub mod error;
pub mod interfaces;
pub mod load;
pub mod manifest;
pub mod meta;
pub mod record;
pub mod state;
pub mod txn;
