# generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
#
# Makefile for zypper-declarative (Rust, cli-tool deployment).
#
# The build target produces a statically-linked binary (cli-tool template
# BINARY-TYPE: static) via crt-static against glibc, built with an explicit
# target triple so host proc-macro crates compile without crt-static, then
# copies the binary to the project root (BINARY-LOCATION: project-root).

BINARY      := zypper-declarative
SPEC_SHA256 := 18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
VERSION     := 0.6.4
TARGET      := x86_64-unknown-linux-gnu
LLM         := claude-opus-4-8

DESTDIR ?=
PREFIX  ?= /usr
BINDIR  := $(PREFIX)/bin
MANDIR  := $(PREFIX)/share/man/man1
# zypper subcommand location (the tool is surfaced as `zypper declarative`).
ZYPPERCMDDIR := $(PREFIX)/lib/zypper/commands

.PHONY: all build test install clean man static dynamic vendor

all: build

# Static release build (glibc + crt-static), copied to the project root.
build: static

static:
	RUSTFLAGS='-C target-feature=+crt-static' cargo build --release --target $(TARGET)
	cp target/$(TARGET)/release/$(BINARY) ./$(BINARY)

# A plain dynamic build (used by the test harness and the compile gate). It does
# not require crt-static and works with the default host target.
dynamic:
	cargo build --release
	cp target/release/$(BINARY) ./$(BINARY)

# Run the independent black-box test suite. The suite invokes the binary at the
# project root; build it first if absent.
test: dynamic man
	cd independent_tests/$(LLM) && cargo test --test cli

man: $(BINARY).1

$(BINARY).1: $(BINARY).1.md
	pandoc $(BINARY).1.md -s -t man -o $(BINARY).1

install: build man
	install -D -m 0755 $(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)
	install -D -m 0644 $(BINARY).1 $(DESTDIR)$(MANDIR)/$(BINARY).1
	# Surface as a zypper subcommand: `zypper declarative` dispatches here.
	install -d $(DESTDIR)$(ZYPPERCMDDIR)
	ln -sf $(BINDIR)/$(BINARY) $(DESTDIR)$(ZYPPERCMDDIR)/zypper-declarative

# Vendor dependencies for an offline build (no network at build time on the
# target). Produces a vendor/ tree and the .cargo/config.toml fragment to use it.
vendor:
	cargo vendor vendor

clean:
	cargo clean
	rm -f ./$(BINARY) $(BINARY).1
