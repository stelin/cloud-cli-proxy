package network

import (
	"encoding/json"
	"fmt"
	"net"
)

// extractProxyServer reads the proxy server address and port from an outbound
// config blob. If the server field contains a domain name it is resolved to an
// IPv4 address so the caller can install routes / firewall rules.
func extractProxyServer(outboundConfig json.RawMessage) (serverIP string, serverPort uint16, err error) {
	var addr struct {
		Server     string `json:"server"`
		ServerPort int    `json:"server_port"`
	}
	if err := json.Unmarshal(outboundConfig, &addr); err != nil {
		return "", 0, fmt.Errorf("extract proxy server: %w", err)
	}
	if addr.Server == "" || addr.ServerPort == 0 {
		return "", 0, fmt.Errorf("proxy server address incomplete: server=%q port=%d", addr.Server, addr.ServerPort)
	}

	ip := net.ParseIP(addr.Server)
	if ip == nil {
		resolved, err := net.ResolveIPAddr("ip4", addr.Server)
		if err != nil {
			return "", 0, fmt.Errorf("resolve proxy server domain %q: %w", addr.Server, err)
		}
		ip = resolved.IP
	}

	return ip.String(), uint16(addr.ServerPort), nil
}
