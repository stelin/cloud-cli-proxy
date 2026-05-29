//go:build !linux

package network

import (
	"context"
	"encoding/json"
	"fmt"
)

// ApplyBypassRuleSet 在非 Linux 平台返回 unsupported 错误。
// bypass reload 需要 nsenter + nft，均为 Linux 独有工具；
// macOS 开发环境走 container_proxy_provider_other.go 的 verifyWorkerNetwork no-op 路径，
// 不会触发 bypass reload。
func ApplyBypassRuleSet(_ context.Context, _ string, _, _ json.RawMessage) error {
	return fmt.Errorf("bypass reload: unsupported on non-Linux platforms (requires nsenter + nft)")
}

// VerifyBypassConsistency 在非 Linux 平台返回 unsupported 错误。
func VerifyBypassConsistency(_ context.Context, _ string) (ConsistencyResult, error) {
	return ConsistencyResult{}, fmt.Errorf("bypass consistency check: unsupported on non-Linux platforms (requires nsenter + nft)")
}
