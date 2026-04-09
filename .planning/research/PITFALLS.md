# Domain Pitfalls：claude-shell 透明 Docker CLI 包装器（追加到既有 Docker 体系）

**Domain:** 在已有「控制面 + host-agent + `--network=none` + WireGuard/sing-box」模型上，增加独立 Go 二进制：透明拉起容器、TTY 透传、指纹与 `/proc` 伪装、sing-box tun、garble 交付与跨平台镜像。
**Researched:** 2026-04-09
**Overall confidence:** MEDIUM–HIGH（Docker/sing-box/官方文档为主；Anthropic 行为以公开文档与 issue 为准，属 MEDIUM）

---

## 总览：与「从零造容器」不同的集成风险

在**已有**系统上追加本地透明包装器时，典型错误不是「会不会用 Docker」，而是：**信号与 TTY 语义、宿主机与容器身份映射、Docker Desktop 与 Linux 生产路径差异、为 tun/nft 追加的 capability 与最小权限的平衡、以及把「反检测」当成可承诺的产品特性**。下列条目按主题编号，每条包含：会出什么问题、征兆、预防、建议由哪类工作项承接（对应 v1.3 里程碑内的子域）。

---

## 1. TTY：raw 模式、信号转发、终端缩放

### 会出什么问题

- **PID 1 与信号：** Linux 下容器内 PID 1 对默认动作为「忽略」的信号（如 `SIGINT`）行为特殊；若包装命令未正确处理 `SIGTERM`/`SIGINT`，表现为 Ctrl+C 无效或容器僵死（Docker 文档明确提示 PID 1 特性）。可用 `--init` 或确保主进程非 PID 1 且正确转发。
- **SIGWINCH：** 分配伪 TTY（`-t`）时，终端窗口变化会向容器内进程转发 `SIGWINCH`。部分守护进程（历史上如 Apache）将 `SIGWINCH` 用作**优雅退出**而非「窗口尺寸变化」，导致一resize 就退出——这是应用语义问题，不是 Docker bug。
- **信号代理：** 默认 `--sig-proxy=true`（foreground attach 场景）会把宿主收到的一些信号传给容器进程；若需要 resize 但不希望某些信号到达子进程，需理解 `--sig-proxy` 与「是否 `-t`」的组合效果。
- **Raw 模式：** 全屏 TUI（含部分 CLI 编辑器）依赖宿主 stty raw；若包装层额外包了一层 shell 或未使用 `-it`，可能出现按键错乱、Ctrl+C 传到错误进程。

### 征兆

- 拉终端窗口后服务无故退出、日志出现 `caught SIGWINCH, shutting down`。
- Ctrl+C 无法结束、需 `docker kill`。
- 交互式 CLI 方向键/补全异常。

### 预防策略

- 对「类 REPL」的 Claude Code：**明确使用 `-i` + `-t`**，并在 Go 侧若用 `exec`/`attach`，保证与 `docker run` 行为一致；评估 **`docker run --init`** 或 entrypoint 用 **`tini`/`dumb-init`** 改善信号与僵尸进程。
- 文档与集成测试覆盖：**resize、SIGINT、EOF**。
- 若子进程对 SIGWINCH 过敏：查阅该进程文档，或避免以 PID 1 直接跑该进程、或调整信号处理（产品层决策，慎用 `--sig-proxy=false` 以免误伤 Ctrl+C）。

### 建议负责范围

**Go 包装器与容器入口（v1.3：本地 `claude` 二进制 + `docker run` 参数契约）**；与既有 SSH 路径解耦，需单独 UAT。

