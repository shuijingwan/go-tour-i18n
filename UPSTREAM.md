# 官方上游基线与同步原则

## 文档用途

本文档用于记录 `go-tour-i18n` 所基于的官方上游版本及后续同步规则。

## 当前上游信息

- 仓库 URL：<https://github.com/golang/website.git>
- 示例本地只读目录：`$HOME/code/go-website-upstream`（可按实际环境调整）
- 分支：`master`
- 固定 commit：`e11dacba76c5aae474746e9eedee19693f492803`
- Go 版本：`go1.26.0 linux/amd64`
- 首次确认日期：2026-08-02

## 基线原则

- 所有初始导入都必须能够追溯到上述固定 commit。
- 官方上游目录必须保持只读和干净。
- 不直接在上游仓库中开发本项目功能。
- 上游发生变化时，不自动覆盖现有译文。
- 同步前先比较结构和课程页面变化。
- 每次同步都应记录旧 commit、新 commit 和同步结果。

## 当前同步状态

尚未执行首次源码导入。

## 基线验证命令

在官方上游只读目录中执行以下只读命令：

```bash
cd "$HOME/code/go-website-upstream"
git branch --show-current
git rev-parse HEAD
git status --short --branch
go version
```
