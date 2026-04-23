package cloudclaude

import (
	"os"
	"path/filepath"
	"testing"
)

// 单元测试 fixture：
//
//   - 通过 Truncate 创建稀疏文件，避免占用真实磁盘空间（Phase 36 D-21）。
//   - 三场景对应 plan 的 behavior 列表（60MB / ignore / 30MB），覆盖
//     Phase 36 D-06 单文件熔断的边界与 ignore 第一层互补语义。
func createFixtureFile(t *testing.T, path string, size int64) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("创建 fixture 文件失败: %v", err)
	}
	defer f.Close()
	if err := f.Truncate(size); err != nil {
		t.Fatalf("Truncate %s 失败: %v", path, err)
	}
}

// applyTestOversizedFilter 镜像 HotSyncEngine.applyOversizedFilter 的 initialSync 行为，
// 让单测可以在不触达 SSH 链路的前提下断言「scanLocalSyncFiles + 单文件熔断」契约。
//
// 与生产实现保持同一阈值与同一语义（>= maxBytes 进入 oversized 并 delete）。
func applyTestOversizedFilter(localFiles map[string]syncFileState, maxBytes int64) []OversizedFile {
	var oversized []OversizedFile
	for rel, state := range localFiles {
		if state.Size >= maxBytes {
			oversized = append(oversized, OversizedFile{Path: rel, SizeBytes: state.Size})
			delete(localFiles, rel)
		}
	}
	return oversized
}

// TestHotSyncOversized_60MB_NotIgnored 断言 60MB 未 ignore 文件进入 OversizedFiles
// 并被 delete 出 localFiles，对应 plan behavior #1。
func TestHotSyncOversized_60MB_NotIgnored(t *testing.T) {
	dir := t.TempDir()
	createFixtureFile(t, filepath.Join(dir, "bigfile.bin"), 60*1024*1024)

	matcher := NewIgnoreMatcher(dir, nil)
	localFiles, err := scanLocalSyncFiles(dir, matcher)
	if err != nil {
		t.Fatalf("scanLocalSyncFiles 失败: %v", err)
	}

	const maxBytes = 50 * 1024 * 1024
	oversized := applyTestOversizedFilter(localFiles, maxBytes)

	if len(oversized) != 1 {
		t.Fatalf("oversized 应含 1 条，got %d: %+v", len(oversized), oversized)
	}
	if oversized[0].Path != "bigfile.bin" {
		t.Errorf("oversized[0].Path 应为 bigfile.bin，got %q", oversized[0].Path)
	}
	if oversized[0].SizeBytes != 60*1024*1024 {
		t.Errorf("oversized[0].SizeBytes 应为 60MB，got %d", oversized[0].SizeBytes)
	}
	if _, ok := localFiles["bigfile.bin"]; ok {
		t.Error("bigfile.bin 应已从 localFiles 中删除")
	}
}

// TestHotSyncOversized_IgnoreHit_NotCounted 断言 ignore 命中的 60MB 文件不进入 OversizedFiles
// （第一层 ignore 已在 scanLocalSyncFiles 内部跳过，第二层熔断观察不到），
// 对应 plan behavior #2。
func TestHotSyncOversized_IgnoreHit_NotCounted(t *testing.T) {
	dir := t.TempDir()
	createFixtureFile(t, filepath.Join(dir, "ignored_big.bin"), 60*1024*1024)

	matcher := NewIgnoreMatcher(dir, []string{"ignored_big.bin"})
	localFiles, err := scanLocalSyncFiles(dir, matcher)
	if err != nil {
		t.Fatalf("scanLocalSyncFiles 失败: %v", err)
	}

	if _, ok := localFiles["ignored_big.bin"]; ok {
		t.Fatalf("ignore 命中的文件不应出现在 scanLocalSyncFiles 结果中: %+v", localFiles)
	}

	const maxBytes = 50 * 1024 * 1024
	oversized := applyTestOversizedFilter(localFiles, maxBytes)

	if len(oversized) != 0 {
		t.Errorf("ignore 命中的文件不应进入 oversized，got %+v", oversized)
	}
}

// TestHotSyncOversized_30MB_NotOversized 断言 30MB 未 ignore 文件不进入 OversizedFiles
// 且保留在 localFiles，对应 plan behavior #3。
func TestHotSyncOversized_30MB_NotOversized(t *testing.T) {
	dir := t.TempDir()
	createFixtureFile(t, filepath.Join(dir, "medium.bin"), 30*1024*1024)

	matcher := NewIgnoreMatcher(dir, nil)
	localFiles, err := scanLocalSyncFiles(dir, matcher)
	if err != nil {
		t.Fatalf("scanLocalSyncFiles 失败: %v", err)
	}

	const maxBytes = 50 * 1024 * 1024
	oversized := applyTestOversizedFilter(localFiles, maxBytes)

	if len(oversized) != 0 {
		t.Errorf("30MB 文件不应进入 oversized，got %+v", oversized)
	}
	if _, ok := localFiles["medium.bin"]; !ok {
		t.Error("medium.bin 应保留在 localFiles 中")
	}
}
