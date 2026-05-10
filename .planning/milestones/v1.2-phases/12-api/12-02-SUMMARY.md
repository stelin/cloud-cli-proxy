---
phase: 12-api
plan: 02
subsystem: ui
tags: [react, tanstack-router, react-query, portal, shadcn-ui]

requires:
  - phase: 12-api-01
    provides: "Backend user host API endpoints (/v1/user/hosts)"
provides:
  - "portalApiFetch 客户端，使用 /v1/user 前缀"
  - "useMyHosts / useMyHostDetail / useRebuildHost React Query hooks"
  - "Portal 主机列表页（卡片网格布局）"
  - "Portal 主机详情页（含出口 IP、重建对话框、状态轮询）"
  - "Topbar portal 路由标题适配和动态角色显示"
affects: [12-api-03, portal-ui]

tech-stack:
  added: []
  patterns: ["portalApiFetch 独立于 admin apiFetch，使用 /v1/user 前缀", "portal 路由文件放在 _portal/portal/ 目录下避免 TanStack Router 路径冲突"]

key-files:
  created:
    - web/admin/src/lib/portal-api.ts
    - web/admin/src/hooks/use-portal-hosts.ts
    - web/admin/src/routes/_portal/portal/index.tsx
    - web/admin/src/routes/_portal/portal/hosts/$hostId.tsx
  modified:
    - web/admin/src/components/layout/topbar.tsx
    - web/admin/src/routeTree.gen.ts

key-decisions:
  - "Portal 路由文件从 _portal/index.tsx 迁移到 _portal/portal/index.tsx，避免与 _dashboard 路由路径冲突"
  - "refetchInterval 使用函数形式基于 query.state.data.status 判断是否轮询"

patterns-established:
  - "Portal API 客户端模式：portalApiFetch 复用 ApiError 但独立 base URL"
  - "Portal hooks 命名约定：useMyXxx 区别于 admin 的 useXxx"

requirements-completed: [PANEL-01, PANEL-02, PANEL-03]

duration: 5min
completed: 2026-03-29
---

# Phase 12 Plan 02: 用户自助面板前端 Summary

**Portal 主机列表卡片页 + 主机详情页（含出口 IP/隧道类型展示和重建确认对话框）+ Topbar 路由标题和角色标签适配**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-29T07:08:02Z
- **Completed:** 2026-03-29T07:13:10Z
- **Tasks:** 3 (2 auto + 1 checkpoint auto-approved)
- **Files modified:** 6

## Accomplishments
- Portal API 客户端 (portalApiFetch) 使用独立的 /v1/user 前缀，与 admin API 完全隔离
- 三个 React Query hooks：useMyHosts（列表）、useMyHostDetail（详情+轮询）、useRebuildHost（重建 mutation）
- 主机列表页以卡片网格展示，包含状态 Badge（4 种颜色）、Globe 图标的出口 IP、创建时间
- 主机详情页展示基本信息、出口 IP 绑定列表（含隧道类型），重建按钮带 AlertDialog 确认
- 重建中状态自动 3 秒轮询，运行中停止轮询
- Topbar 在 /portal 路由下显示"我的面板"，动态路由显示"主机详情"，角色标签基于 JWT 解析

## Task Commits

Each task was committed atomically:

1. **Task 1: Portal API 客户端 + React Query Hooks** - `a1f8abc` (feat)
2. **Task 2: 主机列表页 + 主机详情页 + 重建对话框 + Topbar 适配** - `5977245` (feat)
3. **Task 3: 人工验证** - auto-approved (checkpoint)

## Files Created/Modified
- `web/admin/src/lib/portal-api.ts` - Portal API 客户端，/v1/user 前缀，401 重定向
- `web/admin/src/hooks/use-portal-hosts.ts` - PortalHost/PortalHostDetail 类型 + 三个 hooks
- `web/admin/src/routes/_portal/portal/index.tsx` - 主机列表页，卡片网格布局
- `web/admin/src/routes/_portal/portal/hosts/$hostId.tsx` - 主机详情页，重建对话框，状态轮询
- `web/admin/src/components/layout/topbar.tsx` - 新增 /portal 标题映射和动态角色显示
- `web/admin/src/routeTree.gen.ts` - 路由树自动生成更新

## Decisions Made
- Portal 路由文件从 `_portal/index.tsx` 迁移到 `_portal/portal/index.tsx`，因为 TanStack Router 文件路由系统中 `_portal/index.tsx` 与 `_dashboard/index.tsx` 会解析为相同的 `/` 路径产生冲突
- `useMyHostDetail` 的 `refetchInterval` 使用函数形式 `(query) => ...` 基于当前查询数据状态判断是否开启轮询

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Portal 路由文件路径冲突修复**
- **Found during:** Task 2
- **Issue:** `_portal/index.tsx` 与 `_dashboard/index.tsx` 在 TanStack Router 中解析为同一路径 `/`，导致路由树生成失败
- **Fix:** 将文件从 `_portal/index.tsx` 移动到 `_portal/portal/index.tsx`，`_portal/hosts/$hostId.tsx` 移动到 `_portal/portal/hosts/$hostId.tsx`
- **Files modified:** 路由文件路径变更 + routeTree.gen.ts 重新生成
- **Verification:** `npx @tanstack/router-cli generate` 成功，`tsc --noEmit` 无错误
- **Committed in:** 5977245

**2. [Rule 3 - Blocking] 安装缺失的 typescript devDependency**
- **Found during:** Task 1 验证
- **Issue:** `npx tsc --noEmit` 因项目未安装 typescript 包而失败
- **Fix:** `npm install --save-dev typescript`
- **Files modified:** package.json, package-lock.json (未提交，仅开发依赖)
- **Committed in:** 不影响产出

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** 路由文件路径变更是必要的兼容性修复，不影响运行时行为。

## Issues Encountered
None beyond the deviations above.

## Known Stubs
None - 所有数据通过 portalApiFetch hooks 从后端 API 获取，无硬编码占位数据。

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Portal 主机列表和详情页已就绪，可供后续 Claude 账号信息展示和 KasmVNC 接入扩展
- Topbar 角色显示已动态化，后续可扩展更多用户信息

---
*Phase: 12-api*
*Completed: 2026-03-29*
