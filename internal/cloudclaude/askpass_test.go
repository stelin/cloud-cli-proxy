package cloudclaude

import (
	"os"
	"strings"
	"testing"
)

// 把 HOME 临时切到 t.TempDir，避免污染真实主目录。
func withTempHome(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
}

func Test_AskpassHelper_CreateAndCleanup(t *testing.T) {
	withTempHome(t)

	h, err := NewAskpassHelper()
	if err != nil {
		t.Fatalf("NewAskpassHelper() error = %v", err)
	}
	if h.ScriptPath == "" {
		t.Fatal("ScriptPath 不应为空")
	}

	info, err := os.Stat(h.ScriptPath)
	if err != nil {
		t.Fatalf("stat 脚本失败: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o700 {
		t.Errorf("脚本权限 = %o, want 0700", mode)
	}

	h.Cleanup()
	if _, err := os.Stat(h.ScriptPath); !os.IsNotExist(err) {
		t.Errorf("Cleanup 后脚本仍存在: %v", err)
	}

	h.Cleanup()
}

func Test_AskpassHelper_Env(t *testing.T) {
	withTempHome(t)

	h, err := NewAskpassHelper()
	if err != nil {
		t.Fatalf("NewAskpassHelper() error = %v", err)
	}
	defer h.Cleanup()

	env := h.Env("super-secret-pw")
	if len(env) != 5 {
		t.Fatalf("Env 返回 %d 项，want 5", len(env))
	}

	required := map[string]bool{
		"SSH_ASKPASS=":           false,
		"SSH_ASKPASS_REQUIRE=":   false,
		"DISPLAY=":               false,
		"SETSID=":                false,
		"CLOUD_CLAUDE_SSH_PASS=": false,
	}
	for _, kv := range env {
		for prefix := range required {
			if strings.HasPrefix(kv, prefix) {
				required[prefix] = true
			}
		}
	}
	for prefix, ok := range required {
		if !ok {
			t.Errorf("Env 缺少 %s 项", prefix)
		}
	}

	found := false
	for _, kv := range env {
		if kv == "CLOUD_CLAUDE_SSH_PASS=super-secret-pw" {
			found = true
		}
	}
	if !found {
		t.Error("CLOUD_CLAUDE_SSH_PASS 未透传明文密码")
	}
}

func Test_AskpassHelper_PasswordNotInPath(t *testing.T) {
	withTempHome(t)

	const password = "MyP@ssw0rd!12345"
	h, err := NewAskpassHelper()
	if err != nil {
		t.Fatalf("NewAskpassHelper() error = %v", err)
	}
	defer h.Cleanup()

	if strings.Contains(h.ScriptPath, password) {
		t.Errorf("ScriptPath 含明文密码: %s", h.ScriptPath)
	}

	body, err := os.ReadFile(h.ScriptPath)
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	if strings.Contains(string(body), password) {
		t.Error("脚本内容包含明文密码（应该只引用 $CLOUD_CLAUDE_SSH_PASS）")
	}
}
