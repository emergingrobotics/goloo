# Goloo - Unified VM Provisioning
# Usage: make [target]

BINARY := goloo
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
INSTALL_DIR ?= $(HOME)/bin

.PHONY: help build run-tests clean install

help:
	@echo "Goloo - Unified VM Provisioning"
	@echo ""
	@echo "Build:"
	@echo "  make build                 Build the goloo binary"
	@echo "  make run-tests             Run all tests"
	@echo "  make clean                 Remove build artifacts"
	@echo "  make install               Install to $(INSTALL_DIR)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/goloo

run-tests:
	go test -v ./...

clean:
	rm -rf bin/

install: build
	mkdir -p $(INSTALL_DIR)
	cp bin/$(BINARY) $(INSTALL_DIR)/
