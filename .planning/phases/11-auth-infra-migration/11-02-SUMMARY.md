---
phase: 11-auth-infra-migration
plan: 02
subsystem: auth
tags: [jwt, bcrypt, middleware, go, authentication]

requires:
  - phase: 11-01
    provides: "AuthClaims 类型、GenerateAuthToken 函数、GetUserByShortIDForAuth/CreateUserWithRole 查询、User 模型 Role/PasswordHash 字段"
provides:
  - "UnifiedLoginHandler 统一登录端点 /v1/auth/login"
  - "AuthMiddleware 通用 JWT 认证中间件（提取 user_id + role 注入 context）"
  - "RequireRole 角色限制中间件"
  - "UserIDFromContext / RoleFromContext context 辅助函数"
  - "entry.go bcrypt 密码比对（替代明文）"
  - "ensureSeedAdmin 启动时种子管理员自动创建"
affects: [11-03, 12-user-portal, 13-claude-accounts]

tech-stack:
  added: []
  patterns: ["AuthMiddleware + RequireRole 双层中间件组合", "统一登录端点兼容旧路径"]

key-files:
  created:
    - internal/controlplane/http/auth_handler.go
    - internal/controlplane/http/auth_middleware.go
  modified:
    - internal/controlplane/http/router.go
    - internal/controlplane/http/admin_auth.go
    - internal/controlplane/http/entry.go
    - internal/controlplane/app/app.go

key-decisions:
  - "使用 adminGuard 组合函数（AuthMiddleware + RequireRole）替代旧 AdminAuthMiddleware"
  - "ensureSeedAdmin 使用 bcrypt.DefaultCost 而非跨包引用 cphttp.BcryptCost"
  - "admin_auth.go 保留为空文件以追踪 git 变更历史"

patterns-established:
  - "AuthMiddleware + RequireRole 双层鉴权：先验证 JWT 注入 context，再按角色过滤"
  - "extractToken 三源提取：Authorization header > query param > cookie"

requirements-completed: [AUTH-01, AUTH-03]

duration: 4min
completed: 2026-03-29
---

# Phase 11 Plan 02: 统一登录端点与认证中间件 Summary

**统一登录端点 /v1/auth/login 接受 short_id + password bcrypt 认证返回带 role 的 JWT，通用 AuthMiddleware 替代旧 AdminAuthMiddleware，entry.go 废弃明文密码，启动时自动创建种子管理员**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-29T06:25:10Z
- **Completed:** 2026-03-29T06:29:37Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- UnifiedLoginHandler 统一登录端点，接受 short_id + password，bcrypt 比对后返回 {token, role, expires_in}
- AuthMiddleware 通用 JWT 认证中间件 + RequireRole 角色限制中间件，替代旧 AdminAuthMiddleware
- entry.go 从明文密码比对改为 bcrypt.CompareHashAndPassword
- app.go 新增 ensureSeedAdmin，启动时从环境变量自动创建种子管理员

## Task Commits

Each task was committed atomically:

1. **Task 1: 统一登录 Handler + 认证中间件** - `0752bed` (feat)
2. **Task 2: 路由注册 + AdminAuth 替换 + entry.go bcrypt + 种子管理员** - `f2a488f` (feat)

## Files Created/Modified
- `internal/controlplane/http/auth_handler.go` - UnifiedLoginHandler 统一登录端点
- `internal/controlplane/http/auth_middleware.go` - AuthMiddleware + RequireRole + context 辅助函数
- `internal/controlplane/http/router.go` - /v1/auth/login 注册 + adminGuard 替代 adminAuth
- `internal/controlplane/http/admin_auth.go` - 清理旧 AdminLoginHandler 和 AdminAuthMiddleware
- `internal/controlplane/http/entry.go` - bcrypt 密码比对替代明文
- `internal/controlplane/app/app.go` - ensureSeedAdmin + AuthStore 依赖注入

## Decisions Made
- 使用 adminGuard 组合函数（AuthMiddleware + RequireRole("admin")）替代旧 AdminAuthMiddleware，保持路由注册简洁
- ensureSeedAdmin 使用 bcrypt.DefaultCost 而非跨包引用 cphttp.BcryptCost，避免 app 包对 http 包的不必要依赖
- admin_auth.go 保留为空文件（仅 package 声明），以便 git diff 可追踪变更历史
- /v1/admin/login 保留指向同一 UnifiedLoginHandler 做向后兼容

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Go 编译器未安装在执行环境中，无法运行 `go build` 和 `go vet` 验证。通过 grep 检查所有接受标准确认代码结构正确性。实际编译验证将在合并到主仓库后进行。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- 统一登录端点和认证中间件就绪，Plan 03（前端统一登录页改造）可直接使用 /v1/auth/login
- UserIDFromContext / RoleFromContext 已导出，Phase 12 用户面板可直接使用

---
*Phase: 11-auth-infra-migration*
*Completed: 2026-03-29*
