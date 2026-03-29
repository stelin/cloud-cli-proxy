# Phase 12: 用户自助 API 与前端路由 - Research

**Researched:** 2026-03-28
**Domain:** Go HTTP handler + React (TanStack Router / React Query) 用户面板
**Confidence:** HIGH

## Summary

Phase 12 在现有管理员 API 和前端框架之上构建用户自助面板。后端需要新增 `/v1/user/` 前缀的三个端点（主机列表、主机详情含出口 IP、主机重建），前端需要在 `_portal/` 路由下新增主机列表和详情页面。

技术风险极低：所有后端查询方法（`ListHostsByUserID`、`GetHost`、`GetEgressIPByHost`）和认证基础设施（`AuthMiddleware`、`RequireRole`、`UserIDFromContext`）已就位。前端同样有成熟的模式可复用（React Query hooks、shadcn/ui 组件、TanStack Router 文件路由）。唯一需要新写的是用户级 handler（带归属校验）、用户 API 的 `apiFetch` 变体、以及两个页面组件。

**Primary recommendation:** 严格遵循已有的 `AdminHostsHandler` 模式实现 `UserHostsHandler`，在每个端点内通过 `UserIDFromContext` + 数据库查询实现归属校验，前端通过独立的 `portalApiFetch` 封装 `/v1/user/` 前缀。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** 用户 API 使用 `/v1/user/` 前缀，与管理员 `/v1/admin/` 对称
- **D-02:** 用户路由守卫 `userGuard = authMw(RequireRole("user", "admin"))`，管理员也能访问用户端点
- **D-03:** 每个用户端点内部从 `UserIDFromContext(r.Context())` 获取 user_id，查询时自动过滤只返回属于该用户的资源
- **D-04:** 新增端点：`GET /v1/user/hosts`（我的主机列表）、`GET /v1/user/hosts/{hostID}`（主机详情含出口 IP）、`POST /v1/user/hosts/{hostID}/rebuild`（触发重建）
- **D-05:** 用户主页 `/portal` 展示主机列表，每台主机以卡片形式显示：主机名（hostname）、运行状态（status badge）、出口 IP 地址和隧道类型、创建时间
- **D-06:** 复用现有 `_portal.tsx` 布局（Topbar + 主内容区），不需要侧边栏
- **D-07:** 前端路由结构：`/portal`（主机列表首页）、`/portal/hosts/$hostId`（主机详情页）
- **D-08:** 重建按钮点击后弹出确认对话框，明确警告"重建将重置容器环境，home 目录数据保留"
- **D-09:** 确认后调用 `POST /v1/user/hosts/{hostID}/rebuild`，后端通过 `QueueHostAction` 排队任务
- **D-10:** 重建触发后前端显示任务进度状态（可通过轮询或直接更新主机状态实现），重建期间主机状态变为 rebuilding
- **D-11:** 出口 IP 信息在主机详情页展示，只读模式，显示 IP 地址和隧道类型（wireguard/proxy）
- **D-12:** 在主机列表卡片中也简要展示出口 IP 地址（不展示隧道类型细节）
- **D-13:** 用户请求的 hostID 必须通过 user_id 归属校验，不匹配返回 403
- **D-14:** 管理员角色访问用户端点时跳过归属校验（管理员可查看任意用户资源）

### Claude's Discretion
- 主机卡片的具体样式（阴影、圆角、间距）
- 状态 badge 的颜色映射（running=green, stopped=gray, rebuilding=yellow 等）
- 空状态展示（无主机时的提示文案和样式）
- 主机详情页的具体布局
- 重建进度的轮询间隔

