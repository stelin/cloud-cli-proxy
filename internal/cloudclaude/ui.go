package cloudclaude

import (
	"fmt"
	"io"
	"strings"
)

// ProgressUI 提供极客风终端进度展示。
// 支持扫描阶段实时刷新、hot/cold 分布条、同步进度条。
type ProgressUI struct {
	w        io.Writer
	enabled  bool
	barWidth int
}

// NewProgressUI 创建 ProgressUI。
// noColor=true 或 NO_COLOR 环境变量或 w 非 TTY 时禁用颜色与单行刷新。
func NewProgressUI(w io.Writer, noColor bool) *ProgressUI {
	enabled := false
	if fh, ok := w.(fdHolder); ok {
		enabled = ColorEnabled(noColor, fh)
	}
	return &ProgressUI{
		w:        w,
		enabled:  enabled,
		barWidth: 36,
	}
}

// Stage 输出阶段标题（如 "━━ (1/3) 热同步 ━━"）。
func (p *ProgressUI) Stage(stage string) {
	fmt.Fprintf(p.w, "\n━━ %s ━━\n", stage)
}

// Scanning 在单行实时刷新当前扫描到的文件和累计数量。
func (p *ProgressUI) Scanning(file string, count int) {
	if !p.enabled {
		return
	}
	fmt.Fprintf(p.w, "\r\033[2K  扫描中… %-40s  已发现 %s 个文件",
		truncate(file, 40), formatInt(count))
}

// ScanDone 结束扫描阶段，输出最终统计并换行。
func (p *ProgressUI) ScanDone(total int) {
	fmt.Fprintf(p.w, "\r\033[2K  扫描完成，共 %s 个文件\n", formatInt(total))
}

// Distribution 输出 hot/cold 文件分布条（固定输出，不刷新）。
func (p *ProgressUI) Distribution(hotFiles, hotBytes, coldFiles, coldBytes int64) {
	total := hotFiles + coldFiles
	hotPct := int64(0)
	if total > 0 {
		hotPct = hotFiles * 100 / total
	}

	hotBlocks := int(hotPct) * p.barWidth / 100
	if hotBlocks > p.barWidth {
		hotBlocks = p.barWidth
	}
	coldBlocks := p.barWidth - hotBlocks
	if coldBlocks < 0 {
		coldBlocks = 0
	}

	hotBar := strings.Repeat("█", hotBlocks)
	coldBar := strings.Repeat("█", coldBlocks)

	// 空行 + 标题
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "  文件分布")

	// 进度条
	if p.enabled {
		fmt.Fprintf(p.w, "  [%s%s%s%s%s]\n",
			AnsiOrange, hotBar, AnsiBlue, coldBar, AnsiReset)
	} else {
		fmt.Fprintf(p.w, "  [%s%s]\n", hotBar, coldBar)
	}

	// 统计文字
	hotText := fmt.Sprintf("hot  %d%%  (%s files, %s)", hotPct, formatInt(int(hotFiles)), formatBytes(hotBytes))
	coldText := fmt.Sprintf("cold %d%%  (%s files, %s)", 100-hotPct, formatInt(int(coldFiles)), formatBytes(coldBytes))

	if p.enabled {
		fmt.Fprintf(p.w, "    %s%s%s    %s%s%s\n",
			AnsiOrange, hotText, AnsiReset,
			AnsiBlue, coldText, AnsiReset)
	} else {
		fmt.Fprintf(p.w, "    %s    %s\n", hotText, coldText)
	}
}

// Syncing 在单行实时刷新 hot 同步进度。
func (p *ProgressUI) Syncing(done, total int, currentFile string) {
	if !p.enabled {
		return
	}
	pct := 0
	if total > 0 {
		pct = done * 100 / total
	}
	filled := pct * p.barWidth / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.barWidth-filled)

	fmt.Fprintf(p.w, "\r\033[2K  同步中… [%s] %3d%%  (%s/%s)  → %s",
		bar, pct, formatInt(done), formatInt(total), truncate(currentFile, 35))
}

// SyncDone 结束同步阶段，输出 100% 并换行。
func (p *ProgressUI) SyncDone(done, total int) {
	if p.enabled {
		bar := strings.Repeat("█", p.barWidth)
		fmt.Fprintf(p.w, "\r\033[2K  [%s] 100%%  (%s/%s)  同步完成\n",
			bar, formatInt(done), formatInt(total))
	} else {
		fmt.Fprintf(p.w, "  热同步完成，共 %s 个文件\n", formatInt(done))
	}
}

// formatInt 千位分隔。
func formatInt(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// formatBytes 人类可读字节。
func formatBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// truncate 截断字符串，保留头尾，中间用 … 连接。
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	head := maxLen/2 - 1
	tail := maxLen - head - 3
	return string(runes[:head]) + "…" + string(runes[len(runes)-tail:])
}
