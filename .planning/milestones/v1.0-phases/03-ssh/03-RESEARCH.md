# Phase 3: 启动入口与 SSH 接入 - Research

**Researched:** 2026-03-27
**Domain:** 终端 bootstrap 入口、凭证认证、启动任务进度、SSH readiness gate 与 handoff
**Confidence:** MEDIUM

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** 提供单一短 `curl` 入口，下载并执行受管 bootstrap 脚本；脚本仅负责终端交互与调用控制面接口，不内嵌复杂业务逻辑。
- **D-02:** bootstrap 脚本在终端内采集用户名与密码，密码输入必须关闭回显；认证失败时立即终止，不进入启动流程。
- **D-03:** 认证以 `users.username + password_hash` 为准，并校验账号可用状态（如禁用/过期）；失败场景统一返回终端友好错误码与文案。
- **D-04:** 认证通过后统一走现有异步任务链路触发 `start_host`，复用 `QueueHostAction` 与任务状态机，不新增绕过任务系统的启动路径。
- **D-05:** 终端通过轮询启动状态接口展示进度，基础状态映射为 `pending/running/succeeded/failed`，并补充面向用户的阶段文案。
- **D-06:** 进度阶段固定为“认证通过 → 主机启动中 → 运行时校验中 → SSH 就绪 → 进入会话”，避免模糊提示。
- **D-07:** 只有当启动任务成功且 SSH 就绪检查通过后，才允许向用户返回可接入信息（满足 `RUNT-03` 的就绪门槛要求）。
- **D-08:** 控制面返回标准 SSH 交接载荷（主机、端口、用户、必要连接参数）；bootstrap 脚本直接 `exec ssh` 完成交接，避免用户手工查找连接信息。
- **D-09:** v1 保持 SSH-only，不提供 Web Terminal 或其他替代入口作为回退路径。
- **D-10:** 对 `凭证错误 / 账号不可用 / 未绑定出口 IP / 启动失败 / SSH 未就绪` 建立稳定错误分类，输出可直接在终端展示的中文提示。
- **D-11:** 延续前序阶段“失败不自动重试”原则：系统不做隐式恢复或自动切换，用户侧仅提供明确重试建议。
- **D-12:** 启动失败时保留任务 `last_error_summary` 与关键事件上下文，便于后台与运维快速定位。
- **D-13:** 终端入口对失败类型返回确定的非零退出码，便于脚本化调用与自动化检查。

### Claude's Discretion
- 轮询间隔、超时阈值、退避策略的具体参数。
- 终端进度展示形式（spinner、分段提示、日志密度）。
- SSH 命令参数细节（如 host key 策略与临时配置文件组织方式）。

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ACCS-01 | 用户可以执行一条简短的 `curl` 启动命令，并在终端中看到用户名密码输入提示 | 采用受管 bootstrap 脚本 + 终端静默密码输入（`read -s` / `stty -echo`）+ 控制面认证 API |
| ACCS-02 | 用户在凭证正确后，可以在终端中看到云主机启动中的状态提示 | 复用现有任务状态机（`pending/running/succeeded/failed`）并映射为固定阶段文案 |
| ACCS-03 | 当云主机已就绪且用户有权限时，系统可以直接把用户接入一个可用的 SSH 会话，而无需手工查找主机和端口 | 启动成功后增加 SSH readiness gate；控制面返回标准 handoff 载荷，脚本 `exec ssh` |
| ACCS-04 | 当凭证错误、账号过期、未绑定出口 IP 或启动失败时，用户能收到明确、适合终端展示的错误提示 | 定义稳定错误分类、统一错误码/文案、明确重试建议与非零退出码 |
| RUNT-03 | 在把用户接入会话前，平台会验证容器和 SSH 服务都已经真正就绪 | 在 `start_host` 现有网络 ready 之后新增 SSH 端口/bannner readiness gate，未通过则任务失败 |
</phase_requirements>

## Summary