### Deferred Ideas (OUT OF SCOPE)
- Claude 账号展示 -- Phase 13
- KasmVNC 远程桌面访问入口 -- Phase 14
- 用户面板的更多个性化设置 -- 未来考虑
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PANEL-01 | 用户可在自助面板查看自己的主机列表和运行状态 | `ListHostsByUserID` 已存在，前端通过 React Query + 卡片组件渲染 |
| PANEL-02 | 用户可在自助面板查看自己主机绑定的出口 IP | `GetEgressIPByHost` 和 `GetHostDetail`（含 bindings）已存在，主机详情页展示 |
| PANEL-03 | 用户可在自助面板触发主机重建操作 | `QueueHostAction` + `ActionRebuildHost` 已存在，前端 mutation + 确认对话框 |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go `net/http` + `AuthMiddleware` | Go 1.26.1 | 用户 API 端点和认证守卫 | 项目已有完整的 handler 模式和中间件链 |
| TanStack Router | 已安装 | 文件路由 `_portal/` | 项目已采用，文件路由约定已建立 |
| React Query (TanStack Query) | 已安装 | 数据获取和缓存 | 项目已采用，hooks 模式已建立 |
| shadcn/ui | 已安装 | Card、Badge、Button、AlertDialog 等 | 项目已有全套 UI 组件 |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| lucide-react | 已安装 | 图标（Server、Globe、RefreshCw 等） | 主机卡片和操作按钮 |
| sonner | 已安装 | toast 通知 | 重建操作结果反馈 |

### Alternatives Considered
无需考虑替代方案。所有技术选型已由现有代码库决定。

## Architecture Patterns

### 后端：用户 Handler 结构

```
internal/controlplane/http/
├── user_hosts.go         # UserHostsHandler（List、Get、Rebuild）
├── router.go             # 新增 /v1/user/ 路由注册区块
└── auth_middleware.go     # 已有，直接复用
```

### 前端：Portal 路由结构

```
web/admin/src/
├── routes/
│   └── _portal/
│       ├── index.tsx               # /portal 改为主机列表卡片页
│       └── hosts/
│           └── $hostId.tsx         # /portal/hosts/$hostId 主机详情页
├── hooks/
│   └── use-portal-hosts.ts        # 用户级 API hooks
├── lib/
│   └── portal-api.ts              # portalApiFetch（/v1/user/ 前缀）
└── components/
    └── portal/                    # 用户面板专用组件（可选）
```

### Pattern 1: 用户级 Handler（对标 AdminHostsHandler）

**What:** `UserHostsHandler` struct 持有 store 接口和 queue 接口，每个方法返回 `net/http.Handler`
**When to use:** 所有 `/v1/user/` 端点

```go
// 参考 admin_hosts.go 的 handler 模式
type UserHostStore interface {
    ListHostsByUserID(context.Context, string) ([]repository.Host, error)
    GetHost(context.Context, string) (repository.Host, error)
    GetHostDetail(context.Context, string) (repository.HostDetail, error)
}

type UserHostsHandler struct {
    logger *slog.Logger
    store  UserHostStore
    queue  HostActionQueuer
    events EventRecorder
}
```

### Pattern 2: 归属校验模式

**What:** 每个用户端点在 handler 内部做 user_id 归属检查
**When to use:** 所有涉及 hostID 路径参数的端点

```go
func (h *UserHostsHandler) Get() nethttp.Handler {
    return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
        userID := UserIDFromContext(r.Context())
        role := RoleFromContext(r.Context())
        hostID := r.PathValue("hostID")

        host, err := h.store.GetHost(r.Context(), hostID)
        if err != nil { /* 404 处理 */ }

        // D-13: 归属校验，D-14: 管理员跳过
        if role != "admin" && host.UserID != userID {
            writeJSON(w, nethttp.StatusForbidden, map[string]string{"error": "forbidden"})
            return
        }

        // 继续返回详情...
    })
}
```

### Pattern 3: 用户级 apiFetch

**What:** 独立的 fetch 函数，base URL 为 `/v1/user`
**When to use:** 所有用户面板的数据请求

