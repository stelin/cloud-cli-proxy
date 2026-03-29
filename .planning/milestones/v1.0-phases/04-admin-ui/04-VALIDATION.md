---
phase: 4
slug: admin-ui
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-27
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest (前端) + Go testing (后端) |
| **Config file** | `web/admin/vitest.config.ts` (Wave 0 创建) |
| **Quick run command** | `go test ./internal/controlplane/http/... -v` |
| **Full suite command** | `cd web/admin && npx vitest run && cd ../.. && go test ./internal/controlplane/http/...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/controlplane/http/... -v`
- **After every plan wave:** Run `cd web/admin && npx vitest run && cd ../.. && go test ./internal/controlplane/http/...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | ADMN-03 | unit (Go) | `go test ./internal/controlplane/http/ -run TestDashboardStats -v` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 1 | D-01 | unit (Go) | `go test ./internal/controlplane/http/ -run TestAdminAuth -v` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 2 | USER-01 | unit (Go) | `go test ./internal/controlplane/http/ -run TestAdminUsers -v` | ❌ W0 | ⬜ pending |
| 04-02-02 | 02 | 2 | USER-02 | unit (Go) | `go test ./internal/controlplane/http/ -run TestRotatePassword -v` | ❌ W0 | ⬜ pending |
| 04-03-01 | 03 | 2 | ADMN-01 | unit (Go) | `go test ./internal/controlplane/http/ -run TestAdminEgressIP -v` | ❌ W0 | ⬜ pending |
| 04-03-02 | 03 | 2 | ADMN-02 | unit (Go) | `go test ./internal/controlplane/http/ -run TestAdminBindings -v` | ❌ W0 | ⬜ pending |
| 04-03-03 | 03 | 2 | LIFE-01, LIFE-02, LIFE-03 | unit (Go) | `go test ./internal/controlplane/http/ -run TestAdminHost -v` | ❌ W0 | ⬜ pending |
| 04-03-04 | 03 | 2 | D-13/14 | unit (Go) | `go test ./internal/controlplane/http/ -run TestBindingProtection -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/controlplane/http/admin_auth_test.go` — JWT 登录和中间件测试
- [ ] `internal/controlplane/http/admin_users_test.go` — 用户 CRUD 测试
- [ ] `internal/controlplane/http/admin_egress_ips_test.go` — 出口 IP CRUD 测试
- [ ] `internal/controlplane/http/admin_bindings_test.go` — 绑定管理测试
- [ ] `web/admin/vitest.config.ts` — 前端测试框架配置

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 侧边栏导航布局渲染正确 | D-04 | 视觉验证 | 登录后检查侧边栏显示 5 个导航项 |
| 删除用户二次确认弹窗 | D-08 | UI 交互验证 | 点击删除按钮，确认弹窗出现并要求输入用户名 |
| 密码轮换一次性展示 | D-09 | UI 交互验证 | 点击轮换按钮，确认新密码仅展示一次可复制 |
| 出口 IP 抽屉表单 | D-11 | UI 交互验证 | 点击创建/编辑出口 IP，确认右侧抽屉滑出 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
