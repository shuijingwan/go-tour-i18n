# go-tour-i18n

`go-tour-i18n` 是一个面向多语言扩展的 A Tour of Go 翻译、校验、同步和发布项目。

本项目是社区维护的非官方项目，不由 Google、Go 团队或 go.dev 官方维护，也不表示与官方项目存在隶属、认可或背书关系。原始 A Tour of Go 内容和源码来自 Go 官方上游仓库。

## 当前阶段

- 已从固定官方上游导入可独立运行、测试、解析和渲染的英文 Tour 基线。
- 第一阶段目标语言仍为简体中文 `zh-CN`，但尚未开始中文翻译。
- 当前 module path 为 `github.com/shuijingwan/go-tour-i18n`。
- 尚未提供生产部署配置，也尚未发布正式版本。

当前英文基线包含：

- 7 个一级 `.article`；
- 101 个 standalone 普通课程页面；
- 92 个普通 `.play` 引用；
- 1 个 `.image` 引用；
- 2 个单独记录、且不计入上述 101 页的 `#appengine:` 条件页面。

## 本地运行

需要 Go 1.25 或更高版本。从仓库根目录运行：

```bash
go run -mod=readonly ./tour -http 127.0.0.1:3999 -openbrowser=false
```

然后访问 <http://127.0.0.1:3999/tour/>。

> **安全警告：** 当前本地 Tour 的 `/socket` 会使用运行 Tour 的机器上的 Go 环境编译和执行示例代码。它只用于本地开发验证，只应绑定回环地址；不得将当前 `/socket` 或 Tour 服务直接暴露到公网。当前版本不能视为可安全公网部署的版本。

## 测试

```bash
go test -mod=readonly -count=1 ./...
```

测试覆盖课程示例构建和运行、present 解析、课程结构、引用路径、静态资源、favicon 和 `/_/fmt`。

## 正式网站规划

正式网站不会在 Tour Web 服务器本机执行用户代码。计划通过同源轻量代理访问 Go 官方 Playground，并在 Playground 不可用时降级为只读课程。该代理和 execution provider 尚未实现，本仓库当前也不包含生产部署配置。

## 上游来源

- 官方仓库：<https://github.com/golang/website.git>
- 分支：`master`
- 固定 commit：`e11dacba76c5aae474746e9eedee19693f492803`
- 初始验证环境：`go1.26.0 linux/amd64`

同步原则见 [UPSTREAM.md](UPSTREAM.md)，逐文件来源、模式和 SHA-256 见 [UPSTREAM_MANIFEST.tsv](UPSTREAM_MANIFEST.tsv)。

## 第三方组件

本次原样导入了 Tour 使用的历史前端组件。版本、许可证证据和待复核项见 [THIRD_PARTY.md](THIRD_PARTY.md)。

## 尚未实现

- `zh-CN` 课程翻译；
- 课程正文与公共 UI 的多语言资源分离；
- 翻译状态和自动翻译流水线；
- Playground execution provider 与同源代理；
- 正式发布和生产部署配置。

## 许可证

- 上游原始源码和内容遵循本仓库中的 BSD 风格 [LICENSE](LICENSE)。
- 本项目自行编写的翻译、工具和文档，除另有说明外，也按照该许可证提供。
- [PATENTS](PATENTS) 是上游附带的附加知识产权声明，其适用范围以原文为准。
- 第三方代码和资源适用其各自声明的许可证，不因位于本仓库中而自动适用根 LICENSE。

## English Summary

`go-tour-i18n` is a community-maintained, unofficial multilingual A Tour of Go translation project. The official English Tour baseline has been imported; Simplified Chinese (`zh-CN`) remains the first target language but translation has not started. The current server is for loopback-only local development and is not ready for public deployment.