```typescript
// web/admin/src/lib/portal-api.ts
const PORTAL_API_BASE = "/v1/user";

export async function portalApiFetch<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const token = localStorage.getItem("admin_token");
  const res = await fetch(`${PORTAL_API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init?.headers,
    },
  });
  // 401 处理同 apiFetch
  if (res.status === 401) {
    localStorage.removeItem("admin_token");
    window.location.href = "/login";
    throw new ApiError(401, "Unauthorized");
  }
  if (!res.ok) {
    const body = await res.text();
    throw new ApiError(res.status, body);
  }
  return res.json();
}
```

### Pattern 4: 用户 React Query Hooks

**What:** 独立的 hooks 文件，使用 `portalApiFetch`
**When to use:** Portal 页面的数据获取

```typescript
// web/admin/src/hooks/use-portal-hosts.ts
export function useMyHosts() {
  return useQuery({
    queryKey: ["portal", "hosts"],
    queryFn: () => portalApiFetch<{ hosts: PortalHost[] }>("/hosts"),
  });
}

export function useMyHostDetail(hostId: string) {
  return useQuery({
    queryKey: ["portal", "hosts", hostId],
    queryFn: () => portalApiFetch<PortalHostDetail>(`/hosts/${hostId}`),
    enabled: !!hostId,
  });
}

export function useRebuildHost() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (hostId: string) =>
      portalApiFetch(`/hosts/${hostId}/rebuild`, { method: "POST" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["portal", "hosts"] });
    },
  });
}
```

### Anti-Patterns to Avoid
- **共用 admin apiFetch:** 用户端点的 base URL 是 `/v1/user`，不能复用 `/v1/admin` 的 apiFetch，否则所有请求都会 403
- **在中间件层做归属校验:** 归属校验需要数据库查询拿到 host.user_id，中间件层做会让中间件变得臃肿且不好测试。应在 handler 内部做
- **在前端硬编码角色跳过逻辑:** 前端不应实现 D-14 的管理员跳过逻辑，这完全是后端职责

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 主机列表查询 | 自写 SQL | `Repository.ListHostsByUserID` | 已存在，按 user_id 过滤 |
| 主机详情（含出口 IP） | 自写 JOIN 查询 | `Repository.GetHostDetail` | 已存在，含 bindings 数据 |
| 出口 IP 查询 | 自写绑定查询 | `Repository.GetEgressIPByHost` | 已存在，按 host_id 查 |
| 重建任务排队 | 自写任务系统 | `HostActionQueuer.QueueHostAction` | 已存在，接受 `ActionRebuildHost` |
| JWT 认证 | 自写 token 解析 | `AuthMiddleware` + `RequireRole` | 已存在，含 `UserIDFromContext` |
| 确认对话框 | 自写 modal | shadcn/ui `AlertDialog` | 已存在，管理员删除主机已使用 |
| 卡片组件 | 自写 card | shadcn/ui `Card` | 已存在于项目 |
| 状态标签 | 自写 badge | shadcn/ui `Badge` | 已存在，管理员主机列表已用 |

**Key insight:** Phase 12 的所有底层能力（数据查询、任务排队、认证授权、UI 组件）都已在 Phase 1-11 中实现。此阶段只需做组装和适配。

## Common Pitfalls

### Pitfall 1: 忘记注册 UserHostStore 依赖到 Dependencies struct
**What goes wrong:** `Dependencies` struct 缺少新接口字段导致编译通过但运行时 nil panic
**Why it happens:** `router.go` 的 `Dependencies` struct 需要显式添加新字段
**How to avoid:** 在 `Dependencies` struct 新增 `UserHosts UserHostStore` 字段，并在 `NewRouter` 中做 nil 检查（与 `AdminHosts` 同模式）
**Warning signs:** 端点注册了但请求返回 500

### Pitfall 2: 主机列表返回数据缺少出口 IP 地址
**What goes wrong:** D-12 要求卡片展示出口 IP，但 `ListHostsByUserID` 返回的 `Host` 结构体不含出口 IP
**Why it happens:** 现有 `ListHostsByUserID` 只查 hosts 表，不 JOIN egress_ips
**How to avoid:** 需要新增一个查询方法（如 `ListHostsWithEgressIPByUserID`），或在 handler 层循环调用 `GetEgressIPByHost`。推荐新增查询方法以避免 N+1 问题
**Warning signs:** 主机卡片上出口 IP 显示为空

### Pitfall 3: 重建按钮连续点击导致重复任务
**What goes wrong:** 用户连续点击重建按钮，排入多个 rebuild 任务
**Why it happens:** 前端没有防抖或按钮禁用
**How to avoid:** 确认对话框点击后立即禁用按钮，mutation 的 `isPending` 状态驱动 UI 禁用
**Warning signs:** 同一主机出现多个 rebuilding 任务

### Pitfall 4: Topbar 标题映射不匹配
**What goes wrong:** `/portal` 和 `/portal/hosts/xxx` 页面的 Topbar 显示"管理后台"而不是正确标题
**Why it happens:** 现有 `Topbar` 的 `pageTitles` map 只包含管理员路由
**How to avoid:** 扩展 `pageTitles` 或让 Portal 页面使用自己的 Topbar 变体
**Warning signs:** 用户看到"管理后台"字样

### Pitfall 5: 出口 IP 详情泄露敏感配置
**What goes wrong:** 用户端点返回了 WireGuard 密钥、proxy_config 等敏感字段
**Why it happens:** 直接复用 `GetHostDetail` 或 `GetEgressIPByHost` 返回完整 EgressIP 结构体
**How to avoid:** 用户端点的响应需要过滤敏感字段，只返回 `ip_address`、`tunnel_type`、`label`
**Warning signs:** 前端网络面板可见 wg_private_key 等字段

## Code Examples

### 后端：router.go 注册用户端点

```go
// 在 NewRouter 函数中，admin 路由块之后
userGuard := func(h nethttp.Handler) nethttp.Handler {
    return authMw(RequireRole("user", "admin")(h))
}

