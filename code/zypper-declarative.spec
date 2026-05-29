# generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
# pcd-spec-sha256: 714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
#
# OBS RPM spec for zypper-declarative.

Name:           zypper-declarative
Version:        0.4.0
Release:        0
Summary:        Declarative, reconciling converger surfaced as a zypper subcommand
License:        GPL-2.0-or-later
URL:            https://github.com/mge1512/zypper-declarative
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  go >= 1.23
BuildRequires:  pandoc

# Runtime tools the converger drives (not linked, invoked at runtime).
Requires:       rpm
Requires:       zypper
Recommends:     snapper
Recommends:     systemd

%description
zypper-declarative converges a SUSE system to a desired manifest expressed in
the declarable subset of the SUSE Machinery system-description format
(packages, repositories, services, config_files), inside a single snapshot
transaction, recording what was applied. It provides the verbs apply, diff,
verify, status, and describe, and is surfaced both as the zypper subcommand
`zypper declarative` and as a direct binary.

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
* Fri May 29 2026 Matthias G. Eckermann <pcd@mailbox.org> - 0.4.0-0
- Initial package. YAML opt-in serialisation alongside canonical JSON.