Phase 3 的关键不是“再做一条新启动链路”，而是把现有 Phase 1/2 已经稳定的边界串起来：控制面负责认证与编排、任务系统负责状态推进、host-agent 负责运行时与网络特权动作。当前代码已具备可复用骨架：`QueueHostAction`、`start_host`、`tasks/events`、`last_error_summary`、`net.ready` 事件。研究结论是应在这些骨架之上增量扩展，避免平行链路。

当前最大设计缺口是“SSH handoff 的可达路径”。Phase 2 已提供容器 `mgmt0` 私有管理链路和仅放行 TCP/22 的策略，但外部用户如何拿到可达的 `host:port` 尚未固化。为满足 `ACCS-03` 与 `D-08`，建议在 host-agent 内引入“受控 SSH relay（端口映射或等价代理）”能力，并把可达地址封装为控制面 handoff 载荷的一部分，脚本侧仅 `exec ssh`，不暴露内部网络细节。

认证与终端体验应坚持“低摩擦 + 可运维”：脚本仅做交互与 API 调用，密码不回显，不缓存明文；控制面返回结构化错误码；脚本将错误码映射为稳定退出码。这样既满足用户体验，也能让自动化脚本与运维面板精准定位失败类型。

**Primary recommendation:** 以“控制面认证 + 复用 `start_host` 异步任务 + host-agent SSH readiness gate + 标准化 handoff 载荷 + 稳定错误码/退出码”落地 Phase 3，禁止新增绕过任务系统的启动路径。

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `net/http` | Go 1.25.7（`go.mod`） | bootstrap 认证/状态 API | 已是现有控制面主框架，路由与 JSON 处理成熟可复用 |
| `github.com/jackc/pgx/v5` | v5.7.6（`go.mod`） | 读取用户认证信息、任务/事件聚合 | 现有仓库已全面使用，避免引入第二套 DB 访问层 |
| 现有任务状态机（`tasks` + host-agent worker） | `pending/running/succeeded/failed/canceled`（已落库） | 启动编排与进度反馈 | 与 Phase 1/2 已验证能力一致，符合 D-04/D-05 |
| OpenSSH client/server | 客户端本机 OpenSSH_10.0p2；镜像内 OpenSSH（受管镜像） | 最终 SSH 接入与 handoff | 项目 SSH-only 边界已锁定，避免引入 Web Terminal 新入口 |
| `golang.org/x/crypto/bcrypt` | v0.37.0（项目当前） | 密码哈希校验（`CompareHashAndPassword`） | 标准自适应哈希实现，避免自研密码学逻辑 |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Bash `read -s` / `stty -echo` | GNU bash 3.2.57（本机） | 终端静默密码输入 | bootstrap 脚本采集密码时使用，且需 `trap` 恢复终端回显 |
| curl CLI | 8.7.1（本机） | 下载脚本与调用控制面 API | 推荐 `--fail --show-error --silent` + 受限协议，保证脚本友好失败 |
| 现有 `events` 表 | SQL schema 0001（已存在） | 进度阶段与失败上下文 | 对 `task_id/host_id/type/message/metadata` 聚合输出阶段化状态 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| 复用 `QueueHostAction(start_host)` | 新建同步“直启 API” | 会绕开已存在任务状态机与事件记录，破坏 D-04 |
| 任务轮询进度 | SSE/WebSocket 推送 | 实时性更好，但当前代码基线无长连接基础，Phase 3 成本过高 |
| `bcrypt` 密码校验 | 自定义 hash 或明文比较 | 安全风险不可接受，且不符合 `password_hash` 字段语义 |
| 自动连接不校验 host key | `StrictHostKeyChecking=accept-new` + 独立 known_hosts | 相对安全（首次自动接受，变更拒绝），优于完全关闭校验 |

**Installation:**
```bash
# Phase 3 不要求新增依赖；复用现有 go.mod 即可
go mod tidy
```

