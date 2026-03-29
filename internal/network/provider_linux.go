//go:build linux

package network

import "log/slog"

func newLinuxProvider(logger *slog.Logger) Provider {
	return NewContainerProxyProvider(logger)
}
