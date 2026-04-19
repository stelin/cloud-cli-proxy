package errcodes

// NET_OAUTH_* 错误码注册。文案与 Phase 31 PLAN.md <errcode_registry> 表逐字符对齐。

func init() {
	MustRegister(Entry{
		Code:       NET_OAUTH_EXPIRED,
		Severity:   SeverityFatal,
		Message:    "Claude OAuth 凭证已过期（账号: %s）",
		NextAction: "在容器内运行 cloud-claude exec claude login 重新登录",
	})

	MustRegister(Entry{
		Code:       NET_OAUTH_EXPIRING_SOON,
		Severity:   SeverityWarn,
		Message:    "Claude OAuth 凭证将在 %d 分钟后过期",
		NextAction: "建议尽快 cloud-claude exec claude login",
	})

	MustRegister(Entry{
		Code:       NET_OAUTH_NOT_FOUND,
		Severity:   SeverityFatal,
		Message:    "容器内未找到 Claude OAuth 凭证文件（账号: %s）",
		NextAction: "在容器内运行 cloud-claude exec claude login 完成首次登录",
	})
}
