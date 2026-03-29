---
phase: 12-api
verified: 2026-03-29T07:16:20Z
status: human_needed
score: 4/4 must-haves verified
human_verification:
  - test: "使用普通用户账号登录，验证 /portal 主机卡片列表展示正确"
    expected: "看到自己的主机卡片，含主机名、状态 Badge、出口 IP、创建时间"
    why_human: "需要实际浏览器渲染和后端数据联调才能验证视觉和交互效果"
  - test: "点击主机卡片进入详情页，验证出口 IP 和隧道类型展示"
    expected: "详情页显示主机基本信息、出口 IP 绑定列表（含隧道类型标签）"
    why_human: "需要真实数据和浏览器渲染验证"
  - test: "点击重建按钮，验证确认对话框和重建状态反馈"
    expected: "弹出 AlertDialog 确认框，确认后显示 toast 成功提示，状态自动轮询"
    why_human: "需要实际后端任务队列配合验证交互流程"
  - test: "检查浏览器网络面板，确认 API 响应不含 WireGuard 密钥等敏感字段"
    expected: "/v1/user/hosts/{id} 响应只含 ip_address 和 tunnel_type"
    why_human: "需要真实 API 响应验证字段过滤"
  - test: "使用管理员账号访问 /portal，验证面板与管理后台共存"
    expected: "管理员可同时访问 /portal 和管理后台，Topbar 正确显示角色"
    why_human: "需要多角色登录验证路由守卫隔离"
---

# Phase 12: 用户自助 API 与前端路由 Verification Report

**Phase Goal:** 用户可以在自助面板中查看自己的主机状态、出口 IP 并执行主机重建
**Verified:** 2026-03-29T07:16:20Z
**Status:** human_needed
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | 用户登录后看到自己的主机列表，包含运行状态和基本信息 | VERIFIED | `useMyHosts` hook 调用 `/v1/user/hosts`；后端 `ListHostsWithEgressByUserID` 按 user_id 过滤；前端 `PortalHostList` 渲染卡片含 hostname、StatusBadge、egress_ip、created_at |
| 2 | 用户可以查看每台主机绑定的出口 IP 信息 | VERIFIED | 后端 `UserHostDetail` 响应含 `egress_bindings`（ip_address + tunnel_type）；前端详情页遍历 bindings 渲染 Globe 图标 + IP + 隧道类型标签 |
| 3 | 用户可以对自己的主机触发重建操作，重建过程有状态反馈 | VERIFIED | 后端 `Rebuild()` 调用 `QueueHostAction` + 记录事件 + 返回 202；前端 `useRebuildHost` mutation + `AlertDialog` 确认 + `toast.success` + `refetchInterval: 3000` 轮询 |
| 4 | 用户面板与管理员面板共存于同一 React 应用，通过角色路由守卫隔离 | VERIFIED | `userGuard` 使用 `RequireRole("user", "admin")`；`_portal` layout route 与 `_dashboard` layout route 并存于 routeTree.gen.ts；Topbar 动态显示角色标签 |

