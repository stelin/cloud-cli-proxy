# Technology Stack — claude-shell 子项目增量

**项目：** Cloud CLI Proxy — `claude-shell`（独立 Go 二进制，透明包装容器内 Claude Code + 全隧道代理 + 指纹隔离）  
**调研日期：** 2026-04-09  
**范围：** 仅 v1.3 新增能力；**不重复论证**已交付的控制面（Go 控制面、PostgreSQL、Docker 生命周期、WireGuard、sing-box、nftables、React 管理端）。

**与主线对齐：** 受管镜像已预装 **sing-box v1.13.3**；里程碑要求 **garble 混淆**、容器内 **tun + nftables**、**`/proc` 级指纹伪造**、官方 **Claude Code 安装路径**。

---

## 推荐栈（仅新增/变更）

### 核心：CLI 包装二进制

| 组件 | 建议版本 / 选型 | 用途 | 选型理由 |
|------|-----------------|------|----------|
| Go | **1.26.1**（与仓库/里程碑一致） | `claude-shell` 单二进制 | 与现有模块、garble 主线 README 要求一致（见下文 garble）。若 `go.mod` 仍为 1.25.x，应在合入 claude-shell 前统一到 1.26.x。 |
| CLI 框架 | **`github.com/spf13/cobra` ≥ v1.10.0**（例如 **v1.10.2**，2025-12 前后） | 根命令、`verify` 子命令、持久化 flag、bash/zsh 补全 | **生态默认**：kubectl、Helm、Docker CLI、GitHub CLI 等均基于 Cobra；子命令树、`RunE` 错误传播、与 `pflag` 集成成熟。适合「替换本机 `claude` + 多子命令」的长期演进。 |
| CLI（备选） | **`github.com/urfave/cli/v3`** 或 **stdlib `flag`/`os.Args`** | 极简包装或内部工具 | **urfave**：声明式、依赖面小于 Cobra，适合命令面极少且不需要深度子命令树时。**stdlib**：零依赖、二进制最小；但子命令与帮助文案需自管，里程碑若含 `verify`、多 flag，维护成本高。 |
| Docker 引擎 API | **`github.com/docker/docker/client`**（随上游 **Docker Engine API** 协商版本；`client.FromEnv` + `WithAPIVersionNegotiation()`） | 创建/启动/执行/日志、镜像拉取、bind mount 与 `extra_hosts` | **与 `docker` CLI 同源客户端**：类型化、无每调用 fork `docker` 的开销；便于流式 attach/exec。官方文档明确 CLI 亦基于此包。 |
| YAML | **`go.yaml.in/yaml/v3` v3.0.4+**（或新项目直接 **v4** 稳定版发布后采用） | 若需用户侧 sing-box/代理片段 YAML；或本地配置 | **gopkg.in/yaml.v3 已标记无人维护**（2025-04 起）；YAML 组织接管后的 **`go.yaml.in`** 为 Cobra 等上游迁移目标，安全修复持续。v4 为活跃开发线（若接受 RC/新 API，再评估）。 |
| 加密与资源嵌入 | **`embed` + `crypto/aes`（推荐 AES-GCM，`crypto/cipher`）** | 嵌入默认配置模板、证书或脚本；运行时解密 | 标准库即可满足「嵌入密文 + 运行时解密」；密钥应来自环境变量/外部文件，**避免**把长期密钥硬编码进仓库。 |
| 可选：第三方嵌入加密 | **`github.com/abakum/embed-encrypt`** 等 | 构建期生成 `//encrypted:embed` | 减少明文静态段；需评估与 **garble**、CI 生成步骤的交互；维护面大于手写解密。 |

### 构建与交付

