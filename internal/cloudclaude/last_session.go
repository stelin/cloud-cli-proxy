package cloudclaude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LastSessionSnapshot 是 ~/.cloud-claude/last-session.json 的 schema_version=1 结构。
//
// 由 mount_strategy.go 在每次 mount 路径完成（成功或失败均写）时落盘。
// Phase 34 doctor 第一屏读取展示降级历史与 conflict 计数。
type LastSessionSnapshot struct {
	SchemaVersion       int             `json:"schema_version"`
	Timestamp           time.Time       `json:"timestamp"`
	IntendedMode        string          `json:"intended_mode"`
	ActualMode          string          `json:"actual_mode"`
	DowngradeChain      []DowngradeStep `json:"downgrade_chain"`
	ConflictCount       int             `json:"conflict_count"`
	ClaudeAccountID     string          `json:"claude_account_id,omitempty"`
	ImageVersion        string          `json:"image_version,omitempty"`
	APFSCaseInsensitive bool            `json:"apfs_case_insensitive"`
}

// DowngradeStep 描述单次降级动作。
type DowngradeStep struct {
	From          string `json:"from"`
	To            string `json:"to"`
	ReasonCode    string `json:"reason_code"`
	ReasonMessage string `json:"reason_message"`
}

// WriteLastSession 把 snap 序列化为 JSON 后原子写入 path。
//
// 行为：
//  1. 父目录不存在则 0700 创建（与 ConfigDir 一致）
//  2. SchemaVersion 字段为 0 时强制设置为 1（防御调用方遗漏）
//  3. 写到 path+".tmp" 后 os.Rename → 原子替换
//  4. 失败仅返回 error，不 panic（调用方在 stderr warning 但不阻断 mount 路径）
func WriteLastSession(path string, snap LastSessionSnapshot) error {
	if snap.SchemaVersion == 0 {
		snap.SchemaVersion = 1
	}
	if snap.Timestamp.IsZero() {
		snap.Timestamp = time.Now().UTC()
	}
	if snap.DowngradeChain == nil {
		snap.DowngradeChain = []DowngradeStep{}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("创建 last-session 目录失败: %w", err)
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 last-session 失败: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("写 last-session 临时文件失败: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename last-session 失败: %w", err)
	}
	return nil
}
