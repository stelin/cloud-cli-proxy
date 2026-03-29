//go:build linux

package network

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigureContainerDNS writes a resolv.conf inside the container filesystem
// that points exclusively to the tunnel-side DNS server. This ensures all DNS
// queries go through the controlled egress path, not the host or Docker resolver.
func ConfigureContainerDNS(containerPID uint32, dnsServer string) error {
	resolvPath := filepath.Join("/proc", fmt.Sprintf("%d", containerPID), "root", "etc", "resolv.conf")

	content := fmt.Sprintf(
		"# Managed by cloud-cli-proxy - tunnel DNS only\nnameserver %s\noptions ndots:0 attempts:2 timeout:5\n",
		dnsServer,
	)

	if err := os.WriteFile(resolvPath, []byte(content), 0o644); err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("configure dns: write %s: %v", resolvPath, err),
		}
	}

	return nil
}
