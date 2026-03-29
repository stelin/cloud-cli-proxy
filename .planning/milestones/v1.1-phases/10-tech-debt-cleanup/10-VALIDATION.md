---
phase: 10
slug: tech-debt-cleanup
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-28
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test / vitest |
| **Config file** | go: native / vitest: web/admin/vitest.config.ts |
| **Quick run command** | `go test ./... -short` / `cd web/admin && pnpm test --run` |
| **Full suite command** | `go test ./... && cd web/admin && pnpm test --run` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick run command for affected layer (Go or frontend)
- **After every plan wave:** Run full suite command
- **Before `$gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 10-01-01 | 01 | 1 | SC-1 | integration | `go test ./internal/worker/... -run TestStopHost` | ❌ W0 | ⬜ pending |
| 10-01-02 | 01 | 1 | SC-2 | unit | `go test ./internal/network/... -run TestSingBoxLookPath` | ❌ W0 | ⬜ pending |
| 10-02-01 | 02 | 1 | SC-3 | manual | browser localStorage check | N/A | ⬜ pending |
| 10-02-02 | 02 | 1 | SC-4 | manual | browser UI check | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| localStorage 持久化测试结果 | SC-3 | 需浏览器 localStorage API | 执行代理测试 → 刷新页面 → 验证结果恢复 |
| WireGuard 测试禁用提示 | SC-4 | 需 UI 交互验证 | 切换到 WireGuard 类型出口 IP → 点击测试按钮 → 验证 toast 提示 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
