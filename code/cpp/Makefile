# generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
# Thin wrapper over CMake providing the conventional targets.
SPEC_SHA256 := aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
VERSION     := $(shell cat VERSION)
NAME        := zypper-declarative

BUILD_DIR   := build
# On SLE 15 SP7 select the side-by-side GCC 15 toolchain (the default gcc-c++ is
# GCC 7, too old for clean C++17). On SLE 16 the default is already GCC 15.
# Override on the command line (make CXX=g++) for a host whose default is GCC 15.
CXX         := g++-15

# Translator's own independent test suite.
TR_LLM      := claude-opus-4-8
TR_DIR      := independent_tests/$(TR_LLM)
# Test-author's independent test suite (dual-LLM cross-check).
TA_LLM      := gemini-3-5-flash
TA_DIR      := independent_tests/$(TA_LLM)

.PHONY: build test install clean man dist

build:
	CXX=$(CXX) cmake -S . -B $(BUILD_DIR) -DCMAKE_BUILD_TYPE=Release
	cmake --build $(BUILD_DIR) -j
	cp -f $(BUILD_DIR)/$(NAME) ./$(NAME)

# Build the binary, then compile and run both independent black-box suites.
# Each suite invokes ../../$(NAME) (the project-root binary). The test-author
# harness registers test cases via static initialisers whose construction must
# follow the registry vector's; its translation unit test_utils.cpp is listed
# FIRST on the link line so the registry is constructed before any registration
# runs.
test: build
	$(CXX) -std=c++17 -Wall -Wextra -o $(TR_DIR)/zd_tests \
	    $(TR_DIR)/harness.cpp $(TR_DIR)/zypper_declarative_test.cpp
	cd $(TR_DIR) && ./zd_tests
	$(CXX) -std=c++17 -o $(TA_DIR)/ta_tests \
	    $(TA_DIR)/test_utils.cpp $(TA_DIR)/main.cpp \
	    $(TA_DIR)/test_global.cpp $(TA_DIR)/test_status.cpp \
	    $(TA_DIR)/test_diff.cpp $(TA_DIR)/test_verify.cpp \
	    $(TA_DIR)/test_apply.cpp $(TA_DIR)/test_describe.cpp \
	    $(TA_DIR)/test_model.cpp
	cd $(TA_DIR) && ./ta_tests

man: $(NAME).1

$(NAME).1: $(NAME).1.md
	pandoc $(NAME).1.md -s -t man -o $(NAME).1

install: build man
	install -D -m0755 $(NAME) $(DESTDIR)/usr/lib/zypper/commands/$(NAME)
	install -D -m0644 $(NAME).1 $(DESTDIR)/usr/share/man/man1/$(NAME).1

clean:
	rm -rf $(BUILD_DIR) ./$(NAME) $(NAME).1 \
	    $(TR_DIR)/zd_tests $(TA_DIR)/ta_tests

# Release source tarball: zypper-declarative-$(VERSION).tar.gz containing a
# single top-level directory zypper-declarative-$(VERSION)/ (so rpmbuild's
# default %setup finds it). Build artefacts and VCS dirs are excluded.
dist:
	rm -rf dist-tmp
	mkdir -p dist-tmp/$(NAME)-$(VERSION)
	tar --exclude=./build --exclude='./dist-tmp' --exclude='*.tar.gz' \
	    --exclude='./.git' --exclude='./$(NAME)' \
	    --exclude='*/zd_tests' --exclude='*/ta_tests' \
	    -cf - . | tar -xf - -C dist-tmp/$(NAME)-$(VERSION)
	tar -czf $(NAME)-$(VERSION).tar.gz -C dist-tmp $(NAME)-$(VERSION)
	rm -rf dist-tmp
	@echo "created $(NAME)-$(VERSION).tar.gz"
