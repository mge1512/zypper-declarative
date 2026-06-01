# generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
# Module identity: github.com/mge1512/zypper-declarative (spec META Module:)
cmake_minimum_required(VERSION 3.20)   # 3.28 on SLE 15 SP7; 3.20 is a safe floor
project(zypper-declarative VERSION 0.6.2 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

if(NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Release)
endif()

add_compile_options(-Wall -Wextra)

# The spec SHA256 embedded into every artefact, including the configured header.
set(ZD_SPEC_SHA256 "f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e")
set(ZD_VERSION "${PROJECT_VERSION}")
configure_file(${CMAKE_CURRENT_SOURCE_DIR}/src/meta.hpp.in
               ${CMAKE_CURRENT_BINARY_DIR}/generated/meta.hpp @ONLY)

# --- Dependencies: DYNAMICALLY linked distribution shared libraries ----------
find_package(PkgConfig REQUIRED)

# libzypp: package/rpmdb query and file ownership (NOT librpm, NOT exec rpm).
pkg_check_modules(ZYPP REQUIRED IMPORTED_TARGET libzypp)

# jsoncpp: canonical JSON serialisation/parsing.
pkg_check_modules(JSONCPP REQUIRED IMPORTED_TARGET jsoncpp)

# yaml-cpp: opt-in YAML serialisation (0.6 on SLE 15 SP7, 0.8 on SLE 16).
find_package(yaml-cpp QUIET)
if(NOT yaml-cpp_FOUND)
  pkg_check_modules(YAMLCPP REQUIRED IMPORTED_TARGET yaml-cpp)
endif()

# OpenSSL libcrypto: SHA256 (already in the libzypp dependency closure).
find_package(OpenSSL REQUIRED)

# libsnapper: snapshot/generation operations (soname 5 on SLE 15 SP7, 7 on
# SLE 16). The transaction binding is exercised on a live host; the build links
# it when present so the per-SP RPM picks up the right soname. Optional at
# configure time so the offline build/test gate does not require it.
pkg_check_modules(SNAPPER IMPORTED_TARGET libsnapper)

add_executable(zypper-declarative
  src/main.cpp
  src/cli.cpp
  src/command_runner.cpp
  src/hashing.cpp
  src/manifest.cpp
  src/diff.cpp
  src/describe.cpp
  src/system_reader.cpp
  src/transaction.cpp
  src/commands.cpp
)

target_include_directories(zypper-declarative PRIVATE
  src include ${CMAKE_CURRENT_BINARY_DIR}/generated)

target_link_libraries(zypper-declarative PRIVATE
  PkgConfig::ZYPP
  PkgConfig::JSONCPP
  OpenSSL::Crypto
)

if(yaml-cpp_FOUND)
  target_link_libraries(zypper-declarative PRIVATE yaml-cpp)
else()
  target_link_libraries(zypper-declarative PRIVATE PkgConfig::YAMLCPP)
endif()

if(SNAPPER_FOUND)
  target_link_libraries(zypper-declarative PRIVATE PkgConfig::SNAPPER)
  target_compile_definitions(zypper-declarative PRIVATE ZD_HAVE_SNAPPER=1)
endif()

include(GNUInstallDirs)
install(TARGETS zypper-declarative RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR})
# Surfaced as a zypper subcommand:
install(CODE "message(STATUS \"Also link/copy the binary into /usr/lib/zypper/commands for the 'zypper declarative' subcommand surface\")")
