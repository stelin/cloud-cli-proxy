//go:build linux

package network

import (
	"fmt"
	"net"
	"runtime"
	"time"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const wgKeepaliveInterval = 25 * time.Second

// GenerateWireGuardKeys creates a new WireGuard private/public key pair.
func GenerateWireGuardKeys() (privateKey, publicKey string, err error) {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", "", fmt.Errorf("generate wireguard key: %w", err)
	}
	return key.String(), key.PublicKey().String(), nil
}

// InjectWireGuard creates a WireGuard interface in the host namespace,
// configures it with the given TunnelSpec, then moves it into the container
// namespace. The encrypted UDP socket remains in the host namespace
// (WireGuard birthplace namespace mechanism).
func InjectWireGuard(hostNS, containerNS netns.NsHandle, spec TunnelSpec) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := netns.Set(hostNS); err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("set host netns for wg: %v", err),
		}
	}

	wgLink := &netlink.Wireguard{
		LinkAttrs: netlink.LinkAttrs{Name: spec.InterfaceName},
	}
	if err := netlink.LinkAdd(wgLink); err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("create wg interface %s: %v", spec.InterfaceName, err),
		}
	}

	privKey, err := wgtypes.ParseKey(spec.PrivateKey)
	if err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("parse wg private key: %v", err),
		}
	}

	peerPub, err := wgtypes.ParseKey(spec.PeerPublicKey)
	if err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("parse wg peer public key: %v", err),
		}
	}

	_, allowedNet, err := net.ParseCIDR(spec.AllowedIPs)
	if err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("parse allowed IPs %s: %v", spec.AllowedIPs, err),
		}
	}

	keepalive := wgKeepaliveInterval
	peerCfg := wgtypes.PeerConfig{
		PublicKey:                   peerPub,
		Endpoint:                   spec.PeerEndpoint,
		AllowedIPs:                 []net.IPNet{*allowedNet},
		PersistentKeepaliveInterval: &keepalive,
	}

	if spec.PresharedKey != "" {
		psk, err := wgtypes.ParseKey(spec.PresharedKey)
		if err != nil {
			return &NetworkError{
				Type:    ErrTunnelSetupFailed,
				Message: fmt.Sprintf("parse wg preshared key: %v", err),
			}
		}
		peerCfg.PresharedKey = &psk
	}

	client, err := wgctrl.New()
	if err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("create wgctrl client: %v", err),
		}
	}
	defer client.Close()

	wgConfig := wgtypes.Config{
		PrivateKey: &privKey,
		Peers:      []wgtypes.PeerConfig{peerCfg},
	}
	if err := client.ConfigureDevice(spec.InterfaceName, wgConfig); err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("configure wg device %s: %v", spec.InterfaceName, err),
		}
	}

	link, err := netlink.LinkByName(spec.InterfaceName)
	if err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("find wg link %s: %v", spec.InterfaceName, err),
		}
	}
	if err := netlink.LinkSetNsFd(link, int(containerNS)); err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("move wg to container netns: %v", err),
		}
	}

	if err := netns.Set(containerNS); err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("set container netns for wg: %v", err),
		}
	}
	defer netns.Set(hostNS) //nolint:errcheck

	containerLink, err := netlink.LinkByName(spec.InterfaceName)
	if err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("find wg in container netns: %v", err),
		}
	}

	if spec.TunnelAddress != nil {
		if err := netlink.AddrAdd(containerLink, &netlink.Addr{IPNet: spec.TunnelAddress}); err != nil {
			return &NetworkError{
				Type:    ErrTunnelSetupFailed,
				Message: fmt.Sprintf("add tunnel addr to wg: %v", err),
			}
		}
	}

	if err := netlink.LinkSetUp(containerLink); err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("bring up wg in container: %v", err),
		}
	}

	defaultRoute := &netlink.Route{
		LinkIndex: containerLink.Attrs().Index,
		Dst:       &net.IPNet{IP: net.IPv4zero, Mask: net.CIDRMask(0, 32)},
	}
	if err := netlink.RouteAdd(defaultRoute); err != nil {
		return &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("add default route via wg: %v", err),
		}
	}

	return nil
}
