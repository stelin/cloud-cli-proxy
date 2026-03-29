---
phase: 04-admin-ui
plan: 02
subsystem: api, ui
tags: [go, react, bcrypt, crypto-rand, tanstack-query, shadcn-ui, radix-ui, user-crud, password-rotation]

requires:
  - phase: 04-01
    provides: Admin auth (JWT login, middleware), dashboard stats API, React + Vite scaffold
  - phase: 03-ssh
    provides: bcrypt password hashing pattern, bootstrap user auth model
provides:
  - User CRUD API endpoints (GET/POST/PATCH/DELETE /v1/admin/users)
  - Password rotation API (POST /v1/admin/users/{id}/rotate-password)
  - Repository methods for user lifecycle (GetUser, CreateUser, UpdateUserStatus, DeleteUser, UpdateUserPassword, ListHostsByUserID)
  - Frontend user management pages (list, detail)
  - Create user dialog with zod validation
  - Delete user dialog with username confirmation
  - Password rotation dialog with one-time display
affects: [04-03, user-management, host-lifecycle]

tech-stack:
  added: [radix-ui, shadcn/ui table, dialog, badge, dropdown-menu, alert-dialog, separator]
  patterns: [AdminUserStore interface for handler dependency injection, TanStack Query hooks per resource domain]

key-files:
  created:
    - internal/controlplane/http/admin_users.go
    - web/admin/src/hooks/use-users.ts
    - web/admin/src/routes/_dashboard/users/index.tsx
    - web/admin/src/routes/_dashboard/users/$userId.tsx
    - web/admin/src/components/users/create-user-dialog.tsx
    - web/admin/src/components/users/delete-user-dialog.tsx
    - web/admin/src/components/users/rotate-password-dialog.tsx
    - web/admin/src/components/ui/table.tsx
    - web/admin/src/components/ui/dialog.tsx
    - web/admin/src/components/ui/badge.tsx
    - web/admin/src/components/ui/dropdown-menu.tsx
    - web/admin/src/components/ui/alert-dialog.tsx
    - web/admin/src/components/ui/separator.tsx
  modified:
    - internal/store/repository/models.go
    - internal/store/repository/queries.go
    - internal/controlplane/http/router.go
    - internal/controlplane/app/app.go
    - web/admin/src/components/ui/button.tsx
    - web/admin/package.json

key-decisions:
  - "AdminUserStore 接口解耦 handler 和 repository，便于测试和替换"
  - "密码轮换使用 crypto/rand 生成 20 字符随机密码（含大小写字母、数字和特殊字符）"
  - "删除用户需输入用户名二次确认（D-08），数据库 CASCADE 自动清理关联 hosts 和 bindings"
  - "密码轮换结果仅展示一次，关闭对话框后不可找回（D-09）"
  - "Button 组件升级支持 asChild 属性以兼容 radix-ui v4 组件模式"

patterns-established:
  - "AdminXxxStore 接口模式：handler 通过接口依赖 repository，在 router.go Dependencies 中注入"
  - "use-xxx.ts hooks 模式：每个资源域一个 hooks 文件，封装 useQuery + useMutation"
  - "Dialog 组件模式：受控 open/onOpenChange + mutation hook + toast 反馈"

requirements-completed: [USER-01, USER-02, ADMN-03]

duration: 9min
completed: 2026-03-27
---

# Phase 04 Plan 02: User & Credential Management Summary

**用户 CRUD API（Go bcrypt + crypto/rand 密码轮换）+ React 前端用户管理全页面（列表/详情/创建/删除确认/密码轮换）**

## Performance

- **Duration:** 9 min
- **Started:** 2026-03-27T11:46:43Z
- **Completed:** 2026-03-27T11:55:41Z
- **Tasks:** 2
- **Files modified:** 21

## Accomplishments

- 完整的用户 CRUD API：列表、详情（含主机列表）、创建（bcrypt 哈希）、状态切换（active/disabled）、删除（CASCADE 级联清理）
- 密码轮换 API：crypto/rand 生成 20 字符随机强密码，bcrypt 哈希后存储，返回明文一次性展示
- 前端用户管理完整链路：列表页（表格 + 状态 Badge + 操作菜单）→ 详情页（信息卡 + 操作按钮 + 主机列表）→ 创建/删除/轮换弹窗
- 删除操作需输入用户名二次确认，密码轮换结果仅展示一次并提供复制按钮

