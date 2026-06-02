# generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
cmake_minimum_required(VERSION 3.20)
project(zypper-declarative VERSION 0.6.6 LANGUAGES CXX)

# Spec identity embedded in every generated artefact.
set(ZD_VERSION "0.6.6")
set(ZD_SPEC_SHA256 "51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03")

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

if(NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Release)
endif()

add_compile_options(-Wall -Wextra)

# --- Dependencies (DYNAMICALLY linked distribution libraries) --------------
# Per the C++ decisions hints: dynamic linking against the service pack's own
# sonames; no static binary, no vendoring. Build per-SP via OBS.
#
# Discover jsoncpp, yaml-cpp, and libzypp via pkg-config IMPORTED targets, NOT
# find_package(). On SLE 16 the jsoncpp jsoncppConfig.cmake is Meson-generated
# and omits include(CMakePackageConfigHelpers), so its trailing
# check_required_components(jsoncpp) fails with "Unknown CMake command" and
# breaks the build; yaml-cpp's CMake config has the same per-SP fragility. The
# .pc files (jsoncpp.pc, yaml-cpp.pc, libzypp.pc) are present and stable on both
# SLE 15 SP7 and SLE 16.0, so pkg-config is the robust cross-SP discovery and
# immune to the per-SP CMake-config drift. (decisions hints, 2026-06-02.)
find_package(PkgConfig REQUIRED)

# JSON: jsoncpp (jsoncpp-devel). 1.8.4 on SLE 15 SP7, 1.9.6 on SLE 16.0.
pkg_check_modules(JSONCPP REQUIRED IMPORTED_TARGET jsoncpp)

# YAML: yaml-cpp (libyaml-cpp). 0.6 on SLE 15 SP7, 0.8 on SLE 16.0; restrict to
# the API stable across both. Branch on YAMLCPP_VERSION if ever needed.
pkg_check_modules(YAMLCPP REQUIRED IMPORTED_TARGET yaml-cpp)

# Package database / ownership / per-file baseline: libzypp (libzypp-devel).
pkg_check_modules(ZYPP REQUIRED IMPORTED_TARGET libzypp)

# SHA256/MD5: libcrypto (OpenSSL 3). No major-version/soname split across SPs;
# the CMake config is stable, so find_package is fine here.
find_package(OpenSSL REQUIRED)

# --- Generated metadata header --------------------------------------------
configure_file(${CMAKE_CURRENT_SOURCE_DIR}/src/meta.hpp.in
               ${CMAKE_CURRENT_BINARY_DIR}/generated/meta.hpp @ONLY)

add_executable(zypper-declarative
  src/main.cpp
  src/cli.cpp
  src/command_runner.cpp
  src/hash.cpp
  src/manifest.cpp
  src/diff.cpp
  src/describe.cpp
  src/fullscan.cpp
  src/converge.cpp
  src/commands.cpp
)

target_include_directories(zypper-declarative PRIVATE
  src
  ${CMAKE_CURRENT_BINARY_DIR}/generated
)

# Link the pkg-config IMPORTED targets (carry include dirs and link flags).
target_link_libraries(zypper-declarative PRIVATE
  PkgConfig::JSONCPP
  PkgConfig::YAMLCPP
  PkgConfig::ZYPP
  OpenSSL::Crypto
)

include(GNUInstallDirs)
install(TARGETS zypper-declarative RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR})
# Surfaced as a zypper subcommand.
install(TARGETS zypper-declarative RUNTIME DESTINATION lib/zypper/commands)
install(FILES zypper-declarative.1 DESTINATION ${CMAKE_INSTALL_MANDIR}/man1
        OPTIONAL)
