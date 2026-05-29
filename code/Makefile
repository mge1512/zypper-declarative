# generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
#
# Makefile for zypper-declarative (cli-tool, Go). Single static binary built at
# the project root (BINARY-LOCATION: project-root).

BINARY_NAME   := zypper-declarative
SPEC_SHA256   := 714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
VERSION       := 0.4.0
TESTS_DIR     := independent_tests/claude-opus-4-8
PREFIX        ?= /usr
BINDIR        := $(PREFIX)/bin
MANDIR        := $(PREFIX)/share/man/man1
ZYPPER_CMDDIR := $(PREFIX)/lib/zypper/commands

.PHONY: all build test install clean man

all: build

build:
	CGO_ENABLED=0 go build -o $(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Executable test target: runs the translator test suite (Go discovers *_test.go
# automatically). Exits non-zero on any test failure.
test: build
	cd $(TESTS_DIR) && go test ./...

man: $(BINARY_NAME).1

$(BINARY_NAME).1: $(BINARY_NAME).1.md
	pandoc $(BINARY_NAME).1.md -s -t man -o $(BINARY_NAME).1

install: build man
	install -D -m 0755 $(BINARY_NAME) $(DESTDIR)$(BINDIR)/$(BINARY_NAME)
	install -D -m 0755 $(BINARY_NAME) $(DESTDIR)$(ZYPPER_CMDDIR)/zypper-declarative
	install -D -m 0644 $(BINARY_NAME).1 $(DESTDIR)$(MANDIR)/$(BINARY_NAME).1

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).1
