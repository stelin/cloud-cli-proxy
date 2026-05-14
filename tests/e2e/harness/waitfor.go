//go:build e2e

package harness

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// WaitFor 默认参数。在 Plan 03 范围内固定；其它参数请通过 WithTimeout /
// WithPollInterval / WithDumpHook 显式覆盖。
const (
	DefaultWaitTimeout      = 30 * time.Second
	DefaultWaitPollInterval = 500 * time.Millisecond
)

// waitOptions 是 WaitFor 内部状态，对外通过 WaitOpt functional option 修改。
type waitOptions struct {
	timeout      time.Duration
	pollInterval time.Duration
	dumpHook     DumpHook
}

// WaitOpt 是修改 waitOptions 的 functional option。
type WaitOpt func(*waitOptions)

// WithTimeout 覆盖默认 30s 总超时。
func WithTimeout(d time.Duration) WaitOpt { return func(o *waitOptions) { o.timeout = d } }

// WithPollInterval 覆盖默认 500ms 轮询间隔。<=0 时回退到默认值，避免忙等死循环。
func WithPollInterval(d time.Duration) WaitOpt { return func(o *waitOptions) { o.pollInterval = d } }

// WithDumpHook 注入超时分支调用的 DumpHook。默认 NoopDumpHook（不做任何动作）。
// Plan 04 落地真实 artifactDumper 后由 Scenario 在调用前注入。
func WithDumpHook(h DumpHook) WaitOpt { return func(o *waitOptions) { o.dumpHook = h } }

// WaitFor 反复执行 predicate，直到：
//   - predicate 返回 nil → 立即返回 nil
//   - ctx.Done() → 立即返回 ctx 错（不调 dump hook，因为这是上游主动取消）
//   - timeout 到 → 调 dump hook → 返回 timed out 错（含 last err 与可选 hook err）
//
// 默认 timeout=30s, pollInterval=500ms, dumpHook=NoopDumpHook。
//
// 实现严格用 time.NewTimer + time.NewTicker，禁止 time.Sleep（守护脚本
// scripts/lint-no-bare-sleep.sh 在 Plan 05 落地后会强制扫 tests/e2e/）。
func WaitFor(
	ctx context.Context,
	name string,
	predicate func(context.Context) error,
	applyOpts ...WaitOpt,
) error {
	opts := waitOptions{
		timeout:      DefaultWaitTimeout,
		pollInterval: DefaultWaitPollInterval,
		dumpHook:     NoopDumpHook{},
	}
	for _, fn := range applyOpts {
		fn(&opts)
	}
	if opts.pollInterval <= 0 {
		opts.pollInterval = DefaultWaitPollInterval
	}
	if opts.dumpHook == nil {
		opts.dumpHook = NoopDumpHook{}
	}

	timer := time.NewTimer(opts.timeout)
	defer timer.Stop()
	ticker := time.NewTicker(opts.pollInterval)
	defer ticker.Stop()

	// 入口立刻试一次，避免「目标已经就绪却还要等一个 pollInterval」。
	var lastErr error
	if lastErr = predicate(ctx); lastErr == nil {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			// caller 主动取消 → 不调 dump hook
			return fmt.Errorf("waitfor name=%s: %w (last err: %v)", name, ctx.Err(), lastErr)
		case <-timer.C:
			hookErr := opts.dumpHook.OnWaitForTimeout(ctx, name, lastErr)
			msg := fmt.Sprintf("waitfor name=%s: timed out after %s; last err: %v", name, opts.timeout, lastErr)
			if hookErr != nil {
				msg = fmt.Sprintf("%s; dump hook err: %v", msg, hookErr)
			}
			return errors.New(msg)
		case <-ticker.C:
			if lastErr = predicate(ctx); lastErr == nil {
				return nil
			}
		}
	}
}

// ContainerHandle 抽象 testcontainers.Container 中本包需要的子集，
// 让 Plan 02 Scenario 句柄与 raw testcontainers.Container 都能传入。
//
// 真实的 testcontainers.Container.Exec 签名为：
//
//	Exec(ctx, cmd, options ...tcexec.ProcessOption) (int, io.Reader, error)
//
// 本包给 e2e 用例的简化签名只暴露 cmd []string；Scenario 句柄包装一层
// closure 即可适配。
type ContainerHandle interface {
	Logs(ctx context.Context) (io.ReadCloser, error)
	Exec(ctx context.Context, cmd []string) (int, io.Reader, error)
}

// WaitForLog 反复读容器日志，直到出现 substring 子串或超时。
// 实现采用 io.ReadAll + strings.Contains 简单语义，O(N) 但够用；
// Plan 05 CI 的 e2e 用例规模下不会成为瓶颈。
func WaitForLog(ctx context.Context, c ContainerHandle, substring string, opts ...WaitOpt) error {
	name := fmt.Sprintf("log_contains:%q", substring)
	return WaitFor(ctx, name, func(ctx context.Context) error {
		r, err := c.Logs(ctx)
		if err != nil {
			return fmt.Errorf("read logs: %w", err)
		}
		defer func() { _ = r.Close() }()
		b, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("read logs body: %w", err)
		}
		if !strings.Contains(string(b), substring) {
			return fmt.Errorf("substring %q not found in logs (%d bytes)", substring, len(b))
		}
		return nil
	}, opts...)
}

// WaitForPort 反复 dial host:port，直到成功或超时。单次拨号 1s 超时，
// 让总超时与 pollInterval 决定真实重试节奏。
func WaitForPort(ctx context.Context, host string, port int, opts ...WaitOpt) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	name := fmt.Sprintf("port:%s", addr)
	dialer := &net.Dialer{Timeout: 1 * time.Second}
	return WaitFor(ctx, name, func(ctx context.Context) error {
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return err
		}
		_ = conn.Close()
		return nil
	}, opts...)
}

// WaitForHTTP 反复 GET url，直到响应状态码 == expectStatus 或超时。
// 单次请求 2s 超时；走标准 http.Client，不复用连接（避免 keepalive 假阳）。
func WaitForHTTP(ctx context.Context, url string, expectStatus int, opts ...WaitOpt) error {
	name := fmt.Sprintf("http:%s=%d", url, expectStatus)
	client := &http.Client{Timeout: 2 * time.Second}
	return WaitFor(ctx, name, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("build req: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != expectStatus {
			return fmt.Errorf("http %s: got %d want %d", url, resp.StatusCode, expectStatus)
		}
		return nil
	}, opts...)
}

// WaitForExec 反复在容器内执行 cmd，直到 exit code == expectExitCode 或超时。
func WaitForExec(ctx context.Context, c ContainerHandle, cmd []string, expectExitCode int, opts ...WaitOpt) error {
	name := fmt.Sprintf("exec:%v=%d", cmd, expectExitCode)
	return WaitFor(ctx, name, func(ctx context.Context) error {
		code, _, err := c.Exec(ctx, cmd)
		if err != nil {
			return fmt.Errorf("exec %v: %w", cmd, err)
		}
		if code != expectExitCode {
			return fmt.Errorf("exec %v: got exit code %d want %d", cmd, code, expectExitCode)
		}
		return nil
	}, opts...)
}
