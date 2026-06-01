# pcd-spec-sha256: 87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
#
# OBS RPM spec for zypper-declarative. Distributed via build.opensuse.org; no
# curl-based installation. Builds a single static Rust binary.

Name:           zypper-declarative
Version:        0.6.3
Release:        0
Summary:        Declarative, reconciling converger surfaced as a zypper subcommand
License:        GPL-2.0-or-later
Group:          System/Management
URL:            https://github.com/mge1512/zypper-declarative
Source0:        %{name}-%{version}.tar.xz

BuildRequires:  rust
BuildRequires:  cargo
BuildRequires:  pandoc

ExclusiveArch:  x86_64

%description
zypper-declarative converges a SUSE system to a desired manifest (the declarable
subset of the SUSE Machinery system description: packages, repositories,
services, and /etc config files) inside a single snapshot transaction, recording
what was applied. It provides the verbs apply, diff, verify, status, and
describe, and is surfaced as a "zypper declarative" subcommand and invocable
directly. It performs no direct network I/O of its own; all package retrieval is
delegated to the package manager against declared, pinned, signed repositories.

%prep
%autosetup -n %{name}-%{version}

%build
# Static binary (BINARY-TYPE: static) via the project's .cargo/config.toml, which
# sets a default build target and crt-static. Built offline against vendored deps.
cargo build --release --offline
pandoc %{name}.1.md -s -t man -o %{name}.1

%install
install -D -m 0755 target/x86_64-unknown-linux-gnu/release/%{name} %{buildroot}%{_bindir}/%{name}
install -D -m 0644 %{name}.1 %{buildroot}%{_mandir}/man1/%{name}.1
# zypper subcommand surface: an executable under /usr/lib/zypper/commands.
install -D -m 0755 target/x86_64-unknown-linux-gnu/release/%{name} %{buildroot}%{_prefix}/lib/zypper/commands/%{name}

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%{_prefix}/lib/zypper/commands/%{name}
%{_mandir}/man1/%{name}.1*

%changelog
* Mon Jun 01 2026 Matthias G. Eckermann <pcd@mailbox.org> - 0.6.3-0
- Generated from zypper-declarative.spec.md (Version 0.6.3).
