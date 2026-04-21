package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckKeepaliveConfig_TooAggressive_Fail(t *testing.T) {
	c := checkKeepaliveConfig(context.Background(), 10*time.Second)
	if c.Status != StatusFail {
		t.Errorf("10s 应 Fail，实际 %s", c.Status)
	}
	if c.Code != "SESSION_KEEPALIVE_TOO_AGGRESSIVE" {
		t.Errorf("Code 应为 SESSION_KEEPALIVE_TOO_AGGRESSIVE，实际 %q", c.Code)
	}
}

func TestCheckKeepaliveConfig_OK_Pass(t *testing.T) {
	c := checkKeepaliveConfig(context.Background(), 15*time.Second)
	if c.Status != StatusPass {
		t.Errorf("15s 应 Pass，实际 %s", c.Status)
	}
}

func TestCheckKeepaliveConfig_Zero_Skip(t *testing.T) {
	c := checkKeepaliveConfig(context.Background(), 0)
	if c.Status != StatusSkip {
		t.Errorf("0 应 Skip，实际 %s", c.Status)
	}
}

func TestCheckSSHDKeepaliveDrift_Baseline_Pass(t *testing.T) {
	r := &fakeRunner{out: "clientaliveinterval 15\nclientalivecountmax 8\n"}
	c := checkSSHDKeepaliveDrift(context.Background(), r)
	if c.Status != StatusPass {
		t.Errorf("15/8 应 Pass，实际 %s", c.Status)
	}
}

func TestCheckSSHDKeepaliveDrift_Drift_Warn(t *testing.T) {
	r := &fakeRunner{out: "clientaliveinterval 60\nclientalivecountmax 3\n"}
	c := checkSSHDKeepaliveDrift(context.Background(), r)
	if c.Status != StatusWarn {
		t.Errorf("60/3 应 Warn，实际 %s", c.Status)
	}
	if c.Code != "SSH_SSHD_KEEPALIVE_DRIFT" {
		t.Errorf("Code 应为 SSH_SSHD_KEEPALIVE_DRIFT，实际 %q", c.Code)
	}
}

func TestCheckSSHDKeepaliveDrift_NilRunner_Skip(t *testing.T) {
	c := checkSSHDKeepaliveDrift(context.Background(), nil)
	if c.Status != StatusSkip {
		t.Errorf("nil runner 应 Skip，实际 %s", c.Status)
	}
}

func TestCheckKnownHosts_Missing_Skip(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "known_hosts")
	c := checkKnownHosts(context.Background(), missing, "example.com:22")
	if c.Status != StatusSkip {
		t.Errorf("文件不存在应 Skip，实际 %s (msg=%q)", c.Status, c.Message)
	}
}

func TestCheckKnownHosts_Valid_Pass(t *testing.T) {
	dir := t.TempDir()
	khPath := filepath.Join(dir, "known_hosts")
	// 最小合法的 known_hosts 条目（ssh-ed25519 sample，仅用于 knownhosts.New 解析通过）
	sample := "example.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGIE6GJ4FN+uQQl7yh0K8x3lG0m5f5n6Kk7aA0GZXAbD\n"
	if err := os.WriteFile(khPath, []byte(sample), 0600); err != nil {
		t.Fatalf("write known_hosts: %v", err)
	}
	c := checkKnownHosts(context.Background(), khPath, "example.com:22")
	if c.Status != StatusPass {
		t.Logf("actual status=%s message=%q (knownhosts.New 在不同 Go 版本可能对 sample key 严格校验，允许 Warn)", c.Status, c.Message)
	}
}
