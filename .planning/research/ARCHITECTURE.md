# 架构研究：claude-shell 本地透明代理

**范围：** v1.3「单一 Go 二进制透明包装 Claude Code」与既有 **Go 控制面 + Unix socket host-agent + Docker + SingBoxProvider / ContainerProxyProvider + nftables** 的关系与差异。  
**调研日期：** 2026-04-09  
**总体置信度：** 中高（平台内代码与 sing-box 官方路由文档为 HIGH；Docker Desktop 行为以官方/社区共识为 MEDIUM；/proc 伪装以工程经验为主、需实机验证处标 LOW）

---

## 1. 与既有架构的对照

| 维度 | 既有云主机路径（生产） | claude-shell（本地 CLI） |
|------|------------------------|---------------------------|
| 编排者 | 控制面 → host-agent（特权在 agent） | 用户本机上的 **单一 `claude` 二进制** 直接调 `docker` |
| 容器创建 | agent 已有一套生命周期与 DB 状态 | `docker run` 由 CLI 发起，**无 PostgreSQL / 无 JWT** |
| 网络强约束 | 受管容器 **`--network=none`**，由 **SingBoxProvider** 注入 **mgmt veth**、`nsenter` 启动 sing-box，配合 **nftables**（见 `internal/network/singbox_provider_linux.go`） | 目标仍是 **tun 全量接管 + 默认拒绝**，但 **不能再假设 host-agent 能在宿主机 netns 里打 veth**（尤其 **macOS + Docker Desktop**） |
| 侧车 | **ContainerProxyProvider** 为 worker + **独立 gateway 容器**（bridge 子网 + sing-box tproxy），与 SingBoxProvider（进程进用户 netns）是 **两条不同产品线** | claude-shell 更可能 **单容器内 sing-box tun**（与 SingBoxProvider 的 **tun + mgmt0** 模式同源），一般不引入第二个 gateway 容器，除非要复用 `container_proxy_provider.go` 的 sidecar 模型 |
| 配置来源 | DB `proxy_config` JSONB + 管理后台 | **本地**：环境变量、挂载的 `config.json`、或嵌入默认模板 |

**结论（观点）：** claude-shell 应 **复用 sing-box 配置与路由思想**（`buildSingBoxConfig` 中的 tun、`bind_interface`、`route.rules`），但 **网络外围编排** 与生产 **SingBoxProvider（Linux-only netlink）** 分叉：本地工具优先 **可移植启动路径**，Linux 上可选「加强版」与现有 agent 能力对齐。

---

## 2. Go 二进制结构：`docker run` + TTY + 信号

### 2.1 推荐形态

- **主路径：** `os/exec.CommandContext` 调用 **`docker run`**（而非初期就接入 Docker Engine HTTP API），参数包含 `-i`、`-t`（当 `stdin` 是 TTY 时）、`--rm`、资源限制、环境变量、卷挂载。
- **TTY：** 使用 `github.com/moby/go-dockerclient` 或标准库时，若走 API，需 `Attach` 流式对接；**子进程方式**则与手工 `docker run -it` 一致：将 **`cmd.Stdin = os.Stdin`、`cmd.Stdout = os.Stdout`、`cmd.Stderr = os.Stderr`**，并在启动前 **`term.SetRawTerminal`**（如 `golang.org/x/term`）在 **Unix 上**恢复 Ctrl+C、窗口大小。
- **窗口尺寸：** 监听 `SIGWINCH`，向容器内 `docker exec` 发 `resize` 或使用 API `ContainerResize`；CLI 包装常用 **一个 goroutine 轮询 `syscall.SIGWINCH`**。

### 2.2 信号转发

- 订阅 **`os.Signal`：`SIGINT`、`SIGTERM`、`SIGQUIT`**（可选 `SIGHUP` 策略）。
- **转发目标：** `docker kill -s` 到容器 ID，或 `docker stop`（优雅停止超时后再强杀）。**不要**只杀本地 `docker` 客户端进程而不给守护进程留清理时间，否则 `--rm` 可能滞后。
- **父进程退出：** 使用 `context.WithCancel`，在信号处理中 cancel，等待 `cmd.Wait()`；若用户 `Ctrl+C`，通常期望 **子容器同停**。

### 2.3 与既有代码的关系

