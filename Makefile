# Goloo - Unified VM Provisioning
# Usage: make [target]

BINARY := goloo
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
INSTALL_DIR ?= $(HOME)/bin

# Multipass defaults
NAME ?= dev
CONFIG ?= configs/dev.yaml
IMAGE ?= 24.04
CPUS ?= 4
MEM ?= 4G
DISK ?= 40G
MOUNT_SRC ?=
MOUNT_DST ?=
SNAP ?= snapshot

SSH_KEY_FILE := $(shell if [ -f ~/.ssh/id_ed25519.pub ]; then echo ~/.ssh/id_ed25519.pub; elif [ -f ~/.ssh/id_rsa.pub ]; then echo ~/.ssh/id_rsa.pub; else echo ""; fi)

.PHONY: help build run-tests clean install lint vm dev base python go-dev node claude ssh shell stop start delete purge mount unmount snapshot restore networks validate list info

help:
	@echo "Goloo - Unified VM Provisioning"
	@echo ""
	@echo "Build:"
	@echo "  make build                 Build the goloo binary"
	@echo "  make run-tests             Run all tests"
	@echo "  make clean                 Remove build artifacts"
	@echo "  make install               Install to $(INSTALL_DIR)"
	@echo "  make lint                  Run linter"
	@echo ""
	@echo "Multipass Quick Start:"
	@echo "  make dev                   Create default development VM"
	@echo "  make claude                Create full Claude Code development VM"
	@echo "  make ssh NAME=vmname       SSH into VM"
	@echo "  make list                  List all VMs"
	@echo ""
	@echo "VM Creation:"
	@echo "  make vm NAME=n CONFIG=c    Create VM with custom config"
	@echo "  make base                  Create minimal base VM"
	@echo "  make python                Create Python development VM"
	@echo "  make go-dev                Create Go development VM"
	@echo "  make node                  Create Node.js development VM"
	@echo ""
	@echo "VM Management:"
	@echo "  make info NAME=n           Show VM details"
	@echo "  make shell NAME=n          Open shell in VM"
	@echo "  make stop NAME=n           Stop VM"
	@echo "  make start NAME=n          Start VM"
	@echo "  make delete NAME=n         Delete VM"
	@echo "  make purge NAME=n          Delete VM and purge"

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/goloo

run-tests:
	go test -v ./...

clean:
	rm -rf bin/

install: build
	mkdir -p $(INSTALL_DIR)
	cp bin/$(BINARY) $(INSTALL_DIR)/

lint:
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "Install golangci-lint for more checks"

list:
	@multipass list

info:
	@multipass info $(NAME)

networks:
	@multipass networks

validate:
	@echo "Validating $(CONFIG)..."
	@./scripts/launch.sh --validate-only $(CONFIG)

vm:
	@./scripts/launch.sh --name $(NAME) --config $(CONFIG) --image $(IMAGE) --cpus $(CPUS) --memory $(MEM) --disk $(DISK)

dev:
	@./scripts/launch.sh --name dev --config configs/dev.yaml --image $(IMAGE) --cpus 4 --memory 4G --disk 40G

base:
	@./scripts/launch.sh --name base --config configs/base.yaml --image $(IMAGE) --cpus 2 --memory 2G --disk 20G

python:
	@./scripts/launch.sh --name python-dev --config configs/python-dev.yaml --image $(IMAGE) --cpus 4 --memory 4G --disk 40G

go-dev:
	@./scripts/launch.sh --name go-dev --config configs/go-dev.yaml --image $(IMAGE) --cpus 4 --memory 4G --disk 40G

node:
	@./scripts/launch.sh --name node-dev --config configs/node-dev.yaml --image $(IMAGE) --cpus 4 --memory 4G --disk 40G

claude:
	@./scripts/launch.sh --name claude-dev --config configs/claude-dev.yaml --image $(IMAGE) --cpus 4 --memory 8G --disk 80G

ssh:
	@multipass shell $(NAME)

shell:
	@multipass shell $(NAME)

stop:
	@multipass stop $(NAME)

start:
	@multipass start $(NAME)

delete:
	@multipass delete $(NAME)

purge:
	@multipass delete $(NAME) && multipass purge

mount:
ifndef MOUNT_SRC
	$(error MOUNT_SRC is required)
endif
ifndef MOUNT_DST
	$(error MOUNT_DST is required)
endif
	@multipass mount $(MOUNT_SRC) $(NAME):$(MOUNT_DST)

unmount:
ifndef MOUNT_DST
	$(error MOUNT_DST is required)
endif
	@multipass unmount $(NAME):$(MOUNT_DST)

snapshot:
	@multipass snapshot $(NAME) --name $(SNAP)

restore:
	@multipass restore $(NAME).$(SNAP)
