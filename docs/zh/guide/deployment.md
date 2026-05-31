# 部署指南

推荐使用 Docker Compose 一键部署，无需手动安装 PostgreSQL、Go 或编译源码。

## 1. 安装 Docker

### Linux

```bash
curl -fsSL https://get.docker.com | sh
```

安装后确保当前用户在 `docker` 组：

```bash
sudo usermod -aG docker $USER
# 退出重新登录生效
```

### macOS

安装 [Docker Desktop](https://www.docker.com/products/docker-desktop/)，或通过 Homebrew：

```bash
brew install --cask docker
```

### Windows

安装 [Docker Desktop](https://www.docker.com/products/docker-desktop/)，启用 WSL 2 后端。

## 2. 启动

```bash
git clone https://github.com/ZaneL1u/cloud-cli-proxy.git
cd cloud-cli-proxy

bash deploy/scripts/setup-env.sh
docker compose pull
docker compose up -d
```

`setup-env.sh` 交互式生成所有密码和密钥。支持：

- **内置 PostgreSQL（推荐）**：自动创建，Docker Compose 统一管理，零配置
- **外部 PostgreSQL**：填入已有数据库的连接信息

启动后：

- 管理后台：`http://YOUR_HOST:3000`
- API：`http://YOUR_HOST:8080`

验证：

```bash
curl http://127.0.0.1:8080/healthz
# {"status":"ok"}
```

## 中国大陆用户

由于 `ghcr.io` 在国内访问不稳定，建议使用镜像源或开启代理后拉取。

### 方案一：走代理拉取（推荐）

确保本机已开 TUN 模式或全局代理：

```bash
docker compose pull
docker compose up -d
```

镜像走代理出站，后续无需额外配置。

### 方案二：替换为国内镜像

编辑 `docker-compose.yml`，将 `ghcr.io` 替换为以下任一可用镜像站：

```
ghcr.io/zanel1u/cloud-cli-proxy/control-plane
→ ghcr.nju.edu.cn/zanel1u/cloud-cli-proxy/control-plane
→ ghcr.nuaa.edu.cn/zanel1u/cloud-cli-proxy/control-plane
→ docker.1ms.run/zanel1u/cloud-cli-proxy/control-plane
```

或直接在拉取时指定镜像前缀：

```bash
# 替换 REGISTRY 为可用镜像站
REGISTRY=ghcr.nju.edu.cn

docker pull $REGISTRY/zanel1u/cloud-cli-proxy/control-plane:latest
docker pull $REGISTRY/zanel1u/cloud-cli-proxy/admin:latest

docker tag $REGISTRY/zanel1u/cloud-cli-proxy/control-plane:latest ghcr.io/zanel1u/cloud-cli-proxy/control-plane:latest
docker tag $REGISTRY/zanel1u/cloud-cli-proxy/admin:latest ghcr.io/zanel1u/cloud-cli-proxy/admin:latest

docker compose up -d
```

## 环境变量

使用 `setup-env.sh` 生成后通常不需要手动修改。完整列表见 [配置参考](./configuration)。

## 源码构建

预构建镜像不可用时才需要本地构建：

```bash
docker compose -f docker-compose.yml -f docker-compose.build.yaml --profile build-only build --no-cache
docker compose -f docker-compose.yml -f docker-compose.build.yaml up -d --force-recreate
```

## 宿主机直接部署

对于需要裸机 systemd 部署的场景，提供自动化脚本：

```bash
sudo bash deploy/scripts/deploy.sh
```

会自动完成：创建系统用户 → 构建二进制和镜像 → 生成配置 → 安装 systemd 服务 → 启动。详细步骤见仓库内的 `deploy/scripts/deploy.sh`。
