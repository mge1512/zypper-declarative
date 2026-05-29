# pcd-spec-sha256: 58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
Name:           zypper-declarative
Version:        0.5.0
Release:        0
Summary:        Declarative, reconciling converger surfaced as a zypper subcommand
License:        GPL-2.0-or-later
URL:            https://github.com/mge1512/zypper-declarative
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  go >= 1.23
BuildRequires:  pandoc
ExclusiveArch:  x86_64 aarch64

%description
zypper-declarative converges a SUSE system to a desired manifest (the declarable
subset of the SUSE Machinery system description: packages, repositories,
services, config_files) inside a single snapshot transaction, recording what was
applied. It provides the verbs apply, diff, verify, status, and describe. The
applied record is stored within the generation it describes and is restored on
rollback. Idempotent: a second run against an unchanged manifest and an undrifted
system makes no changes.

%prep
%autosetup

%build
export CGO_ENABLED=0
go build -o %{name} ./cmd/%{name}
pandoc %{name}.1.md -s -t man -o %{name}.1

%install
install -D -m 0755 %{name} %{buildroot}%{_prefix}/lib/zypper/commands/%{name}
install -D -m 0755 %{name} %{buildroot}%{_bindir}/%{name}
install -D -m 0644 %{name}.1 %{buildroot}%{_mandir}/man1/%{name}.1

%files
%license LICENSE
%{_bindir}/%{name}
%{_prefix}/lib/zypper/commands/%{name}
%{_mandir}/man1/%{name}.1*

%changelog
* Fri May 29 2026 Matthias G. Eckermann <pcd@mailbox.org> - 0.5.0-0
- Generated from zypper-declarative.spec.md v0.5.0.
