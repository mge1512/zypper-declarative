# generated from spec: zypper-declarative.spec.md
# Makefile -- thin wrapper over CMake. The build target copies the binary to the
# project root (BINARY-LOCATION: project-root) where tests invoke it via
# ../../zypper-declarative. The test target is executable: it builds the suite
# and runs it, exiting non-zero on any failure.

SPEC_SHA256 = f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
NAME = zypper-declarative
LLM = claude-opus-4-8
BUILDDIR = build

# On SLE 15 SP7 select the side-by-side GCC 15 (the default gcc-c++ is GCC 7,
# too old for clean C++17). On SLE 16 the default toolchain is already GCC 15.
# Note: make pre-defines CXX=g++, so a plain `CXX ?=` would not take effect;
# we therefore pick g++-15 when present unless the caller overrides CXX on the
# command line (make CXX=...) or in the environment (export CXX=...).
ifeq ($(origin CXX),default)
  CXX := $(shell command -v g++-15 >/dev/null 2>&1 && echo g++-15 || echo g++)
endif

PREFIX ?= /usr
BINDIR ?= $(PREFIX)/bin
MANDIR ?= $(PREFIX)/share/man/man1
ZYPPER_CMDDIR ?= $(PREFIX)/lib/zypper/commands

.PHONY: all build test install clean man

all: build

build:
	cmake -S . -B $(BUILDDIR) -DCMAKE_BUILD_TYPE=Release -DCMAKE_CXX_COMPILER=$(CXX)
	cmake --build $(BUILDDIR) -j
	cp -f $(BUILDDIR)/$(NAME) ./$(NAME)

# Executable test target: build the implementation (so the binary exists at the
# project root) then compile and run the black-box suite, which invokes
# ../../$(NAME). Exits non-zero on any test failure.
test: build
	$(CXX) -std=c++17 -Wall -Wextra \
	  -o independent_tests/$(LLM)/run_tests \
	  independent_tests/$(LLM)/test_main.cpp \
	  independent_tests/$(LLM)/test_cli_contract.cpp \
	  independent_tests/$(LLM)/test_manifest_format.cpp \
	  independent_tests/$(LLM)/test_diff_verify.cpp \
	  independent_tests/$(LLM)/test_describe_status.cpp
	cd independent_tests/$(LLM) && ./run_tests

man: $(NAME).1

$(NAME).1: $(NAME).1.md
	pandoc $(NAME).1.md -s -t man -o $(NAME).1

install: build man
	install -d $(DESTDIR)$(BINDIR)
	install -m 0755 $(NAME) $(DESTDIR)$(BINDIR)/$(NAME)
	install -d $(DESTDIR)$(MANDIR)
	install -m 0644 $(NAME).1 $(DESTDIR)$(MANDIR)/$(NAME).1
	install -d $(DESTDIR)$(ZYPPER_CMDDIR)
	ln -sf $(BINDIR)/$(NAME) $(DESTDIR)$(ZYPPER_CMDDIR)/zypper-declarative

clean:
	rm -rf $(BUILDDIR) ./$(NAME) $(NAME).1 \
	  independent_tests/$(LLM)/run_tests
