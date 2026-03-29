# Phase 5: 到期、审计与清理 - Research

**Researched:** 2026-03-27
**Domain:** 后台定时任务、事件审计系统、运行时对账
**Confidence:** HIGH

## Summary

本阶段在已有的控制面基础上增加三大能力：用户到期治理、运维事件全链路记录与展示、以及 DB/Docker 运行时漂移对账。核心实现路径清晰——到期定时器复用现有 `QueueHostAction` 停止链路、事件记录复用已有 `RecordEvent` + JSONB 模式、对账通过 host-agent 通道查询 Docker 状态。

技术风险集中在：(1) Go 后台 goroutine 生命周期管理需与控制面主进程正确绑定，(2) events 表需要扩展 `user_id` 列并补充索引以支持按类型/用户/时间筛选查询，(3) 对账扫描需正确处理 host-agent 通信失败场景。所有技术方案都基于现有代码模式的自然延伸，无需引入新的外部依赖。

**Primary recommendation:** 在 `app.Run()` 中启动带 `context.Context` 绑定的后台 ticker goroutine 组，复用现有 repository 和 runtime 层接口完成到期执行和对账修正；events 表补 `user_id` 列和复合索引后即可支撑全量事件查询 API。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** 在 `users` 表新增 `expires_at TIMESTAMPTZ NULL` 字段；NULL 表示永不过期，非 NULL 值表示账号到期时间点。
- **D-02:** 控制面启动后台定时 goroutine（如每 60 秒一次），扫描 `expires_at <= now() AND status = 'active'` 的用户，将其 `status` 设为 `expired`。该定时器与控制面主进程同生命周期。
- **D-03:** 管理 API 和后台 UI 支持设置、修改和清除用户到期时间（满足 LIFE-04）。管理员也可以将已过期用户重新激活（设置新到期时间并恢复 `active` 状态）。
- **D-04:** Bootstrap 认证入口已在 Phase 3 实现 `expired` 状态拦截（`bootstrap_auth.go`），本阶段无需修改认证拦截逻辑，只需确保到期定时器正确触发状态变更。
- **D-05:** 到期定时器在将用户标记为 `expired` 的同时，检查该用户是否有 `status = 'running'` 的主机，如有则通过现有 `QueueHostAction(ActionStopHost)` 下发停止动作。
- **D-06:** v1 不提供宽限期（grace period），到期即执行。这与项目"宁可失败，不可打穿"的原则一致。
- **D-07:** 过期主机停止事件使用专用事件类型（`host.stop.expired`），与管理员手动停止区分开，便于审计追溯。
- **D-08:** 新增事件类型约定，覆盖认证、启动、生命周期和到期事件（见 CONTEXT.md 完整列表）。
- **D-09:** 沿用现有 `RecordEvent` + JSONB `metadata` 模式。事件统一携带 `user_id`（如适用）和 `operator`（`system` / `admin` / `bootstrap`），按需附加 `host_id`、`reason`、`action` 等上下文字段。
- **D-10:** 现有 worker 中已有的 `net.ready`、`ssh.ready` 等事件保持不变，新增事件类型不影响既有记录行为。
- **D-11:** 侧栏新增「事件日志」入口，展示全局事件时间线列表。独立于现有「任务列表」页面。
- **D-12:** 事件列表支持按事件类型、用户、主机和时间范围筛选。默认按时间倒序展示。
- **D-13:** 事件详情展示 metadata 中的上下文字段，便于排障和支持。
- **D-14:** 后台仪表板概览中增加"最近事件"摘要卡片，展示最近 N 条关键事件。
- **D-15:** 控制面定时扫描 DB 中 `hosts.status = 'running'` 的主机列表，通过 host-agent 查询 Docker 实际容器状态，发现不一致时记录 `reconcile.host.drift` 事件并修正 DB 状态。
- **D-16:** 超过可配置阈值（默认 10 分钟）仍处于 `pending` 或 `running` 状态的任务，由对账定时器标记为 `failed` 并记录 `reconcile.task.stale` 事件。
- **D-17:** 对账定时器与到期定时器共享同一后台调度框架，但执行间隔独立可配。
- **D-18:** 对账操作通过 host-agent 查询 Docker 状态，不在控制面直接持有 Docker 特权。

