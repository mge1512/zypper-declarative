# pcd-spec-sha256: 51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
Name:           zypper-declarative
Version:        0.6.6
Release:        0
Summary:        Declarative system convergence for zypper (Machinery subset)
License:        GPL-2.0-or-later
URL:            https://github.com/mge1512/zypper-declarative
Source0:        %{name}-%{version}.tar.gz

# Build toolchain: C++17. On SLE 15 SP7 the default gcc-c++ is GCC 7, too old
# for clean C++17; use the side-by-side GCC 15 (gcc15-c++) and build with
# g++-15. On SLE 16.0 the default gcc-c++ is already GCC 15.
%if 0%{?sle_version} && 0%{?sle_version} < 160000
BuildRequires:  gcc15-c++
%else
BuildRequires:  gcc-c++
%endif
BuildRequires:  cmake >= 3.20
BuildRequires:  pkgconfig
BuildRequires:  pandoc
BuildRequires:  libzypp-devel
BuildRequires:  libsnapper-devel
BuildRequires:  jsoncpp-devel
BuildRequires:  libyaml-cpp-devel
BuildRequires:  libopenssl-3-devel

%description
zypper-declarative converges a SUSE system to a desired manifest expressed in
the declarable subset of the SUSE Machinery system description (packages,
repositories, services, config_files). It applies inside a single snapshot
transaction, records what was applied, and is idempotent. It also describes the
actual state, computes diffs, and verifies drift. Distributed via OBS; surfaced
as the "zypper declarative" subcommand.

%prep
%setup -q

%build
%if 0%{?sle_version} && 0%{?sle_version} < 160000
export CXX=g++-15
%endif
cmake -S . -B build -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=%{_prefix} \
      -DCMAKE_CXX_COMPILER=%{?sle_version:%{expand:%(test 0%{?sle_version} -lt 160000 && echo g++-15 || echo c++)}}%{!?sle_version:c++}
cmake --build build -j%{?_smp_build_ncpus}
pandoc %{name}.1.md -s -t man -o %{name}.1

%install
install -D -m0755 build/%{name} %{buildroot}%{_bindir}/%{name}
install -D -m0755 build/%{name} %{buildroot}%{_prefix}/lib/zypper/commands/%{name}
install -D -m0644 %{name}.1 %{buildroot}%{_mandir}/man1/%{name}.1

%files
%license LICENSE
%{_bindir}/%{name}
%{_prefix}/lib/zypper/commands/%{name}
%{_mandir}/man1/%{name}.1*

%changelog
* Tue Jun 02 2026 Matthias G. Eckermann <pcd@mailbox.org> - 0.6.6-0
- Initial package generated from zypper-declarative spec v0.6.6.
