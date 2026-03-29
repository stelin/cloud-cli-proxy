package http

import (
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrapScriptHandler(t *testing.T) {
	scriptContent := `#!/usr/bin/env bash
set -euo pipefail

BOOTSTRAP_API="${BOOTSTRAP_API:-http://127.0.0.1:8080}"

printf "用户名: "
read -r username
printf "密码: "
read -r -s password
printf "\n"

response=$(curl --fail --show-error --silent \
  -X POST "${BOOTSTRAP_API}/v1/bootstrap/sessions" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${username}\",\"password\":\"${password}\"}")

error_code=$(echo "$response" | grep -o '"error_code":"[^"]*"' | cut -d'"' -f4)
if [ -n "$error_code" ]; then
  message=$(echo "$response" | grep -o '"message":"[^"]*"' | cut -d'"' -f4)
  printf "\n错误: %s\n" "$message"
  case "$error_code" in
    auth_invalid)   exit 10 ;;
    account_disabled) exit 11 ;;
    account_expired)  exit 12 ;;
    *)              exit 1 ;;
  esac
fi

task_id=$(echo "$response" | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4)
printf "认证通过，主机启动中 (任务: %s)\n" "$task_id"
`

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "cloud-bootstrap.sh")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("returns 200 with script content", func(t *testing.T) {
		handler := NewBootstrapScriptHandler(BootstrapScriptDependencies{
			Logger:     slog.Default(),
			ScriptPath: scriptPath,
		})

		req := httptest.NewRequest(nethttp.MethodGet, "/v1/bootstrap/script", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != nethttp.StatusOK {
			t.Errorf("status = %d, want %d; body = %s", rec.Code, nethttp.StatusOK, rec.Body.String())
		}

		ct := rec.Header().Get("Content-Type")
		if ct != "text/x-shellscript; charset=utf-8" {
			t.Errorf("Content-Type = %q, want %q", ct, "text/x-shellscript; charset=utf-8")
		}

		body := rec.Body.String()
		if !strings.Contains(body, "read -r -s password") {
			t.Error("script must contain 'read -r -s' for silent password input")
		}
		if !strings.Contains(body, "/v1/bootstrap/sessions") {
			t.Error("script must call the bootstrap auth API")
		}
		if !strings.Contains(body, "exit 10") {
			t.Error("script must map auth_invalid to non-zero exit code")
		}
	})

	t.Run("returns 500 when script file missing", func(t *testing.T) {
		handler := NewBootstrapScriptHandler(BootstrapScriptDependencies{
			Logger:     slog.Default(),
			ScriptPath: "/nonexistent/path/script.sh",
		})

		req := httptest.NewRequest(nethttp.MethodGet, "/v1/bootstrap/script", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != nethttp.StatusInternalServerError {
			t.Errorf("status = %d, want %d", rec.Code, nethttp.StatusInternalServerError)
		}
	})
}
