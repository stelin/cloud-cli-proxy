---
phase: 9
slug: frontend-proxy-test
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-28
---

# Phase 9 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest (frontend) / go test (backend) |
| **Config file** | `web/admin/vite.config.ts` / `go test ./...` |
| **Quick run command** | `cd web/admin && npx vitest run --reporter=verbose` |
| **Full suite command** | `cd web/admin && npx vitest run && cd ../.. && go test ./internal/controlplane/...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd web/admin && npx vitest run --reporter=verbose`
- **After every plan wave:** Run full suite command
- **Before `$gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 09-01-01 | 01 | 1 | UI-01 | unit | `vitest run` | ❌ W0 | ⬜ pending |
| 09-01-02 | 01 | 1 | UI-02 | unit | `vitest run` | ❌ W0 | ⬜ pending |
| 09-01-03 | 01 | 1 | UI-03 | unit | `vitest run` | ❌ W0 | ⬜ pending |
| 09-02-01 | 02 | 1 | TEST-01 | unit | `go test ./internal/controlplane/...` | ❌ W0 | ⬜ pending |
| 09-02-02 | 02 | 1 | TEST-02 | unit | `go test ./internal/controlplane/...` | ❌ W0 | ⬜ pending |
| 09-02-03 | 02 | 1 | TEST-03 | unit | `go test ./internal/controlplane/...` | ❌ W0 | ⬜ pending |
| 09-02-04 | 02 | 1 | TEST-04 | unit | `go test ./internal/controlplane/...` | ❌ W0 | ⬜ pending |
| 09-03-01 | 03 | 2 | UI-04 | unit | `vitest run` | ❌ W0 | ⬜ pending |
| 09-03-02 | 03 | 2 | UI-05 | unit | `vitest run` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `web/admin/src/__tests__/egress-ip-drawer.test.tsx` — stubs for UI-01, UI-02, UI-03
- [ ] `internal/controlplane/http/admin_egress_ips_test.go` — test endpoint stubs for TEST-01 to TEST-04
- [ ] `web/admin/src/__tests__/egress-ip-list.test.tsx` — stubs for UI-04, UI-05

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 表单切换视觉效果 | UI-01 | 需人工确认切换动画和布局 | 创建出口 IP → 切换隧道类型 → 验证字段组切换正确 |
| JSON 编辑器双向转换 | UI-03 | 需确认 JSON 格式正确性和错误提示 | 在表单模式填入值 → 切换 JSON 模式 → 修改 → 切回表单模式 |
| 测试结果 Dialog 展示 | UI-04 | 需确认 loading 动画和结果布局 | 点击测试按钮 → 等待结果 → 验证颜色编码和详情展示 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
