package errcodes

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestErrcodesRegistry(t *testing.T) {
	reg := Registry()

	if len(reg) < 15 {
		t.Fatalf("注册表条目不足：want >= 15, got %d", len(reg))
	}

	// PLAN 原表达式 ^[A-Z]+_[A-Z]+_[A-Z0-9]+$ 仅允许 3 段，
	// 但实际 code（如 MOUNT_MUTAGEN_VERSION_SKEW）为 4 段；故放宽为 3+ 段。
	// 见 SUMMARY.md「Deviations from Plan - Rule 1」。
	re := regexp.MustCompile(`^[A-Z]+_[A-Z]+_[A-Z0-9]+(_[A-Z0-9]+)*$`)

	seen := map[Code]struct{}{}
	for code, e := range reg {
		if _, dup := seen[code]; dup {
			t.Errorf("发现重复 code: %s", code)
		}
		seen[code] = struct{}{}

		if string(e.Code) != string(code) {
			t.Errorf("entry.Code (%s) 与 map key (%s) 不一致", e.Code, code)
		}

		if !re.MatchString(string(code)) {
			t.Errorf("code %q 不符合命名规范 ^[A-Z]+_[A-Z]+_[A-Z0-9]+(_[A-Z0-9]+)*$", code)
		}

		if e.Message == "" {
			t.Errorf("code %q Message 不应为空", code)
		}
		if e.NextAction == "" {
			t.Errorf("code %q NextAction 不应为空", code)
		}
		if n := utf8.RuneCountInString(e.NextAction); n > 80 {
			t.Errorf("code %q NextAction 长度 %d > 80 runes: %q", code, n, e.NextAction)
		}
	}
}

func TestFormat_Render(t *testing.T) {
	got := Format(MOUNT_MUTAGEN_VERSION_SKEW, "v0.18.1", "v0.99.99")
	want := "[MOUNT_MUTAGEN_VERSION_SKEW] Mutagen 客户端版本 (v0.18.1) 与容器内 agent 版本 (v0.99.99) 不一致，已降级到 sshfs-only\n  建议: 升级容器镜像到 v3.0.0+ 或重装 cloud-claude"
	if got != want {
		t.Errorf("Format 输出不匹配模板：\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestFormat_UnknownCode(t *testing.T) {
	got := Format("FAKE_CODE_X")
	if !strings.Contains(got, "(unknown code)") {
		t.Errorf("未注册 code 应输出 (unknown code)，实际: %q", got)
	}
	if !strings.Contains(got, "FAKE_CODE_X") {
		t.Errorf("未注册 code 输出应包含原 code 字面量，实际: %q", got)
	}
}

func TestLookup_Hit(t *testing.T) {
	e, ok := Lookup(NET_OAUTH_EXPIRED)
	if !ok {
		t.Fatalf("Lookup(NET_OAUTH_EXPIRED) 应命中，但返回 false")
	}
	if e.Severity != SeverityFatal {
		t.Errorf("NET_OAUTH_EXPIRED Severity want SeverityFatal, got %v", e.Severity)
	}
	if !strings.Contains(e.Message, "OAuth") {
		t.Errorf("NET_OAUTH_EXPIRED Message 应包含 OAuth，实际: %q", e.Message)
	}
}

func TestLookup_Miss(t *testing.T) {
	if _, ok := Lookup("DOES_NOT_EXIST_XX"); ok {
		t.Errorf("Lookup 未注册 code 应返回 false")
	}
}
