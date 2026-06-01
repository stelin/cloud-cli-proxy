# Phase 53 — Deferred Items

执行 Plan 53-01 时发现，但**不在本 Plan scope 内**的预先存在问题。

## D-53-PRE-1: Claude Code 安装段在本地 docker build 中失败

- **发现时机:** Plan 53-01 V1（`docker build -t managed-user:v4-dev`）
- **现象:**
  - `curl https://claude.ai/install.sh` 三次重试均 HTTP 403
  - GitHub fallback `https://github.com/anthropics/claude-code/releases/download/v2.1.142/claude-code-linux-aarch64.tar.gz` HTTP 404（release tarball 命名/结构变了）
  - 在 `RUN ... claude --version` 这一步退出 22
- **根因:** v3.x 既有 Claude Code 安装段（Dockerfile L127-170）依赖的远端资源已经发生变化，与 Plan 53-01 改动无关。
- **范围归属:** SCOPE BOUNDARY — 这是 pre-existing 问题，Plan 53-01 不修。
- **建议处理路径:**
  - 单独立 Plan / Phase 修 Claude Code 安装段（升级 fallback URL/解包路径，或镜像源 mirror）
  - 或在 Phase 53-02 / 53-03 把 Claude Code 安装迁移到 entrypoint 运行时
  - 或在 CI 上补一个有外网穿透 + 正确 release URL 的 build job
- **不阻塞本 Plan 验收:** Dockerfile syntax 已通过 `docker buildx build --check`（0 warning），且 build 中前 18 步全部通过，覆盖了本 Plan 改动的 T2/T3 全部、T1/T4 在 syntax 层面已校验、T5 是注释改动。
