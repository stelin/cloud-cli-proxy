//go:build linux

package network

import (
	"testing"
	"time"
)

func TestDefaultNsConfig_LocksValues(t *testing.T) {
	cfg := defaultNsConfig()
	if cfg.probeWindow != 300*time.Millisecond {
		t.Errorf("default probeWindow = %v, want 300ms", cfg.probeWindow)
	}
	if cfg.maxRetries != 5 {
		t.Errorf("default maxRetries = %d, want 5", cfg.maxRetries)
	}
}

func TestWithProbeWindow_AppliesPositive(t *testing.T) {
	cfg := defaultNsConfig()
	WithProbeWindow(50 * time.Millisecond)(&cfg)
	if cfg.probeWindow != 50*time.Millisecond {
		t.Errorf("probeWindow = %v, want 50ms", cfg.probeWindow)
	}
}

func TestWithProbeWindow_IgnoresZeroOrNegative(t *testing.T) {
	cfg := defaultNsConfig()
	WithProbeWindow(0)(&cfg)
	WithProbeWindow(-1 * time.Second)(&cfg)
	if cfg.probeWindow != 300*time.Millisecond {
		t.Errorf("probeWindow changed despite invalid input: %v", cfg.probeWindow)
	}
}

func TestWithMaxRetries_AppliesPositive(t *testing.T) {
	cfg := defaultNsConfig()
	WithMaxRetries(2)(&cfg)
	if cfg.maxRetries != 2 {
		t.Errorf("maxRetries = %d, want 2", cfg.maxRetries)
	}
}

func TestWithMaxRetries_IgnoresZeroOrNegative(t *testing.T) {
	cfg := defaultNsConfig()
	WithMaxRetries(0)(&cfg)
	WithMaxRetries(-3)(&cfg)
	if cfg.maxRetries != 5 {
		t.Errorf("maxRetries changed despite invalid input: %d", cfg.maxRetries)
	}
}

func TestOptions_Composable(t *testing.T) {
	cfg := defaultNsConfig()
	opts := []Option{
		WithProbeWindow(100 * time.Millisecond),
		WithMaxRetries(3),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.probeWindow != 100*time.Millisecond || cfg.maxRetries != 3 {
		t.Errorf("composed cfg = %+v", cfg)
	}
}
