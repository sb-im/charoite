# SSH private key set up.
CURRENT_USER ?= $(shell whoami)
PRIVATE_KEY_FILE ?= id_ed25519
PRIVATE_KEY_PATH ?= github=${HOME}/.ssh/$(PRIVATE_KEY_FILE)

# Enable docker buildkit.
DOCKER_BUILDKIT = 1
# Project image repo.
IMAGE_REPO ?= ghcr.io/sb-im/charoite
IMAGE_TAG ?= latest

# Version info for binaries
GIT_REVISION := $(shell git rev-parse --short HEAD)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GIT_TAG := $(shell git describe --tags)

# go build flags.
VPREFIX := github.com/SB-IM/charoite/cmd/internal/info
GO_LDFLAGS := -X $(VPREFIX).Branch=$(GIT_BRANCH) -X $(VPREFIX).Version=$(GIT_TAG) -X $(VPREFIX).Revision=$(GIT_REVISION) -X $(VPREFIX).BuildUser=$(shell whoami)@$(shell hostname) -X $(VPREFIX).BuildDate=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_FLAGS := -trimpath -ldflags "-extldflags \"-static\" -s -w $(GO_LDFLAGS)" -a
# See: https://golang.org/doc/gdb#Introduction
DEBUG_GO_FLAGS := -trimpath -race -gcflags "all=-N -l" -ldflags "-extldflags \"-static\" $(GO_LDFLAGS)"
# go build with -race flag must enable cgo.
CGO_ENABLED := 0
# Default is linux.
GOOS ?= linux

# Go build tags.
BUILD_TAGS ?= broadcast

# Default command is the same as build tag.
COMMAND ?= $(BUILD_TAGS)

# Default is debug.
DEBUG ?= true
ifeq ($(DEBUG), true)
	IMAGE_TAG := debug
	GO_FLAGS := $(DEBUG_GO_FLAGS)
	CGO_ENABLED := 1
endif

.PHONY: run
run:
	DEBUG_MQTT_CLIENT=false go run -tags $(BUILD_TAGS) -race ./cmd --debug $(COMMAND) -c config/config.debug.toml

charoite:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) go build -tags $(BUILD_TAGS) $(GO_FLAGS) -o $@ ./cmd

.PHONY: lint
lint:
	@golangci-lint run ./...

.PHONY: image
image:
	@docker build \
	--build-arg DEBUG=$(DEBUG) \
	--build-arg BUILD_TAGS=$(BUILD_TAGS) \
	--ssh $(PRIVATE_KEY_PATH) \
	-t $(IMAGE_REPO):$(IMAGE_TAG)-$(BUILD_TAGS) \
	.

.PHONY: up
up: down
	@docker-compose up -d

.PHONY: down
down:
	@docker-compose down --remove-orphans

.PHONY: clean
clean:
	@rm -rf charoite
