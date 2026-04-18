# Cloud CLI Proxy — Deploy 运维手册

本目录收纳宿主侧部署与运维脚本、配置样例与故障处置指引。

## v3.0 AppArmor override 部署

### 适用范围

仅 **Ubuntu 25.04 及以上**宿主机需要执行本节；Ubuntu 24.04、Debian、CentOS、RHEL 等可跳过。

### 背景

Ubuntu 25.04 起默认加载针对 `fusermount3` 的 AppArmor profile，可能拒绝 `capability dac_override`。受管镜像 v3.0 在容器内以 root 挂载 mergerfs 时需要该能力。若未在宿主追加本地 override，容器内可能出现 `permission denied`，entrypoint 会 fail-fast。

### 修复步骤

在宿主执行（需要 root 权限）：

```bash
sudo tee /etc/apparmor.d/local/fusermount3 >/dev/null <<'APPARMOR'
# Cloud CLI Proxy v3.0 — allow mergerfs DAC override for multi-branch readdir
capability dac_override,
APPARMOR

sudo apparmor_parser -r /etc/apparmor.d/fusermount3
```

修复后建议再执行 `deploy/scripts/host-preflight.sh` 验收，直到 AppArmor fusermount3 检测项输出 OK。

### 自动检测

运行 `deploy/scripts/host-preflight.sh` 会调用 **`check_apparmor_fusermount3`**：

- Ubuntu 25.04+ 且 override 缺失 → 输出 FAIL + 修复指引（**不**自动写文件或执行 `apparmor_parser`）。
- Ubuntu 24.04 / 非 Ubuntu → 自动跳过。
- AppArmor 未启用或 `fusermount3` profile 未加载 → 自动跳过。

调用处使用 `check_apparmor_fusermount3 || true`，检测失败**不会**阻断后续的 `mkdir -p /var/lib/cloud-cli-proxy` 等核心初始化。

> 注：v1 不做自动修复；`AUTO_FIX=1` 由后续阶段引入。

### 参考

- 覆盖路径必须为 **`/etc/apparmor.d/local/fusermount3`**（不是 `docker-default`）。
