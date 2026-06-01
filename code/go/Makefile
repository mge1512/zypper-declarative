# generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
#
# Build automation for zypper-declarative (cli-tool, Go, static binary).

BINARY_NAME    := zypper-declarative
LLM_NAME       := claude-opus-4-8
SPEC_SHA256    := b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
PREFIX         ?= /usr
BINDIR         := $(PREFIX)/bin
MANDIR         := $(PREFIX)/share/man/man1
ZYPPERCMDDIR   := $(PREFIX)/lib/zypper/commands

.PHONY: all build test install clean man

all: build man

# Static binary at the project root, where BINARY-LOCATION: project-root and the
# tests' ../../$(BINARY_NAME) expect it. CGO_ENABLED=0 yields no glibc dependency.
build:
	CGO_ENABLED=0 go build -o $(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Executable test target: runs the black-box independent test suite. Exits
# non-zero on any failure. Go discovers *_test.go automatically.
test: build
	go test ./independent_tests/$(LLM_NAME)/...

# Man page: convert the Markdown source to troff via pandoc.
man: $(BINARY_NAME).1

$(BINARY_NAME).1: $(BINARY_NAME).1.md
	pandoc $(BINARY_NAME).1.md -s -t man -o $(BINARY_NAME).1

install: build man
	install -D -m 0755 $(BINARY_NAME) $(DESTDIR)$(BINDIR)/$(BINARY_NAME)
	# Surface as a zypper subcommand: an executable in /usr/lib/zypper/commands.
	install -d $(DESTDIR)$(ZYPPERCMDDIR)
	ln -sf $(BINDIR)/$(BINARY_NAME) $(DESTDIR)$(ZYPPERCMDDIR)/zypper-declarative
	install -D -m 0644 $(BINARY_NAME).1 $(DESTDIR)$(MANDIR)/$(BINARY_NAME).1

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).1
