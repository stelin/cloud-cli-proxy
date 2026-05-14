---
phase: 51-qual-harden
plan: 51-06
status: completed
completed_at: 2026-05-14
gap_closure:
  - phase-49-gap-1
---

# 51-06 SUMMARY — worker `--cap-drop NET_RAW` + 删 `--cap-add SYS_ADMIN`

## 落地清单

- `internal/runtime/tasks/worker.go` 行 217-220 附近的 `buildCreateArgs` args
  切片：
  - **保留** `--cap-add NET_ADMIN`（按源码依赖：sing-box 在 worker netns 内
    创建 tun0 设备需要 CAP_NET_ADMIN，sing-box 进程通过 `nsenter -n` 进入
    netns 执行，必须容器级保留；运行时 setcap 不可行）。
  - **删除** `--cap-add SYS_ADMIN`（grep 业务代码不依赖；fuse mount 走
    `--device /dev/fuse + apparmor=unconfined`，fusermount setuid root 即可）。
  - **新增** `--cap-drop NET_RAW`（docker 默认 capability 集合含 CAP_NET_RAW，
    必须显式 drop 才能去掉；移除后容器内 SOCK_RAW 创建 PermissionDenied，闭
    Phase 49 LEAK-06 攻击面）。
- `internal/runtime/tasks/worker_caps_test.go`（新）：`TestBuildCreateArgs_
  CapabilitiesLocked` 单测锁定上述三条契约。

## 验证

- `go build ./...` + `GOOS=linux go build ./...` PASS。
- `go test ./internal/runtime/tasks/... -run BuildCreateArgs` PASS。
- `go test ./... -count=1` 全绿。

## NET_ADMIN 保留决策（CONTEXT §Area 4 允许的折中）

CONTEXT §Area 4 明确允许「按源码实际依赖决定：若 host-agent / sing-box 真的
依赖 worker netns 内 CAP_NET_ADMIN，则保留」。grep 结论：

- `internal/network/singbox_provider_linux.go`：sing-box 通过
  `nsenter -t <pid> -n -- sing-box run ...` 在 worker netns 内创建 tun 设备。
  tun 设备 ioctl(TUNSETIFF) 必须有 CAP_NET_ADMIN。
- 在 `host` netns 中通过 `nftables.WithNetNSFd(int(containerNS))` 操作 worker
  netns 的 nft 规则：这条路径靠 host-agent 进程 cap，与 worker 容器内 cap
  无关，本不要求 worker 含 CAP_NET_ADMIN。
- `netlink.LinkAdd` 在 `InjectManagementVeth` 中通过 `runtime.LockOSThread +
  netns.Set(containerNS)` 切到 worker netns 操作 —— 同样靠 host-agent cap。

因此 NET_ADMIN 必须保留以服务 sing-box tun 创建路径；SYS_ADMIN / NET_RAW
均可去掉。

## 与 Phase 49 GAP-1 闭环关系

- LEAK-06 raw socket：Phase 49 fixture `python_raw_socket_perm.txt` 期望
  PermissionError。本 plan 落地后容器内不再有 CAP_NET_RAW，SOCK_RAW
  socket(2) 立即返回 EPERM，用例 Linux runner 转 PASS。
- LEAK-08 capability 审计：Phase 49 fixture `proc_status_clean.txt` 期望
  `CapEff` / `CapBnd` 不含 NET_RAW / NET_ADMIN / SYS_ADMIN。本 plan 落地后
  实际 CapBnd 不含 SYS_ADMIN / NET_RAW，但**仍含 NET_ADMIN**（按 CONTEXT
  允许的折中保留）。
  - **此处与 fixture 严格期望存在差异**：fixture 期望 3 cap 全去掉，实际
    仅 2 cap 去掉。
  - 本 plan 不修改 e2e fixture（CONTEXT 锁定本 phase 不动 tests/e2e），
    fixture 修订属 Phase 49 范围（如后续要求 NET_ADMIN 也运行时 setcap 化，
    需另开 Phase 改造 sing-box 调用链）。
  - 本 SUMMARY 明确披露该折中，VERIFICATION 中列入 human_verification 段
    Linux runner 跑通时按实际情况判定 PASS / 维持 GAP。

## 偏差

- NET_ADMIN 保留（按源码依赖必要 + CONTEXT 允许的折中）。具体见上节。
