#!/usr/bin/env bash
#
# 以伪装身份启动 Claude Code
#
# 用法：
#   ./claude-spoofed.sh                           # 默认伪装
#   SPOOF_HOSTNAME=my-vm ./claude-spoofed.sh      # 自定义主机名
#   SPOOF_DEBUG=1 ./claude-spoofed.sh             # 打印伪装信息
#
# 可配合代理使用（推荐）：
#   HTTPS_PROXY=http://127.0.0.1:8888 ./claude-spoofed.sh
#

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SPOOF_SCRIPT="$SCRIPT_DIR/spoof-fingerprint.js"

if [ ! -f "$SPOOF_SCRIPT" ]; then
  echo "Error: spoof-fingerprint.js not found at $SPOOF_SCRIPT"
  exit 1
fi

# 将 spoof 脚本注入到 Node.js 启动流程
export NODE_OPTIONS="--require $SPOOF_SCRIPT ${NODE_OPTIONS:-}"

# 如果没有设置伪装 hostname，生成一个稳定的
export SPOOF_HOSTNAME="${SPOOF_HOSTNAME:-cloud-vm-$(echo -n "$(whoami)-$(date +%Y%m)" | shasum | cut -c1-6)}"

exec claude "$@"
