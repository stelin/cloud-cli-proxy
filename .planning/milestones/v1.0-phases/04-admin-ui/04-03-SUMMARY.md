---
phase: 04-admin-ui
plan: 03
subsystem: api, ui
tags: [go, react, tanstack-query, shadcn, crud, egress-ip, binding, host-lifecycle]

requires:
  - phase: 04-admin-ui plan 01
    provides: Admin auth, dashboard stats API, React+Vite scaffold
  - phase: 04-admin-ui plan 02
    provides: User CRUD API and frontend, sidebar navigation
provides:
  - Egress IP CRUD API (list/create/update/delete) with IP conflict 409
  - Binding API (bind/unbind) with running-host protection (409 Conflict)
  - Host management API (list with username, detail with bindings)
  - Host lifecycle admin API (start/stop/rebuild via QueueHostAction)
  - Admin tasks list API
  - Egress IP list page with drawer form (create/edit)
  - Host list page with status badges
  - Host detail page with binding manager and lifecycle actions
  - Rebuild dialog with preserve/factory mode
  - Task list page with 5s auto-refresh
affects: [05-hardening, admin-ui-tests]

tech-stack:
  added: [shadcn-sheet, shadcn-select, shadcn-tooltip]
  patterns: [admin-store-interface, running-host-guard, drawer-form]

key-files:
  created:
    - internal/controlplane/http/admin_egress_ips.go
    - internal/controlplane/http/admin_bindings.go
    - internal/controlplane/http/admin_hosts.go
    - web/admin/src/hooks/use-egress-ips.ts
    - web/admin/src/hooks/use-hosts.ts
    - web/admin/src/hooks/use-tasks.ts
    - web/admin/src/routes/_dashboard/egress-ips/index.tsx
    - web/admin/src/routes/_dashboard/hosts/index.tsx
    - web/admin/src/routes/_dashboard/hosts/$hostId.tsx
    - web/admin/src/routes/_dashboard/tasks/index.tsx
    - web/admin/src/components/egress-ips/egress-ip-drawer.tsx
    - web/admin/src/components/hosts/binding-manager.tsx
    - web/admin/src/components/hosts/host-lifecycle-actions.tsx
    - web/admin/src/components/hosts/rebuild-dialog.tsx
  modified:
    - internal/store/repository/models.go
    - internal/store/repository/queries.go
    - internal/controlplane/http/router.go
    - internal/controlplane/app/app.go
    - web/admin/src/lib/api.ts
    - web/admin/src/main.tsx

key-decisions:
  - "AdminEgressIPStore/AdminBindingStore/AdminHostStore 接口与 Plan 01/02 的 AdminUserStore 模式一致，repo 直接实现"
  - "运行中主机保护检查在 handler 层完成（GetHost -> 判断 status），保持与 Phase 2 D-03 一致"
  - "主机生命周期操作复用现有 HostActionQueuer 接口，不新增任务入队路径"
  - "绑定管理放在主机详情页而非单独页面，符合 D-12 操作上下文"

patterns-established:
  - "Admin store interface pattern: 每个资源域定义独立 store 接口，handler 只依赖接口"
  - "Running-host guard: 绑定/解绑操作必须先检查 host.Status != running"
  - "Drawer form pattern: 列表页内的 create/edit 使用 Sheet 抽屉，不跳转页面"

requirements-completed: [ADMN-01, ADMN-02, LIFE-01, LIFE-02, LIFE-03, ADMN-03]

duration: 7min
completed: 2026-03-27
---

# Phase 04 Plan 03: 出口 IP、绑定、主机管理与任务列表 Summary

**出口 IP CRUD + 绑定管理（含运行中主机保护）+ 主机启停重建 + 任务列表的完整前后端实现**

## Performance

- **Duration:** 7 min
- **Started:** 2026-03-27T11:58:06Z
- **Completed:** 2026-03-27T12:04:39Z
- **Tasks:** 2
- **Files modified:** 24

## Accomplishments
- 后端完整实现出口 IP CRUD API，含 IP 唯一冲突检测（409）和已绑定删除保护（FK RESTRICT → 409）
- 绑定/解绑 API 在 handler 层强制执行运行中主机保护检查（status == "running" → 409 Conflict）
- 主机管理 API 支持带用户名的列表查询和包含绑定详情的主机详情
- 前端出口 IP 管理页使用右侧抽屉表单（Sheet），支持 WireGuard 配置字段
- 主机详情页集成绑定选择器和生命周期操作按钮（启动/停止/重建）
- 重建对话框提供 preserve/factory 双模式选择，factory 模式显示红色警告
- 任务列表页每 5 秒自动刷新，展示异步操作结果

## Task Commits

Each task was committed atomically:

1. **Task 1: Go 后端 — 出口 IP CRUD + 绑定 API + 主机管理 API + Repository 扩展** - `59e651b` (feat)
2. **Task 2: 前端 — 出口 IP 列表与抽屉 + 主机列表与详情 + 绑定管理 + 任务列表** - `ab32815` (feat)

## Files Created/Modified