**Version verification:**  
已通过本仓库与本机命令确认：`go version`=1.25.7、`docker --version`=28.5.2、`ssh -V`=OpenSSH_10.0p2、`curl --version`=8.7.1；密码哈希能力来自 `go doc golang.org/x/crypto/bcrypt`（含 `CompareHashAndPassword`、72 字节密码上限）。

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── controlplane/http/           # 入口 API（新增 bootstrap 认证/状态/handoff handler）
├── runtime/                     # QueueHostAction 入口（保持不变）
├── runtime/tasks/               # start_host 执行链路（新增 SSH readiness gate）
├── network/                     # 现有 net.ready 与管理 veth 能力（复用）
└── store/repository/            # 新增认证与启动状态聚合查询

deploy/
└── bootstrap/                   # 受管终端 bootstrap 脚本（curl 下载）
```

### Pattern 1: 脚本薄层 + 控制面厚逻辑
**What:** bootstrap 脚本只做输入采集、API 调用、进度渲染、最终 `exec ssh`，业务判断全部在服务端。  
**When to use:** 所有入口交互与失败映射。  
**Example:**
```bash
# Source: https://www.gnu.org/software/bash/manual/ + `help read`
printf "用户名: "
read -r username
printf "密码: "
read -r -s password
printf "\n"
```

### Pattern 2: 认证成功后只入队 `start_host`
**What:** 控制面认证通过后，调用现有 `QueueHostAction(..., ActionStartHost, ...)`，返回 `task_id` 供轮询。  
**When to use:** `ACCS-02` 进度流程主链路。  
**Example:**
```go
// Source: internal/runtime/runtime_service.go
task, err := s.repo.CreateTask(ctx, repository.CreateTaskParams{
    HostID:      &host.ID,
    Kind:        string(action),
    Status:      repository.TaskStatusPending,
    RequestedBy: requestedBy,
})
```

### Pattern 3: 双门槛 readiness（net ready + ssh ready）
**What:** 保持现有 `net.ready` 成功事件不变，在 `startHost` 末尾追加 SSH 就绪检查（TCP/22 + banner）。  
**When to use:** `RUNT-03` 与 `ACCS-03`。  
**Example:**
```go
// Source: internal/runtime/tasks/worker.go + Phase 3 extension
if err := w.provider.PrepareHost(ctx, spec); err != nil {
    return fmt.Errorf("prepare host network: %w", err)
}
if err := w.waitForSSHReady(ctx, request.HostID); err != nil {
    return fmt.Errorf("ssh readiness gate failed: %w", err)
}
```

### Pattern 4: 阶段化状态 API（而非日志拼接）
**What:** 轮询接口返回 `stage_code/stage_text/task_status/error_code/retryable`，脚本端仅映射显示。  
**When to use:** `ACCS-02`、`ACCS-04`。  
**Example:**
```json
{
  "task_id": "uuid",
  "task_status": "running",
  "stage_code": "runtime_validating",
  "stage_text": "运行时校验中",
  "error_code": "",
  "retryable": false
}
```

### Anti-Patterns to Avoid
- **绕过任务系统直启容器：** 会失去 `tasks/events/last_error_summary` 可观测性，且违背 D-04。
- **以“容器 running”替代 SSH 就绪：** 会导致用户进入半启动状态，违背 RUNT-03。
- **脚本写死复杂业务分支：** 一旦 API 变更会产生多端漂移，排障困难。
- **`StrictHostKeyChecking=no` 常驻：** 会弱化 MITM 防护；应优先 `accept-new` 或预置 known_hosts。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 启动编排 | 自建内存状态机 + goroutine 轮询 | 现有 `QueueHostAction` + `tasks/events` | 已有持久化状态与错误摘要，且与 Phase 1/2 对齐 |
| 认证加密 | 自定义 hash 算法 | `bcrypt.CompareHashAndPassword` | 标准实现，避免密码学坑 |
| 进度来源 | 解析日志文本 | 结构化状态字段 + 事件类型 | 便于脚本稳定渲染与后端演进 |
| readiness 判断 | 仅看 Docker 容器状态 | net.ready + ssh.ready 双门槛 | 直接满足 RUNT-03，减少假就绪 |
| 错误处理 | 单一 “启动失败” 文案 | 稳定错误分类 + 退出码 | 满足 ACCS-04 且可脚本化运维 |

**Key insight:** Phase 3 成功与否主要取决于“复用边界的一致性”，不是新技术；越复用现有任务/事件/host-agent 边界，交付越稳。

## Common Pitfalls

### Pitfall 1: 任务成功过早返回，SSH 尚未可连
**What goes wrong:** `start_host` 一成功就返回 handoff，用户立即 `ssh` 失败。  
**Why it happens:** 只校验容器与网络，未校验 SSH 服务。  
**How to avoid:** 在 `PrepareHost` 后追加 SSH readiness gate（端口连通 + banner）。  
**Warning signs:** 任务 `succeeded`，但用户首连报 `Connection refused`/`No route to host`。

### Pitfall 2: 错误码与终端退出码漂移
**What goes wrong:** 同一种失败场景不同时间返回不同文案/退出码。  
**Why it happens:** API 文案直出，脚本自行猜测错误含义。  
**How to avoid:** 固化 `error_code -> 文案 -> 退出码` 映射表，脚本只消费 code。  
**Warning signs:** 自动化脚本需要字符串模糊匹配错误内容。

### Pitfall 3: 密码输入后终端回显未恢复
**What goes wrong:** 脚本异常退出后用户终端“失声”，后续输入不可见。  
**Why it happens:** 使用 `stty -echo` 未配套 `trap` 恢复。  
**How to avoid:** 始终 `trap 'stty echo' EXIT INT TERM` 或用 `read -s`。  
**Warning signs:** 中断脚本后，终端持续无回显。

### Pitfall 4: SSH host key 策略过于宽松
**What goes wrong:** 为了“无感连接”使用完全禁用 host key 校验。  
**Why it happens:** 忽略 MITM 风险。  
**How to avoid:** 优先 `StrictHostKeyChecking=accept-new` + 独立 `UserKnownHostsFile`；生产可升级为预置主机指纹。  
**Warning signs:** 所有连接均无 host key 记录，且 host key 变化不报错。

### Pitfall 5: 账号过期语义未落地
**What goes wrong:** ACCS-04 要求“账号过期”错误，但当前 schema 无明确 `expires_at` 读取路径。  
**Why it happens:** 把过期治理完全后置到 Phase 5，却在 Phase 3 就需要终端反馈。  
**How to avoid:** Phase 3 先落最小可判定语义（如 `users.status in {active,disabled,expired}`），Phase 5 再补自动到期计算。  
**Warning signs:** 只有 “auth failed”，没有“账号已过期”可区分提示。

## Code Examples

Verified patterns from official sources and current codebase:

### 1) 复用任务状态机而非新链路
```go
// Source: internal/agent/server.go
running := agentapi.TaskStatusUpdate{
    TaskID: request.TaskID,
    Status: "running",
}
if err := s.worker.UpdateTaskStatus(r.Context(), running); err != nil { /* ... */ }

