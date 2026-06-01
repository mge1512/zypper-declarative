# pcd-spec-sha256: f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
#
# RPM spec for zypper-declarative (module identity: github.com/mge1512/zypper-declarative).
# OBS RPM spec. C++ dynamic build against the distribution's supported shared
# libraries (libzypp, jsoncpp, yaml-cpp, libsnapper), per-SP via OBS.
#
# Compiler: on SLE 15 SP7 the default gcc-c++ is GCC 7 (too old for clean
# C++17); BuildRequires gcc15-c++ and build with g++-15. On SLE 16 the default
# toolchain is GCC 15 and the side-by-side package is unnecessary.

Name:           zypper-declarative
Version:        0.6.2
Release:        0
Summary:        Declarative reconciling converger surfaced as a zypper subcommand
License:        GPL-2.0-or-later
Group:          System/Management
URL:            https://github.com/mge1512/zypper-declarative
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  cmake >= 3.20
%if 0%{?sle_version} && 0%{?sle_version} < 160000
BuildRequires:  gcc15-c++
%else
BuildRequires:  gcc-c++
%endif
BuildRequires:  pkgconfig
BuildRequires:  pandoc
BuildRequires:  libzypp-devel
BuildRequires:  jsoncpp-devel
BuildRequires:  libsnapper-devel
%if 0%{?sle_version} && 0%{?sle_version} < 160000
BuildRequires:  libyaml-cpp-devel
%else
BuildRequires:  yaml-cpp-devel
%endif
BuildRequires:  libopenssl-devel

# Runtime shared-library dependencies are resolved automatically by the linker
# (dynamic build); the per-SP package links that SP's own sonames.

%description
zypper-declarative converges a SUSE system to a desired manifest (the declarable
subset of the SUSE Machinery system description: packages, repositories,
services, config_files) inside a single snapshot transaction, recording what was
applied. It provides the verbs apply, diff, verify, status, and describe. It is
surfaced as a zypper subcommand (zypper declarative <verb>) and is also invokable
directly.

%prep
%setup -q

%build
%if 0%{?sle_version} && 0%{?sle_version} < 160000
export CXX=g++-15
cmake -S . -B build -DCMAKE_BUILD_TYPE=Release -DCMAKE_CXX_COMPILER=g++-15
%else
cmake -S . -B build -DCMAKE_BUILD_TYPE=Release
%endif
cmake --build build -j%{?_smp_build_ncpus}
pandoc %{name}.1.md -s -t man -o %{name}.1

%install
DESTDIR=%{buildroot} cmake --install build
install -d %{buildroot}%{_mandir}/man1
install -m 0644 %{name}.1 %{buildroot}%{_mandir}/man1/%{name}.1
install -d %{buildroot}%{_prefix}/lib/zypper/commands
ln -sf %{_bindir}/%{name} %{buildroot}%{_prefix}/lib/zypper/commands/zypper-declarative

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%{_prefix}/lib/zypper/commands/zypper-declarative
%{_mandir}/man1/%{name}.1*

%changelog
* Mon Jun 01 2026 Matthias G. Eckermann <pcd@mailbox.org> - 0.6.2-0
- Initial C++ translation of the zypper-declarative spec (v0.6.2).
