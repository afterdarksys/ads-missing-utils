.DEFAULT_GOAL := build

GO ?= go
BIN_DIR := dist

.PHONY: build test check clean

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -o $(BIN_DIR)/jwalk ./cmd/jwalk
	$(GO) build -trimpath -o $(BIN_DIR)/envsub ./cmd/envsub
	$(GO) build -trimpath -o $(BIN_DIR)/hashsum ./cmd/hashsum

test:
	$(GO) test ./...

check:
	$(GO) vet ./...
	$(GO) test ./...

clean:
	rm -rf $(BIN_DIR)
