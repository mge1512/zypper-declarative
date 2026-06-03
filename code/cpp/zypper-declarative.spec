# pcd-spec-sha256: aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#
# OBS RPM spec for zypper-declarative (C++ implementation, dynamically linked).
# On SLE 15 SP7 build with gcc15-c++ (the default gcc-c++ is GCC 7, too old for
# C++17); on SLE 16.0 the default toolchain (GCC 15) suffices.

Name:           zypper-declarative
Version:        %(cat %{_sourcedir}/VERSION)
Release:        0
Summary:        Declarative convergence of SUSE system state in a snapshot transaction
License:        GPL-2.0-or-later
Group:          System/Management
URL:            https://github.com/mge1512/zypper-declarative
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  cmake
BuildRequires:  pkgconfig
BuildRequires:  pandoc
%if 0%{?sle_version} && 0%{?sle_version} < 160000
BuildRequires:  gcc15-c++
%else
BuildRequires:  gcc-c++
%endif
BuildRequires:  libzypp-devel
BuildRequires:  libsnapper-devel
BuildRequires:  jsoncpp-devel
BuildRequires:  yaml-cpp-devel
BuildRequires:  libopenssl-3-devel

# Runtime: dynamic dependencies are resolved automatically by the linker; the
# tool is surfaced as a zypper subcommand and requires zypper at runtime.
Requires:       zypper

%description
zypper-declarative converges the declarable subset of a SUSE Machinery system
description (packages, repositories, services, config_files) to a desired
manifest inside a single snapshot transaction, recording what was applied.
It links libzypp, libsnapper, jsoncpp, yaml-cpp and libcrypto dynamically and
is surfaced as a zypper subcommand.

%prep
%autosetup -n %{name}-%{version}

%build
%if 0%{?sle_version} && 0%{?sle_version} < 160000
export CXX=g++-15
%endif
cmake -S . -B build \
      -DCMAKE_BUILD_TYPE=Release \
      -DCMAKE_INSTALL_PREFIX=%{_prefix}
cmake --build build %{?_smp_mflags}
pandoc %{name}.1.md -s -t man -o %{name}.1

%install
install -D -m0755 build/%{name} %{buildroot}%{_prefix}/lib/zypper/commands/%{name}
install -D -m0644 %{name}.1 %{buildroot}%{_mandir}/man1/%{name}.1

%files
%license LICENSE
%doc README.md
# /usr/lib/zypper and /usr/lib/zypper/commands are owned by the zypper package
# (Requires: zypper), so they are not claimed here to avoid duplicate ownership.
%{_prefix}/lib/zypper/commands/%{name}
%{_mandir}/man1/%{name}.1*

%changelog
* Wed Jun 03 2026 Matthias G. Eckermann <pcd@mailbox.org> - 0.6.9
- C++17 implementation tracking zypper-declarative.spec.md v0.6.9.
