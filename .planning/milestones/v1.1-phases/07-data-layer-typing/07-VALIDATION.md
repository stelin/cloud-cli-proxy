---
phase: 7
slug: data-layer-typing
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-28
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing Go test infrastructure |
| **Quick run command** | `go test ./internal/...` |
| **Full suite command** | `go test -v -count=1 ./internal/...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/...`
- **After every plan wave:** Run `go test -v -count=1 ./internal/...`
- **Before `$gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 07-01-01 | 01 | 1 | DATA-05 | integration | `go test ./internal/store/...` | ❌ W0 | ⬜ pending |
| 07-01-02 | 01 | 1 | DATA-01, DATA-02 | unit | `go test ./internal/store/repository/...` | ❌ W0 | ⬜ pending |
| 07-02-01 | 02 | 1 | DATA-03 | unit | `go test ./internal/network/...` | ✅ | ⬜ pending |
| 07-02-02 | 02 | 1 | DATA-04 | unit | `go test ./internal/controlplane/http/...` | ❌ W0 | ⬜ pending |
| 07-03-01 | 03 | 2 | DATA-01, DATA-04 | integration | `go test ./internal/controlplane/http/...` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/network/validate_test.go` — extend existing tests for proxy type branching
- [ ] `internal/controlplane/http/admin_egress_ips_test.go` — extend for tunnel_type + proxy_config validation

*Existing infrastructure covers migration and model tests.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| DB migration backward compatibility | DATA-05 | Requires running migration against production-like data | Run migration on test DB with existing WG egress IP rows, verify tunnel_type='wireguard' |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
