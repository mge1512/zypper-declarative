# pcd-spec-sha256: 714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
#
# OBS RPM spec for zypper-declarative (cli-tool, Go, single static binary).

Name:           zypper-declarative
Version:        0.4.0
Release:        0
Summary:        Reconciling converger for declarative SUSE system state
License:        GPL-2.0-or-later
Group:          System/Management
URL:            https://github.com/mge1512/zypper-declarative
Source0:        %{name}-%{version}.tar.gz
BuildRequires:  go >= 1.21
BuildRequires:  pandoc
BuildRequires:  make
Requires:       zypper
Requires:       snapper
Requires:       systemd
ExclusiveArch:  x86_64 aarch64

%description
zypper-declarative converges a SUSE system to a desired manifest inside a
single snapshot transaction and records what was applied. The manifest is the
declarable subset of the SUSE Machinery system description (packages,
repositories, services, config_files), serialised as canonical JSON
(format_version 1) or, optionally, YAML. It is surfaced as a zypper subcommand
and is also invokable directly. Built as a single static binary with no runtime
dependencies of its own beyond the tools it drives.

%prep
%setup -q

%build
export CGO_ENABLED=0
export GOFLAGS=-mod=vendor
go build -o %{name} ./cmd/%{name}
pandoc %{name}.1.md -s -t man -o %{name}.1

%install
install -D -m 0755 %{name} %{buildroot}%{_bindir}/%{name}
install -D -m 0755 %{name} %{buildroot}%{_prefix}/lib/zypper/commands/zypper-declarative
install -D -m 0644 %{name}.1 %{buildroot}%{_mandir}/man1/%{name}.1

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%{_prefix}/lib/zypper/commands/zypper-declarative
%{_mandir}/man1/%{name}.1*

%changelog
* Fri May 29 2026 Matthias G. Eckermann <pcd@mailbox.org> - 0.4.0-0
- Generated from zypper-declarative.spec.md (PCD translator, full-spec build).
