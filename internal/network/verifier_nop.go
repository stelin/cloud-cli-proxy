package network

import "context"

// NopVerifier 在非 Linux 平台跳过网络验证，返回 safe-pass 结果以不阻塞开发流程。
// macOS / Windows 开发者无需具备完整 docker + netns 环境即可启动控制面。
type NopVerifier struct{}

func (v *NopVerifier) Verify(ctx context.Context, containerName string, egress EgressConfig) (VerifyResult, error) {
	return VerifyResult{
		EgressIPMatch: true, DNSCorrect: true,
		LeakBlocked: true, LeakTarget: "1.1.1.1:80",
		BypassEgressOK: true, NonBypassEgressOK: true, PublicDNSBlocked: true,
	}, nil
}
