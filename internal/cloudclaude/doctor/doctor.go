// Package doctor — Phase 34 Plan 02：cloud-claude doctor 五维度自检框架。
//
// Entry point: RunDoctor(ctx, opts) -> *Report
// 串行执行：network → auth → ssh → mount → disk（CONTEXT D-07）
// 远端 check 走 RemoteRunner interface（D-20 lazy connect + StatusSkip 降级）
package doctor

import (
	"context"
	"time"
)

// Status 对标 CONTEXT D-15 JSON schema 字面量 "pass"|"warn"|"fail"|"skip"。
type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
	StatusSkip Status = "skip"
)

// Options 是 RunDoctor 的入参（CONTEXT D-01）。
type Options struct {
	Domain       string        // "network" / "auth" / "ssh" / "mount" / "disk" / "all"（cobra 已校验）
	Fix          bool          // Plan 03 落实；Plan 02 仅注册 flag
	Verbose      bool          // --verbose 每 check 打印时间戳 + Details 全字段
	JSON         bool          // --json 切换 RenderJSON（Text 不输出）
	NoColor      bool          // 显式关闭颜色（叠加在 colors.ColorEnabled 之上）
	Yes          bool          // Plan 03 用：confirmDestructive 跳过交互
	CheckTimeout time.Duration // 单 check timeout；0 则取默认 5s（Verbose 30s）
}

// DowngradeBanner 是第一屏降级历史（CONTEXT D-13 / RESEARCH §5.1）。
type DowngradeBanner struct {
	SnapshotAgeSeconds int64           `json:"snapshot_age_seconds"`
	IntendedMode       string          `json:"intended_mode,omitempty"`
	ActualMode         string          `json:"actual_mode,omitempty"`
	DowngradeChain     []DowngradeStep `json:"downgrade_chain,omitempty"`
	ConflictCount      int             `json:"conflict_count,omitempty"`
	ReconnectCount     int             `json:"reconnect_count,omitempty"`
	TmuxSession        string          `json:"tmux_session,omitempty"`
	ClientRole         string          `json:"client_role,omitempty"`
}

// DowngradeStep 源自 cloudclaude.LastSessionSnapshot.DowngradeChain；doctor 只读不写。
type DowngradeStep struct {
	From          string `json:"from"`
	To            string `json:"to"`
	ReasonCode    string `json:"reason_code,omitempty"`
	ReasonMessage string `json:"reason_message,omitempty"`
}

// Summary 汇总统计（CONTEXT D-13 末段）。
type Summary struct {
	Total int `json:"total"`
	Pass  int `json:"pass"`
	Warn  int `json:"warn"`
	Fail  int `json:"fail"`
	Skip  int `json:"skip"`
}

// Report 是 RunDoctor 返回值 + --json 序列化对象（schema_version=1 锁死，RESEARCH §5.1）。
type Report struct {
	SchemaVersion    int              `json:"schema_version"` // 硬编码 1，不带 omitempty（jq 依赖）
	StartedAt        time.Time        `json:"started_at"`
	DurationMS       int64            `json:"duration_ms"`
	CloudClaudeVer   string           `json:"cloud_claude_version,omitempty"`
	RemoteImageVer   string           `json:"remote_image_version,omitempty"`
	DowngradeHistory *DowngradeBanner `json:"downgrade_history,omitempty"`
	Summary          Summary          `json:"summary"`
	Checks           []Check          `json:"checks"`
}

// RunDoctor 顶层入口。Task 2.2 只创建占位；Task 2.10 实现真实主流程。
//
// 执行顺序（CONTEXT D-07）：network → auth → ssh → mount → disk。
// 远端 conn lazy 建立（CONTEXT D-20），连不上时 RequiresRemote=true 的 check 全部 StatusSkip。
func RunDoctor(ctx context.Context, opts Options) (*Report, error) {
	// Task 2.2 placeholder — 真实主流程由 Task 2.10 落地。
	return &Report{SchemaVersion: 1, StartedAt: time.Now(), Checks: []Check{}}, nil
}
