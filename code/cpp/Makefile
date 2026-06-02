# generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#
# Build orchestration for zypper-declarative (C++17, CMake, dynamic linking
# against the distribution's libzypp / libsnapper / jsoncpp / yaml-cpp /
# libcrypto). The real build is driven by CMake; this Makefile wraps it so the
# cli-tool template's required targets (build, test, install, clean, man) are
# present and `make test` actually runs the suite.
#
# On SLE 15 SP7 the default g++ is GCC 7 (too old for C++17); select g++-15:
#     make build CXX=g++-15
# On SLE 16.0 the default toolchain (GCC 15) suffices.

SPEC_SHA256 := aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
VERSION     := $(shell cat VERSION)
NAME        := zypper-declarative
LLM_NAME    := claude-opus-4-8

BUILD_DIR   := build
CXX         ?= c++
PREFIX      ?= /usr
BINDIR      := $(PREFIX)/bin
MANDIR      := $(PREFIX)/share/man/man1
# zypper subcommand discovery directory
ZYPPER_CMDDIR := $(PREFIX)/lib/zypper/commands

TEST_DIR    := independent_tests/$(LLM_NAME)

.PHONY: all build test install clean man dist

all: build

# --- build: configure + compile via CMake, copy binary to project root -------
build:
	cmake -S . -B $(BUILD_DIR) -DCMAKE_BUILD_TYPE=Release -DCMAKE_CXX_COMPILER=$(CXX)
	cmake --build $(BUILD_DIR) -j
	cp -f $(BUILD_DIR)/$(NAME) ./$(NAME)

# --- test: build the black-box suite and run it (exits non-zero on failure) ---
# The suite invokes the binary at ../../$(NAME) (project root), per the cli-tool
# template BINARY-LOCATION: project-root constraint. `make test` depends on the
# binary existing at the project root, so run `make build` first.
test: build
	$(CXX) -std=c++17 -Wall -Wextra -O2 \
	    -o $(TEST_DIR)/zd_test $(TEST_DIR)/$(NAME)_test.cpp
	cd $(TEST_DIR) && ./zd_test

# --- man: render the markdown man source to troff via pandoc -------------------
man: $(NAME).1

$(NAME).1: $(NAME).1.md
	pandoc $(NAME).1.md -s -t man -o $(NAME).1

# --- install: binary, zypper subcommand link, man page ------------------------
install: build man
	install -D -m 0755 $(NAME) $(DESTDIR)$(BINDIR)/$(NAME)
	install -D -m 0755 $(NAME) $(DESTDIR)$(ZYPPER_CMDDIR)/$(NAME)
	install -D -m 0644 $(NAME).1 $(DESTDIR)$(MANDIR)/$(NAME).1

# --- clean --------------------------------------------------------------------
clean:
	rm -rf $(BUILD_DIR)
	rm -f $(NAME) $(NAME).1
	rm -f $(TEST_DIR)/zd_test

# --- dist: source tarball for OBS / rpmbuild ----------------------------------
# Produces $(NAME)-$(VERSION).tar.gz with a single top-level directory
# $(NAME)-$(VERSION)/ so rpmbuild's default %autosetup finds it. Build
# artefacts and VCS directories are excluded.
dist:
	rm -rf $(NAME)-$(VERSION)
	mkdir -p $(NAME)-$(VERSION)
	cp -a CMakeLists.txt Makefile VERSION src $(NAME).1.md README.md LICENSE \
	      $(NAME).spec debian independent_tests $(NAME)-$(VERSION)/ 2>/dev/null || true
	tar --exclude='build' --exclude='.git' --exclude='*/zd_test' \
	    -czf $(NAME)-$(VERSION).tar.gz $(NAME)-$(VERSION)
	rm -rf $(NAME)-$(VERSION)
