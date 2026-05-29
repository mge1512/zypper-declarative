# pcd-spec-sha256: b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
#
# OBS RPM spec for zypper-declarative.

Name:           zypper-declarative
Version:        0.6.0
Release:        0
Summary:        Declarative, reconciling converger surfaced as a zypper subcommand
License:        GPL-2.0-or-later
Group:          System/Management
URL:            https://github.com/mge1512/zypper-declarative
# Local tarball, not a URL (no network fetch at build time).
Source0:        %{name}-%{version}.tar.xz
BuildRequires:  go >= 1.23
BuildRequires:  pandoc
BuildRequires:  golang-packaging
BuildRoot:      %{_tmppath}/%{name}-%{version}-build

%description
zypper-declarative converges a SUSE system to a desired manifest (the
declarable subset of the SUSE Machinery system description: packages,
repositories, services, and /etc config files) inside a single snapshot
transaction, recording what was applied. It provides the verbs apply, diff,
verify, status, and describe. Idempotent and read-only except for apply.
Distributed via OBS; no curl-based installation.

%prep
%setup -q

%build
# Static binary, no runtime dependencies (RUNTIME-DEPS: none; BINARY-TYPE: static).
export CGO_ENABLED=0
go build -mod=vendor -o %{name} ./cmd/%{name}
pandoc %{name}.1.md -s -t man -o %{name}.1

%install
install -D -m 0755 %{name} %{buildroot}%{_bindir}/%{name}
# Surface as a zypper subcommand.
install -d %{buildroot}%{_prefix}/lib/zypper/commands
ln -sf %{_bindir}/%{name} %{buildroot}%{_prefix}/lib/zypper/commands/zypper-declarative
install -D -m 0644 %{name}.1 %{buildroot}%{_mandir}/man1/%{name}.1

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%{_prefix}/lib/zypper/commands/zypper-declarative
%{_mandir}/man1/%{name}.1*

%changelog
* Fri May 29 2026 Matthias G. Eckermann <pcd@mailbox.org> - 0.6.0-0
- Initial package for zypper-declarative 0.6.0.
