//go:build linux

package network

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func ensureIPForwarding(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "sysctl", "-w", "net.ipv4.ip_forward=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("enable ip forwarding: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func ensureHostMasquerade(ctx context.Context) error {
	check := exec.CommandContext(ctx, "iptables", "-t", "nat", "-C", "POSTROUTING",
		"-s", "10.99.0.0/16", "-j", "MASQUERADE")
	if check.Run() == nil {
		return nil
	}
	add := exec.CommandContext(ctx, "iptables", "-t", "nat", "-A", "POSTROUTING",
		"-s", "10.99.0.0/16", "-j", "MASQUERADE")
	if out, err := add.CombinedOutput(); err != nil {
		return fmt.Errorf("add masquerade rule: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
