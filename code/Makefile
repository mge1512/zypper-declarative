# Makefile for zypper-declarative
# generated from spec: zypper-declarative.spec.md
SPEC_SHA256 = f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2

BINARY_NAME = zypper-declarative
LLM_NAME    = claude-opus-4-8
VERSION     = 0.5.1
PREFIX     ?= /usr
BINDIR      = $(DESTDIR)$(PREFIX)/bin
ZYPPERCMDDIR = $(DESTDIR)$(PREFIX)/lib/zypper/commands
MANDIR      = $(DESTDIR)$(PREFIX)/share/man/man1

.PHONY: all build test install clean man

all: build

# Static binary, no glibc dependency (cli-tool template BINARY-TYPE: static).
# The binary is placed at the project root, where BINARY-LOCATION: project-root
# expects it and where the test suites invoke it as ../../$(BINARY_NAME).
build:
	CGO_ENABLED=0 go build -o $(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Executable test target: runs the independent test suite. Exits non-zero on any
# failure. Builds the binary first via the suite's TestMain.
test:
	go test ./independent_tests/$(LLM_NAME)/...

install: build man
	install -D -m 0755 $(BINARY_NAME) $(BINDIR)/$(BINARY_NAME)
	# Surface as a zypper subcommand: an executable in /usr/lib/zypper/commands.
	install -d $(ZYPPERCMDDIR)
	ln -sf $(PREFIX)/bin/$(BINARY_NAME) $(ZYPPERCMDDIR)/zypper-declarative
	install -D -m 0644 $(BINARY_NAME).1 $(MANDIR)/$(BINARY_NAME).1

# Man page: markdown source converted to troff via pandoc.
man: $(BINARY_NAME).1

$(BINARY_NAME).1: $(BINARY_NAME).1.md
	pandoc $(BINARY_NAME).1.md -s -t man -o $(BINARY_NAME).1

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).1
