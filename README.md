# go-tour-i18n

`go-tour-i18n` 是一个面向多语言扩展的 A Tour of Go 翻译、校验、同步和发布项目。

## 当前阶段

- 第一阶段只交付简体中文 `zh-CN`。
- 当前仓库刚刚初始化。
- 尚未导入 Tour 源码。
- 尚未发布可运行版本。

## 项目目标

- 以完整课程页面作为最小翻译单元。
- 分离课程正文和公共 UI 文案。
- 每种语言独立维护译文、UI 文案、术语和状态。
- 共用 Go 服务端、示例代码和前端基础资源。
- 采用构建时单语言生成。
- 统一执行解析、结构比较、渲染和页面验证。
- 持续记录并同步官方上游版本。

## 当前翻译工作流

```text
完整课程页面
→ 整页翻译
→ candidate
→ 自动校验
→ ready
→ published
```

有限重试后仍失败的页面进入 `blocked`。这类页面可以改用其他翻译来源重新进行整页翻译，但仍必须经过同一套自动校验，才能进入 `ready` 和 `published` 状态。

## 官方上游基线

- 上游仓库：<https://github.com/golang/website.git>
- 上游分支：`master`
- 当前基线 commit：`e11dacba76c5aae474746e9eedee19693f492803`
- 初始验证环境：`go1.26.0 linux/amd64`

基线确认和后续同步原则见 [UPSTREAM.md](UPSTREAM.md)。

## 非官方声明

本项目是由社区维护的非官方项目，不由 Google、Go 团队或 go.dev 官方维护。本项目不表示与官方项目存在隶属、认可或背书关系。原始 A Tour of Go 内容和源码来自上述官方上游仓库。

## 许可证

- 上游原始源码和内容遵循本仓库中的 BSD 风格 [LICENSE](LICENSE)。
- 本项目自行编写的翻译、工具和文档，除另有说明外，也按照该 [LICENSE](LICENSE) 提供。
- [PATENTS](PATENTS) 是上游附带的附加知识产权声明，其适用范围以原文为准。
- 后续导入的第三方代码、字体、图片或其他资产可能适用各自的许可证。

## 项目状态

项目目前仍处于初始化和架构落地阶段，尚不能运行，也没有可供发布的版本。

## English Summary

`go-tour-i18n` is a community-maintained, unofficial multilingual translation project for A Tour of Go. Its first target language is Simplified Chinese (`zh-CN`), and it is currently under initial development.