**Sources (HIGH/MEDIUM):** [Docker run — interactive / tty / stop-signal / init](https://docs.docker.com/engine/reference/commandline/run/)，[Docker attach — sig-proxy](https://docs.docker.com/engine/reference/commandline/attach/)；SIGWINCH 与 Apache 等行为见 [docker-library 讨论](https://github.com/docker-library/php/issues/64)（MEDIUM：应用特定）。

---

## 2. Bind mount 与 UID/GID 错位

### 会出什么问题

- 容器内用户 UID 与宿主机目录所有者不一致时：**写权限拒绝、创建文件归属为 root/意外 UID**，导致 Claude Code 写 `~/.claude`、项目目录或 socket 失败。
- 使用 user namespace / `--user` 时，映射规则与 **volume 上已有文件** ACL 交互复杂，易出现「宿主可读、容器不可写」。

### 征兆

- `Permission denied` 仅在挂载卷上出现；容器内 `id` 与宿主 `ls -n` 不一致。
- 日志或 IDE 报无法写配置目录。

### 预防策略

- 在镜像或 entrypoint 中 **固定非 root 运行**并文档化 UID；或在启动前用 **`--user "$(id -u):$(id -g)"`** 与宿主对齐（需镜像支持该 UID 家目录）。
- 对配置与缓存目录：**命名卷**或启动时 `chown`（谨慎：扩大入口脚本复杂度）。
- 将 **「可写路径契约」** 写入 `verify` 子命令与集成测试。

### 建议负责范围

**镜像用户模型 + 包装器默认挂载表（v1.3）**；与「宿主机项目目录 bind mount」强相关。

**Sources:** 通用 Linux 文件权限行为；**LOW** 若无具体镜像设计需在产品阶段实测。

---

## 3. Docker Desktop：与 Linux 生产环境的差异

### 会出什么问题

- **`--network host`：** 在 macOS/Windows 上 historically **不等同于 Linux host 网卡**（守护进程在 VM 内）；文档与社区长期建议用 **端口映射** 或 **host-gateway**。产品若依赖「host 网络 = 宿主 Linux」，在 Desktop 上会失真。
- **Host 网络模式可用性：** Docker Desktop 设置里 **「Enable host networking」** 等选项影响 `--net=host` 与 localhost 互通行为；团队内若未统一版本/开关，会出现「我机器可复现你机器不行」。
- **`--network none`：** 一般仍可用来去掉默认 bridge，但 **后续再注入 veth/tun** 的路径依赖 **Linux 内核能力**；Desktop 的 VM 内核版本、模块与 nft 行为与生产裸金属可能不同。
- **性能与启动：** 文件系统走 osxfs/virtiofs，**大量小文件或 node_modules bind** 会放大与 Linux 的差异。

### 征兆

- 仅 Mac 上 `host-gateway`、localhost 回连、或 sing-box 路由异常。
- CI（Linux）通过，开发者 Desktop 失败。

### 预防策略

- 在 PROJECT 约束中明确：**生产仍为单台 Linux**；Desktop 仅作开发级 **Tier-2** 支持，文档写明差异与推荐（Linux VM 或远端 dev）。
- 对 `host.docker.internal` / **`--add-host=host.docker.internal:host-gateway`** 建立单一事实来源（compose/run 参数表）。
- 网络正确性门禁以 **Linux 裸机/等同 VM** 为准绳，Desktop 上不重复承诺「三重校验」同级保证。

### 建议负责范围

**文档与验证矩阵 + 可选的 dev override（v1.3）**；核心仍落在 **Linux host-agent 路径**。

**Sources (MEDIUM):** [Docker Desktop Settings — Network / host networking](https://docs.docker.com/desktop/settings-and-maintenance/settings/)；[Docker 网络模式概述（社区/博客类作辅证）](https://oneuptime.com/blog/post/2026-01-25-docker-container-networking-modes/view)。

---

## 4. `/proc` bind mount、`--privileged` 与 `CAP_SYS_ADMIN`

### 会出什么问题

- **在容器内执行 `mount --bind`（含绑定 `/proc` 下某些路径）需要 `CAP_SYS_ADMIN`**；默认 Docker 丢弃 capability，**即使 UID 0 也会 `permission denied`**（mount(2) 语义）。
- **`--privileged`** 会授予**全部 capabilities** 并放宽 device 访问，攻击面远大于「只为 mount」；合规与客户环境可能直接否决。
- **runc 对 /proc 挂载有安全检查**（`checkProcMount`）：部分 `/proc/...` 绑定会被拒绝，错误信息为「cannot be mounted because it is inside /proc」——与「只加 SYS_ADMIN」仍不够的情况并存。
- **绑定伪造的 `cpuinfo`/`meminfo`：** 若用宿主文件覆盖容器内 `/proc/cpuinfo`，需通过 **Docker `-v` 在创建时挂载**，而非依赖容器内再 mount；容器内二次 mount 受 capability 与安全配置约束更大。

### 征兆

- entrypoint 脚本中 `mount --bind` 失败；或仅在加了 `--privileged` 才成功。
- 安全扫描报告 capability 过多。

### 预防策略

- **优先在 `docker run` 层用 `-v` 注入伪造文件**，避免在容器内执行 mount。
- 若必须在容器内 mount：**最小化**为 `--cap-add=SYS_ADMIN` + 必要 `--security-opt`、并审计是否还需 `/dev` 访问；**避免习惯性 `--privileged`**。
- 对 **gVisor、rootless、Podman** 等替代运行时提前声明不支持或需单独测试（嵌套与 mount 行为差异大）。

### 建议负责范围

**容器安全基线评审（v1.3 网络/指纹阶段）**；与「反容器检测」条目联动——伪造 proc 的**工程实现**应偏向 **OCI 挂载** 而非容器内特权 mount。

**Sources (HIGH):** [Docker — runtime privilege and Linux capabilities](https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities)；[mount(2) 需要 CAP_SYS_ADMIN 的通用说明](https://stackoverflow.com/questions/36553617/how-do-i-mount-bind-inside-a-docker-container)（MEDIUM）；[runc checkProcMount 讨论](https://github.com/opencontainers/runc/issues/2826)（MEDIUM）。

---

## 5. sing-box TUN 在容器内：能力与设备、常见误配

### 会出什么问题

- TUN 创建与路由改写通常需要：**`NET_ADMIN`**；访问 `/dev/net/tun` 需将设备暴露进容器（`--device /dev/net/tun` 或等价）。
- sing-box 文档说明：**非特权模式**下 TUN 的地址/MTU **不会自动配置**，需自行保证配置正确（易配错导致「进程起来但无流量」）。
- **`auto_route` + `auto_redirect`（Linux）：** 文档推荐用于与 **Docker bridge** 共存及性能；若与 **nftables 默认拒绝**、mark、策略路由叠加，可能出现 **规则顺序冲突、环路或本机流量未进 tun**。
- **MPTCP：** 1.13+ 对 MPTCP 有特殊处理；未正确 `exclude_mptcp` 时可能出现 **直连泄漏或连接异常**（与 Apple 客户端相关流量文档一致）。
- **UID 分流：** `include_uid` / `exclude_uid` 仅在 Linux 且 `auto_route` 下生效——与容器内跑 claude 的用户 UID 设定强相关，配错则 **部分进程绕开 tun**。

### 征兆

- sing-box 日志显示 tun 已 up，但 `curl` 出口 IP 仍非预期；或仅部分 TCP 走代理。
- DNS 仍走宿主 resolver（泄漏）。

### 预防策略

- 能力/device 清单写死：**`NET_ADMIN` + `/dev/net/tun` + nft 权限模型**（是否与现有 host-agent 统一由实现决定）。
- 使用与生产 **相同版本 sing-box**（PROJECT 已锁 v1.13.3），升级前对照 [Tun 文档](https://sing-box.sagernet.org/configuration/inbound/tun/) 变更日志。
- 将 **strict_route、route_exclude、loopback_address、exclude_mptcp** 纳入 `verify` 与自动化网络测试。

### 建议负责范围

**sing-box + nftables 集成（v1.3 核心）**；与 v1.1 已有代理路径复用时要防 **两套防火墙规则互相踩**。

**Sources (HIGH):** [sing-box Tun inbound 文档](https://sing-box.sagernet.org/configuration/inbound/tun/)。

---

## 6. 容器启动延迟：如何压到可接受

### 会出什么问题

- **冷启动 pull**：首次拉镜像耗时可掩盖「二进制慢」 perception，引发支持成本。
- **层过多 / 未合并 RUN**：每次构建变更导致缓存失效。
- **入口脚本串行**：等 tun 就绪再等 claude，未做并行或 readiness 拆分。
- **Desktop 上文件系统**：bind mount 巨大目录会拖慢启动。

### 征兆

- 用户抱怨「第一次极慢」；后续仍慢则查镜像体积与 entrypoint。

### 预防策略

- **多阶段构建**、删除包管理器缓存、**稳定 base digest**；文档提供 **预拉取** `docker pull`。
- 将 **健康检查** 拆为：网络栈就绪 / Claude Code 进程就绪 / 认证可用。
- 对包装器：**复用已存在容器**（若产品设计允许）比每次 `run --rm` 更快——需权衡隔离与状态。

### 建议负责范围

**镜像流水线 + 包装器默认策略（v1.3）**。

**Sources:** Docker 最佳实践（镜像层）为通用知识；**MEDIUM**。

---

## 7. Claude Code 在容器内：自动更新、登录状态、是否会「弄坏」容器

### 会出什么问题

- **原生安装通道**文档写明：会**后台检查更新**并在下次启动生效；需 **显式关闭** 时应用官方推荐方式。
- 官方推荐在 `settings.json` 的 `env` 中设置 **`DISABLE_AUTOUPDATER=1`**（`autoUpdates` 等旧键曾被弃用或引发混淆，GitHub 有多起 issue）。
- 若仍尝试自更新：可能因 **权限、只读层、网络策略** 导致 **半升级**、二进制消失或路径错乱（issue 社区有「self-update 卸载/找不到命令」类报告）。
- **认证状态**：通常落在用户目录下配置与密钥文件；容器无持久卷时 **每次重建需重新登录**；有卷时需明确 **与宿主隔离** 以防串账号。

### 征兆

- 容器内 `claude` 版本漂移与镜像 tag 不一致；更新失败横幅。
- 无卷时每次启动要求登录。

### 预防策略

- 镜像内交付路径下 **默认设置 `DISABLE_AUTOUPDATER=1`**，由产品节奏统一升级镜像；或在文档中允许更新但 **声明不支持矩阵**。
- **命名卷**挂载 `~/.claude`（或等价路径）并文档化；CI 测「重启容器仍登录」可选。
- 将 **`claude doctor`** 纳入 `verify` 或故障排查手册。

### 建议负责范围

**Claude Code 安装与配置阶段（v1.3）**；与 **镜像版本发布流程** 绑定。

**Sources (MEDIUM):** [Claude Code Advanced setup — Auto-updates](https://code.claude.com/docs/en/setup)；[anthropics/claude-code issues — autoupdate / DISABLE_AUTOUPDATER](https://github.com/anthropics/claude-code/issues/3479)（行为以官方最新文档为准）。

---

## 8. 反检测：Anthropic 侧容器检测与「从内检测难度」

### 会出什么问题

- **删除 `/.dockerenv`、改 cgroup** 等可被用户态篡改的路径，**无法对抗**具备宿主机协作或 hypervisor 视图的检测；对方若在 **客户端二进制或服务器侧启发式** 增加规则，属于持续 arms race。
- **从内检测容器：** 低权限下仍可观察 **cgroup、mountinfo、init 进程、DMI、时钟与 CPU 特征、驱动缺失** 等；难度随检测深度变化，不应在对外承诺中写死「不可检测」。
- **合规风险：** 「规避平台技术措施」可能触及服务条款与法律风险——产品应聚焦 **透明代理与安全隔离的工程诚实表述**，而非对抗性营销。

### 征兆

- 服务端行为突变（额外验证、风控）；内部 `verify` 全绿仍被策略拒绝。

### 预防策略

- 将「反检测」降级为 **最佳努力指纹统一**（与 `/etc/machine-id`、proc 注入目标一致），并在路线图标注 **依赖外部政策**。
- **服务端契约**以官方 API 文档与 ToS 为准；预留 **非欺骗性** 的企业部署叙事（受控出口、审计）。

### 建议负责范围

**产品/合规评审 + 技术实现分离**；工程上归 **v1.3 指纹与 verify**，商业与法律归 **项目决策**。

**Sources:** 无权威第三方「检测难度」指标；本段 **LOW–MEDIUM**（推理与行业常识）。

---

## 9. garble 混淆：已知问题、不能保护的面、运行时开销

### 会出什么问题

- **反射与跨包类型：** historically 为 garble 主要破坏源；近年有 **反射处理 rework**（如 PR/Issue #884、#889 方向），但仍可能在与 **重度反射、Wails、某些 RPC** 组合时出问题。
- **官方自述：** `-literals` 可逆、可能拖慢；`-tiny` 去掉 panic 信息**加大排障难度**；**导出方法**等仍有不混淆策略边界。
- **`runtime/debug.ReadBuildInfo`、`runtime.GOROOT`** 等与构建信息相关的 API 在混淆二进制中可能异常（官方 caveats）。
- **安全预期：** 混淆 **不** 等于加密或防逆向；仅提高成本。

### 征兆

- 仅 garble 构建崩溃或间歇故障；堆栈无符号难以诊断。

### 预防策略

- **CI 双轨：** `go build` 与 `garble build` 全量测试；发布前对 **Docker 驱动路径** 做冒烟。
- 谨慎开启 **`-literals`**；为生产崩溃保留 **可开关** 的诊断构建（内部渠道）。
- 阅读 [garble README caveats](https://github.com/burrowers/garble) 与当前 Go 版本要求。

### 建议负责范围

**发布工程与 CI（v1.3 收尾）**；非功能开发前置条件。

**Sources (HIGH):** [garble 文档与 caveats（pkg.go.dev / 仓库 README）](https://pkg.go.dev/mvdan.cc/garble)；[burrowers/garble issues — reflection](https://github.com/burrowers/garble/issues/884)。

---

## 10. 跨平台：macOS arm64 / amd64 镜像与 Rosetta

### 会出什么问题

- **多架构 manifest：** 若只构建 `linux/amd64`，在 Apple Silicon 上常通过 **qemu/binfmt** 仿真运行，**慢且偶发兼容问题**；应发布 **arm64** 或明确仅支持 Rosetta/x86_64 Docker。
- **Rosetta：** 在 Mac 上运行 amd64 容器依赖 **Rosetta 2 + Docker 设置**；团队需统一，否则「同样 Dockerfile 不同机器」。
- **基础镜像选择：** `FROM` 应使用 **manifest list** 或显式 `--platform` 构建矩阵，避免 CI 与开发者本地架构漂移。

### 征兆

- `exec format error`、极慢、或仅 Intel Mac 可运行。

### 预防策略

- Release 提供 **`linux/arm64` + `linux/amd64`** 或文档写明仅一种；`docker buildx build --platform` 入 CI。
- 在 PROJECT 的「本地 claude-shell」中写清：**推荐 Linux x86_64 生产同源**，Mac 为辅助。

### 建议负责范围

**镜像发布与安装文档（v1.3）**。

**Sources:** Docker buildx/multi-platform 官方文档；**MEDIUM**。

---

## Phase-Specific Warnings（v1.3 映射）

| 子域 | 高风险点 | 缓解 |
|------|----------|------|
| Go 包装器 | TTY/信号/SIGWINCH | §1；集成测试；必要时 `--init` |
| 卷与权限 | UID、`.claude` 持久化 | §2、§7 |
| Desktop 与生产差异 | host 网络、none+tun | §3 |
| 安全挂载 | proc 伪造、privileged 蔓延 | §4 |
| sing-box + nft | 能力、规则顺序、MPTCP/UID | §5 |
| 体验 | 冷启动、镜像体积 | §6 |
| Claude Code | 自动更新、登录 | §7 |
| 交付 | garble、符号、CI | §9 |
| 发布物 | 多架构 | §10 |

---

## 「看起来像做完但其实没有」检查清单

- [ ] Resize 终端后 Claude Code 仍可用，且不会因 SIGWINCH 误杀主进程
- [ ] Ctrl+C / SIGTERM 能在预期时间内结束容器
- [ ] 绑定卷上可写配置与项目目录，UID 行为有文档
- [ ] Linux 与 Docker Desktop 验证矩阵至少覆盖：localhost 回连、DNS、`verify` 出口 IP
- [ ] 未使用 `--privileged` 除非书面论证；`-v` 注入 proc 伪装优先于容器内 mount
- [ ] sing-box：`NET_ADMIN` + `/dev/net/tun` + 路由/nft 与 v1.1 路径无重复冲突
- [ ] `DISABLE_AUTOUPDATER` 策略与镜像版本策略一致
- [ ] garble 构建通过全套测试；发布流水线可复现
- [ ] 镜像 arm64/amd64 声明与 CI 一致

---

## Gaps（需阶段内实测）

- 指定宿主机发行版 + Docker 版本 + Desktop 版本下的 **nft + sing-box auto_redirect** 最小权限集合。
- Claude Code **Bun standalone** 在只读根文件系统 + 可写卷组合下的确切路径与权限。
- garble 与 **Docker API / x/crypto / 插件** 依赖的交互（以项目依赖为准）。

---

## Sources（权威优先）

- Docker：`docker run`（`--tty`、`--interactive`、`--sig-proxy`、`--init`、`--privileged`、`--cap-add`）— https://docs.docker.com/engine/reference/commandline/run/
- Docker：`docker attach`（`--sig-proxy`）— https://docs.docker.com/engine/reference/commandline/attach/
- Docker：runtime privilege and capabilities — https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities
- Docker Desktop：Settings（含网络与 host networking 说明）— https://docs.docker.com/desktop/settings-and-maintenance/settings/
- sing-box：Tun inbound — https://sing-box.sagernet.org/configuration/inbound/tun/
- Claude Code：Setup / Auto-updates、`DISABLE_AUTOUPDATER` — https://code.claude.com/docs/en/setup
- garble：README / caveats — https://github.com/burrowers/garble
- runc：`checkProcMount` 讨论 — https://github.com/opencontainers/runc/issues/2826

---
*专项研究：v1.3 claude-shell 本地透明代理 — 集成陷阱*
*Researched: 2026-04-09*
