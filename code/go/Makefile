# generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
#
# Makefile for zypper-declarative (cli-tool, Go, static binary).

BINARY_NAME := zypper-declarative
SPEC_SHA256 := f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
VERSION     := 0.6.2
LLM_NAME    := claude-opus-4-8

GO          := go
GOFLAGS     :=

.PHONY: all build test install clean man vendor

all: build

# Static binary at the project root (BINARY-LOCATION: project-root).
build:
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Executable test target: actually runs the suite, exits non-zero on failure.
test: build
	$(GO) test ./independent_tests/$(LLM_NAME)/...

# Man page: markdown source -> troff via pandoc.
man: $(BINARY_NAME).1

$(BINARY_NAME).1: $(BINARY_NAME).1.md
	pandoc $(BINARY_NAME).1.md -s -t man -o $(BINARY_NAME).1

# Install the binary as a zypper subcommand and directly invocable.
DESTDIR ?=
PREFIX  ?= /usr
install: build man
	install -D -m 0755 $(BINARY_NAME) $(DESTDIR)$(PREFIX)/bin/$(BINARY_NAME)
	install -D -m 0755 $(BINARY_NAME) $(DESTDIR)$(PREFIX)/lib/zypper/commands/$(BINARY_NAME)
	install -D -m 0644 $(BINARY_NAME).1 $(DESTDIR)$(PREFIX)/share/man/man1/$(BINARY_NAME).1

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).1

vendor:
	$(GO) mod vendor
