# pcd-spec-sha256: f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
Name:           zypper-declarative
Version:        0.6.2
Release:        0
Summary:        Reconciling declarative converger surfaced as a zypper subcommand
License:        GPL-2.0-or-later
Group:          System/Management
URL:            https://github.com/mge1512/zypper-declarative
Source0:        %{name}-%{version}.tar.gz
BuildRequires:  go >= 1.21
BuildRequires:  pandoc
ExclusiveOS:    linux

%description
zypper-declarative converges a system to a desired manifest inside a single
snapshot transaction, recording what was applied. The manifest is the
declarable subset of the SUSE Machinery system description (packages,
repositories, services, config_files), serialised as canonical JSON
(format_version 1) or, optionally, YAML. It is surfaced as a zypper subcommand
and is also directly invocable.

%prep
%setup -q

%build
export CGO_ENABLED=0
go build -o %{name} ./cmd/%{name}
pandoc %{name}.1.md -s -t man -o %{name}.1

%install
install -D -m 0755 %{name} %{buildroot}%{_bindir}/%{name}
install -D -m 0755 %{name} %{buildroot}%{_prefix}/lib/zypper/commands/%{name}
install -D -m 0644 %{name}.1 %{buildroot}%{_mandir}/man1/%{name}.1

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%{_prefix}/lib/zypper/commands/%{name}
%{_mandir}/man1/%{name}.1*

%changelog
* Mon Jun 01 2026 Matthias G. Eckermann <pcd@mailbox.org> - 0.6.2-0
- Generated from zypper-declarative.spec.md v0.6.2.
