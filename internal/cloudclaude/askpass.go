package cloudclaude

import (
	"fmt"
	"os"
	"path/filepath"
)

// AskpassHelper 创建 ssh-askpass 临时 helper 脚本。
// Mutagen fork 出独立 ssh 子进程时通过 SSH_ASKPASS 拿密码。
// 密码经环境变量 CLOUD_CLAUDE_SSH_PASS 透传，不进入 ps 输出的命令行参数。
//
// 安全约束（CONTEXT D-25 + RESEARCH §1.2）：
//   - 脚本文件 0700（仅当前用户可读 / 执行）
//   - 文件名通过 os.CreateTemp 生成，攻击者不可预测
//   - 父进程退出时 defer Cleanup() 删除脚本
type AskpassHelper struct {
	ScriptPath string
	cleanup    func()
}

// NewAskpassHelper 在 ~/.cloud-claude/run/ 下创建临时 helper 脚本。
// 调用方必须在 Mutagen 命令完全退出后调用 Helper.Cleanup() 删除脚本。
func NewAskpassHelper() (*AskpassHelper, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("无法获取用户主目录: %w", err)
	}
	runDir := filepath.Join(home, ".cloud-claude", "run")
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return nil, fmt.Errorf("创建 askpass 目录失败: %w", err)
	}
	f, err := os.CreateTemp(runDir, "ssh-askpass-*.sh")
	if err != nil {
		return nil, err
	}
	const body = "#!/bin/sh\nprintf '%s' \"$CLOUD_CLAUDE_SSH_PASS\"\n"
	if _, err := f.WriteString(body); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return nil, err
	}
	if err := os.Chmod(f.Name(), 0o700); err != nil {
		os.Remove(f.Name())
		return nil, err
	}
	return &AskpassHelper{
		ScriptPath: f.Name(),
		cleanup:    func() { _ = os.Remove(f.Name()) },
	}, nil
}

// Env 返回供 exec.Cmd.Env 使用的 5 个变量；调用方在
// cmd.Env = append(os.Environ(), helper.Env(password)...) 中合并。
//
// 关键：CLOUD_CLAUDE_SSH_PASS 仅作为环境变量传递，
// 永远不进入 mutagen 命令行 argv（避免 ps 泄漏）。
func (h *AskpassHelper) Env(password string) []string {
	return []string{
		"SSH_ASKPASS=" + h.ScriptPath,
		"SSH_ASKPASS_REQUIRE=force",
		"DISPLAY=:0",
		"SETSID=1",
		"CLOUD_CLAUDE_SSH_PASS=" + password,
	}
}

// Cleanup 删除临时脚本。可重复调用（幂等）。
func (h *AskpassHelper) Cleanup() {
	if h.cleanup != nil {
		h.cleanup()
		h.cleanup = nil
	}
}
