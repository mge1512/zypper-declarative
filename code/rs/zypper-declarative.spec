# pcd-spec-sha256: 51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
Name:           zypper-declarative
Version:        0.6.6
Release:        0
Summary:        Declarative convergence of SUSE system state in a snapshot transaction
License:        GPL-2.0-or-later
URL:            https://github.com/mge1512/zypper-declarative
Source0:        %{name}-%{version}.tar.gz
BuildRequires:  cargo
BuildRequires:  rust
BuildRequires:  pandoc
ExclusiveArch:  x86_64

%description
zypper-declarative converges a SUSE system (packages, repositories, services,
and /etc config files) to a desired manifest inside a single snapshot
transaction, recording what was applied. It internalises the SUSE Machinery
system-description capability (describe), computes intent diffs and drift, and
verifies the converged tree. Surfaced as the `zypper declarative` subcommand and
invocable directly. Single static binary, no runtime dependencies of its own.

%prep
%autosetup -n %{name}-%{version}

%build
# Offline, vendored build (no network fetch at build time).
cargo build --release --offline
cp target/x86_64-unknown-linux-gnu/release/%{name} ./%{name}
pandoc %{name}.1.md -s -t man -o %{name}.1

%install
install -D -m 0755 ./%{name} %{buildroot}%{_bindir}/%{name}
install -D -m 0644 %{name}.1 %{buildroot}%{_mandir}/man1/%{name}.1
# zypper subcommand surface
install -d %{buildroot}%{_prefix}/lib/zypper/commands
ln -sf %{_bindir}/%{name} %{buildroot}%{_prefix}/lib/zypper/commands/zypper-declarative

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%{_prefix}/lib/zypper/commands/zypper-declarative
%{_mandir}/man1/%{name}.1*

%changelog
* Tue Jun 02 2026 Matthias G. Eckermann <pcd@mailbox.org> - 0.6.6-0
- Generated from spec zypper-declarative.spec.md (sha256 51284526...dd03).
