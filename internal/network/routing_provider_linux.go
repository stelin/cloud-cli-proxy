//go:build linux

package network

import (
	"context"
	"log/slog"
)

// RoutingProvider implements Provider by delegating to TunnelProvider or
// SingBoxProvider based on the egress TunnelType. This is the single injection
// point used by host-agent — callers don't need to know which provider handles
// a given host.
type RoutingProvider struct {
	tunnel  *TunnelProvider
	singbox *SingBoxProvider
	logger  *slog.Logger
}

// NewRoutingProvider creates a RoutingProvider that owns both a TunnelProvider
// and a SingBoxProvider for WireGuard and proxy egress modes respectively.
func NewRoutingProvider(logger *slog.Logger) *RoutingProvider {
	return &RoutingProvider{
		tunnel:  NewTunnelProvider(logger),
		singbox: NewSingBoxProvider(logger),
		logger:  logger,
	}
}

// PrepareHost routes to the correct provider based on TunnelType.
func (rp *RoutingProvider) PrepareHost(ctx context.Context, spec HostNetworkSpec) error {
	if spec.Egress == nil {
		return &NetworkError{
			Type:    ErrBindingMissing,
			Message: "PrepareHost called without egress config",
			HostID:  spec.HostID,
		}
	}

	switch spec.Egress.TunnelType {
	case TunnelTypeProxy:
		rp.logger.Info("routing to sing-box provider", "host_id", spec.HostID, "tunnel_type", spec.Egress.TunnelType)
		return rp.singbox.PrepareHost(ctx, spec)
	default:
		rp.logger.Info("routing to wireguard provider", "host_id", spec.HostID, "tunnel_type", spec.Egress.TunnelType)
		return rp.tunnel.PrepareHost(ctx, spec)
	}
}

// CleanupHost cleans up artifacts from both providers. Since we may not know
// which provider was used (e.g. after a crash), both are called defensively.
func (rp *RoutingProvider) CleanupHost(ctx context.Context, spec HostNetworkSpec) error {
	rp.tunnel.CleanupHost(ctx, spec)
	rp.singbox.CleanupHost(ctx, spec)
	return nil
}
