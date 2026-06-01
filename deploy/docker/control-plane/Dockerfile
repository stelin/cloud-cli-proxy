FROM node:24-alpine AS frontend-builder
WORKDIR /app
COPY web/admin/package.json web/admin/pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile
COPY web/admin ./
RUN pnpm build

FROM golang:1.26.1 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY --from=frontend-builder /app/dist ./cmd/control-plane/dist

RUN CGO_ENABLED=0 go build -o /usr/local/bin/control-plane ./cmd/control-plane

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends curl ca-certificates docker.io \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/local/bin/control-plane /usr/local/bin/control-plane

WORKDIR /app
COPY internal/store/migrations internal/store/migrations
COPY deploy/docker/managed-user/image.lock deploy/docker/managed-user/image.lock

EXPOSE 8080

CMD ["control-plane"]
