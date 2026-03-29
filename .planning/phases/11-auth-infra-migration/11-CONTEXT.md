# Phase 11: 认证基础设施与数据迁移 - Context

**Gathered:** 2026-03-29 (assumptions mode)
**Status:** Ready for planning

<domain>
## Phase Boundary

用户可以使用自己的凭证登录系统，系统能区分管理员和普通用户角色，Claude 账号数据模型就绪。覆盖 AUTH-01、AUTH-02、AUTH-03、CLAUDE-01。

</domain>

<decisions>
## Implementation Decisions

### JWT 统一认证
- **D-01:** 合并管理员硬编码凭证和用户 bootstrap 认证为统一登录端点，所有角色通过 users 表认证并获取带 role claim 的 JWT
- **D-02:** 登录凭证使用 short_id + 密码，与 AUTH-01 要求一致。管理员也使用 short_id 登录（管理员在 users 表中的 short_id 可设置为如 "admin"）
- **D-03:** 保留环境变量管理员凭证作为超级后备通道（seed 管理员），但主流程走统一登录

### 用户表角色模型
- **D-04:** users 表添加 `role TEXT NOT NULL DEFAULT 'user'` 列，值为 `admin` 或 `user`
- **D-05:** 数据库迁移时自动创建种子管理员记录（从环境变量 ADMIN_USERNAME/ADMIN_PASSWORD 读取），并标记 role=admin
- **D-06:** 现有管理员 API 中间件改为读取 JWT 中的 role claim 鉴权，替代当前的硬编码凭证比对

### 密码体系统一
- **D-07:** 统一使用 bcrypt + password_hash 列，废弃 entry_password 明文字段
- **D-08:** entry flow 改为使用 bcrypt 比对 password_hash，保持 short_id 入口不变

### claude_accounts 数据模型
- **D-09:** 创建独立 claude_accounts 表，外键关联 users(user_id) 和 hosts(host_id)
- **D-10:** 基础字段：id (UUID PK)、user_id (FK)、host_id (FK nullable)、email、display_name、status、created_at、updated_at。具体业务字段（API key、订阅类型等）在 Phase 13 补充

### 前端统一登录
- **D-11:** 管理员和用户共用同一登录页（web/admin/src/routes/login.tsx 改造），输入 short_id + 密码
- **D-12:** 登录后前端从 JWT 解析 role，admin 跳转 /dashboard（现有管理面板），user 跳转 /portal（新增用户面板骨架路由，Phase 12 填充内容）

### 资源隔离中间件
- **D-13:** 新增 UserAuthMiddleware，从 JWT 提取 user_id 和 role，注入请求上下文
- **D-14:** 用户 API 端点校验资源归属（user_id 匹配），不匹配返回 403

### Claude's Discretion
- JWT 过期时间策略（可沿用现有 24h 或调整）
- 种子管理员的 short_id 命名约定
- claude_accounts 表的索引策略
- 登录页 UI 细节（布局、配色保持现有风格即可）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 认证要求
- `.planning/REQUIREMENTS.md` — AUTH-01, AUTH-02, AUTH-03, CLAUDE-01 具体验收标准
- `.planning/ROADMAP.md` §Phase 11 — Success Criteria 4 条

### 现有认证实现
- `internal/controlplane/http/admin_auth.go` — 现有管理员 JWT 生成逻辑（HS256 + RegisteredClaims）
- `internal/controlplane/http/bootstrap_auth.go` — 现有用户 bcrypt 认证流程
- `internal/controlplane/http/entry.go` — entry 明文密码认证（需废弃）
- `internal/controlplane/http/router.go` — 路由注册和中间件挂载点

### 数据模型
- `internal/store/migrations/0001_initial.sql` — users 表原始定义
- `internal/store/migrations/0005_host_env_and_user_entry.sql` — short_id 和 entry_password 字段
- `internal/store/repository/models.go` — User 结构体定义
- `internal/store/repository/queries.go` — 现有用户查询（GetBootstrapUserByUsername, GetUserByShortID 等）

### 前端认证
- `web/admin/src/routes/login.tsx` — 现有管理员登录页
- `web/admin/src/lib/auth.ts` — JWT 存储和解析工具
- `web/admin/src/routes/_dashboard.tsx` — 受保护路由守卫

### 应用配置
- `internal/controlplane/app/app.go` — AdminConfig 环境变量注入点

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `admin_auth.go` 中的 JWT 签名逻辑（HS256 + secret）可直接扩展为带 role claim 的版本
- `bootstrap_auth.go` 中的 bcrypt 比对逻辑可复用为统一登录的密码校验
- `AdminAuthMiddleware` 可重构为通用的 `AuthMiddleware`，根据 role 区分管理员和用户权限
- `web/admin/src/lib/auth.ts` 中的 token 存取逻辑可扩展为解析 role

### Established Patterns
- 数据库迁移使用递增编号 SQL 文件（当前到 0006），新迁移应为 0007
- UUID 主键 + 外键关联是标准模式（users → hosts → bindings）
- Go 结构体 + pgx 手写查询，无 ORM
- 前端 TanStack Router 文件路由 + React Query 数据获取

### Integration Points
- 统一登录端点需在 router.go 中注册，替代或并行 /v1/admin/login
- 种子管理员创建逻辑需在 app.go 启动时执行（读环境变量 → 检查是否已有管理员 → 创建）
- 前端路由守卫需从单一 isAuthenticated() 改为 isAuthenticated() + getRole()

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

- 用户自助面板的具体页面内容 — Phase 12
- Claude 账号 CRUD API 和前端 — Phase 13
- KasmVNC 用户直连 — Phase 14
- Bootstrap 短 URL 重设计 — Phase 15

</deferred>

---

*Phase: 11-auth-infra-migration*
*Context gathered: 2026-03-29*
