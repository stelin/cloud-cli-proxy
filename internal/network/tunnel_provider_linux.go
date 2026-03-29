//go:build linux

package network

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// TunnelProvider implements Provider by wiring each container with a WireGuard
// tunnel, management veth, controlled DNS, and nftables default-deny rules.
type TunnelProvider struct {
	logger *slog.Logger
}

// NewTunnelProvider creates a TunnelProvider backed by the given logger.
func NewTunnelProvider(logger *slog.Logger) *TunnelProvider {
	return &TunnelProvider{logger: logger}
}

// PrepareHost executes the full network wiring pipeline for a container that
// was started with --network=none. Steps run in strict order; any failure
// aborts immediately with no automatic retry (D-04).
func (tp *TunnelProvider) PrepareHost(ctx context.Context, spec HostNetworkSpec) error {
	if spec.Egress == nil {
		return &NetworkError{
			Type:    ErrBindingMissing,
			Message: "PrepareHost called without egress config",
			HostID:  spec.HostID,
		}
	}

	if spec.Egress.Tunnel == nil {
		return &NetworkError{
			Type:     ErrTunnelSetupFailed,
			Message:  "TunnelProvider requires WireGuard tunnel config",
			HostID:   spec.HostID,
			Metadata: map[string]any{"tunnel_type": spec.Egress.TunnelType},
		}
	}

	if spec.Egress.Tunnel.PrivateKey == "" {
		privKey, pubKey, err := GenerateWireGuardKeys()
		if err != nil {
			return &NetworkError{
				Type:    ErrTunnelSetupFailed,
				Message: fmt.Sprintf("generate wg keys: %v", err),
				HostID:  spec.HostID,
			}
		}
		spec.Egress.Tunnel.PrivateKey = privKey
		tp.logger.Info("generated ephemeral WireGuard keys", "host_id", spec.HostID, "public_key", pubKey)
	}

	containerName := fmt.Sprintf("cloudproxy-%s", spec.HostID)
	nsHandle, pid, err := GetContainerNetNS(containerName)
	if err != nil {
		return err
	}
	defer nsHandle.Close()
	spec.ContainerPID = pid

	hostNS, err := netns.Get()
	if err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("get host netns: %v", err),
			HostID:  spec.HostID,
		}
	}
	defer hostNS.Close()

	hostIP, containerIP, err := InjectManagementVeth(hostNS, nsHandle, spec.HostID)
	if err != nil {
		return err
	}
	tp.logger.Info("management veth injected", "host_id", spec.HostID, "host_ip", hostIP, "container_ip", containerIP)

	spec.Egress.Tunnel.InterfaceName = fmt.Sprintf("wg-%s", truncateID(spec.HostID, 8))
	if err := InjectWireGuard(hostNS, nsHandle, *spec.Egress.Tunnel); err != nil {
		return err
	}
	tp.logger.Info("wireguard tunnel injected", "host_id", spec.HostID, "interface", spec.Egress.Tunnel.InterfaceName)

	if err := ConfigureContainerDNS(pid, spec.Egress.Tunnel.DNSServer); err != nil {
		return err
	}
	tp.logger.Info("container DNS configured", "host_id", spec.HostID, "dns_server", spec.Egress.Tunnel.DNSServer)

	wgIfIndex, loIfIndex, mgmtIfIndex, err := resolveContainerIfIndexes(nsHandle, hostNS, spec.Egress.Tunnel.InterfaceName)
	if err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("resolve container interfaces: %v", err),
			HostID:  spec.HostID,
		}
	}

	if err := ApplyFirewallRules(nsHandle, wgIfIndex, loIfIndex, mgmtIfIndex); err != nil {
		return err
	}
	tp.logger.Info("firewall rules applied", "host_id", spec.HostID)

	// Step 8: triple verification (D-09)
	result, verifyErr := VerifyNetworkIntegrity(ctx, spec.ContainerPID, *spec.Egress)
	if verifyErr != nil {
		tp.logger.Error("network verification failed",
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

	tp.logger.Info("network verification passed",
		"host_id", spec.HostID,
		"egress_ip", result.ActualEgressIP,
		"dns_server", result.ActualDNS,
	)

	return nil
}

// CleanupHost removes host-side network artifacts for the given host.
// Errors are logged but not returned to prevent cleanup failures from
// blocking container rebuild operations.
func (tp *TunnelProvider) CleanupHost(_ context.Context, spec HostNetworkSpec) error {
	mgmtName := fmt.Sprintf("mgmt-%s", truncateID(spec.HostID, 8))
	if link, err := netlink.LinkByName(mgmtName); err == nil {
		if err := netlink.LinkDel(link); err != nil {
			tp.logger.Warn("failed to delete management veth", "name", mgmtName, "error", err)
		}
	}

	wgName := fmt.Sprintf("wg-%s", truncateID(spec.HostID, 8))
	if link, err := netlink.LinkByName(wgName); err == nil {
		if err := netlink.LinkDel(link); err != nil {
			tp.logger.Warn("failed to delete wireguard interface", "name", wgName, "error", err)
		}
	}

	return nil
}

// resolveContainerIfIndexes looks up interface indexes inside the container
// namespace for wg, loopback, and management veth.
func resolveContainerIfIndexes(containerNS, hostNS netns.NsHandle, wgIfName string) (wgIdx, loIdx, mgmtIdx int, err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := netns.Set(containerNS); err != nil {
		return 0, 0, 0, err
	}
	defer netns.Set(hostNS) //nolint:errcheck

	wgLink, err := netlink.LinkByName(wgIfName)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("find wg %s: %w", wgIfName, err)
	}

	loLink, err := netlink.LinkByName("lo")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("find lo: %w", err)
	}

	mgmtLink, err := netlink.LinkByName("mgmt0")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("find mgmt0: %w", err)
	}

	return wgLink.Attrs().Index, loLink.Attrs().Index, mgmtLink.Attrs().Index, nil
}