### Claude's Discretion
- 定时器的具体间隔参数和配置方式（环境变量 / 配置文件）。
- 事件列表页的分页策略和批量加载实现细节。
- 对账扫描的并发策略和批量大小。
- 事件 metadata 中各类型的具体字段命名约定。
- 仪表板"最近事件"卡片的展示条数和刷新策略。

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| LIFE-04 | 管理员可以为用户账号或主机分配设置到期时间 | DB 迁移增加 `expires_at` 字段 + 管理 API 扩展 + 后台 UI 支持设置/修改/清除到期时间 |
| LIFE-05 | 已过期用户不能再开启新会话，并且后台能明确显示其已过期状态 | 到期定时器自动设置 `expired` 状态（bootstrap_auth 已有拦截）+ 到期主机自动停止 + 后台展示过期状态 |
| ADMN-04 | 管理员操作和启动结果会被记录为运维事件，便于排障和支持 | 新增事件类型体系 + events 表扩展 user_id 列 + 事件日志页面 + 仪表板事件摘要 |
</phase_requirements>

## Standard Stack

### Core

本阶段无需引入新的外部依赖。所有实现基于已有的 Go 标准库和现有项目依赖。

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go `time.Ticker` | Go 1.25.7 stdlib | 后台定时器（到期扫描、对账扫描） | 标准库原生支持，无需额外调度框架 |
| Go `context.Context` | Go 1.25.7 stdlib | goroutine 生命周期绑定 | 已在项目中广泛使用的取消传播机制 |
| Go `sync.WaitGroup` | Go 1.25.7 stdlib | 优雅关闭时等待后台 goroutine 退出 | 标准并发原语 |
| `pgx/v5` | v5.7.6 | 新增查询和迁移 | 项目已有依赖 |
| TanStack React Query | ^5.75.0 | 事件列表数据获取和轮询 | 项目前端已有依赖 |
| TanStack React Router | ^1.120.0 | 事件日志页面路由 | 项目前端已有依赖 |
| lucide-react | ^0.510.0 | 事件日志侧栏图标 | 项目前端已有依赖 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `time.Ticker` | `robfig/cron` | cron 表达式更灵活，但项目只需固定间隔定时器，引入新依赖不值得 |
| 手写事件查询 SQL | GORM / sqlc | 项目已建立手写 SQL + pgx 模式，保持一致性 |
| 自定义分页 | 基于 cursor 的分页库 | 事件量级在 v1 单宿主机场景下可控，offset 分页足够 |

## Architecture Patterns

### Recommended Project Structure

```
internal/
├── controlplane/
│   ├── app/
│   │   └── app.go              # 增加后台定时器注册和 graceful shutdown
│   ├── http/
│   │   ├── admin_users.go      # 扩展支持 expires_at 和 expired 状态
│   │   ├── admin_events.go     # 新增：事件查询 API handler
│   │   └── router.go           # 注册事件查询路由
│   └── scheduler/
│       ├── scheduler.go        # 新增：后台定时器调度框架
│       ├── expiry.go           # 新增：到期扫描任务
│       └── reconciler.go       # 新增：对账扫描任务
├── store/
│   ├── repository/
│   │   ├── models.go           # User 增加 ExpiresAt 字段
│   │   └── queries.go          # 新增事件查询、到期用户扫描、对账查询
│   └── migrations/
│       └── 0003_expiry_audit.sql  # 新增迁移
web/admin/src/
├── hooks/
│   └── use-events.ts           # 新增：事件列表 hook
├── routes/_dashboard/
│   ├── events/
│   │   └── index.tsx           # 新增：事件日志页面
│   └── index.tsx               # 扩展：增加最近事件卡片
└── components/layout/
    └── sidebar.tsx             # 增加事件日志导航入口
```

