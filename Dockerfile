FROM golang:1.19-alpine AS builder

# See: https://docs.github.com/en/packages/guides/connecting-a-repository-to-a-container-image#connecting-a-repository-to-a-container-image-on-the-command-line
LABEL org.opencontainers.image.source=https://github.com/SB-IM/charoite

RUN apk update && apk add --no-cache \
    build-base \
    git \
    openssh-client

WORKDIR /src

COPY go.mod .

RUN go mod download all; \
    go mod verify

COPY . .

ARG DEBUG=true
ARG BUILD_TAGS=broadcast

RUN make charoite DEBUG=${DEBUG} BUILD_TAGS=${BUILD_TAGS}

FROM alpine AS bin

RUN apk add --no-cache ca-certificates

COPY --from=builder /src/charoite /usr/bin/charoite

RUN addgroup -g 10001 -S charoite && \
    adduser -u 10001 -S charoite -G charoite

USER charoite

VOLUME [ "/etc/charoite" ]

ENTRYPOINT [ "/usr/bin/charoite" ]
