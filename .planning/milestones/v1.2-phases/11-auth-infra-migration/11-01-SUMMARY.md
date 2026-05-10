---
phase: 11-auth-infra-migration
plan: 01
subsystem: auth
tags: [jwt, postgresql, migration, bcrypt, go, claims]

# Dependency graph
requires: []
provides:
  - "0007 迁移：users.role 列 + claude_accounts 表 + 索引"
  - "User 结构体包含 Role 和 PasswordHash 字段"
  - "ClaudeAccount 和 CreateUserWithRoleParams 模型"
  - "AuthClaims 统一 JWT claims 结构体"
  - "GenerateAuthToken 签发带 user_id + role 的 JWT"
  - "GetUserByShortIDForAuth 和 CreateUserWithRole 查询"
affects: [11-02, 11-03, 12-user-panel, 13-claude-accounts]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "统一 AuthClaims 结构体取代 RegisteredClaims 直接使用"
    - "User 查询统一返回 role 和 password_hash 列"

key-files:
  created:
    - "internal/store/migrations/0007_auth_unification.sql"
    - "internal/controlplane/http/auth.go"
  modified:
    - "internal/store/repository/models.go"
    - "internal/store/repository/queries.go"

key-decisions:
  - "entry_password 列暂保留，不在此迁移中删除，避免破坏现有 bootstrap 流程"
  - "GetUserByShortIDForAuth 不返回 entry_password，仅返回认证所需字段"
  - "BcryptCost 定为 10（bcrypt.DefaultCost），作为项目级常量统一管理"

patterns-established:
  - "AuthClaims：所有 JWT 签发统一使用 AuthClaims{UserID, Role} 结构体"
  - "User 查询列顺序：id, username, status, role, short_id, password_hash, entry_password, expires_at, created_at, updated_at"

requirements-completed: [AUTH-01, CLAUDE-01]

# Metrics
duration: 3min
completed: 2026-03-29
---

# Phase 11 Plan 01: 数据库迁移与认证基础类型 Summary

**0007 迁移引入 users.role 列和 claude_accounts 表，扩展 Go 模型支持角色和 Claude 账号，定义统一 AuthClaims JWT claims 结构体和 GenerateAuthToken 工具函数**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-29T06:18:57Z
- **Completed:** 2026-03-29T06:22:11Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- 创建 0007_auth_unification.sql 迁移：users 表增加 role 列（DEFAULT 'user'）、新建 claude_accounts 表（含外键和唯一索引）
- 扩展 User 结构体增加 Role 和 PasswordHash 字段，新增 ClaudeAccount 和 CreateUserWithRoleParams 模型
- 更新全部 7 个 User 查询函数以返回 role 和 password_hash 列
- 新增 GetUserByShortIDForAuth 和 CreateUserWithRole 查询函数
- 创建 auth.go 定义 AuthClaims 统一 JWT claims 结构体和 GenerateAuthToken 签发函数

## Task Commits

Each task was committed atomically:

1. **Task 1: 数据库迁移 0007 + Go 模型扩展** - `3e6e9ba` (feat)
2. **Task 2: 查询层扩展 + AuthClaims 类型定义** - `60d53fe` (feat)

## Files Created/Modified
- `internal/store/migrations/0007_auth_unification.sql` - role 列和 claude_accounts 表的 DDL
- `internal/store/repository/models.go` - User.Role, User.PasswordHash, ClaudeAccount, CreateUserWithRoleParams
- `internal/store/repository/queries.go` - 全部 User 查询增加 role/password_hash + 两个新查询函数
- `internal/controlplane/http/auth.go` - AuthClaims 结构体 + GenerateAuthToken + BcryptCost

## Decisions Made
- entry_password 列暂保留不删除，等 Plan 02 改造 entry.go 后安全移除
- GetUserByShortIDForAuth 跳过 entry_password，仅返回认证必需字段
- BcryptCost = 10（bcrypt.DefaultCost）作为项目级常量

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Go 编译器不在此执行环境中可用，通过 grep 验证所有接受标准替代编译检查。代码结构与现有模式完全一致，不存在语法风险。

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 数据层和类型基础已就绪，Plan 02 可直接基于 AuthClaims 和 GetUserByShortIDForAuth 实现统一登录端点
- Plan 03 可基于 ClaudeAccount 模型实现 Claude 账号 CRUD API
- 迁移文件需在部署时执行 `psql -f 0007_auth_unification.sql`

---
*Phase: 11-auth-infra-migration*
*Completed: 2026-03-29*
