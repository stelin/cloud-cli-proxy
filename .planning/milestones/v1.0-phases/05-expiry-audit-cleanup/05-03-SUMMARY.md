---
phase: 05-expiry-audit-cleanup
plan: 03
subsystem: ui
tags: [react, tanstack-query, tanstack-router, tailwind, shadcn-ui]

requires:
  - phase: 05-01
    provides: "到期时间后端 API (PUT /users/:id/expiry, expires_at 字段)"
  - phase: 05-02
    provides: "事件日志后端 API (GET /events, 事件模型)"
provides:
  - "用户列表页到期时间列和 expired 状态 Badge"
  - "用户详情页到期时间管理 Dialog (设置/修改/清除)"
  - "事件日志页面 (筛选, 分页, metadata 展开)"
  - "侧栏事件日志导航入口"
  - "仪表板最近事件摘要卡片"
affects: []

tech-stack:
  added: []
  patterns:
    - "事件查询 hook 使用 URLSearchParams 构建筛选参数"
    - "共享 eventTypeLabel 映射函数在 use-events.ts 中集中定义"
    - "可展开表格行模式用于展示 metadata 详情"

key-files:
  created:
    - "web/admin/src/hooks/use-events.ts"
    - "web/admin/src/routes/_dashboard/events/index.tsx"
  modified:
    - "web/admin/src/hooks/use-users.ts"
    - "web/admin/src/routes/_dashboard/users/index.tsx"
    - "web/admin/src/routes/_dashboard/users/$userId.tsx"
    - "web/admin/src/components/layout/sidebar.tsx"
    - "web/admin/src/routes/_dashboard/index.tsx"
    - "web/admin/src/routeTree.gen.ts"

key-decisions:
  - "将 eventTypeLabel 映射函数集中在 use-events.ts 中，事件日志页面和仪表板共享使用"
  - "事件详情使用可展开行模式，点击行展开 metadata JSON 键值对"

patterns-established:
  - "事件筛选参数通过 URLSearchParams 动态构建查询字符串"
  - "分页使用 offset/limit 模式配合 total 计数"

requirements-completed: [LIFE-04, LIFE-05, ADMN-04]

duration: 8min
completed: 2026-03-27
---

# Phase 5 Plan 03: 前端到期时间展示与事件日志 Summary

**用户列表/详情页展示和管理到期时间，事件日志页面支持筛选分页和 metadata 展开，仪表板集成最近事件摘要卡片**

## Performance

- **Duration:** 8 min
- **Started:** 2026-03-27T15:00:00Z
- **Completed:** 2026-03-27T15:08:00Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- 用户列表页增加到期时间列和三态 Badge（活跃/已过期/已禁用），过期用户可通过菜单重新激活
- 用户详情页增加到期时间信息行和设置 Dialog，支持设置/修改/清除到期时间
- 新建事件日志页面，支持按事件类型筛选、offset/limit 分页、可展开 metadata 详情行
- 侧栏新增「事件日志」导航入口（ScrollText 图标）
- 仪表板新增最近事件摘要卡片，展示最近 5 条事件，带「查看全部」链接

## Task Commits

Each task was committed atomically:

1. **Task 1: 用户到期时间展示与管理** - `17bf362` (feat)
2. **Task 2: 事件日志页面与仪表板事件摘要** - `276f13c` (feat)

## Files Created/Modified
- `web/admin/src/hooks/use-users.ts` - 增加 expires_at 字段和 useUpdateUserExpiry hook
- `web/admin/src/routes/_dashboard/users/index.tsx` - 到期时间列、expired Badge、重新激活菜单项
- `web/admin/src/routes/_dashboard/users/$userId.tsx` - 到期时间信息行、设置 Dialog、expired 按钮
- `web/admin/src/hooks/use-events.ts` - 事件查询 hook 和类型定义，含 eventTypeLabel 映射
- `web/admin/src/routes/_dashboard/events/index.tsx` - 事件日志页面（筛选、分页、展开行）
- `web/admin/src/components/layout/sidebar.tsx` - 新增事件日志导航入口
- `web/admin/src/routes/_dashboard/index.tsx` - 仪表板最近事件摘要卡片
- `web/admin/src/routeTree.gen.ts` - 注册 events 路由

## Decisions Made
- 将 eventTypeLabel 映射函数集中在 use-events.ts 中，避免在事件日志页面和仪表板间重复定义
- 事件详情使用可展开表格行模式（点击展开 metadata），比模态弹窗更适合快速浏览

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 更新路由树文件注册 events 路由**
- **Found during:** Task 2 (事件日志页面)
- **Issue:** 新增 events/index.tsx 路由文件后，routeTree.gen.ts 未包含该路由，导致路由无法访问
- **Fix:** 手动在 routeTree.gen.ts 中添加 DashboardEventsIndexRoute 的导入、注册和类型声明
- **Files modified:** web/admin/src/routeTree.gen.ts
- **Verification:** npx tsc --noEmit 通过
- **Committed in:** 276f13c (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** 路由树更新是新增路由的必要操作，不影响计划范围。

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 5 所有三个计划（05-01 后端到期/事件、05-02 审计和清理、05-03 前端 UI）已全部完成
- Phase complete, ready for next step

---
*Phase: 05-expiry-audit-cleanup*
*Completed: 2026-03-27*
