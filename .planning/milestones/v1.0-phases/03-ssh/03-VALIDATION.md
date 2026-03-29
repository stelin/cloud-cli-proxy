---
phase: 03
slug: ssh
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-27
---

# Phase 03 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing`（标准库）+ shell 集成脚本 |
| **Config file** | none — Go 原生（shell 用 `bash`） |
| **Quick run command** | `go test ./internal/controlplane/... ./internal/runtime/... -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~90 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/controlplane/... ./internal/runtime/... -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green + Linux 宿主机 bootstrap→SSH 冒烟通过
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | ACCS-01 | shell + API integration | `bash test/bootstrap/test_prompt_and_auth.sh` | ❌ W0 | ⬜ pending |
| 03-01-02 | 01 | 1 | ACCS-04 | unit + integration | `go test ./internal/controlplane/http -run TestBootstrapErrorTaxonomy -count=1` | ❌ W0 | ⬜ pending |
| 03-02-01 | 02 | 2 | ACCS-02 | API contract | `go test ./internal/controlplane/http -run TestBootstrapStageMapping -count=1` | ❌ W0 | ⬜ pending |
| 03-02-02 | 02 | 2 | RUNT-03 | integration | `go test ./internal/runtime/tasks -run TestSSHReadinessGate -count=1` | ❌ W0 | ⬜ pending |
| 03-03-01 | 03 | 3 | ACCS-03 | integration | `go test ./internal/runtime/tasks -run TestStartHostRequiresSSHReady -count=1` | ❌ W0 | ⬜ pending |
| 03-03-02 | 03 | 3 | ACCS-03, ACCS-04 | e2e shell | `bash test/bootstrap/e2e_bootstrap_ssh.sh` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/controlplane/http/bootstrap_auth_test.go` — ACCS-01 / ACCS-04
- [ ] `internal/controlplane/http/bootstrap_status_test.go` — ACCS-02
- [ ] `internal/runtime/tasks/worker_ssh_ready_test.go` — RUNT-03 / ACCS-03
- [ ] `internal/controlplane/http/bootstrap_handoff_test.go` — handoff 契约
- [ ] `test/bootstrap/e2e_bootstrap_ssh.sh` — 终端路径与退出码验证

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 真实 Linux 单宿主机上的 `curl -> SSH` 首次接入体验 | ACCS-01, ACCS-03 | 依赖真实 docker/netns/wireguard 与 ssh 客户端交互 | 在 Linux 宿主机执行 bootstrap，核对密码无回显、阶段提示、自动接入与 known_hosts 行为 |
| 账号禁用/过期提示语义是否符合运营预期 | ACCS-04 | 需要结合运营文案可读性与支持流程评估 | 构造 `users.status` 场景，运行 bootstrap，确认终端提示和下一步指引 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
