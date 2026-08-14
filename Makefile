.DEFAULT_GOAL := build

GO ?= go
BIN_DIR := dist
PREFIX ?= /usr/local
MAN_DIR := man
MAN_BUILD_DIR := $(BIN_DIR)/man
COMMANDS := $(sort $(notdir $(wildcard cmd/*)))
BINARIES := $(addprefix $(BIN_DIR)/,$(COMMANDS))
SOURCES := $(shell find cmd internal -type f -name '*.go')

.PHONY: build test check man man-build man-install clean contrib contrib-update help

build: $(BINARIES)

$(BIN_DIR)/%: $(SOURCES) | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/$*

$(BIN_DIR):
	mkdir -p $@

test:
	$(GO) test ./cmd/... ./internal/... ./tests

check:
	$(GO) vet ./cmd/... ./internal/... ./tests
	$(GO) test ./cmd/... ./internal/... ./tests

man:
	@find $(MAN_DIR) -type f -name '*.[1-9]' -print | sort

man-build:
	@mkdir -p $(MAN_BUILD_DIR)
	@rm -f $(MAN_BUILD_DIR)/*.1.gz
	@for page in $(MAN_DIR)/*.1; do test -s "$$page" || exit 1; gzip -9 -c "$$page" > "$(MAN_BUILD_DIR)/$$(basename "$$page").gz"; done

man-install: man-build
	@mkdir -p $(PREFIX)/share/man/man1
	@install -m 0644 $(MAN_BUILD_DIR)/*.1.gz $(PREFIX)/share/man/man1/

contrib:
	git submodule update --init --recursive

contrib-update:
	git submodule update --remote --merge --recursive

clean:
	rm -rf $(BIN_DIR)

help:
	@printf '%s\n' 'Usage: make {build|test|check|man|man-build|man-install|contrib|contrib-update|clean}'
	@printf '%s\n' 'build creates one binary in dist/ for every cmd/ directory.'
	@printf '%s\n' 'contrib initializes companion-tool submodules under contrib/.'
	@printf '%s\n' 'man lists source man pages packaged with releases.'
	@printf '%s\n' 'man-build creates gzip-compressed pages in dist/man; man-install installs them under PREFIX/share/man/man1 (PREFIX defaults to /usr/local).'
	@printf '%s\n' 'contrib-update refreshes those submodules from their tracked branches.'
