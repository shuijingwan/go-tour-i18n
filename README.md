# go-tour-i18n

`go-tour-i18n` 是一个面向多语言扩展的 A Tour of Go 翻译、校验、同步和发布项目。

本项目是社区维护的非官方项目，不由 Google、Go 团队或 go.dev 官方维护，也不表示与官方项目存在隶属、认可或背书关系。原始 A Tour of Go 内容和源码来自 Go 官方上游仓库。

## 当前阶段

- 已从固定官方上游导入可独立运行、测试、解析和渲染的英文 Tour 基线。
- 已建立 103 个正式发布页面及 2 条条件源审计记录的机器可读目录。
- 第一阶段目标语言为简体中文 `zh-CN`；课程正文已完成，正式状态为 `ready=103`、`pending=0`、`blocked=0`。
- 课程正文、article/lesson 根级 metadata 与公共 UI 分开维护：103 个顶层 `present.Section` 使用 canonical candidate；每个 article 的 `title`、`subtitle` 使用独立 locale metadata；公共 UI 使用独立 UI 本地化资源。
- zh-CN 的 7/7 个正式 article metadata 已完成本地化（title=7/7、subtitle=7/7）。
- 已建立完整页面翻译执行器、术语表、结构保护、candidate 校验、状态管理和 attempt 审计记录。
- 已实现面向 locale 的完整正式投影和本地完整预览；zh-CN 完整投影已通过 103/103 HTTP 页面级验收。
- 当前 module path 为 `github.com/shuijingwan/go-tour-i18n`。
- zh-CN 第一阶段已经正式上线：production publish 已实现并正式部署，浏览器最终验收通过。

当前英文基线包含：

- 7 个一级 `.article`；
- 103 个正式发布课程页面，其中两个由 `#appengine:` 条件源去标记后投影；
- 93 个 `.play` 引用；
- 1 个 `.image` 引用；
- 2 条继续单独保留的 `#appengine:` 条件源审计记录。

翻译的最小单元是一个完整顶层 `present.Section`，不会拆成句子级或多 text 槽位 JSON。页面目录位于 [`data/tour-pages.tsv`](data/tour-pages.tsv)，zh-CN 状态位于 [`locales/zh-CN/status.tsv`](locales/zh-CN/status.tsv)。完整投影能力面向 locale；当前第一阶段完成的 locale 为 zh-CN。语言约定见 [`locales/zh-CN/README.md`](locales/zh-CN/README.md) 和 [LANGUAGES.md](LANGUAGES.md)。

目录中的 `page_id` 是语言状态、候选文件和发布记录的持久身份；`route` 是可能随上游变化的当前访问路径，`article` 和 `section_number` 是可能变化的发布投影位置。当前 103 个 ID 已冻结，不会因插入、删除、重排或改标题而自动重编号。完整规则见 [PAGE_IDENTITY.md](PAGE_IDENTITY.md)。

## 多语言维护工具

检查机器可读页面目录：

```bash
go run -mod=readonly ./cmd/tour-i18n catalog check
```

显式重新生成目录：

```bash
go run -mod=readonly ./cmd/tour-i18n catalog write
```

在导入新的上游版本前，只读预览页面变化：

```bash
go run -mod=readonly ./cmd/tour-i18n upstream preview \
  --source-root /path/to/website
```

预览不会修改目录、语言状态或候选文件。`catalog write` 遇到新增、删除或无法可靠识别的页面时会停止，不会按新位置重编号。

导出一个完整英文源页面：

```bash
go run -mod=readonly ./cmd/tour-i18n page export \
  --id welcome/1 \
  --output /tmp/welcome-1.article
```

检查 zh-CN locale 和 103 页状态：

```bash
go run -mod=readonly ./cmd/tour-i18n status check --locale zh-CN
```

校验一个完整候选页面：

```bash
go run -mod=readonly ./cmd/tour-i18n candidate validate \
  --locale zh-CN \
  --id welcome/1 \
  --file /tmp/candidate.article
```

