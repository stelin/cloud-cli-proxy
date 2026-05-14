---
phase: 51-qual-harden
plan: 51-03
status: completed
completed_at: 2026-05-14
---

# 51-03 SUMMARY — `verifyDNS` 遍历全部 nameserver

## 落地清单

- `internal/network/verify.go`：
  - 新增私有 `parseAllNameservers(rawContent string) []string` 解析函数（按
    原文件顺序返回所有 nameserver IP，跳过注释 / 空行 / 不合法行）。
  - `verifyDNS` 把 `result.ActualDNS` 改为「全部 nameserver 用逗号分隔」（单
    nameserver 退化为原值），便于日志 / metadata 看到 fallback。
  - `DNSCorrect` 判定逻辑保留：仍然走整 buffer 严格相等 + 首行 nameserver
    双保险（Phase 45 WR-07 锁定行为不变）。
- `internal/network/verify_test.go`：新增 6 单测
  - `TestParseAllNameservers_{Empty,SingleNS,MultipleNS,Comments,Garbage}` × 5
  - `TestVerifyDNS_ReportsAllNameservers`（多 nameserver 场景 ActualDNS 含
    逗号分隔列表）

## 验证

- `go build ./...` + `GOOS=linux go build ./...` PASS。
- `go test ./internal/network/...` PASS。
- `go test ./... -count=1` 全绿。
- 既有 `TestFirstNetworkError_DNSLeak{,_NilProxy}` 单测不需要修改即过（fake
  返回单 nameserver `"8.8.8.8"` → ActualDNS 退化为单元素逗号字符串 `"8.8.8.8"`，
  与旧值相同）。

## 偏差

- 无。
