# Phase 3: 启动入口与 SSH 接入 - Context

**Gathered:** 2026-03-27
**Status:** Ready for planning

<domain>
## Phase Boundary

本阶段在现有控制面、任务系统与 host-agent 边界上，交付 `curl -> 认证 -> 启动中提示 -> SSH` 的终端主链路：用户在终端完成认证、触发主机启动、看到进度并在 SSH 真正可用后自动接入会话。后台 CRUD 能力、计费、多宿主机调度和 Web Terminal 不在本阶段范围。

</domain>

<decisions>
## Implementation Decisions

### 启动入口与认证交互
- **D-01:** 提供单一短 `curl` 入口，下载并执行受管 bootstrap 脚本；脚本仅负责终端交互与调用控制面接口，不内嵌复杂业务逻辑。
- **D-02:** bootstrap 脚本在终端内采集用户名与密码，密码输入必须关闭回显；认证失败时立即终止，不进入启动流程。
- **D-03:** 认证以 `users.username + password_hash` 为准，并校验账号可用状态（如禁用/过期）；失败场景统一返回终端友好错误码与文案。

### 启动任务编排与进度通道
- **D-04:** 认证通过后统一走现有异步任务链路触发 `start_host`，复用 `QueueHostAction` 与任务状态机，不新增绕过任务系统的启动路径。
- **D-05:** 终端通过轮询启动状态接口展示进度，基础状态映射为 `pending/running/succeeded/failed`，并补充面向用户的阶段文案。
- **D-06:** 进度阶段固定为“认证通过 → 主机启动中 → 运行时校验中 → SSH 就绪 → 进入会话”，避免模糊提示。

### SSH 就绪门槛与交接
- **D-07:** 只有当启动任务成功且 SSH 就绪检查通过后，才允许向用户返回可接入信息（满足 `RUNT-03` 的就绪门槛要求）。
- **D-08:** 控制面返回标准 SSH 交接载荷（主机、端口、用户、必要连接参数）；bootstrap 脚本直接 `exec ssh` 完成交接，避免用户手工查找连接信息。
- **D-09:** v1 保持 SSH-only，不提供 Web Terminal 或其他替代入口作为回退路径。

### 失败映射与重试体验
- **D-10:** 对 `凭证错误 / 账号不可用 / 未绑定出口 IP / 启动失败 / SSH 未就绪` 建立稳定错误分类，输出可直接在终端展示的中文提示。
- **D-11:** 延续前序阶段“失败不自动重试”原则：系统不做隐式恢复或自动切换，用户侧仅提供明确重试建议。
- **D-12:** 启动失败时保留任务 `last_error_summary` 与关键事件上下文，便于后台与运维快速定位。
- **D-13:** 终端入口对失败类型返回确定的非零退出码，便于脚本化调用与自动化检查。

### Claude's Discretion
- 轮询间隔、超时阈值、退避策略的具体参数。
- 终端进度展示形式（spinner、分段提示、日志密度）。
- SSH 命令参数细节（如 host key 策略与临时配置文件组织方式）。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 阶段范围与验收标准
- `.planning/ROADMAP.md` — Phase 3 的目标、成功标准与 03-01/03-02/03-03 计划拆分。
- `.planning/REQUIREMENTS.md` — `ACCS-01` ~ `ACCS-04` 与 `RUNT-03` 的正式定义。
- `.planning/PROJECT.md` — “一条命令启动 + SSH-only + 受控出网”产品边界。
- `.planning/STATE.md` — 当前阶段位置与近期连续决策背景。

### 前置阶段锁定决策
- `.planning/phases/01-foundation-control-plane-host-agent/01-CONTEXT.md` — 控制面与 host-agent 特权边界、异步任务模型、失败策略。
- `.planning/phases/02-tunnel-egress-enforcement/02-CONTEXT.md` — 启动前后网络校验、无绑定即失败、默认拒绝与事件记录约束。
- `.planning/codebase/ARCHITECTURE.md` — `curl -> 认证 -> 启动任务 -> 就绪校验 -> SSH 接入` 的核心数据流。

### 当前代码锚点
- `internal/controlplane/http/router.go` — 现有 API 路由组织，可扩展启动入口与状态查询端点。
- `internal/runtime/runtime_service.go` — `QueueHostAction` 编排入口与任务创建路径。
- `internal/runtime/tasks/worker.go` — `start_host` 执行流程、网络就绪检查与失败记录行为。
- `internal/store/migrations/0001_initial.sql` — `users.password_hash`、`tasks`、`events` 的数据基础。
- `internal/store/repository/queries.go` — 任务状态读取、错误摘要与事件落库能力。

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/controlplane/http/router.go` 与 `internal/controlplane/http/hosts.go`：已具备标准化路由与 host lifecycle action 入队接口，可复用到启动入口 API。
- `internal/runtime/runtime_service.go`：已封装 host action 入队、镜像规格加载与异步 dispatch，可作为启动流程编排主干。
- `internal/runtime/tasks/worker.go`：已实现 `start_host`、网络校验、事件记录与错误摘要，可在此基础上补齐 SSH 就绪判定。
- `internal/store/repository/queries.go`：已有任务列表与 `last_error_summary` 读取能力，可直接服务终端进度查询。

### Established Patterns
- 控制面使用 `net/http` + JSON 响应，错误通过统一状态码与 message 返回。
- 生命周期动作统一进入异步任务状态机（`pending/running/succeeded/failed/canceled`）。
- 高权限运行时动作集中在 host-agent，不允许控制面直接执行 Docker/网络特权操作。
- 失败路径强调“结构化记录 + 显式人工重试”，不做静默自动恢复。

### Integration Points
- 在 `internal/controlplane/http` 增加终端启动入口与状态查询接口，并接入现有 runtime service。
- 在 repository 增加“按用户名读取认证信息/账号状态”与“按任务聚合启动进度信息”的查询能力。
- 在 `internal/runtime/tasks/worker.go` 的启动链路补充 SSH readiness gate，与现有网络 ready gate 组合为最终放行条件。
- bootstrap 脚本与控制面 API 契约应共享稳定错误码，确保 ACCS-04 的终端失败体验一致。

</code_context>

<specifics>
## Specific Ideas

- 启动体验必须保持 `curl -> 输入凭证 -> 明确 loading -> 自动进入 SSH`，不要要求用户手动拼接连接参数。
- 错误提示优先“可执行下一步”，例如直接给出重试或联系管理员建议，而不是抽象异常名。

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 03-ssh*
*Context gathered: 2026-03-27*