候选校验只读取英文源、candidate 和状态数据，不会自动修改状态、移动页面或创建 `ready`、`blocked`、`published` 记录。

`translate run -dev` 仅用于单页翻译的开发校准：每条命令只执行一次 attempt，并允许从 pending 或 blocked 状态继续。正式批量翻译不得使用 `-dev`。

当且仅当某个 `blocked` 页面耗尽的最近三次正式 attempt 都有完整审计、均为未取得模型响应的 `network:` 失败时，可显式恢复一轮正式额度；恢复会保留原 attempt，并写入 recovery 审计：

```bash
go run -mod=readonly ./cmd/tour-i18n translate recover-network \
  --locale zh-CN \
  --id basics/4
```

该命令拒绝 HTTP/API、模型输出、token、parse、render 或 validator 失败；恢复后仍须单独运行正常的 `translate run`。

## 正式投影与本地预览

完整投影只接受 catalog 中所有页面对应的 canonical `ready` candidate；存在 pending、blocked、缺失、额外或非 canonical candidate 时会失败，不会回退到英文或旧译文。输出为可直接交给官方 Tour 本地服务的完整内容树，不修改 candidate、status 或 catalog。

课程根级 metadata 位于 `locales/<locale>/article-metadata.json`。它独立于 Section candidate、公共 UI 和 status，逐个维护正式 article 的 `title` 与 `subtitle`。完整 build 与 preview 会严格要求 metadata article 集合与 catalog 推导的正式 article 集合一致，且 title/subtitle 均非空；缺失或不完整时失败，不允许回退到 upstream 英文。

构建一个 locale 的完整正式投影（未提供 `--output` 时使用安全临时目录）：

```bash
go run -mod=readonly ./cmd/tour-i18n build \
  --locale zh-CN
```

完整语言本地预览：

```bash
go run -mod=readonly ./cmd/tour-i18n preview \
  --locale zh-CN
```

原有单页 candidate preview 继续可用：

```bash
go run -mod=readonly ./cmd/tour-i18n preview \
  --locale zh-CN \
  --id <page_id>
```

当前 zh-CN 完整投影包含 7 个 article、103 个正式页面，并已完成 103/103 HTTP 页面级验收；7/7 个 article title/subtitle 均已本地化。`welcome/1`、`welcome/4` 和 `welcome/5` 的特殊投影也包含在该验收中。article metadata 不计为额外页面，正式页面数仍为 103。

## Production runtime、publish 与正式上线

production runtime 已完成并正式部署。它使用 `HTTPTransport`，将 `/_/compile` 和 `/_/fmt` 代理到 `https://go.dev/_`；`/_/compile` 按 go.dev 要求保留 form 编码。production 主机不执行用户提交的 Go 程序。本地 `/socket` 不用于 production：普通请求和 WebSocket Upgrade 均返回 404，`/_/share` 当前未启用并返回 404。

使用以下命令生成固定 locale 的 production bundle（当前完成 locale 为 `zh-CN`）：

```bash
go run -mod=readonly ./cmd/tour-i18n publish \
  --locale zh-CN \
  --output <directory>
```

生成目录包含：

```text
<output>/
├── bin/
│   └── tour
├── _content/
├── release.json
└── SHA256SUMS
```

`bin/tour` 在构建时固定 locale，从 binary 自身相邻的 `../_content` 加载内容，不需要运行时 `--locale` 或 `--content`，也不依赖当前工作目录。`release.json` 和 `SHA256SUMS` 是确定性的；相同源码、Go 工具链及 GOOS/GOARCH 下的重复 publish 已逐文件一致，185 个 bundle 文件的 SHA-256 均已验收通过。bundle 不包含 candidate、status、translation-runs 等开发期数据。

当前正式 release 已部署：`/data/go-tour/releases/20260811-zh-CN-925d59d`。项目 commit 为 `925d59d92016e026c92ae60f4535abd9237119ea`。

