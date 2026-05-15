package network

import (
	"os"
	"testing"
)

func TestWorkerContainerName(t *testing.T) {
	got := workerContainerName("my-host")
	want := "cloudproxy-my-host"
	if got != want {
		t.Errorf("workerContainerName = %q, want %q", got, want)
	}
}

func TestGatewayConfigDir_PathSanitization(t *testing.T) {
	if err := os.Unsetenv("DATA_DIR"); err != nil {
		t.Fatalf("unset DATA_DIR: %v", err)
	}
	tests := []struct {
		hostID string
		want   string
	}{
		{"host-1", "/var/lib/cloud-cli-proxy/gateway/host-1"},
		{"host/with/slash", "/var/lib/cloud-cli-proxy/gateway/host/with/slash"},
		{"host..dot", "/var/lib/cloud-cli-proxy/gateway/host..dot"},
		{"host space", "/var/lib/cloud-cli-proxy/gateway/host space"},
	}
	for _, tt := range tests {
		t.Run(tt.hostID, func(t *testing.T) {
			got := GatewayConfigDir(tt.hostID)
			if got != tt.want {
				t.Errorf("GatewayConfigDir(%q) = %q, want %q", tt.hostID, got, tt.want)
			}
		})
	}
}

func TestNewContainerProxyProvider(t *testing.T) {
	p := NewContainerProxyProvider(nil)
	if p == nil {
		t.Fatal("NewContainerProxyProvider returned nil")
	}
	if p.logger != nil {
		t.Error("expected nil logger when passed nil")
	}
}
