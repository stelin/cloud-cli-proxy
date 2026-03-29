package network

import (
	"encoding/json"
	"net"
)

const (
	TunnelTypeWireGuard = "wireguard"
	TunnelTypeProxy     = "proxy"
)

// TunnelSpec holds WireGuard tunnel parameters for a single host-to-egress link.
type TunnelSpec struct {
	InterfaceName string       // wg-<hostID[:8]>
	PrivateKey    string       // per-host WireGuard private key (base64)
	PeerPublicKey string       // remote egress node public key (base64)
	PeerEndpoint  *net.UDPAddr // remote egress node address
	PresharedKey  string       // optional pre-shared key (base64)
	AllowedIPs    string       // typically "0.0.0.0/0"
	TunnelAddress *net.IPNet   // container-side tunnel IP (e.g. 10.0.0.2/24)
	DNSServer     string       // tunnel-side DNS server IP
}

// ProxySpec holds proxy tunnel parameters for sing-box based egress.
type ProxySpec struct {
	OutboundConfig json.RawMessage // sing-box outbound JSON config
	DNSServer      string          // tunnel-side DNS server IP
}

// EgressConfig carries the validated egress binding for a host.
type EgressConfig struct {
	EgressIPID string      // egress_ips.id
	ExpectedIP string      // expected egress IP address (e.g. "1.2.3.4")
	TunnelType string      // "wireguard" or "proxy"
	Tunnel     *TunnelSpec // WireGuard config; nil for proxy mode
	Proxy      *ProxySpec  // proxy config; nil for wireguard mode
}

// HostNetworkSpec carries everything the network Provider needs to wire a container.
type HostNetworkSpec struct {
	HostID       string
	ContainerPID uint32        // container init PID, populated after docker start
	Egress       *EgressConfig // nil when Provider should skip network setup
}
