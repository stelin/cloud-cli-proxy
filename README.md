<div align="center">

# Cloud CLI Proxy

**一条命令，一台云主机，所有流量走指定出口**

[![CI](https://github.com/ZaneL1u/cloud-cli-proxy/actions/workflows/build-images.yml/badge.svg)](https://github.com/ZaneL1u/cloud-cli-proxy/actions/workflows/build-images.yml)
[![Release](https://img.shields.io/github/v/release/ZaneL1u/cloud-cli-proxy)](https://github.com/ZaneL1u/cloud-cli-proxy/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[English](README.en.md) | [Documentation](https://zanel1u.github.io/cloud-cli-proxy/)

</div>

---

Cloud CLI Proxy 是一个面向单宿主机的容器化 SSH 云主机平台。用户通过一条 `curl` 命令即可获得专属 Docker 容器，所有出网流量通过 WireGuard 全隧道路由至指定出口 IP，杜绝 DNS / WebRTC 等任何直连泄漏。

## 核心特性

- **一条命令接入** -- `curl | bash` 启动，自动认证、创建容器、建立 SSH 会话
- **全流量强制出口** -- WireGuard + Linux netns 全隧道，配合 nftables 默认拒绝策略，零泄漏
- **每用户独立环境** -- Docker 容器隔离，预装 Claude Code、KasmVNC 桌面和 Chromium
- **灵活的出口 IP 管理** -- 多出口 IP 池，按用户绑定，支持连通性测试
- **到期自动治理** -- 用户到期后自动停机、禁止登录
- **管理后台** -- React SPA，覆盖用户、主机、出口 IP、事件日志和仪表盘
- **多架构 CI/CD** -- GitHub Actions 自动构建 `linux/amd64` + `linux/arm64` 镜像

## 快速开始

```bash
# 1. 克隆
git clone https://github.com/ZaneL1u/cloud-cli-proxy.git
cd cloud-cli-proxy

# 2. 生成环境配置（交互式，自动生成密码和密钥）
bash deploy/scripts/setup-env.sh

# 3. 启动
docker compose up -d --build

# 4. 验证
curl http://127.0.0.1:8080/healthz
# {"status":"ok"}
```

服务就绪后，管理员通过 API 创建用户和出口 IP，然后将以下命令发给用户：

```bash
curl -sSf http://YOUR_HOST:8080/v1/bootstrap/script | bash
```

## 架构概览

```
终端用户 ──curl──> Control Plane (Go API) ──Docker──> 用户容器 (SSH + VNC + Claude)
                        │                                    │
                   PostgreSQL                          WireGuard 隧道
                                                            │
                                                       指定出口 IP
```

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go, net/http, pgx v5 |
| 前端 | React 19, TypeScript, Vite, Tailwind CSS |
| 数据库 | PostgreSQL 18 |
| 容器 | Docker Engine 28, Ubuntu 24.04 |
| 网络 | WireGuard + Linux netns, nftables, sing-box |
| 桌面 | KasmVNC + Fluxbox + Chromium |

## 文档

完整文档托管在 [GitHub Pages](https://zanel1u.github.io/cloud-cli-proxy/)，包括：

- [快速开始](https://zanel1u.github.io/cloud-cli-proxy/zh/guide/quickstart) -- 部署和首次使用
- [部署指南](https://zanel1u.github.io/cloud-cli-proxy/zh/guide/deployment) -- systemd 原生部署详细步骤
- [配置参考](https://zanel1u.github.io/cloud-cli-proxy/zh/guide/configuration) -- 环境变量和 WireGuard 配置
- [架构说明](https://zanel1u.github.io/cloud-cli-proxy/zh/guide/architecture) -- 系统架构和项目结构
- [API 参考](https://zanel1u.github.io/cloud-cli-proxy/zh/reference/api) -- 完整 Admin API 文档
- [故障排查](https://zanel1u.github.io/cloud-cli-proxy/zh/reference/faq) -- 常见问题和灾难恢复

## 开发

```bash
make setup    # 安装依赖，复制 .env.example
make db       # 启动 PostgreSQL
make dev      # 后端 + 前端热重载
make test     # 全部测试
```

更多命令见 `make help`。

## 贡献

欢迎提交 Issue 和 Pull Request。请使用 [Conventional Commits](https://www.conventionalcommits.org/) 格式提交。

## 许可证

[MIT](LICENSE)
