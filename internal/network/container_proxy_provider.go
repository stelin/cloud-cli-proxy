package network

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type ContainerProxyProvider struct {
	logger *slog.Logger
}

func NewContainerProxyProvider(logger *slog.Logger) *ContainerProxyProvider {
	return &ContainerProxyProvider{logger: logger}
}

// PrepareGateway 在 worker 容器 docker create 之前把 sing-box config 写到 host 端
// SingBoxConfigDir(hostID)/config.json（D-54-2 / Plan 54-02），容器随后通过 ro bind
// mount 把该文件挂到 /etc/sing-box/config.json，entrypoint start_singbox_or_die 读取
// 后从 fs 删除（D-V4-2）。
//
// v4.0 (Phase 54) 改造（54-01）：
//   - 不再创建 cloudproxy-net-* 自定义 bridge（删除 dockerNetworkCreate 调用）
//   - 不再启动 sidecar gateway 容器（删除 dockerRunGateway / waitGatewayHealthy）
//   - 不再写 v3.5 容器 DNS 入口锁占位（resolv.conf / nsswitch.conf 由容器内 sing-box
//     接管，删除 WriteContainerDNSConfig 调用）
//   - 不再写 v4 sing-box 路径下的 cidrs / domains placeholder（由 sing-box config
//     的 route.rule_set 直接拉取，54-02 决定具体格式）
//
// user 容器自带 sing-box（Phase 53 镜像），entrypoint 内 start_singbox_or_die 在
// 容器自身 netns 里建 tun0 并应用全局策略路由；host-agent 只做「config 注入 + verify」。
func (p *ContainerProxyProvider) PrepareGateway(ctx context.Context, spec HostNetworkSpec) error {
	_ = ctx
	if spec.Egress == nil {
		p.logger.Info("container-proxy: no egress config, skipping config inject", "host_id", spec.HostID)
		return nil
	}
	if spec.Egress.Proxy == nil {
		p.logger.Warn("container-proxy: no proxy config, skipping config inject", "host_id", spec.HostID)
		return nil
	}

	dir := SingBoxConfigDir(spec.HostID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("sing-box: mkdir config dir: %w", err)
	}
	if err := writeContainerSingBoxConfig(spec.HostID, spec.Egress); err != nil {
		return fmt.Errorf("sing-box: write config: %w", err)
	}
	p.logger.Info("container-proxy: sing-box config injected", "host_id", spec.HostID, "dir", dir)
	return nil
}

// writeContainerSingBoxConfig 由 Plan 54-02 实现真正逻辑；54-01 留 stub 让单容器
// 链路骨架就位。stub 返回 nil 是 **设计意图**：worker.buildCreateArgs 的 sing-box
// config ro bind mount 在 config.json 不存在时会让 docker create 立即失败，避免
// 静默旁路。
func writeContainerSingBoxConfig(hostID string, egress *EgressConfig) error {
	_ = hostID
	_ = egress
	return nil
}

// PrepareHost 在 worker 容器 docker start 之后调用。
//
// v4.0 (Phase 54) 改造（54-01）：user 容器自带 sing-box（Phase 53），entrypoint
// apply_nft_or_die 在容器内部应用 fail-closed firewall + sing-box 自起 tun0；
// host-agent 不再：
//   - dockerNetworkConnect / disconnect bridge（容器走默认 docker bridge）
//   - configureWorkerEgress（容器内 sing-box 自己配 tun 路由）
//   - applyWorkerFirewall（容器内 entrypoint 自己 apply）
//   - join 控制面到隔离网络（无隔离网络存在）
//
// 仅保留 verifyWorkerNetwork 做出口 IP / DNS / leak 三检，确认 Phase 53 entrypoint
// 启动序列真的把流量导到 tun0。
func (p *ContainerProxyProvider) PrepareHost(ctx context.Context, spec HostNetworkSpec) error {
	if spec.Egress == nil {
		p.logger.Info("container-proxy: no egress config, skipping verify", "host_id", spec.HostID)
		return nil
	}
	if spec.Egress.Proxy == nil {
		p.logger.Warn("container-proxy: no proxy config, skipping verify", "host_id", spec.HostID)
		return nil
	}

	workerName := workerContainerName(spec.HostID)
	result, verifyErr := verifyWorkerNetwork(ctx, workerName, *spec.Egress)
	if verifyErr != nil {
		p.logger.Error("container-proxy: network verification failed",
			"host_id", spec.HostID,
			"egress_ip_match", result.EgressIPMatch,
			"dns_correct", result.DNSCorrect,
			"leak_blocked", result.LeakBlocked,
			"actual_egress_ip", result.ActualEgressIP,
			"actual_dns", result.ActualDNS,
		)
		if netErr, ok := verifyErr.(*NetworkError); ok {
			netErr.HostID = spec.HostID
		}
		return verifyErr
	}
	p.logger.Info("container-proxy: network verification passed (single-container)",
		"host_id", spec.HostID,
		"egress_ip", result.ActualEgressIP,
		"dns_server", result.ActualDNS,
	)
	return nil
}

// CleanupHost 在 host 失败或 stop 路径下被调用。
//
// v4.0 (Phase 54) 改造（54-01）：单容器架构下 host-agent 不再持有 gateway 容器
// 或自定义 bridge 网络，CleanupHost 仅需移除 host 端的 sing-box config 目录。
// 容器本身的销毁由 worker.stopHost / rebuildHost 路径上的 docker stop / rm 负责。
//
// 失败仅 Warn，不阻断后续 cleanup 链路（best-effort 幂等）。
func (p *ContainerProxyProvider) CleanupHost(ctx context.Context, spec HostNetworkSpec) error {
	_ = ctx
	dir := SingBoxConfigDir(spec.HostID)
	if err := os.RemoveAll(dir); err != nil {
		p.logger.Warn("container-proxy: cleanup remove sing-box config dir failed",
			"host_id", spec.HostID, "dir", dir, "error", err)
	}
	return nil
}

// SingBoxConfigDir 返回 host 专属的 sing-box config 目录。
// 路径规则：<DATA_DIR>/gateway/<host_id>/。
//
// 路径名 "gateway" 是 v3.x 历史包袱（D-54-10）：为避免跨包（bypass_reload.go /
// admin_hosts.go / app.go / e2e 等）大改动，54-01 保留物理路径不变，语义重定义
// 为「单容器架构下 host-agent 注入到 user 容器内 /etc/sing-box/config.json 的源
// 路径」。下一里程碑（v4.1）再考虑物理迁移到 sing-box/<host_id>/。
func SingBoxConfigDir(hostID string) string {
	base := os.Getenv("DATA_DIR")
	if base == "" {
		base = "/var/lib/cloud-cli-proxy"
	}
	return filepath.Join(base, "gateway", hostID)
}

// GatewayConfigDir 是 SingBoxConfigDir 的 v4.0 兼容 alias（D-54-9），
// 保留一个里程碑（v4.1 删除）。新代码请使用 SingBoxConfigDir。
//
// Deprecated: use SingBoxConfigDir.
func GatewayConfigDir(hostID string) string {
	return SingBoxConfigDir(hostID)
}

// workerContainerName 是 worker 容器的统一命名规则单点。
func workerContainerName(hostID string) string {
	return "cloudproxy-" + hostID
}
