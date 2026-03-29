package tasks

import (
	"strings"
	"testing"
)

func TestBuildSSHHandoffMetadata(t *testing.T) {
	t.Run("contains required SSH access fields", func(t *testing.T) {
		meta := BuildSSHHandoffMetadata("host-abc123", "clouduser")

		required := []string{"ssh_host", "ssh_port", "ssh_user", "host_id"}
		for _, key := range required {
			if _, ok := meta[key]; !ok {
				t.Errorf("missing required key %q", key)
			}
		}

		if meta["ssh_port"] != 22 {
			t.Errorf("ssh_port = %v, want 22", meta["ssh_port"])
		}
		if meta["ssh_user"] != "clouduser" {
			t.Errorf("ssh_user = %v, want %q", meta["ssh_user"], "clouduser")
		}
		if meta["host_id"] != "host-abc123" {
			t.Errorf("host_id = %v, want %q", meta["host_id"], "host-abc123")
		}
	})

	t.Run("same hostID produces stable results", func(t *testing.T) {
		first := BuildSSHHandoffMetadata("host-xyz789", "clouduser")
		second := BuildSSHHandoffMetadata("host-xyz789", "clouduser")

		if first["ssh_host"] != second["ssh_host"] {
			t.Errorf("ssh_host not stable: %v != %v", first["ssh_host"], second["ssh_host"])
		}
		if first["ssh_port"] != second["ssh_port"] {
			t.Errorf("ssh_port not stable: %v != %v", first["ssh_port"], second["ssh_port"])
		}
	})

	t.Run("ssh_host is valid management network IP", func(t *testing.T) {
		meta := BuildSSHHandoffMetadata("host-abc123", "clouduser")
		host, ok := meta["ssh_host"].(string)
		if !ok || host == "" {
			t.Fatal("ssh_host should be a non-empty string")
		}
		if !strings.HasPrefix(host, "10.99.") {
			t.Errorf("ssh_host = %q, want 10.99.x.x management network", host)
		}
	})

	t.Run("default user propagated correctly", func(t *testing.T) {
		meta := BuildSSHHandoffMetadata("host-abc123", "admin")
		if meta["ssh_user"] != "admin" {
			t.Errorf("ssh_user = %v, want %q", meta["ssh_user"], "admin")
		}
	})
}