正式站点：<https://go-dev.shuijingwanwq.com/>。

生产请求链路为：浏览器 → EdgeOne → Nginx → Go Tour → `go.dev` Playground。Go Tour 由 `go-tour.service` 管理，监听 `127.0.0.1:3999`；生产环境不直接暴露本地 `/socket` 代码执行接口。

## 本地运行

需要 Go 1.25 或更高版本。从仓库根目录运行：

```bash
go run -mod=readonly ./tour -http 127.0.0.1:3999 -openbrowser=false
```

然后访问 <http://127.0.0.1:3999/tour/>。

> **安全警告：** 本地 Tour 的 `/socket` 会使用运行 Tour 的机器上的 Go 环境编译和执行示例代码，只用于本地开发验证，应绑定回环地址，不能直接暴露到公网。production runtime 使用远程 HTTPTransport/`go.dev` Playground 代理，不在 production 主机执行用户代码。

## 测试

```bash
go test -mod=readonly -count=1 ./...
```

测试覆盖课程示例构建和运行、present 解析、课程结构、引用路径、静态资源、favicon 和 `/_/fmt`。

## 实际生产部署状态

zh-CN 第一阶段已经正式上线：<https://go-dev.shuijingwanwq.com/>。

- 正式 release：`/data/go-tour/releases/20260811-zh-CN-925d59d`。
- 项目 commit：`925d59d92016e026c92ae60f4535abd9237119ea`。
- production 执行链路：浏览器 → EdgeOne → Nginx → Go Tour → `https://go.dev/_`。
- systemd 服务：`go-tour.service`，监听 `127.0.0.1:3999`。
- `/tour/welcome/1`、真实 Run、真实 Format 均已通过公网验收；`/socket` 和 `/_/share` 均返回 404。
- production binary 为 Linux amd64、`CGO_ENABLED=0`、静态链接，不依赖服务器 glibc 版本。

部署期间已解决动态链接 glibc 兼容、OneinStack 静态资源 location 抢占、release 目录权限和 Nginx systemd 接管问题。Cloudflare 仅负责权威 DNS，业务 CNAME 使用 DNS only，正式流量经过 EdgeOne，不采用 Cloudflare 双层代理。

## 上游来源

- 官方仓库：<https://github.com/golang/website.git>
- 分支：`master`
- 固定 commit：`e11dacba76c5aae474746e9eedee19693f492803`
- 初始验证环境：`go1.26.0 linux/amd64`

同步原则见 [UPSTREAM.md](UPSTREAM.md)，逐文件来源、模式和 SHA-256 见 [UPSTREAM_MANIFEST.tsv](UPSTREAM_MANIFEST.tsv)。

## 第三方组件

本次原样导入了 Tour 使用的历史前端组件。版本、许可证证据和待复核项见 [THIRD_PARTY.md](THIRD_PARTY.md)。

## 当前边界

- zh-CN 第一阶段已经正式上线；后续工作属于发布后的运维和其他语言扩展，不属于本次上线冻结内容。

## 许可证

- 上游原始源码和内容遵循本仓库中的 BSD 风格 [LICENSE](LICENSE)。
- 本项目自行编写的翻译、工具和文档，除另有说明外，也按照该许可证提供。
- [PATENTS](PATENTS) 是上游附带的附加知识产权声明，其适用范围以原文为准。
- 第三方代码和资源适用其各自声明的许可证，不因位于本仓库中而自动适用根 LICENSE。

## English Summary

`go-tour-i18n` is a community-maintained, unofficial multilingual A Tour of Go translation project. The official English Tour baseline has been imported; Simplified Chinese (`zh-CN`) is the first completed locale, with all 103 projected pages ready, 7/7 article metadata entries localized, completed public UI localization, and a successful final browser acceptance pass. The HTTPTransport-based production runtime and deterministic production publish bundle are implemented and deployed at <https://go-dev.shuijingwanwq.com/>.
