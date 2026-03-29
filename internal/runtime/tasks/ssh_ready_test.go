package tasks

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitForSSHReady(t *testing.T) {
	t.Run("succeeds when check passes immediately", func(t *testing.T) {
		cfg := SSHReadyConfig{
			PollInterval: 10 * time.Millisecond,
			Timeout:      100 * time.Millisecond,
			Check: func(_ context.Context, _ string) error {
				return nil
			},
		}

		err := WaitForSSHReady(context.Background(), "test-container", cfg)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("succeeds after retries", func(t *testing.T) {
		var attempts int32
		cfg := SSHReadyConfig{
			PollInterval: 10 * time.Millisecond,
			Timeout:      500 * time.Millisecond,
			Check: func(_ context.Context, _ string) error {
				n := atomic.AddInt32(&attempts, 1)
				if n < 3 {
					return errors.New("connection refused")
				}
				return nil
			},
		}

		err := WaitForSSHReady(context.Background(), "test-container", cfg)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if got := atomic.LoadInt32(&attempts); got < 3 {
			t.Errorf("expected at least 3 attempts, got %d", got)
		}
	})

	t.Run("returns SSHNotReadyError on timeout", func(t *testing.T) {
		cfg := SSHReadyConfig{
			PollInterval: 10 * time.Millisecond,
			Timeout:      50 * time.Millisecond,
			Check: func(_ context.Context, _ string) error {
				return errors.New("connection refused")
			},
		}

		err := WaitForSSHReady(context.Background(), "test-container", cfg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var sshErr *SSHNotReadyError
		if !errors.As(err, &sshErr) {
			t.Fatalf("expected SSHNotReadyError, got %T: %v", err, err)
		}
		if sshErr.Container != "test-container" {
			t.Errorf("container = %q, want %q", sshErr.Container, "test-container")
		}
		if sshErr.Timeout != 50*time.Millisecond {
			t.Errorf("timeout = %v, want %v", sshErr.Timeout, 50*time.Millisecond)
		}
		if sshErr.LastErr == nil {
			t.Error("expected non-nil LastErr")
		}
	})

	t.Run("returns error when context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cfg := SSHReadyConfig{
			PollInterval: 10 * time.Millisecond,
			Timeout:      1 * time.Second,
			Check: func(_ context.Context, _ string) error {
				return errors.New("connection refused")
			},
		}

		err := WaitForSSHReady(ctx, "test-container", cfg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