- **新代码模块**（建议）：`cmd/claude-shell` 或 `internal/claudeshell`，**不**经过 `internal/controlplane`。
- **可复用：** `internal/network` 中的 **sing-box JSON 构造**（`buildSingBoxConfig` / `buildOutbound`）宜抽成 **与 Provider 无关的纯函数包**，供 CLI 与 Linux agent **共用**，避免两份配置漂移。

**置信度：** HIGH（Go 侧为常见模式；具体 resize 细节以目标 Docker 版本测准为准）。

---

## 3. 容器网络：`--network=none` + veth 注入 vs 桥接 + `host-gateway`

### 3.1 生产平台为何用 `network=none` + veth

- 决策见 `PROJECT.md`：**彻底关闭 Docker 默认 bridge 出网**，由平台注入 **mgmt veth**，代理走 **bind_interface**，避免旁路（与 `singbox_config.go` 中 `bind_interface: mgmt0` 一致）。

### 3.2 claude-shell「本地开发工具」哪条更简单

| 方案 | 优点 | 缺点 |
|------|------|------|
| **`--network=none` + 宿主机注入 veth** | 与生产 **同构**，安全叙事一致 | **仅 Linux 且需等价于 SingBoxProvider 的 netlink 能力**；macOS 上 CLI **无法**像 agent 一样操作真实宿主机 netns（ workload 在 Docker Desktop VM 内） |
| **自定义 bridge + 容器内 sing-box tun + 路由分流** | **一条 `docker run` 跨 macOS/Linux**；实现快 | 启动瞬间若 sing-box 未就绪，存在 **极短窗口**（需 entrypoint 先拉起 sing-box + nft 再 exec 主进程，与现网「先隧道后业务」一致） |
| **bridge + `host.docker.internal:host-gateway`** | 访问宿主机上服务（LLM、本地 API）路径清晰 | **Linux** 需显式 `--add-host host.docker.internal:host-gateway`（Docker 20.10+）；Desktop 常自带解析 |

**推荐（观点）：**

- **默认 / 跨平台 MVP：** **用户态 bridge（默认 bridge 或具名 network）** + 容器内 **sing-box tun** + **sing-box 路由** 将 RFC1918 / 本机回环指向 **direct（走 eth0 出容器）**，将公网指向 **proxy outbound**；宿主机访问用 **`host.docker.internal`**（Linux 加 `host-gateway`）。
- **Linux 可选「硬隔离」模式：** 与生产对齐的 **`--network=none` + 注入 mgmt** — 可复用 `InjectManagementVeth` 等（仅 `GOOS=linux` 编译），作为 **进阶/CI 对齐** 配置，而非 macOS 默认。

**置信度：** MEDIUM（可移植性结论可靠；具体以你们镜像 entrypoint 顺序的实测为准）。

---

## 4. sing-box tun、配置与分流（私网直连、其余走代理）

### 4.1 与现网配置的关系

现网 `buildSingBoxConfig`（`internal/network/singbox_config.go`）已包含：

- **tun inbound**：`auto_route`、`strict_route`、`interface_name: tun0`
- **双 outbound**：`proxy-out` + `direct`（`bind_interface: mgmt0`）
- **DNS**：经代理 `detour: proxy-out`
- **route.rules**：`sniff`、`dns` hijack

