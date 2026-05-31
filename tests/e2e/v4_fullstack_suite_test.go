//go:build e2e

// Package e2e 提供 v4.0 E2E 测试套件。
//
// 使用方式：
//
//	# 针对已有 compose 环境运行全套 E2E
//	go test -tags=e2e ./tests/e2e/ -v -timeout 15m
//
//	# 只运行烟雾测试
//	go test -tags=e2e ./tests/e2e/ -run SmokeSuite -v
//
//	# 只运行网络验证
//	go test -tags=e2e ./tests/e2e/ -run NetVerifySuite -v
package e2e
