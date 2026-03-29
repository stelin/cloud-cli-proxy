---
phase: 11-auth-infra-migration
verified: 2026-03-28T23:45:00Z
status: gaps_found
score: 3/4 must-haves verified
gaps:
  - truth: "用户 API 请求只能访问自己的资源，尝试访问他人资源返回 403"
    status: partial
    reason: "中间件基础设施就绪（AuthMiddleware + RequireRole + UserIDFromContext），但目前没有任何用户级资源端点调用 UserIDFromContext 做资源隔离。UserIDFromContext 仅被定义但未被任何 handler 引用。用户面板 API 尚未实现（Phase 12 范围），因此无法验证 403 行为。"
    artifacts:
      - path: "internal/controlplane/http/auth_middleware.go"
        issue: "UserIDFromContext 已导出但未被任何 handler 使用"
    missing:
      - "至少一个用户级 API 端点（如 GET /v1/user/hosts）调用 UserIDFromContext 并基于 user_id 过滤资源"
      - "当 user_id 不匹配时返回 403 的逻辑"
---

# Phase 11: 认证基础设施与数据迁移 Verification Report

**Phase Goal:** 用户可以使用自己的凭证登录系统，系统能区分管理员和普通用户角色，Claude 账号数据模型就绪
**Verified:** 2026-03-28T23:45:00Z
**Status:** gaps_found
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | 用户使用 short_id + 密码登录后获取 JWT，JWT 中包含 role claim 区分管理员和普通用户 | VERIFIED | auth_handler.go: UnifiedLoginHandler 调用 GetUserByShortIDForAuth + bcrypt.CompareHashAndPassword + GenerateAuthToken(secret, user.ID, user.Role, ...); auth.go: AuthClaims 包含 UserID + Role; 返回 {token, role, expires_in} |
| 2 | 管理员和用户共用同一登录页面，登录后根据角色自动跳转到对应的面板 | VERIFIED | login.tsx: 发送 short_id+password 到 /v1/auth/login，按 data.role === "admin" 跳转 "/" 或 "/portal"; _dashboard.tsx: getRole() !== "admin" 时 redirect 到 /portal; _portal.tsx: 受 isAuthenticated() 保护 |
| 3 | 用户 API 请求只能访问自己的资源，尝试访问他人资源返回 403 | FAILED | UserIDFromContext 已定义但未被任何 handler 调用。没有用户级资源 API 端点存在，无法验证资源隔离和 403 行为。中间件基础设施就绪但未接入实际业务逻辑。 |
| 4 | claude_accounts 表已创建，支持一个用户拥有多个 Claude 账号且每个账号关联一台主机 | VERIFIED | 0007_auth_unification.sql: CREATE TABLE claude_accounts (user_id UUID NOT NULL REFERENCES users, host_id UUID REFERENCES hosts, ...); models.go: ClaudeAccount 结构体; 索引 idx_claude_accounts_user_id + idx_claude_accounts_email |

