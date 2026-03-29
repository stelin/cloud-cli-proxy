---
phase: 2
slug: tunnel-egress-enforcement
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-27
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — Wave 0 installs |
| **Quick run command** | `go test ./internal/network/... -count=1 -short` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/network/... -count=1 -short`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 02-01-T1 | 01 | 1 | NET-01 | unit | `go test ./internal/network/... -count=1 -short` | ❌ W0 | ⬜ pending |
| 02-01-T2 | 01 | 1 | NET-01 | unit | `go test ./internal/network/... -count=1 -short` | ❌ W0 | ⬜ pending |
| 02-02-T1 | 02 | 2 | NET-02, NET-03 | integration | `go test ./internal/network/... -count=1 -short` | ❌ W0 | ⬜ pending |
| 02-02-T2 | 02 | 2 | NET-03, NET-04 | integration | `go test ./internal/network/... -count=1 -short` | ❌ W0 | ⬜ pending |
| 02-03-T1 | 03 | 3 | NET-05 | integration | `go test ./internal/network/... -count=1 -short` | ❌ W0 | ⬜ pending |
| 02-03-T2 | 03 | 3 | NET-05 | integration | `go test ./internal/network/... -count=1 -short` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/network/errors_test.go` — stubs for NET-01 (error types and egress binding)
- [ ] `internal/network/validate_test.go` — stubs for NET-01 (preflight binding validation)
- [ ] `internal/network/namespace_test.go` — stubs for NET-02 (netns + veth + WireGuard injection)
- [ ] `internal/network/firewall_test.go` — stubs for NET-04 (nftables default-deny)
- [ ] `internal/network/dns_test.go` — stubs for NET-03 (DNS controlled resolver)
- [ ] `internal/network/verify_test.go` — stubs for NET-05 (egress IP + DNS + leak triple check)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Container egress IP matches allocated IP (external check) | NET-02 | Requires real WireGuard tunnel and external IP service | `curl ifconfig.me` inside container, compare with allocated IP |
| DNS resolution inside container uses controlled resolver | NET-03 | Requires running container with full namespace setup | `dig @127.0.0.1 example.com` inside container, verify resolver address |
| Direct outbound traffic is blocked (bypass attempt) | NET-04 | Requires live nftables rules in netns | Attempt `curl --interface eth0 ifconfig.me` (should fail) |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
