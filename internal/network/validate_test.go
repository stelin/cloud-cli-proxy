package network

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type mockValidator struct {
	egressIP  EgressIPRecord
	egressErr error
	privKey   string
	pubKey    string
	keysErr   error
}

func (m *mockValidator) GetEgressIPByHost(_ context.Context, _ string) (EgressIPRecord, error) {
	return m.egressIP, m.egressErr
}

func (m *mockValidator) GetHostWgKeys(_ context.Context, _ string) (string, string, error) {
	return m.privKey, m.pubKey, m.keysErr
}

func TestValidateEgressBinding_MissingBinding(t *testing.T) {
	v := &mockValidator{
		egressErr: errors.New("no rows"),
	}

	_, err := ValidateEgressBinding(context.Background(), v, "host-1")
	if err == nil {
		t.Fatal("expected error for missing binding")
	}

	var netErr *NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("expected *NetworkError, got %T", err)
	}
	if netErr.Type != ErrBindingMissing {
		t.Errorf("expected ErrBindingMissing, got %s", netErr.Type)
	}
}

func TestValidateEgressBinding_IncompleteConfig(t *testing.T) {
	v := &mockValidator{
		egressIP: EgressIPRecord{
			ID:        "eip-1",
			IPAddress: "1.2.3.4",
			// WgEndpoint and WgPublicKey are nil → incomplete
			WgAllowedIPs: "0.0.0.0/0",
		},
	}

	_, err := ValidateEgressBinding(context.Background(), v, "host-2")
	if err == nil {
		t.Fatal("expected error for incomplete config")
	}

	var netErr *NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("expected *NetworkError, got %T", err)
	}
	if netErr.Type != ErrTunnelSetupFailed {
		t.Errorf("expected ErrTunnelSetupFailed, got %s", netErr.Type)
	}
}

func TestValidateEgressBinding_Success(t *testing.T) {
	endpoint := "vpn.example.com:51820"
	pubKey := "cGVlcnB1YmtleQ=="
	dns := "10.0.0.1"
	peerAddr := "10.0.0.2/24"

	v := &mockValidator{
		egressIP: EgressIPRecord{
			ID:             "eip-1",
			IPAddress:      "1.2.3.4",
			WgEndpoint:     &endpoint,
			WgPublicKey:    &pubKey,
			WgAllowedIPs:   "0.0.0.0/0",
			WgDNSServer:    &dns,
			WgPeerAddress:  &peerAddr,
		},
		privKey: "cHJpdmtleQ==",
		pubKey:  "bXlwdWJrZXk=",
	}

	cfg, err := ValidateEgressBinding(context.Background(), v, "host-abcdefgh-1234")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.EgressIPID != "eip-1" {
		t.Errorf("EgressIPID: got %q, want %q", cfg.EgressIPID, "eip-1")
	}
	if cfg.ExpectedIP != "1.2.3.4" {
		t.Errorf("ExpectedIP: got %q, want %q", cfg.ExpectedIP, "1.2.3.4")
	}
	if cfg.TunnelType != TunnelTypeWireGuard {
		t.Errorf("TunnelType: got %q, want %q", cfg.TunnelType, TunnelTypeWireGuard)
	}
	if cfg.Tunnel == nil {
		t.Fatal("Tunnel should not be nil for WireGuard type")
	}
	if cfg.Proxy != nil {
		t.Error("Proxy should be nil for WireGuard type")
	}
	if cfg.Tunnel.PeerPublicKey != pubKey {
		t.Errorf("PeerPublicKey: got %q, want %q", cfg.Tunnel.PeerPublicKey, pubKey)
	}
	if cfg.Tunnel.PrivateKey != "cHJpdmtleQ==" {
		t.Errorf("PrivateKey: got %q, want %q", cfg.Tunnel.PrivateKey, "cHJpdmtleQ==")
	}
	if cfg.Tunnel.InterfaceName != "wg-host-abc" {
		t.Errorf("InterfaceName: got %q, want %q", cfg.Tunnel.InterfaceName, "wg-host-abc")
	}
	if cfg.Tunnel.DNSServer != "10.0.0.1" {
		t.Errorf("DNSServer: got %q, want %q", cfg.Tunnel.DNSServer, "10.0.0.1")
	}
}

