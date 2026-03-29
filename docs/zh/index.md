---
layout: home

hero:
  name: Cloud CLI Proxy
  text: 容器化 SSH 云主机平台
  tagline: 一条命令，一台云主机，所有流量走指定出口
  actions:
    - theme: brand
      text: 快速开始
      link: /zh/guide/quickstart
    - theme: alt
      text: GitHub
      link: https://github.com/ZaneL1u/cloud-cli-proxy

features:
  - title: 一条命令接入
    details: curl | bash 启动，自动认证、创建容器、建立 SSH 会话
  - title: 全流量强制出口
    details: WireGuard + Linux netns 全隧道，nftables 默认拒绝策略，零泄漏
  - title: 每用户独立环境
    details: Docker 容器隔离，预装 Claude Code、KasmVNC 桌面和 Chromium
  - title: 灵活的出口 IP 管理
    details: 多出口 IP 池，按用户绑定，支持连通性测试
---
