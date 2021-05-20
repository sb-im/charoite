# SSH private key set up.
CURRENT_USER ?= william
PRIVATE_KEY_FILE ?= id_ed25519
PRIVATE_KEY_PATH ?= github=$(shell getent passwd "$(CURRENT_USER)" | cut -d: -f6)/.ssh/$(PRIVATE_KEY_FILE)

# Enable docker buildkit.
DOCKER_BUILDKIT = 1
# Project image repo.
IMAGE ?= ghcr.io/sb-im/sphinx
IMAGE_TAG ?= latest
# OCI platform.
OCI_PLATFORM ?= linux/arm64
# Docker-compose file.
DOCKER_COMPOSE_FILE ?= docker/docker-compose.yml
# Docker-compose service.
SERVICE ?=

# Version info for binaries.
GIT_REVISION := $(shell git rev-parse --short HEAD)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GIT_TAG := $(shell git describe --tags)

# go build flags.
VPREFIX := github.com/SB-IM/sphinx/cmd/build
GO_LDFLAGS := -X $(VPREFIX).Branch=$(GIT_BRANCH) -X $(VPREFIX).Version=$(GIT_TAG) -X $(VPREFIX).Revision=$(GIT_REVISION) -X $(VPREFIX).BuildUser=$(shell whoami)@$(shell hostname) -X $(VPREFIX).BuildDate=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_FLAGS := -ldflags "-extldflags \"-static\" -s -w $(GO_LDFLAGS)" -a -installsuffix cgo
# See: https://golang.org/doc/gdb#Introduction
DEBUG_GO_FLAGS := -race -gcflags "all=-N -l" -ldflags "-extldflags \"-static\" $(GO_LDFLAGS)"
# go build with -race flag must enable cgo.
CGO_ENABLED := 0

DEBUG ?= false
ifeq ($(DEBUG), true)
	IMAGE_TAG := debug
	GO_FLAGS := $(DEBUG_GO_FLAGS)
	CGO_ENABLED := 1
endif
ifeq ($(OCI_PLATFORM), linux/arm64)
	IMAGE_TAG := $(IMAGE_TAG)-arm64
else ifeq ($(OCI_PLATFORM), linux/amd64)
	IMAGE_TAG := $(IMAGE_TAG)-amd64
endif

.PHONY: run
run:
	@DEBUG_MQTT_CLIENT=false go run -race ./cmd --debug $(SERVICE) -c config/config.dev.toml

sphinx:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux go build $(GO_FLAGS) -o $@ ./cmd

sphinx-hookstream:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm64 go build $(GO_FLAGS) -o $@ ./cmd

.PHONY: lint
lint:
	@golangci-lint run ./...

# binfmt is the requisite for docker cross platform build.
binfmt:
	@docker run --privileged --rm tonistiigi/binfmt --install all

.PHONY: image
image:
	docker buildx build \
	--build-arg DEBUG=$(DEBUG) \
	--platform $(OCI_PLATFORM) \
	--ssh $(PRIVATE_KEY_PATH) \
	-t $(IMAGE):$(IMAGE_TAG) \
	-f docker/Dockerfile \
	.

.PHONY: push
push:
	@docker push $(IMAGE)-arm64

# Note: '--env-file' value is relative to '-f' value's directory.
.PHONY: up
up: down image
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

.PHONY: down
down:
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down --remove-orphans

.PHONY: logs
logs:
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f $(SERVICE)

.PHONY: run-mosquitto
run-mosquitto:
	@docker run -d --rm --name mosquitto -p 1883:1883 -p 9001:9001 -v $$PWD/config/mosquitto.conf:/mosquitto/config/mosquitto.conf eclipse-mosquitto:2

.PHONY: stop-mosquitto
stop-mosquitto:
	@docker stop mosquitto

.PHONY: clean
clean:
	@rm -rf sphinx*
