package network

import (
	"strings"
	"testing"
)

func TestNetworkErrorTypes(t *testing.T) {
	types := []NetworkErrorType{
		ErrBindingMissing,
		ErrEgressIPMismatch,
		ErrDNSLeak,
		ErrLeakNotBlocked,
		ErrEgressUnreachable,
		ErrTunnelSetupFailed,
	}

	for _, typ := range types {
		e := &NetworkError{
			Type:    typ,
			Message: "test message",
			HostID:  "host-abc",
		}

		if !strings.HasPrefix(e.Error(), "["+string(typ)+"]") {
			t.Errorf("Error() for %s: got %q, want prefix [%s]", typ, e.Error(), typ)
		}

		if e.EventType() != string(typ) {
			t.Errorf("EventType() for %s: got %q, want %q", typ, e.EventType(), string(typ))
		}

		meta := e.EventMetadata()
		if meta["host_id"] != "host-abc" {
			t.Errorf("EventMetadata() for %s: missing or wrong host_id, got %v", typ, meta["host_id"])
		}
	}
}

func TestNetworkErrorMetadataMerge(t *testing.T) {
	e := &NetworkError{
		Type:    ErrDNSLeak,
		Message: "leaked to 8.8.8.8",
		HostID:  "host-xyz",
		Metadata: map[string]any{
			"leaked_dns": "8.8.8.8",
		},
	}

	meta := e.EventMetadata()
	if meta["host_id"] != "host-xyz" {
		t.Errorf("host_id not set: got %v", meta["host_id"])
	}
	if meta["leaked_dns"] != "8.8.8.8" {
		t.Errorf("leaked_dns not preserved: got %v", meta["leaked_dns"])
	}
}

func TestNetworkErrorNilMetadata(t *testing.T) {
	e := &NetworkError{
		Type:   ErrBindingMissing,
		HostID: "host-nil",
	}

	meta := e.EventMetadata()
	if meta["host_id"] != "host-nil" {
		t.Errorf("host_id not set for nil metadata: got %v", meta["host_id"])
	}
	if len(meta) != 1 {
		t.Errorf("expected exactly 1 key for nil metadata, got %d", len(meta))
	}
}
