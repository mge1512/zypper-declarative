# generated from spec: zypper-declarative.spec.md sha256:1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e
cmake_minimum_required(VERSION 3.20)   # 3.28 on SLE 15 SP7, 3.31 on SLE 16; 3.20 is a safe floor

# Module identity: project name "zypper-declarative" (spec title; spec META
# Module: github.com/mge1512/zypper-declarative — the C++ artefact identity is
# the binary/project name zypper-declarative, see TRANSLATION_REPORT.md).
project(zypper-declarative CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

if(NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Release)
endif()

add_compile_options(-Wall -Wextra)

# --- Single-source version + embedded spec hash (configure_file) -------------
file(STRINGS "${CMAKE_CURRENT_SOURCE_DIR}/VERSION" ZD_VERSION LIMIT_COUNT 1)
string(STRIP "${ZD_VERSION}" ZD_VERSION)
# The spec SHA256 is embedded once, computed before generation.
set(ZD_SPEC_SHA256 "1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e")
configure_file(
  "${CMAKE_CURRENT_SOURCE_DIR}/src/meta.hpp.in"
  "${CMAKE_CURRENT_BINARY_DIR}/generated/meta.hpp"
  @ONLY)

# --- Dependencies: DYNAMIC distro shared libraries (no static, no vendoring) --
# Prefer pkg-config over find_package for libzypp/jsoncpp/yaml-cpp: their CMake
# configs are fragile / Meson-generated across service packs; the .pc files are
# stable on both SLE 15 SP7 and SLE 16.0. See the C++ decisions hints.
find_package(PkgConfig REQUIRED)
pkg_check_modules(ZYPP    REQUIRED IMPORTED_TARGET libzypp)
pkg_check_modules(JSONCPP REQUIRED IMPORTED_TARGET jsoncpp)
pkg_check_modules(YAMLCPP REQUIRED IMPORTED_TARGET yaml-cpp)
pkg_check_modules(CRYPTO  REQUIRED IMPORTED_TARGET libcrypto)

# libsnapper ships no .pc on every SP; locate the header and library directly.
# soname differs (libsnapper5 on SLE 15 SP7, libsnapper7 on SLE 16.0); link
# whichever is present (OBS builds per-SP).
find_path(SNAPPER_INCLUDE_DIR snapper/Snapper.h)
find_library(SNAPPER_LIBRARY NAMES snapper)
if(NOT SNAPPER_INCLUDE_DIR OR NOT SNAPPER_LIBRARY)
  message(FATAL_ERROR "libsnapper (libsnapper-devel) not found")
endif()

message(STATUS "zypper-declarative ${ZD_VERSION}")
message(STATUS "  libzypp  ${ZYPP_VERSION}")
message(STATUS "  jsoncpp  ${JSONCPP_VERSION}")
message(STATUS "  yaml-cpp ${YAMLCPP_VERSION}")
message(STATUS "  libcrypto ${CRYPTO_VERSION}")
message(STATUS "  snapper  ${SNAPPER_LIBRARY}")

add_executable(zypper-declarative
  src/main.cpp
  src/cli.cpp
  src/command_runner.cpp
  src/manifest.cpp
  src/diff.cpp
  src/actual_state.cpp
  src/transaction.cpp
  src/commands.cpp)

target_include_directories(zypper-declarative PRIVATE
  src
  "${CMAKE_CURRENT_BINARY_DIR}/generated"
  "${SNAPPER_INCLUDE_DIR}")

target_link_libraries(zypper-declarative PRIVATE
  PkgConfig::ZYPP
  PkgConfig::JSONCPP
  PkgConfig::YAMLCPP
  PkgConfig::CRYPTO
  "${SNAPPER_LIBRARY}")

include(GNUInstallDirs)
install(TARGETS zypper-declarative RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR})
# Also installed as a zypper subcommand executable.
install(TARGETS zypper-declarative RUNTIME DESTINATION lib/zypper/commands)
