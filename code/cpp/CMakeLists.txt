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
find_package(PkgConfig REQUIRED)

# JSON: jsoncpp (jsoncpp-devel). Prefer the CMake config, fall back to pkg-config.
find_package(jsoncpp QUIET)
if(NOT jsoncpp_FOUND)
  pkg_check_modules(JSONCPP REQUIRED jsoncpp)
endif()

# YAML: yaml-cpp (libyaml-cpp). 0.6 on SLE 15 SP7, 0.8 on SLE 16.0; restrict to
# the API stable across both.
find_package(yaml-cpp QUIET)
if(NOT yaml-cpp_FOUND)
  pkg_check_modules(YAMLCPP REQUIRED yaml-cpp)
endif()

# Package database / ownership / per-file baseline: libzypp (libzypp-devel).
find_package(Zypp QUIET)
if(NOT Zypp_FOUND)
  pkg_check_modules(ZYPP REQUIRED libzypp)
endif()

# SHA256/MD5: libcrypto (OpenSSL 3).
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

# jsoncpp linkage
if(jsoncpp_FOUND)
  if(TARGET jsoncpp_lib)
    target_link_libraries(zypper-declarative PRIVATE jsoncpp_lib)
  elseif(TARGET JsonCpp::JsonCpp)
    target_link_libraries(zypper-declarative PRIVATE JsonCpp::JsonCpp)
  endif()
else()
  target_include_directories(zypper-declarative PRIVATE ${JSONCPP_INCLUDE_DIRS})
  target_link_libraries(zypper-declarative PRIVATE ${JSONCPP_LIBRARIES})
endif()

# yaml-cpp linkage
if(yaml-cpp_FOUND)
  target_link_libraries(zypper-declarative PRIVATE yaml-cpp)
else()
  target_include_directories(zypper-declarative PRIVATE ${YAMLCPP_INCLUDE_DIRS})
  target_link_libraries(zypper-declarative PRIVATE ${YAMLCPP_LIBRARIES})
endif()

# libzypp linkage
if(Zypp_FOUND)
  target_link_libraries(zypper-declarative PRIVATE zypp)
else()
  target_include_directories(zypper-declarative PRIVATE ${ZYPP_INCLUDE_DIRS})
  target_link_libraries(zypper-declarative PRIVATE ${ZYPP_LIBRARIES})
endif()

target_link_libraries(zypper-declarative PRIVATE OpenSSL::Crypto)

include(GNUInstallDirs)
install(TARGETS zypper-declarative RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR})
# Surfaced as a zypper subcommand.
install(TARGETS zypper-declarative RUNTIME DESTINATION lib/zypper/commands)
install(FILES zypper-declarative.1 DESTINATION ${CMAKE_INSTALL_MANDIR}/man1
        OPTIONAL)