update := s.worker.Execute(r.Context(), request)
_ = s.worker.UpdateTaskStatus(r.Context(), update)
```

### 2) curl 入口的稳健失败参数
```bash
# Source: https://curl.se/docs/manpage.html
curl --fail --show-error --silent --location --proto '=https' \
  "https://example.com/bootstrap.sh"
```

### 3) SSH host key 安全折中
```bash
# Source: https://man.openbsd.org/ssh_config
ssh -o StrictHostKeyChecking=accept-new \
    -o UserKnownHostsFile="$TMPDIR/cloud-proxy-known_hosts" \
    -o ConnectTimeout=8 \
    -o BatchMode=yes \
    "$SSH_USER@$SSH_HOST" -p "$SSH_PORT"
```

### 4) 密码哈希校验
```go
// Source: go doc golang.org/x/crypto/bcrypt
if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
    return ErrAuthInvalid
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 认证后同步阻塞启动并打印杂乱日志 | 异步任务 + 结构化阶段轮询 | 现代控制面常见模式 | 可观测、可重试、可运维 |
| 容器 running 即视为可接入 | network ready + ssh ready 双门槛 | Phase 2 完成网络 gate 后自然演进 | 减少“假就绪”导致的首连失败 |
| 统一“启动失败” | 稳定错误分类 + 退出码 | 运维自动化成熟实践 | 终端与后台可快速定位根因 |
| `StrictHostKeyChecking=no` | `accept-new`/预置 known_hosts | OpenSSH 客户端实践收敛 | 兼顾无感接入与 MITM 基本防护 |