**Score:** 4/4 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/controlplane/http/user_hosts.go` | UserHostsHandler with List, Get, Rebuild | VERIFIED | 175 行，含完整 3 个 handler 方法、归属校验、敏感字段过滤 |
| `internal/controlplane/http/auth_middleware.go` | AuthMiddleware, RequireRole, UserIDFromContext, RoleFromContext | VERIFIED | 107 行，JWT 解析 + 角色校验 + 多来源 token 提取 |
| `internal/store/repository/queries.go` | ListHostsWithEgressByUserID query | VERIFIED | LEFT JOIN 查询避免 N+1，扫描到 UserHostSummary |
| `internal/store/repository/models.go` | UserHostSummary, UserHostDetail, UserEgressBinding | VERIFIED | 3 个类型定义完整，JSON tag 正确 |
| `internal/controlplane/http/router.go` | UserHosts dependency + user routes | VERIFIED | `UserHosts UserHostStore` 字段 + 3 条路由注册 + `userGuard` 中间件 |
| `internal/controlplane/app/app.go` | UserHosts: repo injection | VERIFIED | `UserHosts: repo,` 注入完成 |
| `web/admin/src/lib/portal-api.ts` | portalApiFetch with /v1/user base URL | VERIFIED | 35 行，独立 API 客户端，401 重定向，复用 ApiError |
| `web/admin/src/hooks/use-portal-hosts.ts` | useMyHosts, useMyHostDetail, useRebuildHost | VERIFIED | 62 行，3 个 hooks + 3 个 TS 接口，refetchInterval 支持函数形式 |
| `web/admin/src/routes/_portal/portal/index.tsx` | Portal host list page with cards | VERIFIED | 118 行，卡片网格布局 + 状态 Badge（4 色）+ Globe 图标 + 骨架屏 + 空状态 |
| `web/admin/src/routes/_portal/portal/hosts/$hostId.tsx` | Portal host detail with rebuild | VERIFIED | 220 行，详情 + 出口 IP 绑定列表 + AlertDialog 重建确认 + 轮询 + toast |
| `web/admin/src/components/layout/topbar.tsx` | Portal title + dynamic role | VERIFIED | `/portal` 映射到 "我的面板"，动态路由匹配 "主机详情"，`getRole()` 动态角色 |
| `web/admin/src/routeTree.gen.ts` | Portal routes registered | VERIFIED | `/_portal/portal/` 和 `/_portal/portal/hosts/$hostId` 均已注册 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| router.go | user_hosts.go | userGuard wrapping UserHostsHandler on /v1/user/ | WIRED | 3 条路由：GET /v1/user/hosts, GET /v1/user/hosts/{hostID}, POST /v1/user/hosts/{hostID}/rebuild |
| app.go | router.go | UserHosts: repo | WIRED | `UserHosts: repo,` 在 Dependencies 构造中 |
| use-portal-hosts.ts | portal-api.ts | import portalApiFetch | WIRED | `portalApiFetch<{ hosts: PortalHost[] }>("/hosts")` 等 3 处调用 |
| _portal/portal/index.tsx | use-portal-hosts.ts | useMyHosts hook | WIRED | `const { data, isLoading } = useMyHosts()` |
| _portal/portal/hosts/$hostId.tsx | use-portal-hosts.ts | useMyHostDetail + useRebuildHost | WIRED | 两个 hooks 均被导入和使用 |
| routeTree.gen.ts | _portal/portal/ routes | auto-generated imports | WIRED | `PortalPortalIndexRoute` 和 `PortalPortalHostsHostIdRoute` 均已注册 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| _portal/portal/index.tsx | `data?.hosts` | `useMyHosts()` -> `portalApiFetch("/hosts")` -> `GET /v1/user/hosts` -> `ListHostsWithEgressByUserID` SQL query | Yes -- LEFT JOIN on hosts/bindings/egress_ips with WHERE user_id | FLOWING |
| _portal/portal/hosts/$hostId.tsx | `host` (PortalHostDetail) | `useMyHostDetail(hostId)` -> `portalApiFetch("/hosts/${hostId}")` -> `GET /v1/user/hosts/{hostID}` -> `GetHostDetail` SQL query | Yes -- DB query with sensitive field filtering in handler | FLOWING |
| _portal/portal/hosts/$hostId.tsx | rebuild mutation | `useRebuildHost()` -> `portalApiFetch("/hosts/${hostId}/rebuild", POST)` -> `QueueHostAction` | Yes -- task queue + event recording | FLOWING |

### Behavioral Spot-Checks

Step 7b: SKIPPED (Go binary not compiled in environment; no running server available for API checks)

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| PANEL-01 | 12-01, 12-02 | 用户可在自助面板查看自己的主机列表和运行状态 | SATISFIED | Backend: `ListHostsWithEgressByUserID` returns host list with status; Frontend: `PortalHostList` renders cards with StatusBadge |
| PANEL-02 | 12-01, 12-02 | 用户可在自助面板查看自己主机绑定的出口 IP | SATISFIED | Backend: `UserHostDetail.EgressBindings` with ip_address/tunnel_type; Frontend: detail page renders bindings with Globe icon |
| PANEL-03 | 12-01, 12-02 | 用户可在自助面板触发主机重建操作 | SATISFIED | Backend: `Rebuild()` handler with ownership check + task queue; Frontend: AlertDialog confirm + toast feedback + polling |

No orphaned requirements found.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | - | - | - | - |

No TODO/FIXME/placeholder comments, no empty implementations, no hardcoded empty data, no console.log-only handlers found in any phase 12 files.

### Human Verification Required

### 1. Portal 主机列表视觉和交互验证

**Test:** 使用普通用户账号登录，访问 /portal，验证主机卡片列表
**Expected:** 卡片网格展示主机名、状态 Badge（4 种颜色）、出口 IP（Globe 图标）、创建时间；空状态显示"暂无主机"
**Why human:** 需要实际浏览器渲染和后端数据联调

### 2. 主机详情页出口 IP 展示

**Test:** 点击卡片进入 /portal/hosts/{id}，验证详情页信息
**Expected:** 主机基本信息（hostname、timezone、时间）+ 出口 IP 绑定列表（IP + 隧道类型标签）
**Why human:** 需要真实数据渲染验证

### 3. 重建流程完整验证

**Test:** 点击重建按钮 -> 确认对话框 -> 确认 -> 观察状态
**Expected:** AlertDialog 弹出 -> 确认后 toast "重建任务已提交" -> 状态变为 rebuilding -> 3 秒轮询 -> 恢复 running
**Why human:** 需要后端任务队列实际执行

### 4. 敏感字段泄露检查

**Test:** 在浏览器 Network 面板检查 /v1/user/hosts/{id} 响应
**Expected:** 响应只含 id、hostname、status、timezone、timestamps、egress_bindings（ip_address + tunnel_type），不含 WireGuard 密钥
**Why human:** 需要实际 API 响应验证

### 5. 多角色路由隔离

**Test:** 分别用 admin 和 user 角色登录，验证路由守卫
**Expected:** admin 可访问 /portal 和管理后台；user 只能访问 /portal；Topbar 角色标签正确
**Why human:** 需要多角色登录验证

### Gaps Summary

无代码层面的 gap。所有后端 API 端点、前端页面组件、数据 hooks、路由注册、依赖注入均已实现且正确连接。敏感字段过滤和归属校验逻辑完整。

唯一待确认的是人工验证项：实际浏览器渲染效果、端到端交互流程、多角色路由隔离行为。这些无法通过静态代码分析验证。

---

_Verified: 2026-03-29T07:16:20Z_
_Verifier: Claude (gsd-verifier)_
