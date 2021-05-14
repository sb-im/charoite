# Project name.
PROJECT_NAME:=skywalker

# SSH private key set up.
CURRENT_USER?=william
PRIVATE_KEY_FILE?=id_ed25519
PRIVATE_KEY_PATH?=github=/home/$(CURRENT_USER)/.ssh/$(PRIVATE_KEY_FILE)
PROJECT_DIR?=/home/$(CURRENT_USER)/go/src/github.com/SB-IM/skywalker

# Project image repo.
IMAGE?=ghcr.io/sb-im/skywalker:latest-dev

# Docker-compose file.
DOCKER_COMPOSE_FILE?=docker/docker-compose.yml

# Docker-compose service.
SERVICE?=

# Skywalker service commands.
COMMAND?=broadcast

.PHONY: run
run:
	@DEBUG_MQTT_CLIENT=false go run -race ./cmd --debug $(COMMAND) -c config/config.dev.toml

skywalker:
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
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

.PHONY: down
down:
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down --remove-orphans

.PHONY: broker
broker:
	@docker run -d --rm --name mosquitto -p 1883:1883 -p 9001:9001 -v $(PROJECT_DIR)/config/mosquitto.conf:/mosquitto/config/mosquitto.conf eclipse-mosquitto:2

.PHONY: stop-broker
stop-broker:
	@docker stop mosquitto

.PHONY: logs
logs:
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f $(SERVICE)

.PHONY: e2e-broadcast
e2e-broadcast:
	@go run ./e2e/broadcast

.PHONY: clean
clean:
	@rm -rf skywalker
