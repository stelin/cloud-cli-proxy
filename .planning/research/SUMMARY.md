# Research Summary: v1.2 用户自助面板与 Bootstrap 重设计

**Project:** Cloud CLI Proxy
**Domain:** 用户认证体系扩展、前端角色路由、KasmVNC 代理、Claude 账号模型、Bootstrap 流程优化
**Researched:** 2026-03-28
**Overall confidence:** HIGH

## Executive Summary

v1.2 的六项新功能都可以在现有架构基础上增量集成，不需要对控制面、host-agent 或网络层做结构性改动。核心变更集中在三个层面：认证/授权（JWT role claim + 中间件链）、前端路由（TanStack Router RBAC 布局拆分）、以及数据模型（claude_accounts 表）。

现有 JWT 认证体系的扩展最为关键——用共享 secret + role claim 的方式，既保持管理员路径不变，又为用户面开辟独立的 API 命名空间（`/v1/me/`）。这是所有后续功能的前置依赖。

KasmVNC 代理已在管理员面完整实现（HTTP 反向代理 + WebSocket hijack），用户面只需加一层权限校验即可复用。Nginx 需要补上 WebSocket 升级头配置，这是当前缺失的。

Bootstrap 重设计（短 URL + SSE 实时状态）是用户体验层面的改进，技术上最简单（Nginx URL 重写 + Go 标准库 SSE），但排在构建顺序末尾，因为它改动的是用户入口，需要前面功能稳定后再触碰。

## Key Findings

**Stack:** 无新依赖引入。JWT role claim 用 golang-jwt/jwt/v5（已有），SSE 用 Go net/http 标准库，前端 RBAC 用 TanStack Router 内置 beforeLoad。
**Architecture:** 共享 JWT + role claim，`/v1/admin/` 和 `/v1/me/` 双命名空间，Nginx 做短 URL 重写和 WebSocket 升级。
**Critical pitfall:** 统一登录切换时必须保持管理员路径完全等价——先部署新中间件链，验证管理员功能无回归后再开放用户面。

## Implications for Roadmap

Based on research, suggested phase structure:

1. **认证基础设施 + DB 迁移** - 所有后续功能的前置依赖
   - Addresses: 用户登录认证体系、JWT role claim、中间件链替换、claude_accounts 表
   - Avoids: 在没有统一认证的情况下分头开发用户面 API

2. **用户自助 API + 前端路由** - 用户面的骨架
   - Addresses: 用户自助面板、角色路由守卫、统一登录页
   - Avoids: 前后端认证不一致

3. **Claude 账号管理** - 独立 CRUD，不阻塞其他功能
   - Addresses: Claude 账号数据模型、管理员管理页面、用户查看
   - Avoids: 数据模型变更影响已上线功能

4. **KasmVNC 用户面** - 依赖用户路由和权限
   - Addresses: 用户直接访问 KasmVNC 远程桌面
   - Avoids: WebSocket 代理在没有权限校验时暴露

5. **Bootstrap 重设计** - 用户入口体验，最后动
   - Addresses: 短 URL、SSE 实时状态推送、欢迎艺术字、自动 SSH 接入
   - Avoids: 在功能未稳定时改动用户首次接触点

**Phase ordering rationale:**
- Phase 1 是硬依赖：没有角色认证，用户面 API 无法鉴权
- Phase 2 依赖 Phase 1 的中间件和登录端点
- Phase 3 可以和 Phase 2 并行，但建议顺序执行以降低 Git 冲突
- Phase 4 依赖 Phase 2 的用户路由结构
- Phase 5 相对独立，但用户入口变更应该在功能稳定后进行

**Research flags for phases:**
- Phase 1: 标准模式，不需要额外研究
- Phase 4: Nginx WebSocket 配置需要实测验证超时和缓冲设置
- Phase 5: SSE 在 shell 脚本中的消费需要考虑 bash 兼容性（macOS 默认 bash 3.x vs Linux bash 5.x）

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | 无新依赖，全部使用现有库 |
| Features | HIGH | 需求清晰，现有代码提供了完整参考 |
| Architecture | HIGH | 现有 VNC 代理和认证代码已验证核心模式 |
| Pitfalls | MEDIUM | Nginx SSE/WebSocket 配置和 shell 兼容性需实测 |

## Gaps to Address

- macOS 默认 bash 版本对 SSE 消费的兼容性需要在 Phase 5 时测试
- 用户密码目前 `password_hash` 列已存在但部分用户可能只有 `entry_password`，需要确认迁移策略
- KasmVNC 在容器内的 `:6080` 端口是否在所有用户镜像中预配置，需确认镜像模板

---
*Research completed: 2026-03-28*
*Ready for roadmap: yes*
