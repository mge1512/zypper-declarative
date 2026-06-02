# generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
#
# Build orchestration for zypper-declarative (C++17, CMake, dynamic linking).
# On SLE 15 SP7 select the side-by-side GCC 15 toolchain (CXX=g++-15); on
# SLE 16.0 the default toolchain is already GCC 15.

SPEC_SHA256 := 51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
NAME        := zypper-declarative
VERSION     := 0.6.6
BUILD_DIR   := build

LLM_NAME    := claude-opus-4-8
TESTDIR     := independent_tests/$(LLM_NAME)

# Pick GCC 15 for C++17 when a side-by-side g++-15 exists (SLE 15 SP7); else
# the default toolchain (SLE 16.0 ships GCC 15 as default). Note: a bare
# `CXX ?=` does not work here because Make predefines CXX=g++.
CXX := $(shell command -v g++-15 2>/dev/null || command -v c++)

.PHONY: all build test install clean man

all: build

# build: configure + compile with CMake, then copy the binary to the project
# root (BINARY-LOCATION: project-root), where both test suites invoke it.
build:
	cmake -S . -B $(BUILD_DIR) -DCMAKE_BUILD_TYPE=Release \
		-DCMAKE_CXX_COMPILER=$(CXX)
	cmake --build $(BUILD_DIR) -j
	cp -f $(BUILD_DIR)/$(NAME) ./$(NAME)

# test: compile and run the black-box test suite, which invokes ../../$(NAME).
# Exits non-zero on any test failure.
test: build
	$(CXX) -std=c++17 -Wall -Wextra -o $(TESTDIR)/run_tests \
		$(TESTDIR)/zypper_declarative_test.cpp
	cd $(TESTDIR) && ./run_tests
	rm -f $(TESTDIR)/run_tests

# man: render the troff man page from its Markdown source.
man: $(NAME).1

$(NAME).1: $(NAME).1.md
	pandoc $(NAME).1.md -s -t man -o $(NAME).1

install: build man
	install -D -m0755 $(NAME) $(DESTDIR)/usr/bin/$(NAME)
	install -D -m0755 $(NAME) $(DESTDIR)/usr/lib/zypper/commands/$(NAME)
	install -D -m0644 $(NAME).1 $(DESTDIR)/usr/share/man/man1/$(NAME).1

clean:
	rm -rf $(BUILD_DIR) ./$(NAME) $(NAME).1 $(TESTDIR)/run_tests
