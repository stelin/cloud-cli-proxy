---
phase: 04-admin-ui
plan: 01
subsystem: ui, auth, api
tags: [jwt, react, vite, tanstack-router, tanstack-query, tailwindcss, shadcn, admin]

requires:
  - phase: 01-control-plane-core
    provides: "PostgreSQL schema (users, hosts, egress_ips tables) and Repository pattern"
  - phase: 03-bootstrap-ssh
    provides: "User model with password_hash and status fields"
provides:
  - "JWT admin login API (POST /v1/admin/login)"
  - "JWT auth middleware for admin routes"
  - "Dashboard stats API (GET /v1/admin/dashboard/stats)"
  - "DashboardStats and AdminConfig repository types"
  - "React SPA scaffold with TanStack Router file-based routing"
  - "Admin login page with form validation"
  - "Dashboard layout with sidebar (5 nav items) and topbar"
  - "Dashboard overview page with 3 stats cards"
  - "apiFetch wrapper with automatic Bearer token injection"
  - "Auth utilities (isAuthenticated, setToken, clearToken, logout)"
affects: [04-admin-ui, 05-expiry-governance]

tech-stack:
  added: [golang-jwt/jwt/v5, react 19.2, vite 8, tanstack-router, tanstack-query, tailwindcss 4, shadcn/ui, react-hook-form, zod, sonner, lucide-react]
  patterns: [JWT HS256 auth, admin config via env vars with graceful disable, apiFetch with automatic 401 redirect, file-based routing, beforeLoad auth guard]

key-files:
  created:
    - internal/controlplane/http/admin_auth.go
    - internal/controlplane/http/admin_dashboard.go
    - web/admin/src/lib/api.ts
    - web/admin/src/lib/auth.ts
    - web/admin/src/routes/login.tsx
    - web/admin/src/routes/_dashboard.tsx
    - web/admin/src/routes/_dashboard/index.tsx
    - web/admin/src/components/layout/sidebar.tsx
    - web/admin/src/components/layout/topbar.tsx
  modified:
    - internal/store/repository/models.go
    - internal/store/repository/queries.go
    - internal/controlplane/http/router.go
    - internal/controlplane/app/app.go
    - cmd/control-plane/main.go
    - go.mod
    - go.sum

key-decisions:
  - "Admin API 通过 ADMIN_JWT_SECRET 环境变量启用，未设置时自动禁用，不影响现有服务"
  - "管理员凭证使用 hmac.Equal 常量时间比对，防止时序攻击"
  - "前端使用 TanStack Router 文件路由 + TanStack Query 数据获取，统一管理路由和服务端状态"
  - "401 响应自动清除 token 并重定向到登录页，apiFetch 封装统一处理"

patterns-established:
  - "Admin route pattern: POST /v1/admin/login (public) + AdminAuthMiddleware protects /v1/admin/* (private)"
  - "Frontend auth guard: beforeLoad + isAuthenticated() + redirect to /login"
  - "Stats card pattern: useQuery → 3-card grid with loading skeleton"

requirements-completed: [ADMN-03]

duration: 3min
completed: 2026-03-27
---

# Phase 04 Plan 01: Admin UI 骨架 Summary

**Go 端 JWT 登录 API + 认证中间件 + 仪表板统计 API，React 19 SPA 脚手架含登录页、5 项侧边栏导航和 3 卡片仪表板概览**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-27T11:21:00Z
- **Completed:** 2026-03-27T11:43:00Z
- **Tasks:** 2
- **Files modified:** 34

## Accomplishments
- JWT 管理员登录 API 和 Bearer token 认证中间件，HS256 签名、24h 过期
- 仪表板统计 API 返回活跃用户数、运行中主机数和可用出口 IP 数
- React 19 + Vite 8 + TypeScript SPA 完整工程脚手架
- 登录页面带 zod 表单校验，仪表板概览页带 3 张统计卡片
- 侧边栏包含 5 个导航项（仪表板、用户管理、出口 IP、主机管理、任务列表），当前路由高亮

## Task Commits

Each task was committed atomically:

1. **Task 1: Go 后端 — JWT 认证登录 API + 认证中间件 + 仪表板统计 API + 配置连线** - `f267855` (feat)
2. **Task 2: 前端 — React SPA 工程脚手架 + 登录页 + 侧边栏布局 + 仪表板概览** - `3b55392` (feat)

## Files Created/Modified
- `internal/controlplane/http/admin_auth.go` - JWT 登录 handler 和认证中间件
- `internal/controlplane/http/admin_dashboard.go` - 仪表板统计 handler 和接口定义
- `internal/store/repository/models.go` - 新增 DashboardStats 和 AdminConfig 类型
- `internal/store/repository/queries.go` - 新增 GetDashboardStats 查询
- `internal/controlplane/http/router.go` - 注册 admin 路由（login + dashboard/stats）
- `internal/controlplane/app/app.go` - 连线 AdminConfig 到 Dependencies
- `cmd/control-plane/main.go` - 读取 ADMIN_* 环境变量
- `web/admin/package.json` - React 19 + 全部依赖声明
- `web/admin/vite.config.ts` - TanStack Router + Tailwind + API proxy
- `web/admin/src/lib/api.ts` - apiFetch 封装，自动注入 Bearer token
- `web/admin/src/lib/auth.ts` - token 管理和 isAuthenticated 判断
- `web/admin/src/routes/login.tsx` - 管理员登录页面
- `web/admin/src/routes/_dashboard.tsx` - 受保护布局路由 + beforeLoad auth guard
- `web/admin/src/routes/_dashboard/index.tsx` - 仪表板概览，3 张统计卡片
- `web/admin/src/components/layout/sidebar.tsx` - 5 项导航侧边栏
- `web/admin/src/components/layout/topbar.tsx` - 顶栏 + 页面标题映射
- `web/admin/src/main.tsx` - 应用入口，QueryClient + Router 初始化

## Decisions Made
- Admin API 通过 ADMIN_JWT_SECRET 环境变量启用，未设置时优雅降级（仅 warn 日志），不影响现有 bootstrap 流程
- 管理员凭证使用 hmac.Equal 常量时间比对，防止时序攻击
- 前端使用 TanStack Router 文件路由 + TanStack Query 数据获取
- 登录 API 直接用 fetch（不走 apiFetch），因为登录时还没有 token
- 401 响应在 apiFetch 中统一处理：清除 token + 重定向到 /login

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 后台骨架完整：登录→仪表板概览链路已通
- 侧边栏 5 个导航项中，仅"仪表板"有实际页面，其余 4 项（用户管理、出口 IP、主机管理、任务列表）等待后续计划实现
- Admin API 受 JWT 保护，后续 CRUD 页面可直接复用 apiFetch + AdminAuthMiddleware

## Self-Check: PASSED

- All 17 key files verified present
- Commit f267855 verified in git log
- Commit 3b55392 verified in git log
- `go build ./cmd/control-plane` exit 0
- `go vet ./internal/...` exit 0
- `npx vite build` exit 0

---
*Phase: 04-admin-ui*
*Completed: 2026-03-27*
