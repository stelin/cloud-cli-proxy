---
phase: 51-qual-harden
plan: 51-01
status: completed
completed_at: 2026-05-14
---

# 51-01 SUMMARY — `verifyEgressIP` 多源轮询

## 落地清单

- `internal/network/verify.go`：
  - 新增包级 `egressIPSources = []string{ip.me, ifconfig.io, ipinfo.io/ip}`。
  - 新增 `verifyEgressIPMulti(ctx, prefix, expectedIP, sources, *result)` 并发探测
    + 多数派投票。
  - 新增 `voteEgressIP(results []string) (winner string, ok bool)` 简单多数派函数
    （≥2 一致 / 单源也接受 / tie / 全空 → ok=false）。
  - `verifyEgressIP` 旧签名保留，内部委托 `verifyEgressIPMulti(单源)`，对外行为
    与 Phase 47 之前完全一致。
  - `VerifyNetworkIntegrity` Check 1 切换到 `verifyEgressIPMulti(..., egressIPSources, ...)`。
- `internal/network/verify_test.go`：新增 8 个单测
  - `TestVoteEgressIP_*` × 5（3-of-3 / 2-of-3 / 1-1-1 tie / 全空 / nil）
  - `TestVerifyEgressIPMulti_*` × 3（majority match / majority mismatch / all timeout）

## 验证

- `go build ./...` + `GOOS=linux go build ./...` PASS。
- `go vet ./...` PASS。
- `go test ./internal/network/... -count=1 -v` PASS（含新增 8 单测）。
- `go test ./... -count=1` 全绿。
- 既有 `TestVerifyNetworkIntegrity_*` 单测零修改通过（fake nsenter 不存在 → 多
  source 全 err → vote fail → EgressIPMatch=false + ActualEgressIP=""，与旧路径
  同行为）。

## 偏差

- CONTEXT §Area 1 提到「直接复用 Phase 46 `Vote`」，但 `Vote` 定义在
  `tests/e2e/helpers.go` 包 —— production 代码（`internal/network`）不能 import
  tests 包。本地新增私有 `voteEgressIP` 复刻同语义（多数派 ≥2 一致），与
  e2e 包语义对齐。已在代码注释明确披露。
