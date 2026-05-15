//go:build linux

package network

import (
	"context"
	"fmt"
)

// verifyWorkerNetwork 在 worker 容器 netns 中跑出口 IP / DNS / leak 三检。
//
// v4.0 (Phase 54) 改造（54-01）：删除 applyWorkerFirewall / cleanupWorkerFirewall
// 入口（D-54-7）。entrypoint apply_nft_or_die 在容器内部用 root 身份 apply 全套
// fail-closed nft 规则；host-agent 不再进入 worker netns 跑 ApplyWorkerFirewallRules，
// 也不再持有清理责任（容器销毁即规则销毁）。
func verifyWorkerNetwork(ctx context.Context, workerName string, egress EgressConfig) (VerifyResult, error) {
	_, pid, err := GetContainerNetNS(workerName)
	if err != nil {
		return VerifyResult{}, fmt.Errorf("get worker pid: %w", err)
	}
	return VerifyNetworkIntegrity(ctx, pid, egress)
}
