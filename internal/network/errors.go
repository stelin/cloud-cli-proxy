package network

import "fmt"

// NetworkErrorType classifies network failures for event recording and operational alerting.
type NetworkErrorType string

const (
	ErrBindingMissing    NetworkErrorType = "net.binding_missing"
	ErrEgressIPMismatch  NetworkErrorType = "net.egress_ip_mismatch"
	ErrDNSLeak           NetworkErrorType = "net.dns_leak"
	ErrLeakNotBlocked    NetworkErrorType = "net.leak_not_blocked"
	ErrEgressUnreachable NetworkErrorType = "net.egress_unreachable"
	ErrTunnelSetupFailed NetworkErrorType = "net.tunnel_setup_failed"
)

// NetworkError represents a structured network failure with metadata suitable for event recording.
type NetworkError struct {
	Type     NetworkErrorType
	Message  string
	HostID   string
	Metadata map[string]any
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

func (e *NetworkError) EventType() string {
	return string(e.Type)
}

func (e *NetworkError) EventMetadata() map[string]any {
	if e.Metadata == nil {
		return map[string]any{"host_id": e.HostID}
	}
	m := make(map[string]any, len(e.Metadata)+1)
	for k, v := range e.Metadata {
		m[k] = v
	}
	m["host_id"] = e.HostID
	return m
}
