# 架构说明

## 系统架构

```
┌─────────────┐   curl    ┌────────────────┐    Docker     ┌──────────────────┐
│   终端用户   │ ────────> │  Control Plane │ ───────────>  │  用户容器         │
│  (SSH 客户端)│ <──────── │  (Go API)      │              │  SSH + VNC + Claude│
└─────────────┘   SSH     └───────┬────────┘              └────────┬─────────┘
                                  │                                │
                           ┌──────┴───────┐               ┌───────┴────────┐
                           │  PostgreSQL   │               │  WireGuard /   │
                           │  (状态持久化) │               │  sing-box 隧道 │
                           └──────────────┘               └───────┬────────┘
                                                                  │
                                                          ┌───────┴────────┐
                                                          │  指定出口 IP    │
                                                          └────────────────┘
```

## 核心组件

| 组件 | 职责 |
|------|------|
| **Control Plane** | HTTP API、用户认证、会话管理、到期扫描、状态协调 |
| **Host Agent** | Docker 容器生命周期、WireGuard 隧道、nftables 防火墙、网络命名空间 |
| **用户容器** | OpenSSH Server、Shell 工具、Claude Code、KasmVNC 桌面 |
| **PostgreSQL** | 用户、主机、出口 IP 绑定、会话、到期时间和审计事件 |

## 架构原则

- **单宿主机优先** -- 不为 v1 提前引入多节点调度复杂度
- **网络强约束优先** -- 先保证"所有流量都必须走指定出口"
- **启动体验优先** -- 建立在真实可验证的运行时正确性之上

## 关键边界

- Web / API 层不直接持有过宽的宿主机特权
- Docker 和网络 namespace 的操作集中在 host-agent 中执行
- 用户容器的默认出网必须被隧道网络接管，不能保留旁路

## 项目结构

```
cloud-cli-proxy/
├── cmd/
│   ├── control-plane/          # 控制面 API 入口
│   └── host-agent/             # 宿主机代理入口
├── internal/
│   ├── controlplane/           # HTTP 路由、业务逻辑、到期扫描、状态协调
│   ├── agent/                  # host-agent 服务端
│   ├── network/                # WireGuard / nftables / sing-box 配置
│   ├── runtime/                # Docker 容器生命周期管理
│   ├── sshproxy/               # SSH 代理（转发到容器 22 端口）
│   └── store/                  # 数据库迁移和查询（pgx）
├── web/admin/                  # React 管理后台（TanStack Router）
├── deploy/
│   ├── docker/                 # 4 个 Dockerfile
│   ├── compose/                # 开发用 Compose
│   ├── bootstrap/              # 用户 curl 引导脚本
│   ├── scripts/                # setup-env.sh、deploy.sh、backup.sh
│   └── systemd/                # systemd 服务单元
├── docs/                       # VitePress 文档站
├── docker-compose.yml          # 生产 Compose
├── Makefile                    # 开发命令入口
└── .github/workflows/          # CI/CD
```

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go, net/http 标准库, pgx v5 |
| 前端 | React 19, TypeScript, Vite, Tailwind CSS, TanStack Router/Query |
| 数据库 | PostgreSQL 18 |
| 容器 | Docker Engine 28, Ubuntu 24.04 用户镜像 |
| 网络 | WireGuard + Linux netns, nftables, sing-box |
| 桌面 | KasmVNC 1.4.0 + Fluxbox + Chromium |
