---
phase: quick
plan: 260328-u4q
subsystem: docs
tags: [documentation, vitepress, github-pages, readme]
key-files:
  created:
    - docs/.vitepress/config.mts
    - docs/index.md
    - docs/zh/index.md
    - docs/en/index.md
    - docs/zh/guide/quickstart.md
    - docs/zh/guide/deployment.md
    - docs/zh/guide/configuration.md
    - docs/zh/guide/architecture.md
    - docs/zh/reference/api.md
    - docs/zh/reference/faq.md
    - docs/en/guide/quickstart.md
    - docs/en/guide/deployment.md
    - docs/en/guide/configuration.md
    - docs/en/guide/architecture.md
    - docs/en/reference/api.md
    - docs/en/reference/faq.md
    - .github/workflows/docs.yml
  modified:
    - README.md
    - README.en.md
    - package.json
    - pnpm-lock.yaml
    - .gitignore
  removed:
    - docs/deployment-guide.md
    - docs/operations-manual.md
    - docs/recovery-runbook.md
decisions:
  - "VitePress 配置使用 .mts 扩展名以兼容 ESM"
  - "中文为默认语言（/zh/ 路径），英文为备选（/en/ 路径）"
  - "旧 docs/*.md 内容迁移到 VitePress 结构后删除原文件"
metrics:
  duration: "8m"
  completed: "2026-03-29"
  tasks: 3
  files_created: 17
  files_modified: 5
  files_removed: 3
---

# Quick Task 260328-u4q: README 重写 + VitePress 文档站 + GitHub Pages 部署

**概要：** 将 README.md/README.en.md 从 ~490 行缩减为 ~100 行的开源风格简介，创建 VitePress i18n 文档站并配置 GitHub Pages 自动部署。

## 完成的任务

### 任务 1：重写 README.md 和 README.en.md

- 从 ~490 行缩减到 ~100 行，参照 Docker / Traefik / Coolify 风格
- 添加居中的 badges（CI 状态、Release 版本、MIT 许可证）
- 核心特性以 bullet points 呈现
- 快速开始精简为 4 步
- 链接到 VitePress 文档站获取完整文档
- **提交：** `7c6c86f`

### 任务 2：创建 VitePress 文档站

- 安装 vitepress 为 devDependency
- 在 root package.json 添加 `docs:dev`、`docs:build`、`docs:preview` 脚本
- 配置 i18n：中文为默认语言（`/zh/`），英文为备选（`/en/`）
- 创建指南页面：快速开始、部署指南、配置参考、架构说明
- 创建参考页面：API 参考、故障排查
- 将旧文档内容完整迁移到 VitePress 结构
- 删除旧的 `docs/deployment-guide.md`、`docs/operations-manual.md`、`docs/recovery-runbook.md`
- **提交：** `c28f369`

### 任务 3：添加 GitHub Pages 工作流

- 创建 `.github/workflows/docs.yml`
- 在 push 到 main 且 `docs/` 变更时触发
- 使用 pnpm + Node.js 24 构建 VitePress
- 通过 `actions/configure-pages` + `actions/upload-pages-artifact` + `actions/deploy-pages` 部署
- **提交：** `e0c9b5e`

### 修复：ESM 兼容性

- 将 `config.ts` 重命名为 `config.mts` 解决 ESM require 错误
- 将 `docs/.vitepress/dist/` 和 `cache/` 加入 `.gitignore`
- **提交：** `53b30a5`

## 偏差记录

### 自动修复

**1. [Rule 3 - Blocking] VitePress 配置文件 ESM 兼容性**
- **发现于：** 任务 2 构建验证阶段
- **问题：** `config.ts` 无法被 esbuild 的 `require()` 加载，因为 vitepress 是 ESM-only 包
- **修复：** 重命名为 `config.mts` 使 vitepress 以 ESM 模式加载配置
- **文件：** `docs/.vitepress/config.mts`, `.gitignore`
- **提交：** `53b30a5`

## 已知 Stubs

无。

## 验证

- VitePress 构建成功：`pnpm docs:build` 输出 "build complete in 3.03s"
- 所有 17 个文档页面生成到 `docs/.vitepress/dist/`
- GitHub Pages 工作流语法正确