### Pattern 1: 后台定时器调度框架

**What:** 统一的后台定时任务管理器，与控制面主进程同生命周期。
**When to use:** 任何需要定期执行的后台工作（到期扫描、对账扫描）。

```go
// internal/controlplane/scheduler/scheduler.go
type Job struct {
    Name     string
    Interval time.Duration
    Fn       func(ctx context.Context) error
}

type Scheduler struct {
    logger *slog.Logger
    jobs   []Job
}

func (s *Scheduler) Run(ctx context.Context) {
    var wg sync.WaitGroup
    for _, job := range s.jobs {
        wg.Add(1)
        go func(j Job) {
            defer wg.Done()
            ticker := time.NewTicker(j.Interval)
            defer ticker.Stop()
            for {
                select {
                case <-ctx.Done():
                    return
                case <-ticker.C:
                    if err := j.Fn(ctx); err != nil {
                        s.logger.Error("scheduler job failed", "job", j.Name, "error", err)
                    }
                }
            }
        }(job)
    }
    wg.Wait()
}
```

**Key point:** 通过 `ctx.Done()` 与控制面主进程的 signal 处理绑定，保证 graceful shutdown。

### Pattern 2: 到期扫描单次执行

**What:** 单次到期扫描的执行逻辑，由调度器定期调用。
**When to use:** 每次到期定时器触发时。

```go
func (e *ExpiryScanner) Scan(ctx context.Context) error {
    expiredUsers, err := e.repo.ListExpiredActiveUsers(ctx) // expires_at <= now() AND status = 'active'
    if err != nil {
        return err
    }
    for _, user := range expiredUsers {
        // 1. 标记用户为 expired
        e.repo.UpdateUserStatus(ctx, user.ID, "expired")
        e.repo.RecordEvent(ctx, RecordEventParams{...Type: "user.expired"...})
        
        // 2. 停止该用户的运行中主机
        hosts, _ := e.repo.ListHostsByUserID(ctx, user.ID)
        for _, host := range hosts {
            if host.Status == "running" {
                e.queue.QueueHostAction(ctx, host.ID, ActionStopHost, "system:expiry")
                e.repo.RecordEvent(ctx, RecordEventParams{...Type: "host.stop.expired"...})
            }
        }
    }
    return nil
}
```

### Pattern 3: 事件查询 API 筛选

**What:** 支持多条件组合筛选的事件列表查询。
**When to use:** 管理后台事件日志页面。

```go
type ListEventsParams struct {
    EventType string    // 按事件类型筛选
    UserID    string    // 按用户 ID 筛选
    HostID    string    // 按主机 ID 筛选
    Since     time.Time // 时间范围起始
    Until     time.Time // 时间范围结束
    Limit     int       // 分页大小
    Offset    int       // 分页偏移
}

// 动态构建 WHERE 子句，参数化查询防注入
func (r *Repository) ListEvents(ctx context.Context, params ListEventsParams) ([]Event, int, error) {
    // 查询总数 + 分页数据，返回 (events, totalCount, error)
}
```

### Pattern 4: 对账扫描

**What:** 比对 DB 状态和 Docker 实际状态，修正漂移。
**When to use:** 定期对账和管理员手动触发。

```go
func (r *Reconciler) ReconcileHosts(ctx context.Context) error {
    dbHosts, _ := r.repo.ListRunningHosts(ctx)  // hosts WHERE status = 'running'
    for _, host := range dbHosts {
        containerName := containerNameForHost(host.ID)
        status, err := r.agent.InspectContainer(ctx, containerName)
        if err != nil || status != "running" {
            r.repo.UpdateHostStatus(ctx, host.ID, "stopped")
            r.repo.RecordEvent(ctx, RecordEventParams{
                Type: "reconcile.host.drift",
                Metadata: map[string]any{
                    "host_id": host.ID, "db_status": "running", "actual_status": status,
                },
            })
        }
    }
    return nil
}
```

### Anti-Patterns to Avoid

