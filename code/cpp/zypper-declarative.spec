# pcd-spec-sha256: 1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e
#
# RPM spec for zypper-declarative. OBS build target (build.opensuse.org).
# C++17, built with CMake, dynamically linked against the distribution's
# libzypp / libsnapper / jsoncpp / yaml-cpp / libcrypto. No curl-based install.
#
# Per-SP compiler note: on SLE 15 SP7 the default g++ is GCC 7 (too old for
# C++17); BuildRequires gcc15-c++ and build with g++-15. On SLE 16.0 the
# default toolchain is GCC 15 and no special selection is needed.

Name:           zypper-declarative
Version:        %(cat %{_sourcedir}/VERSION 2>/dev/null || echo 0.6.8)
Release:        0
Summary:        Declarative convergence of SUSE system state in a snapshot transaction
License:        GPL-2.0-or-later
Group:          System/Management
URL:            https://github.com/mge1512/zypper-declarative
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  cmake >= 3.20
BuildRequires:  pkgconfig
BuildRequires:  pandoc
# C++17 toolchain. On SLE 15 SP7 use the side-by-side GCC 15:
%if 0%{?sle_version} && 0%{?sle_version} < 160000
BuildRequires:  gcc15-c++
%else
BuildRequires:  gcc-c++
%endif
# Dynamically linked distribution libraries (devel packages).
BuildRequires:  libzypp-devel
BuildRequires:  libsnapper-devel
BuildRequires:  jsoncpp-devel
BuildRequires:  libyaml-cpp-devel
BuildRequires:  libopenssl-3-devel

%description
zypper-declarative converges a SUSE system to a declarative manifest describing
the declarable subset of the SUSE Machinery system description (packages,
repositories, services, and /etc config files), inside a single snapshot
transaction. The manifest is JSON (Machinery format_version 1) by default, with
YAML as an opt-in serialisation. It reads the live system into the same model
itself, so no separate collector is required. It is surfaced as a zypper
subcommand and is also invokable directly.

%prep
%autosetup

%build
%if 0%{?sle_version} && 0%{?sle_version} < 160000
export CXX=g++-15
%endif
cmake -S . -B build \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_INSTALL_PREFIX=%{_prefix} \
    %{?with_g15:-DCMAKE_CXX_COMPILER=g++-15}
cmake --build build -j%{?_smp_build_ncpus}
pandoc %{name}.1.md -s -t man -o %{name}.1

%install
install -D -m 0755 build/%{name} %{buildroot}%{_bindir}/%{name}
# zypper subcommand discovery directory
install -D -m 0755 build/%{name} %{buildroot}%{_prefix}/lib/zypper/commands/%{name}
install -D -m 0644 %{name}.1 %{buildroot}%{_mandir}/man1/%{name}.1

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%{_prefix}/lib/zypper/commands/%{name}
%{_mandir}/man1/%{name}.1*

%changelog
* Tue Jun 02 2026 Matthias G. Eckermann <pcd@mailbox.org> - 0.6.8-0
- Generated from spec zypper-declarative.spec.md v0.6.8
  (sha256:1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e).
- on_unreadable knob now honored on diff, verify, and apply (v0.6.8).
