# generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
#
# Build, test, install, clean, and man-page targets for zypper-declarative.
# The binary is a single static Rust binary (BINARY-TYPE: static), placed at the
# project root (BINARY-LOCATION: project-root) where both the translator's tests
# and the test-author's tests invoke it as ../../zypper-declarative.

BINARY      := zypper-declarative
SPEC_SHA256 := 87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
VERSION     := 0.6.3
LLM_NAME    := claude-opus-4-8
TARGET      := x86_64-unknown-linux-gnu

DESTDIR ?=
PREFIX  ?= /usr
BINDIR  := $(PREFIX)/bin
MANDIR  := $(PREFIX)/share/man/man1

# Static linking against glibc is configured in .cargo/config.toml (default build
# target + crt-static), so a plain `cargo build --release` produces a static
# binary at target/$(TARGET)/release/$(BINARY).
RELEASE_BIN := target/$(TARGET)/release/$(BINARY)

.PHONY: all build test install clean man

all: build

build:
	cargo build --release
	cp $(RELEASE_BIN) ./$(BINARY)

# `make test` is executable: it builds the binary, then runs the black-box
# integration suite, which invokes ../../$(BINARY). Exits non-zero on any failure.
test: build
	cargo test --lib --release
	cargo test --test '*' --manifest-path independent_tests/$(LLM_NAME)/Cargo.toml

install: build man
	install -D -m 0755 ./$(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)
	install -D -m 0644 $(BINARY).1 $(DESTDIR)$(MANDIR)/$(BINARY).1
	# Surface as a zypper subcommand: an executable under /usr/lib/zypper/commands.
	install -D -m 0755 ./$(BINARY) $(DESTDIR)$(PREFIX)/lib/zypper/commands/$(BINARY)

man: $(BINARY).1

$(BINARY).1: $(BINARY).1.md
	pandoc $(BINARY).1.md -s -t man -o $(BINARY).1

clean:
	cargo clean
	rm -f ./$(BINARY) $(BINARY).1