- **在定时器回调中使用 `context.Background()` 代替传入的 ctx:** 会导致 graceful shutdown 时定时器无法及时退出。
- **在到期扫描中直接调用 Docker API:** 违反特权边界（D-18），必须通过 host-agent。
- **在事件查询中使用字符串拼接构建 SQL:** 项目已有参数化查询模式，拼接会引入 SQL 注入风险。
- **在前端轮询事件列表时使用过短的间隔:** 事件日志不需要实时性，10-15 秒轮询或手动刷新即可。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 定时任务调度 | 完整的 cron 系统 | `time.Ticker` + context 取消 | v1 只需固定间隔，无需 cron 表达式 |
| SQL 动态筛选 | 字符串拼接 WHERE | 条件参数化构建 | 项目已有安全的参数化查询模式 |
| 前端分页 | 自定义分页逻辑 | TanStack Query 的 `keepPreviousData` + URL query params | 框架已提供分页友好的缓存机制 |
| 时间格式化 | 自定义日期处理 | `toLocaleDateString("zh-CN")` | 前端已有此模式（见 tasks/index.tsx） |

## Common Pitfalls

### Pitfall 1: Goroutine 泄漏

**What goes wrong:** 后台定时器的 goroutine 在控制面关闭时未正确退出，导致进程 hang 住。
**Why it happens:** 缺少 context 取消传播或 WaitGroup 等待。
**How to avoid:** 所有后台 goroutine 必须监听 `ctx.Done()`，`app.Run()` 在收到 shutdown 信号后 cancel context 并等待所有 goroutine 退出。
**Warning signs:** 进程退出时日志中无定时器关闭确认日志。

### Pitfall 2: 到期扫描的并发安全

**What goes wrong:** 两次相邻的到期扫描可能同时处理同一用户，导致重复下发停止动作。
**Why it happens:** 扫描间隔短于单次扫描执行时间。
**How to avoid:** 在定时器回调中使用互斥锁或"下次触发时如果上次还在执行则跳过"的策略。更简单的方案：`UpdateUserStatus` 的 SQL 加 `WHERE status = 'active'` 条件，利用数据库行级锁保证幂等。
**Warning signs:** 日志中同一用户的 `user.expired` 事件出现多次。

### Pitfall 3: Events 表查询性能

**What goes wrong:** 事件量增长后，无索引的 `type`/`user_id`/`created_at` 列导致全表扫描。
**Why it happens:** 现有 events 表只有主键索引，没有针对查询场景的覆盖索引。
**How to avoid:** 迁移中为 events 表添加 `(created_at DESC)`、`(type, created_at DESC)` 和 `(user_id, created_at DESC)` 索引。
**Warning signs:** 事件列表页面加载缓慢。

### Pitfall 4: 对账扫描中 host-agent 不可用

**What goes wrong:** host-agent 进程未运行或 Unix socket 不可用时，对账扫描失败并可能错误修正 DB 状态。
**Why it happens:** 对账将"无法查询 Docker 状态"误判为"容器不存在"。
**How to avoid:** 严格区分"通信失败"和"容器不存在"两种情况。通信失败时记录错误日志但不修改 DB 状态；只有明确得到"容器不存在/已停止"响应时才修正。
**Warning signs:** 重启 host-agent 后大量主机状态被错误标记为 stopped。

### Pitfall 5: 用户状态扩展遗漏

**What goes wrong:** `admin_users.go` 的 `UpdateStatus` 只允许 `active`/`disabled`，新增 `expired` 后未同步扩展验证逻辑。
**Why it happens:** 硬编码的状态白名单。
**How to avoid:** D-03 要求管理员可将 expired 用户重新激活——需要扩展 `UpdateStatus` 的 `PATCH /v1/admin/users/{userID}` 端点，使其支持 `active`/`disabled`/`expired` 之间的合法转换。同时需要新增独立的到期时间设置端点或将 `expires_at` 合并到用户更新 API。
**Warning signs:** 管理员无法在 UI 中重新激活过期用户。

