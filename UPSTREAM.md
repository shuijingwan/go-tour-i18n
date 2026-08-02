# 官方上游基线与同步原则

## 文档用途

本文档记录 `go-tour-i18n` 所基于的官方上游版本、首次 Tour 范围导入及后续同步规则。

## 当前上游信息

- 仓库 URL：<https://github.com/golang/website.git>
- 示例本地只读目录：`$HOME/code/go-website-upstream`（可按实际环境调整）
- 分支：`master`
- 固定 commit：`e11dacba76c5aae474746e9eedee19693f492803`
- Go 版本：`go1.26.0 linux/amd64`
- 首次确认日期：2026-08-02
- 首次 Tour 范围源码导入日期：2026-08-02

## 导入范围

本项目不是完整的 `golang/website` 镜像。首次导入只提取独立 Tour 运行、测试、解析和渲染所需的源码、英文课程、示例、模板、静态资源和测试辅助代码，并尽量保留上游相对路径。

逐文件的原样复制或改造关系记录在 [UPSTREAM_MANIFEST.tsv](UPSTREAM_MANIFEST.tsv)。其中 `exact` 表示逐字节一致，`adapted` 表示为独立 module、内部 import path 或 Tour 边界进行了最小改造。

## 当前英文基准

- 7 个一级 article；
- 101 个 standalone 普通课程页面；
- 92 个普通 `.play` 引用；
- 1 个 `.image` 引用；
- 2 个 `welcome.article` 中的 `#appengine:` 条件页面，不计入 101 个 standalone 页面。

## 基线原则

- 所有导入必须能够追溯到固定 commit。
- 官方上游目录保持只读和干净，不在其中开发本项目功能。
- 原样文件按字节和 SHA-256 验证。
- 上游发生变化时，不自动覆盖未来译文。
- 本次导入后尚未执行第二次上游同步。

后续每次同步必须：

1. 记录并比较旧 commit 与新 commit；
2. 比较逐文件 manifest 和哈希；
3. 检查课程页面数量、顺序和 present 结构；
4. 检查 `.play` 引用和示例文件哈希；
5. 单独处理新增、删除、移动或重排的课程页面；
6. 记录同步结果，但不把尚未实现的工具或流程描述为已经存在。

## 基线验证命令

在官方上游只读目录中执行：

```bash
cd "$HOME/code/go-website-upstream"
git branch --show-current
git rev-parse HEAD
git status --short --branch
go version
```