if deps.UserHosts != nil {
    userHostsHandler := NewUserHostsHandler(deps.Logger, deps.UserHosts, deps.HostActions, deps.EventRecorder)
    mux.Handle("GET /v1/user/hosts", userGuard(userHostsHandler.List()))
    mux.Handle("GET /v1/user/hosts/{hostID}", userGuard(userHostsHandler.Get()))
    mux.Handle("POST /v1/user/hosts/{hostID}/rebuild", userGuard(userHostsHandler.Rebuild()))
}
```

### 后端：用户主机列表（含出口 IP 摘要）

```go
// 响应格式 — 过滤掉敏感字段
type UserHostSummary struct {
    ID        string    `json:"id"`
    Hostname  string    `json:"hostname"`
    Status    string    `json:"status"`
    EgressIP  string    `json:"egress_ip"`      // 仅 IP 地址
    CreatedAt time.Time `json:"created_at"`
}
```

### 前端：TanStack Router 文件路由

```typescript
// web/admin/src/routes/_portal/index.tsx
// createFileRoute("/_portal/portal") 已有骨架，替换为主机列表
export const Route = createFileRoute("/_portal/portal")({
  component: PortalHostList,
});
```

```typescript
// web/admin/src/routes/_portal/hosts/$hostId.tsx — 新建
export const Route = createFileRoute("/_portal/portal/hosts/$hostId")({
  component: PortalHostDetail,
});
```

### 前端：重建确认对话框

```typescript
// 复用 shadcn/ui AlertDialog（与管理员删除主机确认框模式相同）
<AlertDialog open={rebuildOpen} onOpenChange={setRebuildOpen}>
  <AlertDialogContent>
    <AlertDialogHeader>
      <AlertDialogTitle>确认重建主机？</AlertDialogTitle>
      <AlertDialogDescription>
        重建将重置容器环境，home 目录数据保留。重建过程中主机将暂时不可访问。
      </AlertDialogDescription>
    </AlertDialogHeader>
    <AlertDialogFooter>
      <AlertDialogCancel>取消</AlertDialogCancel>
      <AlertDialogAction
        disabled={rebuildMutation.isPending}
        onClick={() => rebuildMutation.mutate(hostId)}
      >
        {rebuildMutation.isPending ? "重建中..." : "确认重建"}
      </AlertDialogAction>
    </AlertDialogFooter>
  </AlertDialogContent>
