package network

import "context"

// DockerVerifier 通过 docker exec 在容器内执行真实网络验证。
// 适用于 Linux 生产环境，要求宿主机具有 docker 执行权限。
type DockerVerifier struct{}

func (v *DockerVerifier) Verify(ctx context.Context, containerName string, egress EgressConfig) (VerifyResult, error) {
	return VerifyNetworkIntegrityDocker(ctx, containerName, egress)
}
