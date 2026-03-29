package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
)

// EgressIPRecord is a storage-agnostic representation of an egress IP row.
// The network package defines its own record type so it never imports the store package.
type EgressIPRecord struct {
	ID             string
	IPAddress      string
	TunnelType     string
	ProxyConfig    json.RawMessage
	WgEndpoint     *string
	WgPublicKey    *string
	WgPresharedKey *string
	WgAllowedIPs   string
	WgDNSServer    *string
	WgPeerAddress  *string
}

// EgressValidator abstracts the repository queries needed for binding validation.
type EgressValidator interface {
	GetEgressIPByHost(ctx context.Context, hostID string) (EgressIPRecord, error)
	GetHostWgKeys(ctx context.Context, hostID string) (wgPrivateKey string, wgPublicKey string, err error)
}

// ValidateEgressBinding checks that the host has a valid egress IP binding with
// complete tunnel configuration. Returns a fully-populated EgressConfig on success
// or a typed NetworkError on failure.
func ValidateEgressBinding(ctx context.Context, v EgressValidator, hostID string) (EgressConfig, error) {
	record, err := v.GetEgressIPByHost(ctx, hostID)
	if err != nil {
		return EgressConfig{}, &NetworkError{
			Type:    ErrBindingMissing,
			Message: "no egress IP bound to host",
			HostID:  hostID,
		}
	}

	switch record.TunnelType {
	case TunnelTypeProxy:
		return validateProxyBinding(record, hostID)
	default:
		return validateWireGuardBinding(ctx, v, record, hostID)
	}
}

func validateWireGuardBinding(ctx context.Context, v EgressValidator, record EgressIPRecord, hostID string) (EgressConfig, error) {
	if record.WgEndpoint == nil || *record.WgEndpoint == "" ||
		record.WgPublicKey == nil || *record.WgPublicKey == "" {
		return EgressConfig{}, &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: "incomplete tunnel config: missing endpoint or public key",
			HostID:  hostID,
			Metadata: map[string]any{
				"egress_ip_id":   record.ID,
				"has_endpoint":   record.WgEndpoint != nil && *record.WgEndpoint != "",
				"has_public_key": record.WgPublicKey != nil && *record.WgPublicKey != "",
			},
		}
	}

	privKey, _, err := v.GetHostWgKeys(ctx, hostID)
	if err != nil {
		return EgressConfig{}, &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: "failed to read host WireGuard keys",
			HostID:  hostID,
		}
	}

	endpoint, _ := net.ResolveUDPAddr("udp", *record.WgEndpoint)

	var tunnelAddr *net.IPNet
	if record.WgPeerAddress != nil && *record.WgPeerAddress != "" {
		_, tunnelAddr, _ = net.ParseCIDR(*record.WgPeerAddress)
	}

	var psk string
	if record.WgPresharedKey != nil {
		psk = *record.WgPresharedKey
	}

	var dns string
	if record.WgDNSServer != nil {
		dns = *record.WgDNSServer
	}

	return EgressConfig{
		EgressIPID: record.ID,
		ExpectedIP: record.IPAddress,
		TunnelType: TunnelTypeWireGuard,
		Tunnel: &TunnelSpec{
			InterfaceName: "wg-" + truncateID(hostID, 8),
			PrivateKey:    privKey,
			PeerPublicKey: *record.WgPublicKey,
			PeerEndpoint:  endpoint,
			PresharedKey:  psk,
			AllowedIPs:    record.WgAllowedIPs,
			TunnelAddress: tunnelAddr,
			DNSServer:     dns,
		},
	}, nil
}

func validateProxyBinding(record EgressIPRecord, hostID string) (EgressConfig, error) {
	if record.ProxyConfig == nil || len(record.ProxyConfig) == 0 {
		return EgressConfig{}, &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: "proxy type requires non-empty proxy_config",
			HostID:  hostID,
			Metadata: map[string]any{"egress_ip_id": record.ID},
		}
	}

	var parsed map[string]any
	if err := json.Unmarshal(record.ProxyConfig, &parsed); err != nil {
		return EgressConfig{}, &NetworkError{
			Type:    ErrTunnelSetupFailed,
			Message: fmt.Sprintf("proxy_config is not valid JSON: %v", err),
			HostID:  hostID,
			Metadata: map[string]any{"egress_ip_id": record.ID},
		}
	}

	dnsServer, _ := parsed["dns_server"].(string)

	return EgressConfig{
		EgressIPID: record.ID,
		ExpectedIP: record.IPAddress,
		TunnelType: TunnelTypeProxy,
		Proxy: &ProxySpec{
			OutboundConfig: record.ProxyConfig,
			DNSServer:      dnsServer,
		},
	}, nil
}

func truncateID(id string, n int) string {
	if len(id) <= n {
		return id
	}
	return id[:n]
}
