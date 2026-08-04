.DEFAULT_GOAL := build

GO ?= go
BIN_DIR := dist
COMMANDS := $(sort $(notdir $(wildcard cmd/*)))
BINARIES := $(addprefix $(BIN_DIR)/,$(COMMANDS))
SOURCES := $(shell find cmd internal -type f -name '*.go')

.PHONY: build test check clean help

build: $(BINARIES)

$(BIN_DIR)/%: $(SOURCES) | $(BIN_DIR)
	$(GO) build -trimpath -o $@ ./cmd/$*

$(BIN_DIR):
	mkdir -p $@

test:
	$(GO) test ./...

check:
	$(GO) vet ./...
	$(GO) test ./...

clean:
	rm -rf $(BIN_DIR)

help:
	@printf '%s\n' 'Usage: make {build|test|check|clean}'
	@printf '%s\n' 'build creates one binary in dist/ for every cmd/ directory.'
