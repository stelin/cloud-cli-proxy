//go:build e2e

package harness

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newFlakyPredicate 返回一个 predicate，前 failN 次返回错误，第 failN+1 次返回 nil。
// 用 atomic 计数确保并发安全（虽然 WaitFor 内部 ticker 是串行调用）。
func newFlakyPredicate(failN int) (func(context.Context) error, func() int32) {
	var calls atomic.Int32
	return func(_ context.Context) error {
			n := calls.Add(1)
			if n <= int32(failN) {
				return fmt.Errorf("flaky attempt %d", n)
			}
			return nil
		},
		func() int32 { return calls.Load() }
}

// recordingHook 记录 OnWaitForTimeout 是否被调用，并可配置其返回值。
type recordingHook struct {
	called  atomic.Bool
	returns error
}

func (h *recordingHook) OnWaitForTimeout(_ context.Context, _ string, _ error) error {
	h.called.Store(true)
	return h.returns
}

func failIf(t *testing.T, cond bool, format string, args ...any) {
	t.Helper()
	if cond {
		t.Fatalf(format, args...)
	}
}

// TestWaitFor_SucceedsImmediately：predicate 第一次返回 nil → WaitFor 立即返回 nil。
func TestWaitFor_SucceedsImmediately(t *testing.T) {
	pred, callCount := newFlakyPredicate(0)
	err := WaitFor(context.Background(), "test_immediate", pred,
		WithTimeout(2*time.Second),
		WithPollInterval(50*time.Millisecond),
	)
	failIf(t, err != nil, "expected nil err, got %v", err)
	failIf(t, callCount() != 1, "expected 1 predicate call, got %d", callCount())
}

// TestWaitFor_PredicateFailsUntilTimeout：predicate 永远失败 → 超时分支返回错。
// 错误文本必须含 "timed out after 200ms"、"last err: nope" 与 "name=test1"。
func TestWaitFor_PredicateFailsUntilTimeout(t *testing.T) {
	pred := func(_ context.Context) error { return errors.New("nope") }
	err := WaitFor(context.Background(), "test1", pred,
		WithTimeout(200*time.Millisecond),
		WithPollInterval(20*time.Millisecond),
	)
	failIf(t, err == nil, "expected timeout err, got nil")
	msg := err.Error()
	failIf(t, !strings.Contains(msg, "timed out after 200ms"), "missing timeout fragment: %q", msg)
	failIf(t, !strings.Contains(msg, "last err: nope"), "missing last err fragment: %q", msg)
	failIf(t, !strings.Contains(msg, "name=test1"), "missing name fragment: %q", msg)
}

// TestWaitFor_WithTimeoutOverridesDefault：用 50ms 覆盖默认 30s，验证 WithTimeout 真实生效。
func TestWaitFor_WithTimeoutOverridesDefault(t *testing.T) {
	start := time.Now()
	err := WaitFor(context.Background(), "test_to",
		func(_ context.Context) error { return errors.New("nope") },
		WithTimeout(50*time.Millisecond),
		WithPollInterval(10*time.Millisecond),
	)
	elapsed := time.Since(start)
	failIf(t, err == nil, "expected timeout err")
	failIf(t, elapsed >= 500*time.Millisecond, "elapsed %v should be < 500ms (default 30s would have been used)", elapsed)
}

// TestWaitFor_WithPollIntervalOverridesDefault：10ms poll + 100ms timeout
// → predicate 至少被调 5 次（入口 1 + ticker ≥ 4）。
func TestWaitFor_WithPollIntervalOverridesDefault(t *testing.T) {
	pred, callCount := newFlakyPredicate(1000) // 永远失败
	_ = WaitFor(context.Background(), "test_poll", pred,
		WithTimeout(100*time.Millisecond),
		WithPollInterval(10*time.Millisecond),
	)
	failIf(t, callCount() < 5, "expected ≥ 5 predicate calls (entry + ≥ 4 ticks), got %d", callCount())
}

