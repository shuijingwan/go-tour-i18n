# go-tour-i18n

`go-tour-i18n` 是一个面向多语言扩展的 A Tour of Go 翻译、校验、同步和发布项目。

本项目是社区维护的非官方项目，不由 Google、Go 团队或 go.dev 官方维护，也不表示与官方项目存在隶属、认可或背书关系。原始 A Tour of Go 内容和源码来自 Go 官方上游仓库。

## 在线入口

- 在线体验：[A Tour of Go 简体中文](https://go-dev.shuijingwanwq.com/)
- 官方英文版：[A Tour of Go](https://go.dev/tour/)
- 开发记录：[《A Tour of Go 多语言翻译项目》](https://www.shuijingwanwq.com/series/go-tour-chinese-edition-development-series/)
- 问题反馈：[GitHub Issues](https://github.com/shuijingwan/go-tour-i18n/issues)

正式站点为 <https://go-dev.shuijingwanwq.com/>：根路径 `/` 是项目首页，课程位于 `/tour/`。博客页面回链至正式站点和 GitHub 仍由仓库外的博客维护工作完成。

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
- 已完成翻译输入保护与 Translation Engine 的匿名质量实验；ChatGPT 是下一阶段优先验证的高质量翻译来源，但尚未接入正式翻译执行路径。
- 现有 103 个 canonical candidate 与 production 均未因实验替换。下一阶段将研究自动化 ChatGPT retranslation staging，先生成独立候选并完成比较与校验，不立即覆盖现有译文。详见 [翻译质量实验](docs/TRANSLATION_QUALITY_EXPERIMENTS.md)。

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

### ChatGPT 重译批次约定

ChatGPT 执行重译批次时，应主动从 GitHub 仓库 `main` 分支读取批次输入和目标语言当前最新的术语规则。用户无需在每个新会话中重新复制术语表。正常读取顺序为：

1. `data/retranslation-runs/<locale>/<batch-id>/manifest.json`；
2. manifest 指向的 `data/retranslation-runs/<locale>/<batch-id>/inputs/*.article`；
3. `locales/<locale>/glossary.yaml`。

整体工作流为：完整顶层 `present.Section` → Default protected input → retranslation batch → ChatGPT 读取批次和最新 locale glossary → 每页独立整页翻译 → restore → batch 内 candidate staging → validator。当前已经实现批次输入导出，以及仓库内 `raw-responses/` 的 restore、隔离 candidate staging 和统一 candidate 校验；从聊天自动导入 response、将通过的暂存 candidate 提升为 canonical candidate 等后续自动化衔接仍属于计划中的流程，不应视为现有能力。

一个批次可以包含多个页面（默认 10 页），但每个完整顶层 `present.Section` 始终是独立翻译单元，不得将批次内多个页面合并成一个翻译单元。

正式翻译使用 `main` 分支中目标语言当前最新的 `glossary.yaml`。批次无需额外绑定 glossary commit ID、glossary SHA，也无需复制术语表；如需追溯历史时期使用的规则，使用 Git 历史即可，当前不另建术语版本绑定、数据库或状态机制。这一约定以 GitHub 仓库作为 ChatGPT 获取批次输入和项目规则的主要来源之一，减少人工复制粘贴，并由 Git 历史承担必要的版本追溯。

处理最早一个 raw response 完整且尚未处理的批次：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation process --locale zh-CN
```

处理前会根据当前 Catalog 和 glossary 重新生成 Default protected input，并要求它与批次保存的 input 字节完全一致；随后使用同一份保护映射 restore，并调用正式 `ValidateCandidate`。结果只写入该批次自己的 `candidates/`、`validation/` 和 `result.json`，不会修改 locale 的 canonical candidate 或 status。默认严格保持批次顺序：最早未处理批次缺少 raw response 时会停止，不会跳到后续批次；`--batch-id` 只用于显式调试或恢复。

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

production runtime 已完成并正式部署。页面访问链路为：浏览器 → EdgeOne → 阿里云 Nginx → Go Tour。正常 production Run / Format 由浏览器直接请求自建 ZgoCloud 固定代理 `https://play.go-dev.shuijingwanwq.com:8443/compile` 和 `/fmt`，再由该代理转发到官方 Playground `play.golang.org`；不再经过阿里云 Go 服务端的 `/_/compile`、`/_/fmt`。旧服务端路径仍保留为兼容/回滚路径，并按 go.dev 要求保留 form 编码。ZgoCloud 只是本项目自建的固定反向代理，不是 Go 官方服务。production 主机不执行用户提交的 Go 程序。本地 `/socket` 不用于 production：普通请求和 WebSocket Upgrade 均返回 404，`/_/share` 当前未启用并返回 404。

使用以下命令生成固定 locale 的 production bundle（当前完成 locale 为 `zh-CN`）：

```bash
go run -mod=readonly ./cmd/tour-i18n publish \
  --locale zh-CN \
  --published-at 2026-08-12T00:00:00Z \
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

`--published-at` 是当前 locale production bundle 的发布时间，必须使用 RFC 3339 UTC；它不是 Git commit、服务启动或请求时间。源码 `_content/tour/site-metadata.json` 仅为 `go run ./tour` 和 preview 提供开发态 metadata，不保存生产发布时间；publish 会在 bundle 内生成真实的站点 metadata。`bin/tour` 在构建时固定 locale，从 binary 自身相邻的 `../_content` 加载内容，不需要运行时 `--locale` 或 `--content`，也不依赖当前工作目录。`release.json`、bundle 内的站点 metadata 和 `SHA256SUMS` 由相同输入确定；相同源码、发布时间、Go 工具链及 GOOS/GOARCH 下的重复 publish 逐文件一致。bundle 不包含 candidate、status、translation-runs 等开发期数据。

当前正式 release 已部署：`/data/go-tour/releases/20260813-zh-CN-1fe73f5`。项目 commit 为 `1fe73f54615b3d05d09133f6e25aa91c5fb07f75`。

正式站点：<https://go-dev.shuijingwanwq.com/>。

页面请求链路为：浏览器 → EdgeOne → Nginx → Go Tour。Run 使用 `https://play.go-dev.shuijingwanwq.com:8443/compile`，Format 使用 `https://play.go-dev.shuijingwanwq.com:8443/fmt`，链路随后为 ZgoCloud Nginx → `play.golang.org`。旧的阿里云 Go 服务端 `/_/compile`、`/_/fmt` 仍保留作为兼容/回滚路径。Go Tour 由 `go-tour.service` 管理，监听 `127.0.0.1:3999`；生产环境不直接暴露本地 `/socket` 代码执行接口。

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

- 正式 release：`/data/go-tour/releases/20260813-zh-CN-1fe73f5`。
- 项目 commit：`1fe73f54615b3d05d09133f6e25aa91c5fb07f75`。
- 页面访问链路：浏览器 → EdgeOne → 阿里云 Nginx → Go Tour。
- 正常 production Run / Format 链路：浏览器 → `https://play.go-dev.shuijingwanwq.com:8443/compile` 或 `/fmt` → ZgoCloud Nginx → `play.golang.org`。
- 旧阿里云 Go 服务端 `/_/compile`、`/_/fmt` 仍保留作为兼容/回滚路径；production `/socket` 仍禁用，生产主机不执行用户提交的 Go 代码。
- systemd 服务：`go-tour.service`，监听 `127.0.0.1:3999`。
- `/tour/welcome/1`、真实 Run、真实 Format 均已通过公网验收；`/socket` 和 `/_/share` 均返回 404。
- production binary 为 Linux amd64、`CGO_ENABLED=0`、静态链接，不依赖服务器 glibc 版本。

部署期间已解决动态链接 glibc 兼容、OneinStack 静态资源 location 抢占、release 目录权限和 Nginx systemd 接管问题。Cloudflare 仅负责权威 DNS，业务 CNAME 使用 DNS only，正式流量经过 EdgeOne，不采用 Cloudflare 双层代理。

## 上游来源

- 官方仓库：<https://github.com/golang/website.git>
- 分支：`master`
- 固定 commit：`e11dacba76c5aae474746e9eedee19693f492803`
- upstream commit 时间：`2026-07-23 04:05:40`（北京时间；`2026-07-22T20:05:40Z`）
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