**Deprecated/outdated:**
- “日志文本即接口”：不应再让脚本依赖日志字符串判定阶段。
- “新增平行启动 API”：与现有任务系统重复且容易产生状态不一致。

## Open Questions

1. **SSH handoff 的外部可达路径最终采用哪种 relay 机制**
   - What we know: Phase 2 已有容器 `mgmt0` 与仅 SSH 放行策略；外部用户仍需可达 `host:port` 才能满足 D-08。
   - What's unclear: 使用 host-agent 管理 DNAT/端口映射，还是用户态 TCP relay 进程。
   - Recommendation: 以 host-agent 统一托管 relay（仍在特权边界内），并把 `host/port` 持久化到 host 元数据供 handoff 输出。

2. **“账号过期”在 Phase 3 的最小实现语义**
   - What we know: `users.status` 已存在，`expires_at` 尚未在当前 schema 中可用。
   - What's unclear: Phase 3 是否引入最小到期字段，还是仅使用状态值。
   - Recommendation: Phase 3 先采用状态值落地 ACCS-04；Phase 5 再补完整到期时间治理与自动流转。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | 控制面/agent 开发与测试 | ✓ | go1.25.7 | — |
| Docker CLI + daemon | 启动任务与运行时操作 | ✓ | 28.5.2 | — |
| SSH client | bootstrap 最终 handoff | ✓ | OpenSSH_10.0p2 | — |
| curl | bootstrap 下载与 API 调用 | ✓ | 8.7.1 | — |
| Bash | bootstrap 交互脚本 | ✓ | 3.2.57 | zsh/sh（需适配） |
| timeout | readiness 超时控制（脚本/探测） | ✓ | coreutils 9.10 | Go `context.WithTimeout` |
| nsenter | netns 内探测（现有网络校验链路） | ✗（当前机器） | — | 在 Linux 宿主机执行 |
| ip (iproute2) | Linux 网络接线与排障 | ✗（当前机器） | — | 在 Linux 宿主机执行 |
| nft | 防火墙规则调试/运维 | ✗（当前机器） | — | 在 Linux 宿主机执行 |
| wg | WireGuard 运维诊断 | ✗（当前机器） | — | 在 Linux 宿主机执行 |
| psql / pg_isready | 本地数据库运维检查 | ✗（当前机器） | — | 使用应用健康检查或容器内客户端 |

**Missing dependencies with no fallback:**
- None（开发可继续），但“真实 netns/WireGuard/SSH E2E”必须在 Linux 宿主机验证。

**Missing dependencies with fallback:**
- `nsenter/ip/nft/wg`：在 Linux 单宿主机或 CI Linux runner 执行集成验证。
- `psql/pg_isready`：短期可由应用健康检查替代，后续补齐运维工具链。

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing`（标准库） |
| Config file | none — Go 原生 |
| Quick run command | `go test ./internal/controlplane/... ./internal/runtime/... -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| ACCS-01 | `curl` 入口可输入用户名/密码且密码不回显 | shell + API integration | `bash test/bootstrap/test_prompt_and_auth.sh` | ❌ Wave 0 |
| ACCS-02 | 认证后显示分阶段进度（pending/running/succeeded/failed 映射） | unit + API contract | `go test ./internal/controlplane/http -run TestBootstrapStageMapping -count=1` | ❌ Wave 0 |
| ACCS-03 | 就绪后自动 SSH handoff（无需手工查 host/port） | integration | `go test ./internal/runtime/tasks -run TestStartHostRequiresSSHReady -count=1` | ❌ Wave 0 |
| ACCS-04 | 凭证错误/账号不可用/无绑定/启动失败有稳定提示与退出码 | unit + integration | `go test ./internal/controlplane/http -run TestBootstrapErrorTaxonomy -count=1` | ❌ Wave 0 |
| RUNT-03 | 接入前验证容器与 SSH 真正就绪 | integration | `go test ./internal/runtime/tasks -run TestSSHReadinessGate -count=1` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/controlplane/... ./internal/runtime/... -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green + Linux 主机上完成 bootstrap->SSH 冒烟