// TestWaitFor_DumpHookCalledOnTimeout：超时分支必须调用 dump hook（hook 返回 nil）。
// 错误文案不应含 "dump hook err"（因为 hook 没报错）。
func TestWaitFor_DumpHookCalledOnTimeout(t *testing.T) {
	hook := &recordingHook{}
	err := WaitFor(context.Background(), "test_hook_ok",
		func(_ context.Context) error { return errors.New("nope") },
		WithTimeout(50*time.Millisecond),
		WithPollInterval(10*time.Millisecond),
		WithDumpHook(hook),
	)
	failIf(t, err == nil, "expected timeout err")
	failIf(t, !hook.called.Load(), "dump hook OnWaitForTimeout should have been called")
	failIf(t, strings.Contains(err.Error(), "dump hook err"), "hook returned nil but err mentions hook err: %q", err.Error())
}

// TestWaitFor_DumpHookErrorMerged：hook 返回 error → 错误文案同时含 last err 与 dump hook err。
func TestWaitFor_DumpHookErrorMerged(t *testing.T) {
	hook := &recordingHook{returns: errors.New("hookboom")}
	err := WaitFor(context.Background(), "test_hook_err",
		func(_ context.Context) error { return errors.New("nope") },
		WithTimeout(50*time.Millisecond),
		WithPollInterval(10*time.Millisecond),
		WithDumpHook(hook),
	)
	failIf(t, err == nil, "expected timeout err")
	msg := err.Error()
	failIf(t, !strings.Contains(msg, "last err: nope"), "missing last err: %q", msg)
	failIf(t, !strings.Contains(msg, "dump hook err: hookboom"), "missing dump hook err: %q", msg)
}

// TestWaitFor_CtxCancelImmediateReturn：外部已取消的 ctx → WaitFor 在入口
// predicate 失败后立刻命中 <-ctx.Done()，快速返回 ctx.Err 包装错。
//
// 故意不用 goroutine + time.Sleep 模拟延迟取消（避免触发 Plan 05
// scripts/lint-no-bare-sleep.sh 守护）。直接在调用前 cancel 即可：
// entry predicate 返回 "flaky" 后，for-select 第一次迭代必然走 <-ctx.Done()。
func TestWaitFor_CtxCancelImmediateReturn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 提前 cancel，模拟 caller 主动放弃

	pred := func(_ context.Context) error { return errors.New("flaky") }

	start := time.Now()
	err := WaitFor(ctx, "test_cancel", pred,
		WithTimeout(5*time.Second),
		WithPollInterval(100*time.Millisecond),
	)
	elapsed := time.Since(start)

	failIf(t, err == nil, "expected ctx canceled err")
	failIf(t, !errors.Is(err, context.Canceled), "err must wrap context.Canceled, got %v", err)
	failIf(t, elapsed >= 1*time.Second, "elapsed %v should be sub-second (caller cancel), not waiting until timeout", elapsed)
}

// TestWaitFor_PollIntervalZeroFallsBackToDefault：WithPollInterval(0) 不应卡死，
// 应回退到 DefaultWaitPollInterval；最终走超时分支。
func TestWaitFor_PollIntervalZeroFallsBackToDefault(t *testing.T) {
	start := time.Now()
	err := WaitFor(context.Background(), "test_zero_poll",
		func(_ context.Context) error { return errors.New("nope") },
		WithTimeout(100*time.Millisecond),
		WithPollInterval(0),
	)
	elapsed := time.Since(start)

	failIf(t, err == nil, "expected timeout err, not deadlock")
	// 100ms 超时 + 默认 500ms poll → 入口 1 次 + 0 次 tick（500ms > 100ms timeout）；总耗时 ~100ms。
	failIf(t, elapsed >= 1*time.Second, "elapsed %v should be ~100ms (timer wins), not deadlocked", elapsed)
}
