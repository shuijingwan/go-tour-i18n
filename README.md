# go-tour-i18n

`go-tour-i18n` 是一个面向多语言扩展的 A Tour of Go 翻译、校验、同步和发布项目。

本项目是社区维护的非官方项目，不由 Google、Go 团队或 go.dev 官方维护，也不表示与官方项目存在隶属、认可或背书关系。原始 A Tour of Go 内容和源码来自 Go 官方上游仓库。

## 在线入口

- 在线体验：[A Tour of Go 简体中文](https://go-dev.shuijingwanwq.com/)
- 在线体验：[A Tour of Go 日本語](https://ja-go-dev.shuijingwanwq.com/)
- 官方英文版：[A Tour of Go](https://go.dev/tour/)
- 开发记录：[《A Tour of Go 多语言翻译项目》](https://www.shuijingwanwq.com/series/go-tour-chinese-edition-development-series/)
- 问题反馈：[GitHub Issues](https://github.com/shuijingwan/go-tour-i18n/issues)

正式社区语言站为简体中文 <https://go-dev.shuijingwanwq.com/> 和日本語 <https://ja-go-dev.shuijingwanwq.com/>：根路径 `/` 是项目首页，课程位于 `/tour/`。博客页面回链至正式站点和 GitHub 仍由仓库外的博客维护工作完成。

## 当前阶段

- 已从固定官方上游导入可独立运行、测试、解析和渲染的英文 Tour 基线。
- 已建立 103 个正式发布页面及 2 条条件源审计记录的机器可读目录。
- 第一阶段目标语言为简体中文 `zh-CN`；统一 locale workflow status 当前包含 103 个 ready Page 和 19 个 ready Example，另有 74 个无需翻译自然语言注释的 Example 不进入 status。
- 课程正文、article/lesson 根级 metadata 与公共 UI 分开维护：103 个顶层 `present.Section` 使用 canonical candidate；每个 article 的 `title`、`subtitle` 使用独立 locale metadata；公共 UI 使用独立 UI 本地化资源。
- zh-CN 的 7/7 个正式 article metadata 已完成本地化（title=7/7、subtitle=7/7）。
- 已建立完整页面翻译执行器、术语表、结构保护、candidate 校验、状态管理和 attempt 审计记录。
- 已实现面向 locale 的完整正式投影和本地完整预览；zh-CN 完整投影已通过 103/103 HTTP 页面级验收。
- 当前 module path 为 `github.com/shuijingwan/go-tour-i18n`。
- zh-CN 第一阶段已有 production release 在线运行；production publish、自动部署和浏览器验收能力均已完成。
- ja-JP 已完成 production release 部署、Cloudflare Free 公网接入、SEO 全量验收和 Playground 浏览器验收，并正式上线。
- 在匿名质量实验之后，已完成 zh-CN 全部 103 页的 ChatGPT 整页重译、批次处理、统一 validator、语义质量审计和 canonical promotion。当前仓库中的 103 个 canonical candidate 均为本轮正式提升后的 ChatGPT 译文，最终语义质量审计为 A=103、B/C/D=0。
- 本轮 ChatGPT 103 页 canonical 已生成正式 production release、完成生产部署和线上最终验收。正式课程路由 103/103、article endpoint 7/7、真实 Run 与 Format 均通过；当前在线内容已经是本轮正式提升后的 ChatGPT 译文。详见 [翻译质量实验](docs/TRANSLATION_QUALITY_EXPERIMENTS.md) 和 [项目状态](docs/PROJECT_STATE.md)。

当前英文基线包含：

- 7 个一级 `.article`；
- 103 个正式发布课程页面，其中两个由 `#appengine:` 条件源去标记后投影；
- 93 个 `.play` 引用；
- 1 个 `.image` 引用；
- 2 条继续单独保留的 `#appengine:` 条件源审计记录。

页面翻译的最小单元是一个完整顶层 `present.Section`，Example 翻译单元是完整 `.go` 文件。页面目录位于 [`data/tour-pages.tsv`](data/tour-pages.tsv)，统一的 zh-CN workflow 状态位于 [`locales/zh-CN/status.tsv`](locales/zh-CN/status.tsv)。完整 projection 统一要求 103 个 Page 和 19 个 eligible Example 全部 ready；当前 122/122 TranslationUnit 均为 ready。语言约定见 [`locales/zh-CN/README.md`](locales/zh-CN/README.md) 和 [LANGUAGES.md](LANGUAGES.md)。

系统还会从正式页面经过现有 present 解析确认的 `.play` 指令发现示例，并在 [`data/tour-examples.tsv`](data/tour-examples.tsv) 中记录第二种 source unit。每个 example 的 source 是完整 `.go` 文件，`source_sha256` 覆盖文件全部字节，而不是只覆盖注释。当前 93 个 Example 中有 19 个含普通可翻译自然语言注释，已进入统一 status 并达到 ready；其余 74 个只参与 source tracking，projection 保留 upstream 原文。统一 promotion 可将通过验证的 Example 提升为 canonical `.go` candidate，projection 会将 ready Example 覆盖到完整内容树。

翻译运行器现已通过 `Catalog.Unit` 和 `TranslationUnit` 取得翻译源及其版本哈希；课程页面随后继续进入原有 Page 专用保护、校验和状态流程。当前运行器只开放 page 翻译，识别到 example 时会在创建 attempt 或调用模型前明确拒绝，不能据此认为示例翻译已经实现。

翻译单元的输入准备会按类型分派：page 保持原有保护行为；example 使用 Go scanner 识别真实注释，将代码、字符串、布局、注释分隔符和机器语义注释替换为现有保护 token，只开放普通自然语言注释 payload，并继续使用同一 locale glossary 的 `keep` 规则保护其中必须保持原样的术语。example 输入能够通过现有 restore 逐字节恢复完整 `.go` 文件，并已接入 batch process/retry、统一 validator、canonical promotion、status 和 projection；当前 19 个 eligible Example 已具备翻译 evidence 与 canonical candidate。

Page 和 Example 是两种一级 `TranslationUnit`。统一控制面使用 `unit_id` 和 `unit_kind`；Page 的持久 ID 仍遵循既有冻结规则，`route`、`article` 和 `section_number` 只描述当前投影位置。完整 Page 身份规则见 [PAGE_IDENTITY.md](PAGE_IDENTITY.md)。

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

上游同步前先运行 `upstream preview` 检测变化。完成经确认的人工 `_content/tour` source 复制后，如 Page route 与 conditional identity 未变，可显式重建三个 source catalog：

```bash
go run -mod=readonly ./cmd/tour-i18n catalog write --allow-source-change
```

该模式仅更新 `data/tour-pages.tsv`、`data/tour-conditional-pages.tsv` 和 `data/tour-examples.tsv`；不会修改 locale status、candidate 或 review evidence，新增、删除、移动或无法识别的 Page 仍会被拒绝。

导出一个完整英文源页面：

```bash
go run -mod=readonly ./cmd/tour-i18n page export \
  --id welcome/1 \
  --output /tmp/welcome-1.article
```

检查 zh-CN locale 的统一状态（103 个 Page、19 个 eligible Example）：

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

### 正式 TranslationUnit 翻译流程

当前默认生产 Translation Engine 为 **Codex GPT-5.6 Sol + High**，ChatGPT 统一承担 Quality Check。Codex 直接读取仓库中的 batch 输入并写入 `raw-responses/`，不经过 ZIP、JSON wrapper 或本地导入环节。

正式模型输入由当前 batch 的 `manifest.json`、manifest 列出的全部 `inputs/*` 与 `locales/<locale>/glossary.yaml` 共同构成，三者不可拆分。Glossary 必须在翻译前读取并完整遵守，不只是 validator 的后置检查材料。

当前流程为：

```text
export
→ Codex High 翻译
→ raw-responses/
→ process
→ automatic validation
→ ChatGPT Quality Check
→ Final Review
→ promotion
```

Retry 只处理 `restore_failed` 与 `validation_failed`，且 `retranslation retry` 本身不调用模型、不生成译文。Automatic validation 通过后的 B/C/D 质量问题必须进入新的 revision batch，不得使用 retry。所有 TranslationUnit 的 ChatGPT Quality Check 均为 A 后才能进入 Final Review；Final Review 也只有 A 才允许 promotion。

完整导航见 [多语言翻译流程](docs/TRANSLATION_WORKFLOW.md)，输入/输出契约见 [翻译任务规范](docs/TRANSLATION_TASK_SPEC.md)，Codex 规则见 [Codex TranslationUnit 翻译执行规范](docs/CODEX_TRANSLATION.md)，执行顺序见 [Retranslation 执行手册](docs/RETRANSLATION_RUNBOOK.md)。历史上 zh-CN 103 页由 ChatGPT 完成并提升为 canonical 的事实继续保留。

## 正式投影与本地预览

完整投影只接受 workflow 中 103 Page + 19 eligible Example 对应的 canonical `ready` candidate；存在 pending、blocked、缺失、额外或非 canonical candidate 时会失败，不会回退到英文或旧译文。投影先复制完整 upstream 内容树，再用 ready Page 重建 article、用 ready Example canonical `.go` 覆盖对应源路径；其余 74 个 Example 保留 upstream 原文。该过程不修改 candidate、status 或 catalog。

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

production runtime 已在 zh-CN 和 ja-JP 正式部署。zh-CN 页面经 EdgeOne 接入，ja-JP 页面经 Cloudflare Free 接入，随后均由阿里云 Nginx 转发至各自 Go Tour 服务。正常 production Run / Format 由浏览器直接请求自建 ZgoCloud 固定代理 `https://play.go-dev.shuijingwanwq.com:8443/compile` 和 `/fmt`，再由该代理转发到官方 Playground `play.golang.org`；代理已允许 `https://go-dev.shuijingwanwq.com` 与 `https://ja-go-dev.shuijingwanwq.com` 两个正式 Origin。旧服务端路径仍保留为兼容/回滚路径，并按 go.dev 要求保留 form 编码。ZgoCloud 只是本项目自建的固定反向代理，不是 Go 官方服务。production 主机不执行用户提交的 Go 程序。本地 `/socket` 不用于 production：普通请求和 WebSocket Upgrade 均返回 404，`/_/share` 当前未启用并返回 404。

使用以下命令生成固定 locale 的 production bundle（当前正式上线 locale 为 `zh-CN` 和 `ja-JP`）：

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

`--published-at` 是当前 locale production bundle 的发布时间，必须使用 RFC 3339 UTC；它不是 Git commit、服务启动或请求时间。源码 `_content/tour/site-metadata.json` 仅为 `go run ./tour` 和 preview 使用的开发态 metadata，不保存生产发布时间；publish 会在 bundle 内生成真实的站点 metadata。`bin/tour` 在构建时固定 locale，从 binary 自身相邻的 `../_content` 加载内容，不需要运行时 `--locale` 或 `--content`，也不依赖当前工作目录。schema v2 `release.json` 同时记录 `translation_units`、`pages`、`eligible_examples` 和 `articles`；`translation_units` 与 `eligible_examples` 由当前 Catalog 的统一 workflow 动态计算。`release.json`、bundle 内的站点 metadata 和 `SHA256SUMS` 由相同输入确定；相同源码、发布时间、Go 工具链及 GOOS/GOARCH 下的重复 publish 逐文件一致。bundle 不包含 candidate、status、translation-runs 等开发期数据。

当前 zh-CN 正式 release 已部署：`/data/go-tour/releases/20260824-zh-CN-7f6474b0`。该 release 使用 upstream `645042eb697eaf69e33a9af00c6b5b3fffdead5a`，包含 103 个 ready Page 与 19 个 ready eligible Example；历史上正式提升后的 103 页 ChatGPT canonical 事实保持不变。

当前 ja-JP 正式 release 为 `/data/go-tour-ja-JP/releases/20260824-ja-JP-164fecdd`，由 `go-tour-ja-JP.service` 在 `127.0.0.1:4000` 提供服务；非中文 production 静态资源使用 <https://assets-go-dev.shuijingwanwq.com/>。

zh-CN 正式站点：<https://go-dev.shuijingwanwq.com/>。

zh-CN 页面请求链路为：浏览器 → EdgeOne → Nginx → Go Tour。Run 使用 `https://play.go-dev.shuijingwanwq.com:8443/compile`，Format 使用 `https://play.go-dev.shuijingwanwq.com:8443/fmt`，链路随后为 ZgoCloud Nginx → `play.golang.org`。旧的阿里云 Go 服务端 `/_/compile`、`/_/fmt` 仍保留作为兼容/回滚路径。Go Tour 由 `go-tour.service` 管理，监听 `127.0.0.1:3999`；生产环境不直接暴露本地 `/socket` 代码执行接口。

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

zh-CN 与 ja-JP 均已正式上线：<https://go-dev.shuijingwanwq.com/>、<https://ja-go-dev.shuijingwanwq.com/>。

以下是当前正式 production 部署状态：

- zh-CN 正式 release：`/data/go-tour/releases/20260824-zh-CN-7f6474b0`。
- zh-CN Upstream commit：`645042eb697eaf69e33a9af00c6b5b3fffdead5a`。
- zh-CN 正式状态：`ready=122`、`pending=0`、`blocked=0`、`pages=103`、`eligible_examples=19`、`articles=7`；103/103 课程路由与 7/7 article endpoint 已完成线上验收。
- zh-CN 页面访问链路：浏览器 → EdgeOne → 阿里云 Nginx → Go Tour。
- 正常 production Run / Format 链路：浏览器 → `https://play.go-dev.shuijingwanwq.com:8443/compile` 或 `/fmt` → ZgoCloud Nginx → `play.golang.org`。
- 旧阿里云 Go 服务端 `/_/compile`、`/_/fmt` 仍保留作为兼容/回滚路径；production `/socket` 仍禁用，生产主机不执行用户提交的 Go 代码。
- systemd 服务：`go-tour.service`，监听 `127.0.0.1:3999`。
- `/tour/welcome/1`、真实 Run、真实 Format 均已通过公网验收；`/socket` 和 `/_/share` 均返回 404。
- sitemap 包含首页、`/tour/list` 和 103 个课程页面，共 105 个唯一 URL。
- production binary 为 Linux amd64、`CGO_ENABLED=0`、静态链接，不依赖服务器 glibc 版本。
- ja-JP 正式 release：`/data/go-tour-ja-JP/releases/20260824-ja-JP-164fecdd`；data root 为 `/data/go-tour-ja-JP`，systemd 服务为 `go-tour-ja-JP.service`，监听 `127.0.0.1:4000`，公网使用 Cloudflare Free。
- ja-JP sitemap 已验证 105/105，host mismatch=0、HTTP failure=0；`robots.txt` 正确指向 <https://ja-go-dev.shuijingwanwq.com/sitemap.xml>。
- ja-JP Playground compile、fmt 和浏览器实际运行均已通过；Playground 允许 zh-CN 与 ja-JP 两个正式 Origin。
- 非中文共享静态资源由 <https://assets-go-dev.shuijingwanwq.com/> 提供；旧 `assets.go-dev.shuijingwanwq.com` 已废弃并清理。

部署期间已解决动态链接 glibc 兼容、OneinStack 静态资源 location 抢占、release 目录权限和 Nginx systemd 接管问题。当前 CDN 接入明确区分：zh-CN 使用 EdgeOne；ja-JP 使用 Cloudflare Free；`assets-go-dev.shuijingwanwq.com` 使用 Cloudflare 代理。不存在 Cloudflare 与 EdgeOne 双层代理。

## 上游来源

- 官方仓库：<https://github.com/golang/website.git>
- 分支：`master`
- 固定 commit：`645042eb697eaf69e33a9af00c6b5b3fffdead5a`
- upstream commit 时间：`2026-08-20 13:56:11`（北京时间；`2026-08-20T05:56:11Z`）
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
