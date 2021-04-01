# SSH private key set up.
CURRENT_USER?=william
PRIVATE_KEY_FILE?=id_ed25519
PRIVATE_KEY_PATH?=github=/home/$(CURRENT_USER)/.ssh/$(PRIVATE_KEY_FILE)
PROJECT_DIR?=/home/${CURRENT_USER}/go/src/github.com/SB-IM/sphinx

# Project image repo.
IMAGE_LIVESTREAM?=ghcr.io/sb-im/sphinx:livestream-amd64-latest

.PHONY: run
run:
	@DEBUG_MQTT_CLIENT=false go run -race ./cmd --debug livestream -c config/config.dev.yaml

.PHONY: lint
lint:
	@golangci-lint run ./...

.PHONY: image
image:
	@docker build \
	--ssh $(PRIVATE_KEY_PATH) \
	-t $(IMAGE_LIVESTREAM) \
	-f docker/Dockerfile.livestream.dev .

# Note: '--env-file' value is relative to '-f' value's directory.
.PHONY: up
up: down image
	@docker-compose -f docker/docker-compose.dev.yaml up -d

.PHONY: down
down:
	@docker-compose -f docker/docker-compose.dev.yaml down

.PHONY: logs
logs:
	@docker-compose -f docker/docker-compose.dev.yaml logs -t -f

.PHONY: broker
broker:
	@docker run -d --rm --name mosquitto -p 1883:1883 -p 9001:9001 -v ${PROJECT_DIR}/config/mosquitto.conf:/mosquitto/config/mosquitto.conf eclipse-mosquitto:2

.PHONY: stop-broker
stop-broker:
	@docker stop mosquitto
