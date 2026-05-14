//go:build e2e

// Package harness 提供 Cloud CLI Proxy e2e 套件的可复用基础设施。
//
// 当前包含 BaseSuite（生命周期 hook + 上下文 + 日志器 + 项目根定位）。
// 后续 plan 会陆续补充 Scenario builder（Plan 02）、waitFor helper（Plan 03）、
// artifact dump（Plan 04）。
package harness

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/stretchr/testify/suite"
)

// BaseSuite 是所有 e2e suite 的基础。它提供：
//
//   - Ctx / Cancel：suite 生命周期上下文，SetupSuite 创建、TearDownSuite 取消。
//   - Logger：结构化日志器，key-value 风格与项目主代码（log/slog）保持一致。
//   - ProjectRoot：仓库根目录（基于 runtime.Caller 反推），便于 fixture 文件按
//     项目相对路径定位；禁止任何代码硬编码 /Users/... 这类绝对路径。
//
// 子 suite 通过结构体嵌入 *BaseSuite 复用以上字段，并按需在自己的
// SetupSuite/TearDownSuite 中先调用基类版本：
//
//	func (s *MySuite) SetupSuite() {
//	    s.BaseSuite = &harness.BaseSuite{}
//	    s.BaseSuite.SetT(s.T())
//	    s.BaseSuite.SetupSuite()
//	}
//	func (s *MySuite) TearDownSuite() { s.BaseSuite.TearDownSuite() }
type BaseSuite struct {
	suite.Suite

	Ctx         context.Context
	Cancel      context.CancelFunc
	Logger      *slog.Logger
	ProjectRoot string
}

// SetupSuite 在整个 suite 跑第一个用例之前执行一次。
func (s *BaseSuite) SetupSuite() {
	s.Ctx, s.Cancel = context.WithCancel(context.Background())
	s.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	s.ProjectRoot = projectRootFromCaller()
	s.Logger.Info("e2e BaseSuite ready", "project_root", s.ProjectRoot)
}

// TearDownSuite 在整个 suite 跑完最后一个用例之后执行一次。
func (s *BaseSuite) TearDownSuite() {
	if s.Cancel != nil {
		s.Cancel()
	}
}

// SetupTest 留作 Plan 02+ 的 Scenario 注入点（当前为空）。
func (s *BaseSuite) SetupTest() {}

// TearDownTest 留作 Plan 04 失败 artifact dump 钩点（当前为空）。
// Plan 04 会在此检查 s.T().Failed() 并调用 dump.Collect(...)。
func (s *BaseSuite) TearDownTest() {}

// projectRootFromCaller 通过 runtime.Caller 反推仓库根目录（go.mod 所在目录）。
// 不依赖 git，也不依赖 CWD（go test 在不同目录下 CWD 不稳定）。
func projectRootFromCaller() string {
	_, file, _, _ := runtime.Caller(0) // tests/e2e/harness/suite.go
	// 向上 3 级回到仓库根：harness/ → e2e/ → tests/ → <root>
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