</AlertDialog>
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 无用户面板 | 用户自助面板 + 角色路由守卫 | Phase 11 (认证基础设施) | Phase 12 可直接构建在此之上 |

**已就位的基础设施:**
- `AuthMiddleware` + `RequireRole` + `UserIDFromContext`: Phase 11 产出
- `_portal.tsx` 布局 + `_dashboard.tsx` 角色路由守卫: Phase 11 产出
- `ListHostsByUserID` / `GetHost` / `GetHostDetail` / `GetEgressIPByHost` / `QueueHostAction`: Phase 1-10 产出

## Open Questions

1. **主机列表含出口 IP 的查询方式**
   - What we know: `ListHostsByUserID` 不含出口 IP，`GetEgressIPByHost` 按单个 host 查
   - What's unclear: 是否需要新增一个 JOIN 查询，还是在 handler 层遍历调用
   - Recommendation: 新增 `ListHostsWithEgressIPByUserID` 查询方法，一次 SQL 返回主机+出口 IP 地址。如果用户主机数量少（通常 1-3 台），handler 层遍历也可接受

2. **Topbar 组件适配**
   - What we know: 当前 Topbar 硬编码了管理员路由的标题映射
   - What's unclear: 是扩展现有 Topbar 还是为 Portal 创建独立 Topbar
   - Recommendation: 扩展现有 Topbar 的 `pageTitles` 映射，增加 `/portal` 系列路由的标题。将右侧"管理员"文字改为根据角色动态显示

## Project Constraints (from CLAUDE.md)

- 所有面向用户的回复、计划、状态更新使用中文
- Go handler 使用 `net/http` 标准库，不引入额外 HTTP 框架
- 数据库使用 pgx v5 手写 SQL
- 前端使用 React 19.2 + Vite + TanStack Router + React Query
- 单宿主机部署，不考虑分布式
- 用户容器网络必须走全隧道出网（此 phase 不涉及，但 API 响应不能泄露隧道配置详情给用户）

## Sources

### Primary (HIGH confidence)
- 项目源码 `internal/controlplane/http/admin_hosts.go` -- 管理员 handler 实现模式
- 项目源码 `internal/controlplane/http/router.go` -- 路由注册和中间件链模式
- 项目源码 `internal/controlplane/http/auth_middleware.go` -- 认证中间件和角色守卫
- 项目源码 `internal/store/repository/queries.go` -- 现有数据查询方法
- 项目源码 `internal/store/repository/models.go` -- 数据模型定义
- 项目源码 `web/admin/src/hooks/use-hosts.ts` -- 前端 React Query hooks 模式
- 项目源码 `web/admin/src/lib/api.ts` -- apiFetch 封装模式
- 项目源码 `web/admin/src/routes/_portal.tsx` -- Portal 布局
- 项目源码 `web/admin/src/routes/_dashboard.tsx` -- 角色路由守卫参考
- 项目源码 `web/admin/src/routes/_dashboard/hosts/index.tsx` -- 管理员主机列表页面参考

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - 全部使用项目已有的技术栈和库
- Architecture: HIGH - 严格遵循已建立的 handler/hook/路由模式
- Pitfalls: HIGH - 基于代码审查发现的具体问题

**Research date:** 2026-03-28
**Valid until:** 2026-04-28 (项目代码模式稳定)
