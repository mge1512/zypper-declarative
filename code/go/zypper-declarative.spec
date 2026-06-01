# pcd-spec-sha256: 27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
Name:           zypper-declarative
Version:        0.6.5
Release:        0
Summary:        Declarative system convergence for zypper-managed systems

License:        GPL-2.0-or-later
URL:            https://github.com/mge1512/zypper-declarative
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  go >= 1.26
BuildRequires:  pandoc

%description
zypper-declarative converges a SUSE system to a desired manifest (the declarable
subset of the SUSE Machinery system description: packages, repositories,
services, config_files) inside a single snapshot transaction, recording what was
applied. It provides the verbs apply, diff, verify, status, and describe, and is
surfaced as the "zypper declarative" subcommand.

%prep
%autosetup -n %{name}-%{version}

%build
CGO_ENABLED=0 go build -o %{name} ./cmd/%{name}
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
* Tue Jun 02 2026 Matthias G. Eckermann <pcd@mailbox.org> - 0.6.5-0
- Initial package generated from zypper-declarative.spec.md (PCD translator).