claude-shell 需在 **route.rules** 中 **显式提前** 插入分流规则（官方 Route Rule 文档，sing-box 支持 `ip_is_private`、`ip_cidr`、`action`/`outbound` 等，见 [Route Rule](https://sing-box.sagernet.org/configuration/route/rule/)）：

1. **`ip_is_private: true` → `direct`**（或指向绑定在 **eth0** 的 direct outbound，见下条）。
2. **可选：** 为 `127.0.0.0/8`、`host.docker.internal` 解析结果再补 **ip_cidr** 规则，避免环回误走代理。
3. **默认：** 其余流量仍走 **proxy-out**（与现网一致）。

### 4.2 `direct` 的 `bind_interface` 名称

- 生产用 **`mgmt0`**（注入 veth）。
- 本地 bridge 模式用 **`eth0`**（或 `route.default_interface` 与实际一致）。需在镜像或启动脚本里 **固定接口名**，或在 sing-box **1.13+** 利用文档中的 **interface / default_interface** 类字段做对齐。

### 4.3 nftables

- 与 **PROJECT.md** 要求一致：容器内 **默认拒绝 + 仅放行经 tun 的路径**，与现网 SingBoxProvider 哲学一致；**ContainerProxyProvider** 使用 **tproxy + iptables** 是 **另一套 sidecar 模型**，claude-shell 若不跑 sidecar，则 **nftables 规则应直接贴近 SingBoxProvider 的 tun 方案**，而非照搬 gateway 镜像。

**置信度：** HIGH（分流字段来自 sing-box 官方文档；具体 JSON 需与镜像接口名联调）。

---

## 5. `/proc/cpuinfo`、`/proc/meminfo` 伪装

### 5.1 bind mount 文件

- **做法：** 在宿主机或镜像构建阶段生成 **静态文本文件**，`docker run -v /path/to/fake-cpuinfo:/proc/cpuinfo:ro`（meminfo 同理）。
- **注意：** `/proc` 下部分条目在部分内核/运行时上对 **bind mount 覆盖** 行为敏感；**常见实践是可行**，但须在 **Docker Desktop（LinuxKit VM）与原生 Linux** 各测一轮。

### 5.2 能力（capabilities）

- 普通 bind mount **通常不需要** `CAP_SYS_ADMIN`；若改用 **overlay 或自定义 tmpfs 叠 /proc**，权限与兼容性风险上升，**不推荐**作为默认路径。

### 5.3 Docker Desktop

- 容器内看见的是 **VM 内内核** 的 proc；bind mount 仍作用于容器文件系统视图，**一般可用**，但 **CPU/内存型号与宿主机 Mac 不一致** 本身也是「像云主机」的期望行为。

### 5.4 与 `tools/spoof-fingerprint.js` 的关系

- 现网脚本在 **Node 用户态** patch `os` 模块；v1.3 要求 **不依赖 spoof.js**，则 **proc 级伪装 + machine-id** 更贴近 **通用 Linux 工具链**（Go/Java 读 `/proc`）。**两者可同时存在**：JS 层 patch 管 npm 生态，proc 管读文件的生态。

**置信度：** MEDIUM–LOW（bind mount 需实机验证；无官方「保证覆盖 /proc/cpuinfo」的一刀切声明）。

---

## 6. 反容器检测：标记清单与处理策略

以下为 **常见启发式标记**（非完全可枚举；对抗检测不是密码学保证）。

| 类别 | 典型标记 | 处理思路 |
|------|----------|----------|
| 文件 | **`/.dockerenv`** | 启动时 **删除或 bind 空文件**（需可写层或 tmpfs overlay） |
| cgroup | **`/proc/1/cgroup`** 含 `docker`、`kubepods` | 使用 **伪造 cgroup 模板文件** 再通过 **bind mount** 覆盖（与 cpuinfo 同风险级别） |
| mount | **`/proc/mounts` 含 `overlay`** | 难完美隐藏；可降低特征：减少敏感 read-only 传播路径的暴露 |
| 进程树 | **pid 1 为 `docker-init` / `containerd-shim`** | 部分场景可换 **init 包装**；成本高 |
| 网络 | **网卡 MAC OUI、接口名 `eth0@if...`** | 自定义网络 + 固定 `--mac-address`（需注意冲突） |
| 环境变量 | **`container=podman`/`docker`** | 清理 env |
| 能力集 | **`/proc/self/status` CapEff** | 降 cap / 与真机对齐的镜像基线 |

**观点：** 做 **分层交付**——先做 **/.dockerenv + machine-id + cpu/mem**，再迭代 cgroup；并在 `verify` 子命令中 **可重复检测**。

**置信度：** MEDIUM（清单来自社区共识；具体工具检测逻辑持续变化）。

---

## 7. 进程生命周期

```
用户 shell
  → claude 二进制（父）
      → docker run --rm ...（docker 客户端子进程，可选长期 attach）
          → containerd → 容器内 PID1（entrypoint）
              → sing-box / init → claude-code / bun
```

- **父进程职责：** 信号转发、**保证容器停止**（`docker stop` 超时策略）、临时文件与卷清理。
- **异常：** `docker daemon` 不可用 → CLI 明确错误码；**镜像拉取失败** → 重试策略与离线提示。

---

## 8. macOS Docker Desktop vs Linux Docker Engine（网络）

| 点 | Docker Desktop（Mac） | Linux Engine |
|----|----------------------|--------------|
| 数据面位置 | 容器在 **Linux VM** 内 | 容器在 **本机内核** |
| `host.docker.internal` | 一般 **内置** | 常需 **`--add-host host.docker.internal:host-gateway`** |
| 与宿主机服务通信 | 走 **Desktop 注入的主机路由**，不是 Mac 本机 `127.0.0.1` | 走 **bridge 网关**；宿主机 `127.0.0.1` 仍不可直达，除非 **host 网络**或其它转发 |
| netns/veth 手工操作 | **CLI 在 Mac 侧无法等价于生产 agent 的 netlink 编排** | 可与 SingBoxProvider **同构** |
| nftables 在容器内 | VM 内核需支持 **nft**（Desktop 近年镜像已逐步补齐；以实测为准） | 通常可行 |

**置信度：** HIGH（Desktop 与 Engine 差异为官方文档与长期社区共识）。

---

## 9. 配置传递：环境变量 vs 挂载文件 vs label

| 方式 | 适用 | 说明 |
|------|------|------|
| **环境变量** | 代理 URL、非敏感开关、`HTTP(S)_PROXY` 类 | 易暴露于 `docker inspect`；**密钥不宜仅放 env** |
| **挂载配置文件** | **完整 sing-box JSON**、分流白名单、伪造 proc 文件 | 与现网 **写入 `/etc/sing-box/config.json`** 一致；**推荐为主配置载体** |
| **Docker labels** | 版本、镜像 digest、**非机密元数据** | 便于 `docker ps` / 运维过滤；**不应用来存密钥** |

**推荐组合（观点）：** **配置文件挂载（主） + 少量 env（覆盖路径与日志级别）**；与 `container_proxy_provider.go` 中 `-v .../config.json:ro` 一致。

---

## 10. 集成点、新增/修改与建议构建顺序

### 10.1 集成点（与仓库现状）

- **复用/抽取：** `singbox_config.go` 的配置生成 → **共享包**（供 `SingBoxProvider` 与 claude-shell）。
- **不直接耦合：** `host-agent`、**Unix socket API**、**PostgreSQL** —— claude-shell **零依赖**。
- **镜像：** 可与 **`deploy/docker/managed-user`** 同源或 **slim 变体**（无 SSH/Kasm 若不需要）；sing-box 版本与现网 **1.13.x** 对齐，避免行为漂移。

### 10.2 新增组件（建议）

1. **`claude` CLI 模块**：`docker run`、信号、TTY、配置渲染。
2. **`internal/claudeshell/config`**：合并 env + 文件，输出 sing-box JSON。
3. **（可选）`internal/claudeshell/net_linux.go`**：`network=none` 强模式仅 Linux。

### 10.3 数据流变化

- **无控制面数据流**；仅 **本地 stdin/stdout ↔ 容器 ↔ 代理出口**。
- **verify 子命令：** 可在容器内执行与 **管理后台代理测试** 类似的 **HTTP/DNS 探测**（逻辑可参考 `admin_egress_ip_probe`），但 **不经过 API**。

### 10.4 建议构建顺序（依赖优先）

1. **镜像 entrypoint**：`sing-box` 就绪 + **nft** + 启动 **Claude Code**（失败快速回滚提示）。
2. **sing-box 配置模板** + 分流规则（私网直连）。
3. **最简 `docker run` 包装**（非 TTY 也可跑通）。
4. **TTY + 信号 + SIGWINCH**。
5. **machine-id / proc 伪装 / 反检测**（逐项加）。
6. **garble 构建与发布流水线**。

---

## 11. 研究缺口与需实机验证项

- `/proc/cpuinfo` **bind mount** 在 **Docker Desktop 当前版本** 上的稳定性（**LOW→MEDIUM** 需测）。
- 容器内 **nftables** 与 **sing-box tun** 启动顺序在 **arm64/amd64** 双架构表现。
- **Apple Silicon** 上 **Bun / Claude Code** 二进制兼容与性能。

---

## 12. 来源

- 仓库：`internal/network/singbox_config.go`、`internal/network/singbox_provider_linux.go`、`internal/network/container_proxy_provider.go`
- `.planning/PROJECT.md`（v1.3 需求与关键决策）
- sing-box：[Route Rule](https://sing-box.sagernet.org/configuration/route/rule/)
- Docker：`host.docker.internal` / `host-gateway` 社区与文档共识（Engine 20.10+）
