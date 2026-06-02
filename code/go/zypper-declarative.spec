# pcd-spec-sha256: 51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
Name:           zypper-declarative
Version:        0.6.6
Release:        0
Summary:        Declarative convergence of SUSE system state via Machinery manifests
License:        GPL-2.0-or-later
Group:          System/Management
URL:            https://github.com/mge1512/zypper-declarative
Source0:        %{name}-%{version}.tar.gz
BuildRequires:  go >= 1.21
BuildRequires:  pandoc
ExclusiveArch:  x86_64 aarch64

%description
zypper-declarative converges a SUSE system to a desired manifest (the declarable
subset of the SUSE Machinery system description: packages, repositories,
services, config_files) inside a single snapshot transaction, recording what was
applied. It provides apply, diff, verify, status, and describe verbs, reads live
system state itself, and serialises manifests as canonical Machinery JSON (or
opt-in YAML). Surfaced as a zypper subcommand and invokable directly.

%prep
%setup -q

%build
# Static binary, no glibc dependency (BINARY-TYPE: static).
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
* Tue Jun 02 2026 Matthias G. Eckermann <pcd@mailbox.org> - 0.6.6-0
- Generated from zypper-declarative.spec.md (spec v0.6.6).
