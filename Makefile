.DEFAULT_GOAL := build

GO ?= go
BIN_DIR := dist
COMMANDS := $(sort $(notdir $(wildcard cmd/*)))
BINARIES := $(addprefix $(BIN_DIR)/,$(COMMANDS))
SOURCES := $(shell find cmd internal -type f -name '*.go')

.PHONY: build test check clean contrib contrib-update help

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

contrib:
	git submodule update --init --recursive

contrib-update:
	git submodule update --remote --merge --recursive

clean:
	rm -rf $(BIN_DIR)

help:
	@printf '%s\n' 'Usage: make {build|test|check|contrib|contrib-update|clean}'
	@printf '%s\n' 'build creates one binary in dist/ for every cmd/ directory.'
	@printf '%s\n' 'contrib initializes companion-tool submodules under contrib/.'
	@printf '%s\n' 'contrib-update refreshes those submodules from their tracked branches.'
