# generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
#
# Makefile for zypper-declarative (cli-tool, Go, static binary).

BINARY_NAME    := zypper-declarative
SPEC_SHA256    := 58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
VERSION        := 0.5.0
LLM_NAME       := claude-opus-4-8

.PHONY: build test install clean man

# build: produce the single static binary at the project root
# (BINARY-LOCATION: project-root), with no glibc dependency.
build:
	CGO_ENABLED=0 go build -o $(BINARY_NAME) ./cmd/$(BINARY_NAME)

# test: run the translator test suite. The suite builds the binary under test
# via its own TestMain at the canonical project-root location.
test: build
	go test ./independent_tests/$(LLM_NAME)/...

# install: install the binary as a zypper subcommand and directly on PATH.
install: build
	install -D -m 0755 $(BINARY_NAME) $(DESTDIR)/usr/lib/zypper/commands/$(BINARY_NAME)
	install -D -m 0755 $(BINARY_NAME) $(DESTDIR)/usr/bin/$(BINARY_NAME)
	install -D -m 0644 $(BINARY_NAME).1 $(DESTDIR)/usr/share/man/man1/$(BINARY_NAME).1

# man: render the troff man page from its Markdown source via pandoc.
man:
	pandoc $(BINARY_NAME).1.md -s -t man -o $(BINARY_NAME).1

# clean: remove build outputs.
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).1