## Code Examples

### 1. 数据库迁移 (0003_expiry_audit.sql)

```sql
-- 用户到期字段
ALTER TABLE users ADD COLUMN expires_at TIMESTAMPTZ;

-- 事件表扩展 user_id 列
ALTER TABLE events ADD COLUMN user_id UUID REFERENCES users (id) ON DELETE SET NULL;

-- 事件查询索引
CREATE INDEX idx_events_created_at ON events (created_at DESC);
CREATE INDEX idx_events_type_created_at ON events (type, created_at DESC);
CREATE INDEX idx_events_user_id_created_at ON events (user_id, created_at DESC);
CREATE INDEX idx_events_host_id_created_at ON events (host_id, created_at DESC);

-- 到期扫描索引
CREATE INDEX idx_users_expires_at_status ON users (expires_at, status) WHERE expires_at IS NOT NULL AND status = 'active';

-- 对账扫描索引
CREATE INDEX idx_hosts_status ON hosts (status) WHERE status = 'running';
CREATE INDEX idx_tasks_stale ON tasks (status, updated_at) WHERE status IN ('pending', 'running');
```

### 2. User 模型扩展

```go
type User struct {
    ID        string     `json:"id"`
    Username  string     `json:"username"`
    Status    string     `json:"status"`
    ExpiresAt *time.Time `json:"expires_at,omitempty"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
}
```

### 3. RecordEventParams 扩展

```go
type RecordEventParams struct {
    TaskID   *string
    HostID   *string
    UserID   *string        // 新增：关联用户 ID
    Level    string
    Type     string
    Message  string
    Metadata map[string]any
}
```

### 4. 事件查询 API 响应格式

```json
{
  "events": [
    {
      "id": "...",
      "type": "user.expired",
      "level": "info",
      "message": "用户已过期",
      "user_id": "...",
      "host_id": null,
      "metadata": {"operator": "system", "reason": "expires_at reached"},
      "created_at": "2026-03-27T12:00:00Z"
    }
  ],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

### 5. 后台调度器接入 app.Run()

```go
func (a *App) Run(ctx context.Context) error {
    // ... migrations ...
    
    sched := scheduler.New(a.logger, []scheduler.Job{
        {Name: "expiry-scan", Interval: expiryInterval, Fn: expiryScanner.Scan},
        {Name: "reconcile", Interval: reconcileInterval, Fn: reconciler.Run},
    })
    
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()
    
    go sched.Run(ctx)  // 后台定时器，ctx cancel 时自动退出
    
    // ... HTTP server ...
}
```

### 6. 前端事件日志 Hook

```typescript
export interface EventItem {
  id: string;
  type: string;
  level: string;
  message: string;
  user_id: string | null;
  host_id: string | null;
  metadata: Record<string, unknown>;
  created_at: string;
}

export function useEvents(params: { type?: string; userId?: string; limit?: number; offset?: number }) {
  const searchParams = new URLSearchParams();
  if (params.type) searchParams.set("type", params.type);
  if (params.userId) searchParams.set("user_id", params.userId);
  // ...
  return useQuery({
    queryKey: ["events", params],
    queryFn: () => apiFetch<{ events: EventItem[]; total: number }>(`/events?${searchParams}`),
    refetchInterval: 15000,
  });
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Go `time.After` 循环 | `time.NewTicker` + `select` | Go 1.23+ ticker 自动 GC | 无需手动 `Stop()` 防泄漏（但仍建议显式 Stop） |
| 全量扫描到期用户 | 部分索引 + WHERE 条件 | PostgreSQL 11+ | 到期扫描只命中 `active` 且有 `expires_at` 的行 |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + httptest (后端)；项目未建立前端测试框架 |
| Config file | 无独立配置文件，使用 `go test ./...` |
| Quick run command | `go test ./internal/controlplane/... ./internal/store/... -count=1 -short` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| LIFE-04 | 管理 API 支持设置/修改/清除到期时间 | unit | `go test ./internal/controlplane/http/ -run TestAdminUsers -count=1` | ❌ Wave 0 |
| LIFE-05 | 到期定时器将 active+expired_at<=now 标记为 expired | unit | `go test ./internal/controlplane/scheduler/ -run TestExpiry -count=1` | ❌ Wave 0 |
| LIFE-05 | 过期用户运行中主机被停止 | unit | `go test ./internal/controlplane/scheduler/ -run TestExpiryHostStop -count=1` | ❌ Wave 0 |
| ADMN-04 | 事件查询 API 支持筛选和分页 | unit | `go test ./internal/controlplane/http/ -run TestAdminEvents -count=1` | ❌ Wave 0 |
| ADMN-04 | 对账发现漂移并记录事件 | unit | `go test ./internal/controlplane/scheduler/ -run TestReconcile -count=1` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/controlplane/... -count=1 -short`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/controlplane/scheduler/scheduler_test.go` — 定时器调度框架基础测试
- [ ] `internal/controlplane/scheduler/expiry_test.go` — 到期扫描逻辑测试
- [ ] `internal/controlplane/scheduler/reconciler_test.go` — 对账扫描逻辑测试
- [ ] `internal/controlplane/http/admin_events_test.go` — 事件查询 API 测试

## Open Questions

1. **host-agent 容器状态查询 API 是否已有？**
   - What we know: host-agent 已有 `HostActionRequest` 的 `Dispatch` 接口，用于执行 start/stop/rebuild。
   - What's unclear: 是否已有独立的"仅查询容器状态"端点，还是需要新增 `inspect` 动作。
   - Recommendation: 检查 `agentapi` 包中是否有 inspect 相关的 action 类型。如果没有，对账需要新增 `ActionInspectHost` 或复用 worker 中的 `containerExists` 逻辑通过 agent 暴露。

2. **定时器间隔的配置方式**
   - What we know: D-17 要求独立可配，Claude's Discretion 中列出。
   - Recommendation: 使用环境变量 `EXPIRY_SCAN_INTERVAL`（默认 60s）和 `RECONCILE_INTERVAL`（默认 120s），与现有 `ADMIN_JWT_SECRET` 等配置方式保持一致。

3. **事件列表分页上限**
   - What we know: v1 单宿主机场景下事件量可控。
   - Recommendation: 默认每页 50 条，最大 200 条。使用 offset 分页即可。

## Sources

### Primary (HIGH confidence)
- 项目现有代码：`internal/store/repository/queries.go` — RecordEvent、ListPendingTasks 模式
- 项目现有代码：`internal/controlplane/app/app.go` — 控制面启动和 graceful shutdown 模式
- 项目现有代码：`internal/runtime/runtime_service.go` — QueueHostAction 复用入口
- 项目现有代码：`internal/controlplane/http/bootstrap_auth.go` — expired 状态拦截已就绪
- 项目现有代码：`internal/controlplane/http/admin_users.go` — 用户管理 API 模式
- 项目现有代码：`internal/store/migrations/` — 迁移文件命名和应用模式
- 项目现有代码：`web/admin/src/hooks/use-tasks.ts` — TanStack Query 轮询模式
- 项目现有代码：`web/admin/src/routes/_dashboard/tasks/index.tsx` — 列表页面模式
- 项目现有代码：`web/admin/src/components/layout/sidebar.tsx` — 侧栏导航模式
- Go 标准库文档：`time.NewTicker`、`context.WithCancel`、`sync.WaitGroup`

### Secondary (MEDIUM confidence)
- PostgreSQL 文档：部分索引 (CREATE INDEX ... WHERE) 适合稀疏条件查询场景

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 全部基于项目现有依赖，无新引入
- Architecture: HIGH — 定时器、事件记录、对账模式都是成熟的 Go 编程范式
- Pitfalls: HIGH — 基于代码审查发现的具体风险点，每个都有明确的预防策略

**Research date:** 2026-03-27
**Valid until:** 2026-04-27 (项目依赖稳定，无快速变动风险)
