//go:build !linux

package network

import "log/slog"

func newLinuxProvider(logger *slog.Logger) Provider {
	logger.Info("using container-proxy provider (non-Linux)")
	return NewContainerProxyProvider(logger)
}