func TestValidateEgressBinding_KeysError(t *testing.T) {
	endpoint := "vpn.example.com:51820"
	pubKey := "cGVlcnB1YmtleQ=="

	v := &mockValidator{
		egressIP: EgressIPRecord{
			ID:           "eip-1",
			IPAddress:    "1.2.3.4",
			WgEndpoint:   &endpoint,
			WgPublicKey:  &pubKey,
			WgAllowedIPs: "0.0.0.0/0",
		},
		keysErr: errors.New("db error"),
	}

	_, err := ValidateEgressBinding(context.Background(), v, "host-3")
	if err == nil {
		t.Fatal("expected error when keys lookup fails")
	}

	var netErr *NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("expected *NetworkError, got %T", err)
	}
	if netErr.Type != ErrTunnelSetupFailed {
		t.Errorf("expected ErrTunnelSetupFailed, got %s", netErr.Type)
	}
}

func TestValidateEgressBinding_ProxySuccess(t *testing.T) {
	proxyConfig := json.RawMessage(`{"type":"socks","server":"proxy.example.com","server_port":1080,"dns_server":"10.0.0.1"}`)

	v := &mockValidator{
		egressIP: EgressIPRecord{
			ID:           "eip-proxy-1",
			IPAddress:    "5.6.7.8",
			TunnelType:   TunnelTypeProxy,
			ProxyConfig:  proxyConfig,
			WgAllowedIPs: "0.0.0.0/0",
		},
	}

	cfg, err := ValidateEgressBinding(context.Background(), v, "host-proxy-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.TunnelType != TunnelTypeProxy {
		t.Errorf("TunnelType: got %q, want %q", cfg.TunnelType, TunnelTypeProxy)
	}
	if cfg.EgressIPID != "eip-proxy-1" {
		t.Errorf("EgressIPID: got %q, want %q", cfg.EgressIPID, "eip-proxy-1")
	}
	if cfg.ExpectedIP != "5.6.7.8" {
		t.Errorf("ExpectedIP: got %q, want %q", cfg.ExpectedIP, "5.6.7.8")
	}
	if cfg.Tunnel != nil {
		t.Error("Tunnel should be nil for proxy type")
	}
	if cfg.Proxy == nil {
		t.Fatal("Proxy should not be nil for proxy type")
	}
	if cfg.Proxy.OutboundConfig == nil {
		t.Error("Proxy.OutboundConfig should not be nil")
	}
	if cfg.Proxy.DNSServer != "10.0.0.1" {
		t.Errorf("Proxy.DNSServer: got %q, want %q", cfg.Proxy.DNSServer, "10.0.0.1")
	}
}

func TestValidateEgressBinding_ProxyMissingConfig(t *testing.T) {
	v := &mockValidator{
		egressIP: EgressIPRecord{
			ID:           "eip-proxy-2",
			IPAddress:    "5.6.7.8",
			TunnelType:   TunnelTypeProxy,
			ProxyConfig:  nil,
			WgAllowedIPs: "0.0.0.0/0",
		},
	}

	_, err := ValidateEgressBinding(context.Background(), v, "host-proxy-2")
	if err == nil {
		t.Fatal("expected error for proxy type with nil proxy_config")
	}

	var netErr *NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("expected *NetworkError, got %T", err)
	}
	if netErr.Type != ErrTunnelSetupFailed {
		t.Errorf("expected ErrTunnelSetupFailed, got %s", netErr.Type)
	}
}

func TestValidateEgressBinding_ProxyInvalidJSON(t *testing.T) {
	v := &mockValidator{
		egressIP: EgressIPRecord{
			ID:           "eip-proxy-3",
			IPAddress:    "5.6.7.8",
			TunnelType:   TunnelTypeProxy,
			ProxyConfig:  json.RawMessage(`{invalid json`),
			WgAllowedIPs: "0.0.0.0/0",
		},
	}

	_, err := ValidateEgressBinding(context.Background(), v, "host-proxy-3")
	if err == nil {
		t.Fatal("expected error for invalid proxy_config JSON")
	}

	var netErr *NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("expected *NetworkError, got %T", err)
	}
	if netErr.Type != ErrTunnelSetupFailed {
		t.Errorf("expected ErrTunnelSetupFailed, got %s", netErr.Type)
	}
}
