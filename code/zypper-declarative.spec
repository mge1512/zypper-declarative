# pcd-spec-sha256: f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
Name:           zypper-declarative
Version:        0.5.1
Release:        0
Summary:        Declarative, reconciling converger for SUSE systems, surfaced as a zypper subcommand
License:        GPL-2.0-or-later
URL:            https://github.com/mge1512/zypper-declarative
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  go >= 1.22
BuildRequires:  pandoc

%description
zypper-declarative converges a SUSE system (SL Micro 6.2, SLES 16.1) to a
desired manifest expressed in the declarable subset of the SUSE Machinery
system-description format (packages, repositories, services, config_files),
inside a single snapshot transaction, recording what was applied. It provides
the verbs apply, diff, verify, status, and describe, and is surfaced as the
zypper subcommand "zypper declarative".

%prep
%setup -q

%build
# Static binary, no runtime dependencies of its own (cli-tool BINARY-TYPE: static).
export CGO_ENABLED=0
go build -mod=vendor -o %{name} ./cmd/%{name}
pandoc %{name}.1.md -s -t man -o %{name}.1

%install
install -D -m 0755 %{name} %{buildroot}%{_bindir}/%{name}
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
* Fri May 29 2026 Matthias G. Eckermann <pcd@mailbox.org> - 0.5.1-0
- Generated from zypper-declarative.spec.md v0.5.1.