**Score:** 3/4 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/store/migrations/0007_auth_unification.sql` | role 列和 claude_accounts 表 DDL | VERIFIED | ALTER TABLE users ADD COLUMN role; CREATE TABLE claude_accounts; 3 indexes |
| `internal/store/repository/models.go` | User.Role, ClaudeAccount 结构体 | VERIFIED | User.Role, User.PasswordHash, ClaudeAccount, CreateUserWithRoleParams 全部存在 |
| `internal/store/repository/queries.go` | GetUserByShortIDForAuth, CreateUserWithRole | VERIFIED | 两个函数存在；所有 User 查询包含 role 列 |
| `internal/controlplane/http/auth.go` | AuthClaims + GenerateAuthToken | VERIFIED | AuthClaims{UserID, Role}; GenerateAuthToken 签发 HS256 JWT; BcryptCost 常量 |
| `internal/controlplane/http/auth_handler.go` | UnifiedLoginHandler | VERIFIED | ServeHTTP 完整实现：解码请求 -> GetUserByShortIDForAuth -> bcrypt 比对 -> GenerateAuthToken -> 返回 token+role |
| `internal/controlplane/http/auth_middleware.go` | AuthMiddleware + RequireRole + context 辅助 | VERIFIED | AuthMiddleware 解析 JWT 注入 context; RequireRole 角色过滤; UserIDFromContext + RoleFromContext 导出 |
| `internal/controlplane/http/router.go` | /v1/auth/login 路由注册 | VERIFIED | POST /v1/auth/login + POST /v1/admin/login 指向同一 handler; adminGuard = AuthMiddleware + RequireRole("admin"); AuthStore 依赖注入 |
| `internal/controlplane/app/app.go` | ensureSeedAdmin 启动逻辑 | VERIFIED | ensureSeedAdmin 在 Run() 中迁移后调用; 使用 GetUserByShortIDForAuth 检查 + CreateUserWithRole 创建; bcrypt 哈希密码 |
| `internal/controlplane/http/entry.go` | bcrypt 密码比对 | VERIFIED | bcrypt.CompareHashAndPassword(user.PasswordHash, body.Password) 替代明文比对 |
| `web/admin/src/lib/auth.ts` | parseTokenPayload, getRole, isAdmin | VERIFIED | 全部 3 个函数存在且实现完整 |
| `web/admin/src/routes/login.tsx` | 统一登录页 short_id + /v1/auth/login | VERIFIED | schema 使用 short_id; fetch /v1/auth/login; 按 role 跳转 |
| `web/admin/src/routes/_dashboard.tsx` | 管理员路由守卫 | VERIFIED | getRole() !== "admin" 时 redirect 到 /portal |
| `web/admin/src/routes/_portal.tsx` | 用户面板布局骨架 | VERIFIED | isAuthenticated() 保护; PortalLayout 包含 Topbar + Outlet |
| `web/admin/src/routes/_portal/index.tsx` | 用户面板首页占位 | VERIFIED | 显示 "我的面板" 占位内容 |
| `internal/controlplane/http/admin_auth.go` | 旧代码已清理 | VERIFIED | 仅保留 package 声明和注释，AdminLoginHandler/AdminAuthMiddleware 已删除 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| auth_handler.go | queries.go | GetUserByShortIDForAuth | WIRED | handler 调用 h.store.GetUserByShortIDForAuth |
| auth_middleware.go | auth.go | AuthClaims 解析 JWT | WIRED | ParseWithClaims 使用 &AuthClaims{} |
| router.go | auth_handler.go | /v1/auth/login | WIRED | mux.Handle("POST /v1/auth/login", loginHandler) |
| app.go | queries.go | CreateUserWithRole 种子管理员 | WIRED | a.repo.CreateUserWithRole(ctx, ...) |
| login.tsx | /v1/auth/login | fetch POST | WIRED | fetch("/v1/auth/login", {...}) |
| login.tsx | auth.ts | setToken + role 跳转 | WIRED | setToken(data.token); navigate based on data.role |
| _dashboard.tsx | auth.ts | getRole 角色检查 | WIRED | import { getRole } from "@/lib/auth"; role check in beforeLoad |
| 0007 migration | queries.go | SQL 列名与 Go Scan 对齐 | WIRED | role 列在 SELECT 和 Scan 中对齐 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|-------------------|--------|
| login.tsx | loginMutation response | /v1/auth/login (auth_handler.go) | Yes - queries DB via GetUserByShortIDForAuth, returns real token+role | FLOWING |
| _dashboard.tsx | role (beforeLoad) | auth.ts getRole() -> localStorage JWT | Yes - parsed from real JWT payload | FLOWING |
| _portal/index.tsx | N/A (static content) | N/A | N/A - placeholder for Phase 12 | N/A |

### Behavioral Spot-Checks

Step 7b: SKIPPED (Go compiler not available in execution environment; no runnable entry points for static verification)

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-----------|-------------|--------|----------|
| AUTH-01 | 11-01, 11-02 | 用户可使用 short_id + 密码登录，获取带 role claim 的 JWT | SATISFIED | UnifiedLoginHandler + AuthClaims + GetUserByShortIDForAuth + bcrypt 全链路就绪 |
| AUTH-02 | 11-03 | 管理员和用户使用同一登录页，登录后根据角色跳转不同面板 | SATISFIED | login.tsx 统一页面 + role-based redirect + _dashboard admin guard + _portal 骨架 |
| AUTH-03 | 11-02 | 用户只能访问自己的资源，不能看到其他用户的数据 | BLOCKED | 中间件基础设施就绪但无用户级资源 API 调用 UserIDFromContext 做隔离 |
| CLAUDE-01 | 11-01 | claude_accounts 数据模型，一用户多账号，每账号关联一台主机 | SATISFIED | 迁移 DDL + ClaudeAccount 模型 + 外键 + 索引 |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| web/admin/src/routes/_portal/index.tsx | 10-13 | 占位内容 "您的主机和资源信息将在后续版本中显示" | Info | Phase 12 范围，不影响 Phase 11 目标 |
| internal/controlplane/http/auth_middleware.go | 19 | UserIDFromContext 已导出但未被任何 handler 使用 | Warning | AUTH-03 基础设施就绪但未接入业务 |

### Human Verification Required

### 1. 统一登录端到端流程

**Test:** 启动应用，访问登录页，使用 short_id + password 登录
**Expected:** admin 角色跳转到 dashboard，user 角色跳转到 /portal
**Why human:** 需要运行应用并在浏览器中验证路由跳转和 JWT 签发

### 2. 种子管理员自动创建

**Test:** 设置 ADMIN_USERNAME 和 ADMIN_PASSWORD 环境变量，启动应用
**Expected:** 首次启动时数据库中自动创建 admin 角色用户，再次启动不重复创建
**Why human:** 需要运行应用连接 PostgreSQL 验证

### 3. Go 编译通过

**Test:** 执行 `go build ./internal/...` 和 `go vet ./internal/...`
**Expected:** 无错误
**Why human:** Go 编译器不在当前验证环境中

### Gaps Summary

Phase 11 的 4 个成功标准中 3 个已完全验证。AUTH-03（用户资源隔离返回 403）的中间件基础设施已就绪（AuthMiddleware 注入 user_id 到 context，RequireRole 可按角色过滤，UserIDFromContext 可在 handler 中获取当前用户 ID），但由于用户面板 API 端点尚未实现（属 Phase 12 范围），目前没有任何 handler 实际调用 UserIDFromContext 来限制用户只能访问自己的资源。

这是一个边界问题：Phase 11 提供了所有认证和授权基础设施，但 AUTH-03 所要求的"尝试访问他人资源返回 403"的可观测行为在没有用户级资源 API 的情况下无法实现。建议在 Phase 12 实现用户面板 API 时同步完成资源隔离逻辑，届时重新验证 AUTH-03。

---

_Verified: 2026-03-28T23:45:00Z_
_Verifier: Claude (gsd-verifier)_
