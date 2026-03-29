---
phase: 5
slug: expiry-audit-cleanup
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-27
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — uses standard Go test tooling |
| **Quick run command** | `go test ./internal/...` |
| **Full suite command** | `go test ./internal/... -v -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/...`
- **After every plan wave:** Run `go test ./internal/... -v -count=1`
- **Before `$gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 05-01-01 | 01 | 1 | LIFE-04 | unit | `go test ./internal/store/...` | ❌ W0 | ⬜ pending |
| 05-01-02 | 01 | 1 | LIFE-05 | unit | `go test ./internal/controlplane/...` | ❌ W0 | ⬜ pending |
| 05-02-01 | 02 | 1 | ADMN-04 | unit | `go test ./internal/controlplane/...` | ❌ W0 | ⬜ pending |
| 05-03-01 | 03 | 2 | ADMN-04 | unit | `go test ./internal/controlplane/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Test stubs for expiry timer and user status transition
- [ ] Test stubs for event recording and querying
- [ ] Test stubs for reconciliation logic

*Existing infrastructure covers Go test tooling — no framework install needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Event log page renders correctly | ADMN-04 | Frontend visual verification | Navigate to admin panel, check events page shows timeline with filters |
| Dashboard event card displays | ADMN-04 | Frontend visual verification | Check dashboard shows recent events card |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
