#!/bin/sh
set -e

# Sidecar gateway: sing-box tun 模式，拦截所有转发过来的流量。
# net.ipv4.ip_forward=1 由 docker run --sysctl 注入。

exec /usr/local/bin/sing-box run -c /etc/sing-box/config.json