## Task Commits

Each task was committed atomically:

1. **Task 1: Go 后端 — 用户 CRUD API + 密码轮换 API + Repository 扩展** - `6593c84` (feat)
2. **Task 2: 前端 — 用户列表页 + 详情页 + 创建/删除/密码轮换功能** - `d463aeb` (feat)

## Files Created/Modified

- `internal/controlplane/http/admin_users.go` - AdminUsersHandler：List/Get/Create/UpdateStatus/Delete/RotatePassword handlers
- `internal/store/repository/models.go` - 新增 CreateUserParams 类型
- `internal/store/repository/queries.go` - 新增 GetUser/CreateUser/UpdateUserStatus/DeleteUser/UpdateUserPassword/ListHostsByUserID 方法
- `internal/controlplane/http/router.go` - 注册 /v1/admin/users 路由组（6 个端点）
- `internal/controlplane/app/app.go` - 注入 AdminUsers: repo 依赖
- `web/admin/src/hooks/use-users.ts` - TanStack Query hooks：useUsers/useUser/useCreateUser/useUpdateUserStatus/useDeleteUser/useRotatePassword
- `web/admin/src/routes/_dashboard/users/index.tsx` - 用户列表页
- `web/admin/src/routes/_dashboard/users/$userId.tsx` - 用户详情页
- `web/admin/src/components/users/create-user-dialog.tsx` - 创建用户弹窗（zod 校验）
- `web/admin/src/components/users/delete-user-dialog.tsx` - 删除确认弹窗（输入用户名确认）
- `web/admin/src/components/users/rotate-password-dialog.tsx` - 密码轮换弹窗（一次性展示 + 复制）
- `web/admin/src/components/ui/*.tsx` - 新增 table/dialog/badge/dropdown-menu/alert-dialog/separator 组件
- `web/admin/src/components/ui/button.tsx` - 升级支持 asChild 属性

## Decisions Made

- AdminUserStore 接口解耦 handler 和 repository，保持与 DashboardStatsGetter 一致的依赖注入模式
- 密码轮换使用 crypto/rand 而非 math/rand，保证密码学安全随机性
- username 唯一冲突通过错误信息匹配 "unique"/"duplicate" 返回 409
- Button 组件升级支持 asChild（使用 radix-ui Slot）以兼容新版 shadcn v4 AlertDialog/Dialog 组合模式
- 删除用户 Repository 方法通过 RowsAffected == 0 判断是否存在，而非先查询再删除

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] shadcn CLI 将组件写入错误目录**
- **Found during:** Task 2（安装 shadcn 组件）
- **Issue:** shadcn CLI 将文件写入了 `web/admin/@/components/ui/` 而非 `web/admin/src/components/ui/`
- **Fix:** 手动复制文件到正确目录并删除错误目录
- **Files modified:** web/admin/src/components/ui/ 下的 6 个组件文件
- **Verification:** vite build 通过

**2. [Rule 2 - Missing Critical] Button 组件缺少 asChild 支持**
- **Found during:** Task 2（创建 AlertDialog 组件）
- **Issue:** 新版 shadcn v4 的 AlertDialogAction/Cancel 使用 `<Button asChild>`，但原始 Button 不支持 asChild
- **Fix:** 更新 Button 组件引入 radix-ui Slot，支持 asChild 属性
- **Files modified:** web/admin/src/components/ui/button.tsx
- **Verification:** vite build 通过，AlertDialog 正常渲染

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 missing critical)
**Impact on plan:** 两处修复均为工具链兼容性问题，不影响业务功能范围。

## Issues Encountered

- shadcn 安装过程中 dialog.json 下载出现网络错误，但组件文件已成功创建。通过手动移动文件解决。

## User Setup Required

None - no external service configuration required.

## Known Stubs

None — all API endpoints and frontend pages are fully wired with real data sources.

## Next Phase Readiness

- 用户管理全链路就绪，Plan 03 可直接在用户详情页基础上扩展主机生命周期操作和出口 IP 管理
- AdminXxxStore 接口模式已建立，Plan 03 新增的出口 IP 和主机管理 handler 可直接沿用

---
*Phase: 04-admin-ui*
*Completed: 2026-03-27*
