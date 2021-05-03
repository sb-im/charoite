# Project name.
PROJECT_NAME:=skywalker

# SSH private key set up.
CURRENT_USER?=william
PRIVATE_KEY_FILE?=id_ed25519
PRIVATE_KEY_PATH?=github=/home/$(CURRENT_USER)/.ssh/$(PRIVATE_KEY_FILE)
PROJECT_DIR?=/home/$(CURRENT_USER)/go/src/github.com/SB-IM/skywalker

# Project image repo.
IMAGE?=ghcr.io/sb-im/skywalker:latest-dev

.PHONY: run-broadcast
run-broadcast:
	@DEBUG_MQTT_CLIENT=false go run -race ./cmd --debug broadcast -c config/config.dev.toml

.PHONY: run-turn
run-turn:
	@DEBUG_MQTT_CLIENT=false go run -race ./cmd --debug turn -c config/config.dev.toml

.PHONY: build
build:
	@go build -race -o $(PROJECT_NAME) ./cmd

.PHONY: lint
lint:
	@golangci-lint run ./...

.PHONY: image
image:
	@docker build \
	--ssh $(PRIVATE_KEY_PATH) \
	-t $(IMAGE) \
	-f docker/Dockerfile.dev .

.PHONY: push
push:
	@docker push $(IMAGE)

# Note: '--env-file' value is relative to '-f' value's directory.
.PHONY: up
up: down image
	@docker-compose -f docker/docker-compose.dev.yaml up -d

.PHONY: down
down:
	@docker-compose -f docker/docker-compose.dev.yaml down --remove-orphans

.PHONY: broker
broker:
	@docker run -d --rm --name mosquitto -p 1883:1883 -p 9001:9001 -v $(PROJECT_DIR)/config/mosquitto.conf:/mosquitto/config/mosquitto.conf eclipse-mosquitto:2

.PHONY: stop-broker
stop-broker:
	@docker stop mosquitto

.PHONY: logs
logs:
	@docker-compose -f docker/docker-compose.dev.yaml logs -f

.PHONY: log-livestream
log-livestream:
	@docker-compose -f docker/docker-compose.dev.yaml logs -f livestream

.PHONY: log-broadcast
log-broadcast:
	@docker-compose -f docker/docker-compose.dev.yaml logs -f broadcast

.PHONY: e2e-broadcast
e2e-broadcast:
	@go run ./e2e/broadcast
