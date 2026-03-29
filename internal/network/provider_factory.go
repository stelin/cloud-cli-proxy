package network

import "log/slog"

func NewProvider(logger *slog.Logger) Provider {
	return newLinuxProvider(logger)
}
