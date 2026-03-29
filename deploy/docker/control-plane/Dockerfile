FROM golang:1.26.1 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 go build -o /usr/local/bin/control-plane ./cmd/control-plane

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       curl ca-certificates docker.io \
       iproute2 procps nftables wireguard-tools nsenter \
    && rm -rf /var/lib/apt/lists/*

RUN ARCH=$(dpkg --print-architecture) \
    && curl -fsSL -o /tmp/sing-box.tar.gz \
        "https://github.com/SagerNet/sing-box/releases/download/v1.11.7/sing-box-1.11.7-linux-${ARCH}.tar.gz" \
    && tar -xzf /tmp/sing-box.tar.gz -C /tmp \
    && mv /tmp/sing-box-*/sing-box /usr/local/bin/sing-box \
    && chmod +x /usr/local/bin/sing-box \
    && rm -rf /tmp/sing-box*

COPY --from=builder /usr/local/bin/control-plane /usr/local/bin/control-plane

WORKDIR /app
COPY deploy/docker/managed-user/image.lock deploy/docker/managed-user/image.lock

EXPOSE 8080

CMD ["control-plane"]
