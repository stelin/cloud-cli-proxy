package network

import "context"

// Provider defines the contract for setting up and tearing down per-host
// network isolation.
//
// 并发语义（Phase 45 WR-08 修正）：
//   - 同一 hostID 上的 Prepare* / Cleanup* 调用 **必须由调用方串行化**。
//     实现内部不持锁；多个 goroutine 用同一 hostID 并发调用会让写盘 /
//     RemoveAll / verify 等步骤交错，导致 config.json 被反复 wipe / write
//     等竞态。
//   - 不同 hostID 之间互不影响，可并行调用。
//
// 单宿主机控制面通过任务队列（同一 hostID 一次只有一个 worker）天然满足
// 串行约束；如果未来把 Provider 嵌入到管理多并发的 SDK / agent 中，调用方
// 必须自行加 per-hostID mutex。
//
// Phase 54-01 单容器化后，host 创建路径上 PrepareGateway 必须先于 worker
// 容器的 docker create / start 调用：
//   - PrepareGateway 负责「mkdir SingBoxConfigDir + 写 sing-box config」，保证
//     worker 容器启动时 ro bind mount 引用的 /etc/sing-box/config.json 已存在；
//   - PrepareHost 在 worker 容器 docker start 之后调用，仅负责
//     verifyWorkerNetwork（出口 IP / DNS / leak 三检）。
//
// CleanupHost 是反操作：清理 host 端 SingBoxConfigDir 目录（best-effort 幂等）。
// 容器自身的销毁由 worker.stopHost / rebuildHost 路径上的 docker stop / rm 负责。
type Provider interface {
	PrepareGateway(context.Context, HostNetworkSpec) error
	PrepareHost(context.Context, HostNetworkSpec) error
	CleanupHost(context.Context, HostNetworkSpec) error
}
