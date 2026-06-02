# generated from spec: zypper-declarative.spec.md
# Makefile for zypper-declarative (cli-tool, Go, static binary)

SPEC_SHA256 := 51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
BINARY_NAME := zypper-declarative
LLM_NAME    := claude-opus-4-8
PREFIX      ?= /usr
BINDIR      ?= $(PREFIX)/bin
MANDIR      ?= $(PREFIX)/share/man
ZYPPERCMDDIR ?= $(PREFIX)/lib/zypper/commands

.PHONY: all build test install clean man vendor

all: build

# Static binary (BINARY-TYPE: static; CGO_ENABLED=0), built at the project root
# (BINARY-LOCATION: project-root) where the tests invoke it as ../../zypper-declarative.
build:
	CGO_ENABLED=0 go build -o $(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Executable test target: runs the suite and exits non-zero on any failure.
test: build
	go test ./independent_tests/$(LLM_NAME)/...

man: $(BINARY_NAME).1

$(BINARY_NAME).1: $(BINARY_NAME).1.md
	pandoc $(BINARY_NAME).1.md -s -t man -o $(BINARY_NAME).1

install: build man
	install -D -m 0755 $(BINARY_NAME) $(DESTDIR)$(BINDIR)/$(BINARY_NAME)
	install -D -m 0755 $(BINARY_NAME) $(DESTDIR)$(ZYPPERCMDDIR)/$(BINARY_NAME)
	install -D -m 0644 $(BINARY_NAME).1 $(DESTDIR)$(MANDIR)/man1/$(BINARY_NAME).1

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).1

vendor:
	go mod vendor
