# pcd-spec-sha256: 18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
#
# OBS RPM spec for zypper-declarative (Rust, cli-tool deployment).

Name:           zypper-declarative
Version:        0.6.4
Release:        0
Summary:        Declarative reconciling converger for SUSE systems
License:        GPL-2.0-or-later
URL:            https://github.com/mge1512/zypper-declarative
Source0:        %{name}-%{version}.tar.xz

BuildRequires:  cargo
BuildRequires:  rust
BuildRequires:  pandoc
# Static binary; no runtime library dependency for the tool's own logic. The
# tool drives zypper, snapper, systemctl, and rpm at run time (Requires below).
Requires:       zypper
Requires:       snapper
Requires:       systemd
Requires:       rpm

%description
zypper-declarative converges a SUSE system toward a desired manifest inside a
single snapshot transaction, recording what was applied. It is surfaced as a
zypper subcommand (zypper declarative) and is also invokable directly. The
desired state is the declarable subset of the SUSE Machinery system description
(packages, repositories, services, config_files). The same shared model is
produced by describe (the actual state), stored as the applied record, and
consumed by apply, diff, and verify.

%prep
%autosetup -n %{name}-%{version}

%build
# Static linking against glibc via crt-static; built with an explicit target so
# host proc-macro crates compile without crt-static.
export RUSTFLAGS='-C target-feature=+crt-static'
cargo build --release --target x86_64-unknown-linux-gnu --offline
cp target/x86_64-unknown-linux-gnu/release/%{name} ./%{name}
# Man page.
pandoc %{name}.1.md -s -t man -o %{name}.1

%install
install -D -m 0755 %{name} %{buildroot}%{_bindir}/%{name}
install -D -m 0644 %{name}.1 %{buildroot}%{_mandir}/man1/%{name}.1
# Surface as a zypper subcommand: `zypper declarative` dispatches to this binary.
install -d %{buildroot}%{_prefix}/lib/zypper/commands
ln -sf %{_bindir}/%{name} %{buildroot}%{_prefix}/lib/zypper/commands/zypper-declarative

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%{_prefix}/lib/zypper/commands/zypper-declarative
%{_mandir}/man1/%{name}.1*

%changelog
* Mon Jun 01 2026 Matthias G. Eckermann <pcd@mailbox.org> - 0.6.4-0
- Translation from zypper-declarative.spec.md (spec sha256
  18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd).