### Backend (Go)
- `internal/store/repository/models.go` - 新增 CreateEgressIPParams, UpdateEgressIPParams, HostDetail, BindingWithIP, HostWithUsername 类型
- `internal/store/repository/queries.go` - 新增 ListEgressIPs, CreateEgressIP, UpdateEgressIP, DeleteEgressIP, BindEgressIPToHost, UnbindEgressIPFromHost, GetBindingHostID, GetHostDetail, ListHostsWithUsername 方法
- `internal/controlplane/http/admin_egress_ips.go` - 出口 IP CRUD handlers（List/Get/Create/Update/Delete）
- `internal/controlplane/http/admin_bindings.go` - 绑定/解绑 handlers（含运行中主机保护检查）
- `internal/controlplane/http/admin_hosts.go` - 主机管理 handlers（List/Get/Start/Stop/Rebuild）
- `internal/controlplane/http/router.go` - 注册新 admin 路由
- `internal/controlplane/app/app.go` - 注入新依赖

### Frontend (React)
- `web/admin/src/hooks/use-egress-ips.ts` - 出口 IP 查询和变更 hooks
- `web/admin/src/hooks/use-hosts.ts` - 主机查询、生命周期、绑定 hooks
- `web/admin/src/hooks/use-tasks.ts` - 任务列表 hook（5s 自动刷新）
- `web/admin/src/routes/_dashboard/egress-ips/index.tsx` - 出口 IP 列表页
- `web/admin/src/routes/_dashboard/hosts/index.tsx` - 主机列表页
- `web/admin/src/routes/_dashboard/hosts/$hostId.tsx` - 主机详情页
- `web/admin/src/routes/_dashboard/tasks/index.tsx` - 任务列表页
- `web/admin/src/components/egress-ips/egress-ip-drawer.tsx` - 出口 IP 创建/编辑抽屉
- `web/admin/src/components/hosts/binding-manager.tsx` - 绑定管理器（运行中保护）
- `web/admin/src/components/hosts/host-lifecycle-actions.tsx` - 生命周期操作按钮
- `web/admin/src/components/hosts/rebuild-dialog.tsx` - 重建模式选择对话框
- `web/admin/src/components/ui/sheet.tsx` - shadcn Sheet 组件
- `web/admin/src/components/ui/select.tsx` - shadcn Select 组件
- `web/admin/src/components/ui/tooltip.tsx` - shadcn Tooltip 组件
- `web/admin/src/lib/api.ts` - 修复 204 No Content 响应处理
- `web/admin/src/main.tsx` - 添加 TooltipProvider 包裹

## Decisions Made
- AdminEgressIPStore/AdminBindingStore/AdminHostStore 接口与 Plan 01/02 的 AdminUserStore 模式一致，repo 直接实现全部接口
- 运行中主机保护检查在 handler 层完成（GetHost → 判断 status），保持与 Phase 2 D-03 一致
- 主机生命周期操作复用现有 HostActionQueuer 接口，不新增任务入队路径
- 绑定管理放在主机详情页而非单独页面，符合 D-12 操作上下文

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 修复 apiFetch 对 204 No Content 的处理**
- **Found during:** Task 2（前端 hooks 创建）
- **Issue:** `apiFetch` 对所有成功响应调用 `res.json()`，但 DELETE 端点返回 204 No Content 无 body，导致 JSON 解析失败
- **Fix:** 在 `res.json()` 前增加 `if (res.status === 204) return undefined as T` 判断
- **Files modified:** `web/admin/src/lib/api.ts`
- **Verification:** `npx vite build` 通过，DELETE mutation 不会因 JSON 解析报错
- **Committed in:** `ab32815`（Task 2 commit）

**2. [Rule 3 - Blocking] shadcn 组件安装到错误路径**
- **Found during:** Task 2（前端构建）
- **Issue:** `npx shadcn add` 将 sheet/select/tooltip 组件安装到 `web/admin/@/components/ui/` 而非 `web/admin/src/components/ui/`
- **Fix:** 手动移动文件到正确路径并清理错误目录
- **Files modified:** `web/admin/src/components/ui/sheet.tsx`, `select.tsx`, `tooltip.tsx`
- **Verification:** `npx vite build` 构建通过
- **Committed in:** `ab32815`（Task 2 commit）

**3. [Rule 2 - Missing Critical] 添加 TooltipProvider 包裹**
- **Found during:** Task 2（Tooltip 组件集成）
- **Issue:** shadcn Tooltip 组件需要 TooltipProvider 包裹才能正常工作
- **Fix:** 在 `main.tsx` 中用 TooltipProvider 包裹 RouterProvider
- **Files modified:** `web/admin/src/main.tsx`
- **Verification:** `npx vite build` 构建通过
- **Committed in:** `ab32815`（Task 2 commit）

---

**Total deviations:** 3 auto-fixed (1 missing critical, 2 blocking)
**Impact on plan:** All auto-fixes necessary for correctness. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 04 全部 3 个计划完成，后台管理界面功能齐全
- 管理员可完整管理用户、出口 IP、绑定、主机生命周期和任务状态
- 准备进入 Phase 05 硬化阶段

---
*Phase: 04-admin-ui*
*Completed: 2026-03-27*
