# generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
cmake_minimum_required(VERSION 3.20)

# Version single-sourced from the top-level VERSION file.
file(STRINGS "${CMAKE_CURRENT_SOURCE_DIR}/VERSION" ZD_VERSION_RAW LIMIT_COUNT 1)
string(STRIP "${ZD_VERSION_RAW}" ZD_VERSION)

project(zypper-declarative VERSION ${ZD_VERSION} LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

if(NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Release)
endif()

# Spec SHA256 embedded in the binary's version output and all artefacts.
set(ZD_SPEC_SHA256 "aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3")

# Dynamically linked distribution shared libraries (no static, no vendoring).
# Discover via pkg-config IMPORTED targets (the .pc files are the stable
# cross-service-pack discovery; per-SP CMake configs are fragile).
find_package(PkgConfig REQUIRED)
pkg_check_modules(ZYPP    REQUIRED IMPORTED_TARGET libzypp)
pkg_check_modules(JSONCPP REQUIRED IMPORTED_TARGET jsoncpp)
pkg_check_modules(YAMLCPP REQUIRED IMPORTED_TARGET yaml-cpp)
pkg_check_modules(CRYPTO  REQUIRED IMPORTED_TARGET libcrypto)

# libsnapper ships no .pc on some service packs; discover the headers/lib and
# detect its version from the installed Version.h for the Plugins::Report
# API-break guard.
find_path(SNAPPER_INCLUDE_DIR NAMES snapper/Snapper.h)
find_library(SNAPPER_LIBRARY NAMES snapper)
if(NOT SNAPPER_INCLUDE_DIR OR NOT SNAPPER_LIBRARY)
  message(FATAL_ERROR "libsnapper-devel not found (snapper/Snapper.h and libsnapper)")
endif()

# Detect snapper SONAME major from Version.h. The header defines
# LIBSNAPPER_MAJOR as the soname major (5 => 0.8.x one-arg API; 7 => 0.12.x
# two-arg Plugins::Report API added; 8 => 0.13.x one-arg removed). The values
# may be quoted (LIBSNAPPER_MAJOR="5").
set(SNAPPER_SONAME_MAJOR 0)
if(EXISTS "${SNAPPER_INCLUDE_DIR}/snapper/Version.h")
  file(READ "${SNAPPER_INCLUDE_DIR}/snapper/Version.h" _snapver_h)
  string(REGEX MATCH "LIBSNAPPER_MAJOR=?[ \t]*\"?([0-9]+)\"?" _m "${_snapver_h}")
  if(NOT CMAKE_MATCH_1 STREQUAL "")
    set(SNAPPER_SONAME_MAJOR "${CMAKE_MATCH_1}")
  endif()
endif()
message(STATUS "libsnapper soname major detected: ${SNAPPER_SONAME_MAJOR}")

# soname major >= 7 (snapper >= 0.12) carries the trailing Plugins::Report&
# parameter on mutating calls. Branch in the build (where the version is
# known), not via a header probe.
set(ZD_SNAPPER_REPORT OFF)
if(SNAPPER_SONAME_MAJOR GREATER_EQUAL 7)
  set(ZD_SNAPPER_REPORT ON)
endif()

# Generate meta.hpp (version + spec hash) via configure_file.
configure_file(${CMAKE_CURRENT_SOURCE_DIR}/src/meta.hpp.in
               ${CMAKE_CURRENT_BINARY_DIR}/generated/meta.hpp @ONLY)

add_executable(zypper-declarative
  src/main.cpp
  src/cli.cpp
  src/config.cpp
  src/diagnostic.cpp
  src/command_runner.cpp
  src/hashing.cpp
  src/manifest_io.cpp
  src/diffdrift.cpp
  src/describe.cpp
  src/package_db.cpp
  src/transaction.cpp
  src/converge.cpp
  src/verbs.cpp
)

target_include_directories(zypper-declarative PRIVATE
  src
  include
  ${CMAKE_CURRENT_BINARY_DIR}/generated
  ${SNAPPER_INCLUDE_DIR})

target_compile_options(zypper-declarative PRIVATE -Wall -Wextra)

if(ZD_SNAPPER_REPORT)
  target_compile_definitions(zypper-declarative PRIVATE ZD_SNAPPER_REPORT_PARAM)
endif()

target_link_libraries(zypper-declarative PRIVATE
  PkgConfig::ZYPP
  PkgConfig::JSONCPP
  PkgConfig::YAMLCPP
  PkgConfig::CRYPTO
  ${SNAPPER_LIBRARY})

include(GNUInstallDirs)
install(TARGETS zypper-declarative
        RUNTIME DESTINATION ${CMAKE_INSTALL_PREFIX}/lib/zypper/commands)
