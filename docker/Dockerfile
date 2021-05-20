FROM golang:1.16-alpine AS builder

# See: https://docs.github.com/en/packages/guides/connecting-a-repository-to-a-container-image#connecting-a-repository-to-a-container-image-on-the-command-line
LABEL org.opencontainers.image.source=https://github.com/SB-IM/sphinx

RUN apk update && apk add --no-cache \
    build-base \
    git \
    openssh-client

WORKDIR /src

COPY go.mod .

# See: https://docs.docker.com/develop/develop-images/build_enhancements/
# http://blog.oddbit.com/post/2019-02-24-docker-build-learns-about-secr/
RUN --mount=type=ssh,id=github git config --global url."git@github.com:".insteadOf "https://github.com/"; \
    mkdir -p -m 0600 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts; \
    go env -w GOPRIVATE="github.com/SB-IM"; \
    go mod download; \
    go mod verify

COPY . .

ARG DEBUG=false

RUN DEBUG=${DEBUG} make sphinx

FROM alpine AS bin

RUN apk add --no-cache ca-certificates

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /src/sphinx /usr/bin/sphinx

RUN addgroup -g 10001 -S sphinx && \
    adduser -u 10001 -S sphinx -G sphinx

USER sphinx

EXPOSE 5004/udp 5005/udp

ENV DEBUG_MQTT_CLIENT=false

VOLUME [ "/etc/sphinx" ]

ENTRYPOINT [ "/usr/bin/sphinx" ]
