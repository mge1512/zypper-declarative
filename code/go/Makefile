# generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
#
# Makefile for zypper-declarative (cli-tool, Go). Static binary (CGO_ENABLED=0).

BINARY_NAME = zypper-declarative
SPEC_SHA256 = 27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
LLM_NAME    = claude-opus-4-8
PREFIX     ?= /usr
BINDIR     ?= $(PREFIX)/bin
MANDIR     ?= $(PREFIX)/share/man
ZYPPERCMDS ?= $(PREFIX)/lib/zypper/commands

.PHONY: all build test install clean man

all: build

build:
	CGO_ENABLED=0 go build -o $(BINARY_NAME) ./cmd/$(BINARY_NAME)

test: build
	go test ./independent_tests/$(LLM_NAME)/...

install: build man
	install -D -m 0755 $(BINARY_NAME) $(DESTDIR)$(BINDIR)/$(BINARY_NAME)
	install -D -m 0755 $(BINARY_NAME) $(DESTDIR)$(ZYPPERCMDS)/$(BINARY_NAME)
	install -D -m 0644 $(BINARY_NAME).1 $(DESTDIR)$(MANDIR)/man1/$(BINARY_NAME).1

man: $(BINARY_NAME).1

$(BINARY_NAME).1: $(BINARY_NAME).1.md
	pandoc $(BINARY_NAME).1.md -s -t man -o $(BINARY_NAME).1

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).1
