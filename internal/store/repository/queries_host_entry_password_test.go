package repository

import (
	"strings"
	"testing"
)

// TestAllHostReadQueriesIncludeEntryPassword 锁定 Phase 29.1 的根因修复：
// 任何读 Host 的 SQL 必须把 entry_password 列纳入 SELECT，
// 否则下游 runtime / worker / admin 链路会把容器密码降级为 "workspace"。
//
// 测试只断言 SQL 文本契约（字符串 strings.Contains），不连接 DB、
// 不打印任何密码明文样本（线上敏感字段，T-29.1-05-log 防泄漏）。
func TestAllHostReadQueriesIncludeEntryPassword(t *testing.T) {
	queries := map[string]string{
		"getHostSQL":                  getHostSQL,
		"listHostsSQL":                listHostsSQL,
		"listHostsByUserIDSQL":        listHostsByUserIDSQL,
		"listHostsWithUsernameSQL":    listHostsWithUsernameSQL,
		"listRunningHostsSQL":         listRunningHostsSQL,
		"listRunningHostsByUserIDSQL": listRunningHostsByUserIDSQL,
	}
	for name, q := range queries {
		if !strings.Contains(q, "entry_password") {
			t.Errorf("%s 必须包含 entry_password 列（Phase 29.1 根因修复）；实际 SQL:\n%s", name, q)
		}
	}
}
