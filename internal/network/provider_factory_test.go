package network

import (
	"log/slog"
	"testing"
)

func TestNewProvider(t *testing.T) {
	logger := slog.Default()
	p := NewProvider(logger)
	if p == nil {
		t.Fatal("NewProvider returned nil")
	}
}

func TestNewLinuxProvider(t *testing.T) {
	logger := slog.Default()
	p := newLinuxProvider(logger)
	if p == nil {
		t.Fatal("newLinuxProvider returned nil")
	}
}
