# Deployment Guide

Docker Compose is the recommended way to deploy. No need to install PostgreSQL, Go, or compile from source.

## 1. Install Docker

### Linux

```bash
curl -fsSL https://get.docker.com | sh
```

Add your user to the `docker` group:

```bash
sudo usermod -aG docker $USER
# Log out and back in for it to take effect
```

### macOS

Install [Docker Desktop](https://www.docker.com/products/docker-desktop/), or via Homebrew:

```bash
brew install --cask docker
```

### Windows

Install [Docker Desktop](https://www.docker.com/products/docker-desktop/) with WSL 2 backend enabled.

## 2. Start

```bash
git clone https://github.com/ZaneL1u/cloud-cli-proxy.git
cd cloud-cli-proxy

bash deploy/scripts/setup-env.sh
docker compose pull
docker compose up -d
```

`setup-env.sh` interactively generates all passwords and secrets. It supports:

- **Built-in PostgreSQL (recommended)**: auto-created, managed by Docker Compose, zero config
- **External PostgreSQL**: provide your existing database connection details

After startup:

- Admin dashboard: `http://YOUR_HOST:3000`
- API: `http://YOUR_HOST:8080`

Verify:

```bash
curl http://127.0.0.1:8080/healthz
# {"status":"ok"}
```

## Users in Mainland China

`ghcr.io` may be slow or unreachable from within mainland China. Use one of the following workarounds.

### Option 1: Pull through a proxy (recommended)

If your machine already has a TUN-mode proxy or global proxy set up:

```bash
docker compose pull
docker compose up -d
```

Images will be pulled through the proxy. No further configuration needed.

### Option 2: Use a mirror registry

Replace `ghcr.io` with one of the available mirrors:

```
ghcr.io/zanel1u/cloud-cli-proxy/control-plane
→ ghcr.nju.edu.cn/zanel1u/cloud-cli-proxy/control-plane
→ ghcr.nuaa.edu.cn/zanel1u/cloud-cli-proxy/control-plane
→ docker.1ms.run/zanel1u/cloud-cli-proxy/control-plane
```

Or pull and re-tag with a mirror prefix:

```bash
REGISTRY=ghcr.nju.edu.cn

docker pull $REGISTRY/zanel1u/cloud-cli-proxy/control-plane:latest
docker pull $REGISTRY/zanel1u/cloud-cli-proxy/admin:latest

docker tag $REGISTRY/zanel1u/cloud-cli-proxy/control-plane:latest ghcr.io/zanel1u/cloud-cli-proxy/control-plane:latest
docker tag $REGISTRY/zanel1u/cloud-cli-proxy/admin:latest ghcr.io/zanel1u/cloud-cli-proxy/admin:latest

docker compose up -d
```

## Environment Variables

After running `setup-env.sh`, manual changes are usually unnecessary. See [Configuration](./configuration) for the full reference.

## Building from Source

Only needed when prebuilt images are unavailable:

```bash
docker compose -f docker-compose.yml -f docker-compose.build.yaml --profile build-only build --no-cache
docker compose -f docker-compose.yml -f docker-compose.build.yaml up -d --force-recreate
```

## Bare-metal Deployment

For scenarios that require native systemd deployment:

```bash
sudo bash deploy/scripts/deploy.sh
```

This automates: creating the system user → building binaries and images → generating config → installing systemd units → starting services. See `deploy/scripts/deploy.sh` in the repo for details.