| 组件 | 建议版本 / 选型 | 用途 | 选型理由 |
|------|-----------------|------|----------|
| 二进制混淆 | **`mvdan.cc/garble`**（安装：`go install mvdan.cc/garble@latest`；发布标签见 [GitHub Releases](https://github.com/burrowers/garble/releases)） | 发布构建：`garble build -literals …` | **官方 README（master）写明：Requires Go 1.26 or later。** 与项目 Go 版本对齐后可用。v0.15.0 发布说明曾写「支持 Go 1.25」等与当前 README 不完全一致，**以构建时 `garble -h` + README 为准**。 |
| 交叉编译 | `GOOS`/`GOARCH` + 与主线一致静态链接策略 | 分发单文件给用户机 | 与现有发布流程一致即可。 |

### 容器内网络（与已有 sing-box 能力对齐）

| 组件 | 版本 | 用途 | 备注 |
|------|------|------|------|
| sing-box | **1.13.3**（与受管镜像一致；升级需回归 tun） | 容器内 **tun inbound** + outbound 代理链 | 官方 [Tun 配置](https://sing-box.sagernet.org/configuration/inbound/tun/)：Linux 上 **`auto_route` + `auto_redirect`（nftables）** 为推荐组合，且文档说明可缓解 **TUN 与 Docker bridge 冲突**。容器内需 **`CAP_NET_ADMIN`** 及 tun 设备权限。 |
| 路由/例外 | 配置字段 `route_exclude_address` / `route_address` 等 | 仅外网走代理；RFC1918、`host-gateway` 回宿主机 | 与里程碑「本地流量回连宿主机」一致；具体 CIDR 与 `extra_hosts` 由 claude-shell 生成配置时注入。 |

### 指纹与 `/proc`（Linux）

| 技术 | 用途 | 说明 |
|------|------|------|
| **bind mount 静态文件** | 覆盖 `/proc/cpuinfo`、`/proc/meminfo` 等 | 里程碑已列「bind mount」路径；实现简单、无额外守护进程；内容需与预设 CPU/RAM 故事一致。 |
| **lxcfs**（宿主机运行） | 提供 cgroup 一致的 `/proc/cpuinfo`、`meminfo`、`stat` 等 | 典型做法：宿主机运行 `lxcfs`，将 `/var/lib/lxcfs/proc/...` **逐个 bind** 进容器；**不要**盲目把整个 `/var/lib/lxcfs/proc` 盖在容器 `/proc` 上（易破坏 `/proc/self` 等）。见社区文章与实践（如 Podman + lxcfs 绑定单文件）。 |
| **自研 FUSE** | 完全自定义 `/proc` 视图 | 成本高，仅在有强定制且无法接受静态文件时使用。 |

### Claude Code 安装与运行（官方行为摘要）

| 项目 | 内容 |
|------|------|
| 推荐安装 | 官方文档：**Native Install** — `curl -fsSL https://claude.ai/install.sh \| bash`（macOS/Linux/WSL）；PowerShell 为 `irm https://claude.ai/install.ps1 \| iex`。见 [Anthropic 安装文档](https://docs.anthropic.com/en/docs/claude-code/setup)。 |
| 已弃用 | **`npm install -g @anthropic-ai/claude-code`** 标记为 deprecated；里程碑要求不依赖 npm，与之一致。 |
| 容器注意 | 官方文档写明：**Alpine / musl** 等环境需 **`libgcc`、`libstdc++`、`ripgrep`** 等；无头环境可用 **`ANTHROPIC_API_KEY`**。安装后验证：`claude --version`、`claude doctor`（名称以文档为准）。 |
| 与「透明包装」关系 | `claude-shell` 宿主机二进制应 **exec 进容器内** 已安装的 `claude`，或通过 `docker exec` 转发 stdin/stdout；需统一 **工作目录与 TTY** 行为。 |

---

## 分项结论（对应调研问题）

### 1. Go CLI 框架：Cobra vs urfave/cli vs stdlib

- **默认推荐：`spf13/cobra`（v1.10.x）** — 子命令（如 `verify`）、flag 继承、帮助与补全生态与现有行业工具一致；维护活跃（例如依赖迁移至 `go.yaml.in/yaml/v3`）。
- **`urfave/cli`** — 更轻、API 不同；适合命令面极薄时。
- **`stdlib`** — 最小依赖；适合原型，若产品化多子命令则易膨胀。

**集成：** 与 Docker client、配置模块同进程，无冲突；避免在同一二进制再引入 **Viper** 除非确有大量配置源（里程碑以 YAML/环境变量为主时可不用）。

### 2. garble：版本、兼容性、限制

- **安装路径：** `mvdan.cc/garble`（模块路径），`go install mvdan.cc/garble@latest`。
- **Go 版本：** 上游 **README 要求 Go 1.26+**；项目锁定 **Go 1.26.1** 时与之对齐。**每次升级 Go 小版本需重跑 garble 回归**（garble 依赖 linker 补丁，新版本 Go 可能短暂滞后，见上游 issue/PR 历史）。
- **已知限制（摘自上游 README，HIGH 置信度）：**
  - 导出方法当前不混淆（接口约束）；`GOGARBLE` 仅按包模式。
  - **`runtime/debug.ReadBuildInfo`、`runtime.GOROOT` 等**在混淆二进制中不可用或受限。
  - **`//go:linkname` 注入 runtime 的包**可能直接构建失败。
  - **Go plugin 不支持**。
  - 需要 **git** 以给 linker 打补丁。
  - `-tiny` 会去掉 panic 栈等，排障更难。
- **构建成本：** 约 **2× `go build`** 时间；独立 `GARBLE_CACHE`。

### 3. Docker SDK（Go）vs 调用 `docker` CLI

- **推荐：Moby/Docker 官方 Go API**（`github.com/docker/docker/client`）：与引擎对话、流式 attach/exec、错误类型化；**无**每次 `exec.Command("docker", …)` 的进程与解析成本。
- **shell out：** 仅适合快速脚本或运维 one-off；产品路径应使用 SDK，便于测试 mock与错误处理。
- **注意：** 避免无条件 `ImagePull`（本地已有镜像时仍会拉取元数据，历史上曾导致「SDK 慢」误判）；按镜像存在性分支。

### 4. YAML 配置库

- **新代码优先：`go.yaml.in/yaml/v3`（≥ v3.0.4）** 或跟进 **v4** 正式版。
- **避免**继续引入已无人维护的 **`gopkg.in/yaml.v3`** 作为直接依赖。

### 5. 嵌入并加密资源（`go:embed` + AES）

- **模式：** `//go:embed` 密文 blob → `init` 或首次使用时 **`AES-GCM`（`crypto/cipher`）** 解密到 `[]byte` 或 `memfile`。
- **密钥：** 运行时从环境变量/用户配置读取；构建脚本仅注入占位或 CI 秘密。
- **与 garble：** `-literals` 会处理字符串字面量；敏感常量仍应避免以明文字面量出现在源码中。

### 6. sing-box：容器内 tun 模式配置要点

- **inbound：** `type: tun`，配置 `interface_name`、`address`（1.10+ 合并 IPv4/IPv6）、`mtu`、`stack`（`system` / `gvisor` / `mixed` 按镜像与内核选择）。
- **Linux：** `auto_route: true`，并启用 **`auto_redirect: true`**（文档称改善路由与性能，并减少与 Docker 网络冲突）。
- **分流：** `route_exclude_address`（或 deprecated 的 inet4/inet6 等价字段）列出私网段；与 **nftables 默认拒绝**、**host-gateway** 回宿主机策略一致。
- **权限：** 容器需 tun、nft 所需 capability；与现有 ** `--network=none` + 自建网络栈** 的架构需在设计文档中明确是否复用或分支（避免双栈冲突）。

### 7. `/proc` 伪造技术

| 方法 | 优点 | 缺点 |
|------|------|------|
| 静态 bind mount | 实现快、无守护进程 | 与真实 cgroup 不一致时可能被交叉验证 |
| lxcfs 单文件 bind | 与 limits 更一致 | 需宿主机安装运行 lxcfs |
| 全盘 FUSE 替换 `/proc` | 灵活 | 复杂度高、易踩内核/容器边界 |

**建议：** 里程碑「系统级指纹」用 **静态文件 + 选择性 bind**；若要与内存限额展示一致，再叠加 **lxcfs**。

### 8. Claude Code 官方安装与容器兼容性

- **推荐路径：** `install.sh` / `install.ps1` 的 **Native** 流程；**npm 全局安装为 deprecated**。
- **容器内：** 使用 **glibc** 系基础镜像或按文档补齐 **musl 依赖**；CI 可缓存安装目录以减少重复下载。
- **无 UI 服务器：** 设 **`ANTHROPIC_API_KEY`** 或按文档使用 OAuth/登录流程（以官方最新说明为准）。

---

## 不建议引入（本里程碑）

| 项 | 原因 |
|----|------|
| **spf13/viper**（除非配置源爆炸） | 增加间接依赖与初始化顺序；仅用 YAML + env 时过重。 |
| **子进程 `docker` CLI 作为主路径** | 可维护性与性能均弱于官方 Go client。 |
| **继续依赖 `gopkg.in/yaml.v3`** | 上游已停止维护；应迁移至 `go.yaml.in`。 |
| **在二进制内硬编码长期密钥** | 违背基本密钥管理；审计不通过。 |

---

## 集成点（与现有仓库）

- **不替代**控制面 HTTP API；`claude-shell` 为**用户侧**独立交付物，通过 Docker API 与本地 Engine 交互。
- **复用概念**：sing-box JSON 模型、nftables「默认拒绝」哲学与 v1.1 文档一致；实现上可能与 `host-agent` 分流，避免在未抽象库的情况下复制大段代码（具体以阶段规划为准）。
- **版本锁**：sing-box **1.13.3** 与镜像一致；升级需同时更新镜像与 claude-shell 模板测试。

---

## 置信度

| 主题 | 置信度 | 依据 |
|------|--------|------|
| Cobra / Docker Go client / sing-box tun 字段 | **高** | 官方文档与 pkg.go.dev |
| garble 限制与 Go 版本要求 | **高** | burrowers/garble README + releases |
| `go.yaml.in` 替代 `gopkg.in` | **高** | pkg.go.dev 与 Cobra 迁移提交 |
| Claude Code 安装方式 | **高** | Anthropic 官方安装文档 |
| lxcfs + Docker 单文件 bind 实践 | **中** | 社区文章与 lxcfs issue；生产需在本项目目标发行版上实测 |
| garble 与 **Docker client 反射/接口** 组合 | **中** | 一般可构建；若遇问题可用 `GOGARBLE` 缩小范围或升级 garble |

---

## 来源（权威优先）

- Cobra：https://github.com/spf13/cobra  
- garble：https://github.com/burrowers/garble  
- Docker Engine Go SDK：https://pkg.go.dev/github.com/docker/docker/client  
- go-yaml（维护版）：https://pkg.go.dev/go.yaml.in/yaml/v3  
- sing-box Tun：https://sing-box.sagernet.org/configuration/inbound/tun/  
- Claude Code 安装：https://docs.anthropic.com/en/docs/claude-code/setup  
- lxcfs：https://github.com/lxc/lxcfs  

---

*本文件仅供路线图/阶段规划消费；具体模块边界与代码位置以实现阶段 PLAN 为准。*
