---
phase: 11-auth-infra-migration
plan: 03
subsystem: ui
tags: [react, tanstack-router, jwt, auth, role-routing]

requires:
  - phase: 11-01
    provides: 统一用户表 + short_id 字段
  - phase: 11-02
    provides: /v1/auth/login 统一登录端点（返回 role 字段）
provides:
  - auth.ts parseTokenPayload/getRole/isAdmin 角色解析函数
  - 统一登录页发送 short_id 到 /v1/auth/login 并按 role 跳转
  - 管理员路由守卫增加 admin 角色校验
  - 用户面板骨架路由 _portal 及首页占位
affects: [12-user-portal, 13-bootstrap-redesign]

tech-stack:
  added: []
  patterns: [JWT payload 角色解析, 文件路由角色守卫, 布局路由认证保护]

key-files:
  created:
    - web/admin/src/routes/_portal.tsx
    - web/admin/src/routes/_portal/index.tsx
  modified:
    - web/admin/src/lib/auth.ts
    - web/admin/src/routes/login.tsx
    - web/admin/src/routes/_dashboard.tsx
    - web/admin/src/routeTree.gen.ts

key-decisions:
  - "portal 路由使用 _portal 布局前缀 + 显式 /portal 路径，与 _dashboard 模式一致"
  - "admin 用户访问 portal 不做阻止（仅非 admin 被 dashboard 拒绝）"

patterns-established:
  - "角色路由守卫: beforeLoad 中 getRole() 校验，不匹配则 redirect"
  - "JWT payload 解析: atob(token.split('.')[1]) 提取 role/user_id/sub/exp"

requirements-completed: [AUTH-02]

duration: 5min
completed: 2026-03-28
---

# Phase 11 Plan 03: 前端统一登录与角色路由 Summary

**统一登录页改用 short_id 发送到 /v1/auth/login，按 JWT role 字段跳转 dashboard 或 portal，管理员路由守卫增加角色校验**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-28T12:05:02Z
- **Completed:** 2026-03-28T12:10:02Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- auth.ts 新增 parseTokenPayload/getRole/isAdmin 三个角色相关函数
- 登录页改为 short_id + password 发送到统一 /v1/auth/login 端点，按 role 跳转 dashboard 或 portal
- _dashboard.tsx 路由守卫增加 admin 角色校验，非管理员重定向到 /portal
- 创建 _portal.tsx 用户面板布局路由（含认证保护）和 _portal/index.tsx 首页占位

## Task Commits

Each task was committed atomically:

1. **Task 1: auth.ts 扩展 + 登录页改造** - `b981727` (feat)
2. **Task 2: 管理员路由守卫角色校验 + 用户面板骨架** - `72a3981` (feat)

## Files Created/Modified
- `web/admin/src/lib/auth.ts` - 新增 parseTokenPayload/getRole/isAdmin 函数
- `web/admin/src/routes/login.tsx` - 统一登录页，short_id + /v1/auth/login + 角色跳转
- `web/admin/src/routes/_dashboard.tsx` - 管理员路由守卫增加 admin 角色校验
- `web/admin/src/routes/_portal.tsx` - 用户面板布局路由（认证保护）
- `web/admin/src/routes/_portal/index.tsx` - 用户面板首页占位
- `web/admin/src/routeTree.gen.ts` - 注册 portal 路由到路由树

## Decisions Made
- portal 路由使用 _portal 布局前缀 + 显式 /portal 路径注册，与 _dashboard 模式保持一致
- admin 用户不被阻止访问 portal（仅非 admin 被 dashboard 守卫拒绝并重定向到 portal）

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 更新 routeTree.gen.ts 注册 portal 路由**
- **Found during:** Task 2
- **Issue:** TanStack Router 文件路由需要在 routeTree.gen.ts 中注册新路由，否则 /portal 路径不存在
- **Fix:** 手动更新 routeTree.gen.ts 添加 PortalRoute 和 PortalIndexRoute 的导入、注册和类型声明
- **Files modified:** web/admin/src/routeTree.gen.ts
- **Verification:** TypeScript 编译通过
- **Committed in:** 72a3981 (Task 2 commit)

**2. [Rule 3 - Blocking] portal index 路由路径从 / 改为 /portal**
- **Found during:** Task 2
- **Issue:** _portal 是 pathless 布局路由，其 index 子路由若使用 path: '/' 会与 _dashboard index 冲突
- **Fix:** 将 _portal/index.tsx 的 createFileRoute 路径设为 "/_portal/portal"，routeTree 中映射到 /portal 全路径
- **Files modified:** web/admin/src/routes/_portal/index.tsx, web/admin/src/routeTree.gen.ts
- **Verification:** TypeScript 编译通过，无路径冲突
- **Committed in:** 72a3981 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** 两项修复均为路由注册必要步骤，不影响功能范围。

## Issues Encountered
- node_modules 不存在，需先执行 npm install 安装依赖后才能运行 tsc 类型检查

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- 用户面板骨架就绪，Phase 12 可填充实际内容（主机列表、Claude 账号信息等）
- 管理员和用户的路由隔离已建立，后续只需在各自布局下添加子路由

## Self-Check: PASSED

All 5 created/modified files verified present. Both task commits (b981727, 72a3981) verified in git log.

---
*Phase: 11-auth-infra-migration*
*Completed: 2026-03-28*
