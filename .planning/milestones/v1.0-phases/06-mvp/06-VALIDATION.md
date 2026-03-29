---
phase: 6
slug: mvp
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-27
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` 标准库 (1.25.7) + BATS (1.x) |
| **Config file** | 无需配置文件（Go 标准工具链） |
| **Quick run command** | `go test ./internal/...` |
| **Full suite command** | `go test -tags integration ./... && bats tests/smoke/` |
| **Estimated runtime** | ~30 seconds (快速层) / ~90 seconds (含集成测试) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/...`
- **After every plan wave:** Run `go test -tags integration ./... && bats tests/smoke/`
- **Before `$gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 06-01-01 | 01 | 1 | ACCS-01 | smoke (BATS) | `bats tests/smoke/bootstrap.bats` | ❌ W0 | ⬜ pending |
| 06-01-02 | 01 | 1 | ACCS-03 | integration | `go test -tags integration ./tests/integration/ -run TestBootstrapHandoff` | ❌ W0 | ⬜ pending |
| 06-01-03 | 01 | 1 | NET-05 | unit + integration | `go test ./internal/network/... && go test -tags integration ./tests/integration/ -run TestNetworkVerification` | ✅ (unit) / ❌ (integration) | ⬜ pending |
| 06-01-04 | 01 | 1 | ADMN-03 | unit | `go test ./internal/controlplane/http/ -run TestAdmin` | ❌ W0 | ⬜ pending |
| 06-01-05 | 01 | 1 | ADMN-04 | unit | `go test ./internal/controlplane/http/ -run TestAdminEvents` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `tests/integration/helpers_test.go` — TestMain + testcontainers PostgreSQL 初始化
- [ ] `tests/integration/bootstrap_test.go` — bootstrap 全链路集成测试 (ACCS-01, ACCS-03)
- [ ] `tests/integration/admin_api_test.go` — Admin API CRUD 集成测试 (ADMN-03)
- [ ] `internal/controlplane/http/admin_users_test.go` — Admin Users handler 单元测试
- [ ] `internal/controlplane/http/admin_egress_ips_test.go` — Admin EgressIPs handler 单元测试
- [ ] `internal/controlplane/http/admin_hosts_test.go` — Admin Hosts handler 单元测试
- [ ] `internal/controlplane/http/admin_events_test.go` — Admin Events handler 单元测试
- [ ] `internal/controlplane/scheduler/expiry_test.go` — 到期扫描器单元测试
- [ ] `internal/controlplane/scheduler/reconciler_test.go` — 对账器单元测试
- [ ] `tests/smoke/bootstrap.bats` — bootstrap 脚本错误码契约测试
- [ ] BATS 安装：`npm install --save-dev bats bats-assert bats-support`

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| WireGuard 隧道连通性 | NET-05 | 需要真实 WireGuard 端点 | 在目标宿主机上运行 host-preflight.sh 并验证 wg show 输出 |
| SSH 会话交互体验 | ACCS-03 | 需要真实 SSH 服务端 | 执行 bootstrap 脚本完成完整登录流程 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
