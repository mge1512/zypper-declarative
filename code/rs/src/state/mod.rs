// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// BEHAVIOR/INTERNAL: describe-actual-state — the single live-state reader.
// Reads the actual state of the four declarable scopes under a given root and
// returns a Manifest in the shared schema. Every verb that needs actual state
// obtains it through this behaviour (or a supplied dump). No other module reads
// live system state.

pub mod configfiles;
pub mod fullscan;
pub mod packages;
pub mod repos;
pub mod services;

use crate::config::{OnUnreadable, Scope};
use crate::error::{Diagnostic, Domain};
use crate::interfaces::CommandRunner;
use crate::manifest::{
    ChangedManagedFilesScope, ConfigFilesScope, Manifest, PackagesScope, RepositoriesScope,
    ServicesScope, UnmanagedFilesScope,
};
use std::collections::HashSet;

pub struct DescribeOptions<'a> {
    pub root: String,
    pub on_unreadable: OnUnreadable,
    pub scope: Scope,
    pub keep_list: HashSet<String>,
    pub content_store: Option<String>,
    pub created_at: String,
    pub runner: &'a dyn CommandRunner,
}

pub struct DescribeResult {
    pub manifest: Manifest,
    pub diagnostics: Vec<Diagnostic>,
}

/// describe-actual-state: read packages, repositories, services, config_files
/// (and, under scope=full, the two observational scopes).
pub fn describe_actual_state(opts: &DescribeOptions) -> Result<DescribeResult, Diagnostic> {
    let mut manifest = Manifest::new_actual(opts.created_at.clone());
    let mut diagnostics: Vec<Diagnostic> = Vec::new();
    let strict = matches!(opts.on_unreadable, OnUnreadable::Error);

    // 1. packages
    match packages::read_packages(opts.runner, &opts.root) {
        packages::PackagesResult::Records(recs) => {
            if !recs.is_empty() {
                let mut scope: PackagesScope = PackagesScope::with_attr("package_system", "rpm");
                scope.elements = recs;
                manifest.packages = Some(scope);
            }
        }
        packages::PackagesResult::Unreadable(src) => {
            handle_unreadable(strict, Domain::Packages, &src, &mut diagnostics)?;
        }
    }

    // 2. repositories
    match repos::read_repositories(&opts.root) {
        repos::ReposResult::Records(recs) => {
            if !recs.is_empty() {
                let mut scope: RepositoriesScope =
                    RepositoriesScope::with_attr("repository_system", "zypp");
                scope.elements = recs;
                manifest.repositories = Some(scope);
            }
        }
        repos::ReposResult::Unreadable(src) => {
            handle_unreadable(strict, Domain::Repositories, &src, &mut diagnostics)?;
        }
    }

    // 3. services
    match services::read_services(opts.runner, &opts.root) {
        services::ServicesResult::Records(recs) => {
            if !recs.is_empty() {
                let mut scope: ServicesScope = ServicesScope::with_attr("init_system", "systemd");
                scope.elements = recs;
                manifest.services = Some(scope);
            }
        }
        services::ServicesResult::Unreadable(src) => {
            handle_unreadable(strict, Domain::Services, &src, &mut diagnostics)?;
        }
    }

    // 4. config_files
    match configfiles::read_config_files(
        opts.runner,
        &opts.root,
        &opts.on_unreadable,
        &opts.keep_list,
        opts.content_store.as_deref(),
    ) {
        Ok(output) => {
            for d in output.diagnostics {
                diagnostics.push(Diagnostic::warning(Domain::Files, d));
            }
            if !output.records.is_empty() {
                let mut scope: ConfigFilesScope = ConfigFilesScope::default();
                scope.elements = output.records;
                manifest.config_files = Some(scope);
            }
        }
        Err(configfiles::ConfigFilesError::Unreadable(src)) => {
            // read_config_files only returns Err under on_unreadable=error.
            return Err(Diagnostic::error(
                Domain::Files,
                format!("unreadable source {}", src),
            ));
        }
    }

    // 4a. full-scan integrity
    if matches!(opts.scope, Scope::Full) {
        match fullscan::full_scan(
            opts.runner,
            &opts.root,
            &opts.on_unreadable,
            &opts.keep_list,
        ) {
            Ok(fs) => {
                for d in fs.diagnostics {
                    diagnostics.push(Diagnostic::warning(Domain::Files, d));
                }
                if !fs.changed.is_empty() {
                    let mut scope: ChangedManagedFilesScope = ChangedManagedFilesScope::default();
                    scope.elements = fs.changed;
                    manifest.changed_managed_files = Some(scope);
                }
                if !fs.unmanaged.is_empty() {
                    let mut scope: UnmanagedFilesScope = UnmanagedFilesScope::default();
                    scope.elements = fs.unmanaged;
                    manifest.unmanaged_files = Some(scope);
                }
            }
            Err(fullscan::FullScanError::Unreadable(src)) => {
                return Err(Diagnostic::error(
                    Domain::Files,
                    format!("unreadable source {}", src),
                ));
            }
        }
    }

    Ok(DescribeResult {
        manifest,
        diagnostics,
    })
}

fn handle_unreadable(
    strict: bool,
    domain: Domain,
    source: &str,
    diagnostics: &mut Vec<Diagnostic>,
) -> Result<(), Diagnostic> {
    if strict {
        Err(Diagnostic::error(
            domain,
            format!("unreadable source {}", source),
        ))
    } else {
        diagnostics.push(Diagnostic::warning(
            domain,
            format!("omitting unreadable source {}", source),
        ));
        Ok(())
    }
}
