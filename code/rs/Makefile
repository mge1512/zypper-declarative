# Makefile for zypper-declarative
# generated from spec: zypper-declarative.spec.md
# sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03

SPEC_SHA256 = 51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
BINARY      = zypper-declarative
LLM_NAME    = claude-opus-4-8
TARGET      = x86_64-unknown-linux-gnu
RELEASE_BIN = target/$(TARGET)/release/$(BINARY)
DESTDIR    ?=
PREFIX     ?= /usr
BINDIR      = $(DESTDIR)$(PREFIX)/bin
MANDIR      = $(DESTDIR)$(PREFIX)/share/man/man1
# zypper subcommand surface (an executable in /usr/lib/zypper/commands)
ZYPPER_CMDDIR = $(DESTDIR)$(PREFIX)/lib/zypper/commands

.PHONY: all build test install clean man

all: build

# Build the static release binary and copy it to the project root, where
# BINARY-LOCATION: project-root (and the test suites) expect it.
build:
	cargo build --release --offline || cargo build --release
	cp $(RELEASE_BIN) ./$(BINARY)

# Run the independent black-box test suite. cargo test exits non-zero on any
# test failure. The implementation must be built first (it is the binary under
# test); `build` is a prerequisite.
test: build
	cargo test --test '*' --manifest-path independent_tests/$(LLM_NAME)/Cargo.toml

install: build man
	install -d $(BINDIR)
	install -m 0755 ./$(BINARY) $(BINDIR)/$(BINARY)
	install -d $(MANDIR)
	install -m 0644 $(BINARY).1 $(MANDIR)/$(BINARY).1
	# Surface as a zypper subcommand: `zypper declarative ...`.
	install -d $(ZYPPER_CMDDIR)
	ln -sf $(PREFIX)/bin/$(BINARY) $(ZYPPER_CMDDIR)/zypper-declarative

man: $(BINARY).1

$(BINARY).1: $(BINARY).1.md
	pandoc $(BINARY).1.md -s -t man -o $(BINARY).1

clean:
	cargo clean
	rm -f ./$(BINARY) $(BINARY).1
