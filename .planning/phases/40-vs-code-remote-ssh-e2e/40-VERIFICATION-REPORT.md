# Phase 40: VS Code Remote-SSH E2E 验证报告

**验证日期:** 2026-05-08
**测试环境:** (待填写)
**容器名:** (待填写)
**egress 配置:** (待填写: 有 / 无)

---

## UAT 脚本结果

运行命令:
```bash
# 自动检测容器
bash tests/scripts/uat-vscode-remote-ssh.sh --confirm-destructive

# 或指定容器和出口 IP
bash tests/scripts/uat-vscode-remote-ssh.sh --confirm-destructive \
  --container=CONTAINER_NAME \
  --expected-egress-ip=1.2.3.4
```

| 场景 | 状态 | 详情 |
|------|------|------|
| sing-box 进程检测 | (待填写) | |
| 出口 IP 验证 | (待填写) | |
| DNS 泄漏检测 | (待填写) | |
| VS Code Server 检测 | (待填写) | |
| sshd 进程检测 | (待填写) | |
| sing-box 日志检查 | (待填写) | |

JSON 报告: (粘贴完整 JSON 或链接到 benchmarks/ 目录)

---

## 手动测试结果

详细操作步骤见 `40-MANUAL-CHECKLIST.md`。

| 步骤 | 场景 | 状态 | 实际结果 |
|------|------|------|----------|
| 1 | VS Code 连接 | (待填写) | |
| 2 | 文件浏览 | (待填写) | |
| 3 | 终端操作 | (待填写) | |
| 4 | 端口转发 | (待填写) | |
| 5 | 扩展安装 | (待填写) | |
| 6 | 出口 IP 验证 | (待填写) | |
| 7 | DNS 泄漏验证 | (待填写) | |
| 8 | VS Code 更新流量 | (待填写) | |
| 9 | 端口转发出口 IP | (待填写) | |

---

## 发现的问题

(如果发现问题，列出每个问题的描述、原因、修复方案)

## 修复记录

(如果做了代码修复，记录修改的文件和内容)

## 结论

(总结验证结果: 全部通过 / 部分通过 / 需要修复)

---

*Phase: 40-vs-code-remote-ssh-e2e*
*Report created: 2026-05-08*
