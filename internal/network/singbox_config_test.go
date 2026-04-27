//go:build linux

package network

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildSingBoxConfig(t *testing.T) {
	outbound := json.RawMessage(`{"type":"socks","server":"1.2.3.4","server_port":1080}`)
	spec := ProxySpec{
		OutboundConfig: outbound,
		DNSServer:      "10.0.0.1",
	}

	cfg, err := buildSingBoxConfig(spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal(cfg, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Check top-level keys
	for _, key := range []string{"log", "dns", "inbounds", "outbounds", "route"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("missing top-level key %q", key)
		}
	}

	// Log level should be "warn"
	log, _ := parsed["log"].(map[string]any)
	if level, _ := log["level"].(string); level != "warn" {
		t.Errorf("log.level = %q, want %q", level, "warn")
	}

	// DNS should have strategy ipv4_only
	dns, _ := parsed["dns"].(map[string]any)
	if strategy, _ := dns["strategy"].(string); strategy != "ipv4_only" {
		t.Errorf("dns.strategy = %q, want %q", strategy, "ipv4_only")
	}

	// DNS server should point to spec.DNSServer
	servers, _ := dns["servers"].([]any)
	if len(servers) != 1 {
		t.Fatalf("expected 1 DNS server, got %d", len(servers))
	}
	srv := servers[0].(map[string]any)
	if srv["server"] != "10.0.0.1" {
		t.Errorf("dns server = %q, want %q", srv["server"], "10.0.0.1")
	}
	if srv["detour"] != "proxy-out" {
		t.Errorf("dns detour = %q, want %q", srv["detour"], "proxy-out")
	}

	// Route should have default_interface = "mgmt0"
	route, _ := parsed["route"].(map[string]any)
	if defIface, _ := route["default_interface"].(string); defIface != "mgmt0" {
		t.Errorf("route.default_interface = %q, want %q", defIface, "mgmt0")
	}

	// Route rules should include sniff and hijack-dns
	rules, _ := route["rules"].([]any)
	if len(rules) < 2 {
		t.Fatalf("expected at least 2 route rules, got %d", len(rules))
	}

	// Outbounds: proxy-out + direct
	outbounds, _ := parsed["outbounds"].([]any)
	if len(outbounds) != 2 {
		t.Errorf("expected 2 outbounds, got %d", len(outbounds))
	}

	// Direct outbound should have bind_interface = "mgmt0"
	var directFound bool
	for _, o := range outbounds {
		om, _ := o.(map[string]any)
		if tag, _ := om["tag"].(string); tag == "direct" {
			directFound = true
			if bind, _ := om["bind_interface"].(string); bind != "mgmt0" {
				t.Errorf("direct outbound bind_interface = %q, want %q", bind, "mgmt0")
			}
		}
	}
	if !directFound {
		t.Error("direct outbound not found")
	}
}

func TestBuildOutbound(t *testing.T) {
	userConfig := json.RawMessage(`{"type":"socks","server":"1.2.3.4","server_port":1080,"dns_server":"10.0.0.1","custom_field":"value"}`)

	out, err := buildOutbound(userConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// dns_server should be stripped
	if _, ok := parsed["dns_server"]; ok {
		t.Error("dns_server should have been stripped")
	}

	// tag should be set to "proxy-out"
	if tag, _ := parsed["tag"].(string); tag != "proxy-out" {
		t.Errorf("tag = %q, want %q", tag, "proxy-out")
	}

	// bind_interface should be set to "mgmt0"
	if bind, _ := parsed["bind_interface"].(string); bind != "mgmt0" {
		t.Errorf("bind_interface = %q, want %q", bind, "mgmt0")
	}

	// Original fields should be preserved
	if typ, _ := parsed["type"].(string); typ != "socks" {
		t.Errorf("type = %q, want %q", typ, "socks")
	}
	if server, _ := parsed["server"].(string); server != "1.2.3.4" {
		t.Errorf("server = %q, want %q", server, "1.2.3.4")
	}
	if custom, _ := parsed["custom_field"].(string); custom != "value" {
		t.Errorf("custom_field = %q, want %q", custom, "value")
	}
}

func TestBuildOutbound_InvalidJSON(t *testing.T) {
	userConfig := json.RawMessage(`{invalid json`)
	_, err := buildOutbound(userConfig)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse outbound config") {
		t.Errorf("expected 'parse outbound config' in error, got: %v", err)
	}
}

func TestBuildOutbound_EmptyConfig(t *testing.T) {
	userConfig := json.RawMessage(`{}`)
	out, err := buildOutbound(userConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if tag, _ := parsed["tag"].(string); tag != "proxy-out" {
		t.Errorf("tag = %q, want %q", tag, "proxy-out")
	}
	if bind, _ := parsed["bind_interface"].(string); bind != "mgmt0" {
		t.Errorf("bind_interface = %q, want %q", bind, "mgmt0")
	}
}

func TestBuildSingBoxConfig_TunInbound(t *testing.T) {
	outbound := json.RawMessage(`{"type":"socks","server":"1.2.3.4","server_port":1080}`)
	spec := ProxySpec{
		OutboundConfig: outbound,
		DNSServer:      "10.0.0.1",
	}

	cfg, err := buildSingBoxConfig(spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(cfg, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	inbounds, _ := parsed["inbounds"].([]any)
	if len(inbounds) != 1 {
		t.Fatalf("expected 1 inbound, got %d", len(inbounds))
	}
	tunIn := inbounds[0].(map[string]any)
	if typ, _ := tunIn["type"].(string); typ != "tun" {
		t.Errorf("inbound type = %q, want %q", typ, "tun")
	}
	// Verify stack is "system"
	if stack, _ := tunIn["stack"].(string); stack != "system" {
		t.Errorf("tun stack = %q, want %q", stack, "system")
	}
}

func TestBuildSingBoxConfig_MinimalProxySpec(t *testing.T) {
	// A minimal outbound with just required fields
	outbound := json.RawMessage(`{"type":"http","server":"1.1.1.1","server_port":8080}`)
	spec := ProxySpec{
		OutboundConfig: outbound,
		DNSServer:      "8.8.8.8",
	}

	cfg, err := buildSingBoxConfig(spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(cfg, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Verify it contains proxy-out outbound with the original type
	outbounds, _ := parsed["outbounds"].([]any)
	var proxyOut map[string]any
	for _, o := range outbounds {
		om, _ := o.(map[string]any)
		if tag, _ := om["tag"].(string); tag == "proxy-out" {
			proxyOut = om
			break
		}
	}
	if proxyOut == nil {
		t.Fatal("proxy-out outbound not found")
	}
	if typ, _ := proxyOut["type"].(string); typ != "http" {
		t.Errorf("proxy outbound type = %q, want %q", typ, "http")
	}
}

func TestBuildSingBoxConfig_DifferentDNSServer(t *testing.T) {
	outbound := json.RawMessage(`{"type":"socks","server":"1.2.3.4","server_port":1080}`)
	spec := ProxySpec{
		OutboundConfig: outbound,
		DNSServer:      "1.2.3.4",
	}

	cfg, err := buildSingBoxConfig(spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	json.Unmarshal(cfg, &parsed)
	dns, _ := parsed["dns"].(map[string]any)
	servers, _ := dns["servers"].([]any)
	srv := servers[0].(map[string]any)
	if srv["server"] != "1.2.3.4" {
		t.Errorf("dns server = %q, want %q", srv["server"], "1.2.3.4")
	}
}
