---
phase: 8
slug: singboxprovider
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-28
---

# Phase 8 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — uses standard Go test conventions |
| **Quick run command** | `go test ./internal/network/... -run TestSingBox -count=1` |
| **Full suite command** | `go test ./internal/network/... ./internal/runtime/... ./internal/agent/... -count=1` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/network/... -run TestSingBox -count=1`
- **After every plan wave:** Run `go test ./internal/network/... ./internal/runtime/... ./internal/agent/... -count=1`
- **Before `$gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 08-01-01 | 01 | 1 | SING-05 | build | `docker build -f deploy/docker/managed-user/Dockerfile .` | ❌ W0 | ⬜ pending |
| 08-01-02 | 01 | 1 | SING-06 | unit | `go test ./internal/network/... -run TestRoutingProvider` | ❌ W0 | ⬜ pending |
| 08-02-01 | 02 | 2 | SING-01 | unit | `go test ./internal/network/... -run TestSingBoxConfig` | ❌ W0 | ⬜ pending |
| 08-02-02 | 02 | 2 | SING-02 | unit | `go test ./internal/network/... -run TestSingBoxDNS` | ❌ W0 | ⬜ pending |
| 08-02-03 | 02 | 2 | SING-03 | unit | `go test ./internal/network/... -run TestFirewallProxy` | ❌ W0 | ⬜ pending |
| 08-02-04 | 02 | 2 | SING-07 | unit | `go test ./internal/network/... -run TestSingBoxCleanup` | ❌ W0 | ⬜ pending |
| 08-03-01 | 03 | 3 | SING-04 | integration | `go test ./internal/network/... -run TestVerifyProxy` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/network/singbox_provider_test.go` — stubs for SING-01, SING-02, SING-07
- [ ] `internal/network/routing_provider_test.go` — stubs for SING-06
- [ ] `internal/network/firewall_test.go` — extend with proxy mode tests for SING-03

*Existing go test infrastructure covers framework needs.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Full traffic routing through tun0 in live container | SING-01 | Requires real Docker + netns + proxy server | Deploy test container with proxy egress, run `curl ipify.org` from within |
| DNS leak prevention in live environment | SING-02 | Requires real DNS resolution through proxy | Run DNS leak test from container, verify no direct DNS queries |
| sing-box process cleanup on container stop | SING-07 | Requires Docker lifecycle events | `docker stop <container>`, verify no orphan sing-box processes on host |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
