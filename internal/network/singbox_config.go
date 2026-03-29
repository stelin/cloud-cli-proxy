//go:build linux

package network

import (
	"encoding/json"
	"fmt"
)

type singBoxConfig struct {
	Log       singBoxLog        `json:"log"`
	DNS       singBoxDNS        `json:"dns"`
	Inbounds  []json.RawMessage `json:"inbounds"`
	Outbounds []json.RawMessage `json:"outbounds"`
	Route     singBoxRoute      `json:"route"`
}

type singBoxLog struct {
	Level string `json:"level"`
}

type singBoxDNS struct {
	Servers  []singBoxDNSServer `json:"servers"`
	Strategy string             `json:"strategy,omitempty"`
}

type singBoxDNSServer struct {
	Tag    string `json:"tag"`
	Type   string `json:"type"`
	Server string `json:"server"`
	Detour string `json:"detour,omitempty"`
}

type singBoxRoute struct {
	Rules            []singBoxRouteRule `json:"rules"`
	DefaultInterface string             `json:"default_interface,omitempty"`
}

type singBoxRouteRule struct {
	Action   string `json:"action,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

// buildSingBoxConfig produces a complete sing-box JSON configuration from the
// given ProxySpec. The result is ready to be written to /etc/sing-box/config.json.
func buildSingBoxConfig(spec ProxySpec) ([]byte, error) {
	proxyOutbound, err := buildOutbound(spec.OutboundConfig)
	if err != nil {
		return nil, fmt.Errorf("build proxy outbound: %w", err)
	}

	directOutbound, err := json.Marshal(map[string]any{
		"type":           "direct",
		"tag":            "direct",
		"bind_interface": "mgmt0",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal direct outbound: %w", err)
	}

	tunInbound, err := json.Marshal(map[string]any{
		"type":           "tun",
		"tag":            "tun-in",
		"interface_name": "tun0",
		"address":        []string{"172.18.0.1/30"},
		"mtu":            1500,
		"auto_route":     true,
		"strict_route":   true,
		"stack":          "system",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal tun inbound: %w", err)
	}

	cfg := singBoxConfig{
		Log: singBoxLog{Level: "warn"},
		DNS: singBoxDNS{
			Servers: []singBoxDNSServer{
				{
					Tag:    "proxy-dns",
					Type:   "udp",
					Server: spec.DNSServer,
					Detour: "proxy-out",
				},
			},
			Strategy: "ipv4_only",
		},
		Inbounds:  []json.RawMessage{tunInbound},
		Outbounds: []json.RawMessage{proxyOutbound, directOutbound},
		Route: singBoxRoute{
			Rules: []singBoxRouteRule{
				{Action: "sniff"},
				{Protocol: "dns", Action: "hijack-dns"},
			},
			DefaultInterface: "mgmt0",
		},
	}

	return json.MarshalIndent(cfg, "", "  ")
}

// buildOutbound merges the user-provided outbound JSON with required sing-box
// fields (tag and bind_interface) and strips the non-sing-box "dns_server" key.
func buildOutbound(userConfig json.RawMessage) (json.RawMessage, error) {
	var m map[string]any
	if err := json.Unmarshal(userConfig, &m); err != nil {
		return nil, fmt.Errorf("parse outbound config: %w", err)
	}

	delete(m, "dns_server")
	m["tag"] = "proxy-out"
	m["bind_interface"] = "mgmt0"

	return json.Marshal(m)
}
