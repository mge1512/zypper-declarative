# generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
#
# Makefile for zypper-declarative (cli-tool, Go, static binary).

BINARY_NAME := zypper-declarative
SPEC_SHA256 := 714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
VERSION     := 0.4.0
LLM_NAME    := claude-opus-4-8

PREFIX  ?= /usr
BINDIR  ?= $(PREFIX)/bin
MANDIR  ?= $(PREFIX)/share/man
# The zypper subcommand directory (executable surfaced as `zypper declarative`).
ZYPPERCMDDIR ?= $(PREFIX)/lib/zypper/commands

GO       ?= go
PANDOC   ?= pandoc

.PHONY: all build test install clean man

all: build

# Static binary, no glibc dependency (CGO_ENABLED=0). Binary lands at the
# project root, the canonical BINARY-LOCATION the tests invoke.
build:
	CGO_ENABLED=0 $(GO) build -o $(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Executable test target: runs the translator's independent test suite.
# `go test` discovers *_test.go automatically and exits non-zero on failure.
test: build
	$(GO) test ./independent_tests/$(LLM_NAME)/...

man: $(BINARY_NAME).1

$(BINARY_NAME).1: $(BINARY_NAME).1.md
	$(PANDOC) $(BINARY_NAME).1.md -s -t man -o $(BINARY_NAME).1

install: build man
	install -D -m 0755 $(BINARY_NAME) $(DESTDIR)$(BINDIR)/$(BINARY_NAME)
	install -D -m 0755 $(BINARY_NAME) $(DESTDIR)$(ZYPPERCMDDIR)/$(BINARY_NAME)
	install -D -m 0644 $(BINARY_NAME).1 $(DESTDIR)$(MANDIR)/man1/$(BINARY_NAME).1

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).1
