//go:build !linux

package network

import (
	"context"
)

// verifyWorkerNetwork 在非 linux 平台返回空结果（macOS 开发机 / Windows 等）。
// v4.0 (Phase 54) 改造（54-01）：删除 applyWorkerFirewall / cleanupWorkerFirewall
// stub（D-54-7）；容器内 entrypoint 自己 apply nft，host-agent 不再入 worker netns。
func verifyWorkerNetwork(_ context.Context, _ string, _ EgressConfig) (VerifyResult, error) {
	return VerifyResult{}, nil
}