### Wave 0 Gaps
- [ ] `internal/controlplane/http/bootstrap_auth_test.go` — 覆盖 ACCS-01/ACCS-04
- [ ] `internal/controlplane/http/bootstrap_status_test.go` — 覆盖 ACCS-02
- [ ] `internal/runtime/tasks/worker_ssh_ready_test.go` — 覆盖 RUNT-03/ACCS-03
- [ ] `internal/controlplane/http/bootstrap_handoff_test.go` — 覆盖 handoff 载荷契约
- [ ] `test/bootstrap/e2e_bootstrap_ssh.sh` — 覆盖真实终端路径与退出码

## Project Constraints (from .cursor/rules/)

- `.cursor/rules/` 目录不存在，未发现额外 Cursor rule 文件。
- 继承仓库级约束（`CLAUDE.md` 与既有 phase context）：
  - 控制面不得直接持有 Docker/网络特权，特权动作必须收敛在 host-agent。
  - v1 保持单宿主机与 SSH-only，不引入 Web Terminal。
  - 网络强约束优先，不能破坏 Phase 2 的隧道与防泄漏模型。
  - 默认不自动重试，失败需结构化记录并可运维定位。
  - 面向用户的文案与提示保持中文。

## Sources

### Primary (HIGH confidence)
- `.planning/phases/03-ssh/03-CONTEXT.md` — 本阶段锁定决策与边界
- `.planning/REQUIREMENTS.md` — `ACCS-01~04`、`RUNT-03` 定义
- `internal/runtime/runtime_service.go` — `QueueHostAction` 入队主干
- `internal/runtime/tasks/worker.go` — `start_host` 流程、`net.ready` 事件与失败摘要
- `internal/network/firewall.go` / `internal/network/namespace.go` — 管理 veth 与 SSH 仅放行规则
- `internal/controlplane/http/router.go` / `tasks.go` / `hosts.go` — 现有 API 路由与任务查询能力
- `deploy/docker/managed-user/sshd_config` / `entrypoint.sh` / `image.lock` — SSH 运行时契约
- [curl man page](https://curl.se/docs/manpage.html) — `--fail/--show-error/--silent/--proto/--location` 语义
- [OpenSSH ssh_config](https://man.openbsd.org/ssh_config) — `StrictHostKeyChecking`、`UserKnownHostsFile`、`BatchMode` 行为
- [OpenSSH ssh](https://man.openbsd.org/ssh) — known_hosts 与主机指纹校验行为
- `go doc golang.org/x/crypto/bcrypt`（本机） — `CompareHashAndPassword`、`ErrPasswordTooLong` 等 API

### Secondary (MEDIUM confidence)
- `.planning/research/ARCHITECTURE.md` — 启动入口与 SSH handoff 推荐结构
- `.planning/phases/02-tunnel-egress-enforcement/02-RESEARCH.md` — 管理 veth 与 SSH 路径的前置假设

### Tertiary (LOW confidence)
- None — 关键结论均由现有代码/官方文档支撑。

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 以仓库现有依赖与已运行命令为依据，变更面小。
- Architecture: MEDIUM — SSH relay 的具体落地方式（DNAT vs 用户态代理）仍需在计划阶段锁定。
- Pitfalls: HIGH — 来源于现有代码边界、已完成 Phase 2 行为与官方 ssh/curl 文档。

**Research date:** 2026-03-27  
**Valid until:** 2026-04-10（该阶段涉及入口安全与接入策略，建议 14 天内复核一次）
