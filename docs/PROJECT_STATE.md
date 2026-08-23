# 项目状态

更新时间：2026-08-23（北京时间）

## 基线与架构

- 官方 upstream 基线为 `golang/website` 的 `master` 分支 commit `645042eb697eaf69e33a9af00c6b5b3fffdead5a`；翻译运行时使用仓库内固定的最小 Tour 源码闭包，外部 checkout 仅用于同步与校验。
- 当前目录包含 103 个正式发布页面和 2 条单独保留的 `#appengine:` 条件源审计记录；两个条件 Section 去标记后同时投影为 `welcome/4`、`welcome/5`。
- 唯一维护 CLI 是 `cmd/tour-i18n`。
- 页面身份使用 `data/tour-pages.tsv` 中冻结的持久 `page_id`，不会以页面位置或临时语义 key 替代。
- 项目根 `.env` 的安全加载已完成：系统环境变量优先，`.env` 不覆盖已有值；`/.env` 被 Git 忽略，`.env.example` 可提交。

## 翻译执行与校验

- 2026-08-23 起，正式 TranslationUnit 翻译流程切换为 `export → Codex GPT-5.6 Sol High → raw-responses/ → process → automatic validation`，ChatGPT 统一承担逐 TranslationUnit Quality Check。仓库根 `AGENTS.md` 是 Codex 自动导航入口，`docs/CODEX_TRANSLATION.md` 是当前 Codex 执行规范。
- 正式模型输入由 batch `manifest.json`、manifest 列出的全部 `inputs/*` 与 `locales/<locale>/glossary.yaml` 共同构成，缺一不可；glossary 必须在生成译文前读取。
- Retry 只用于 `restore_failed`、`validation_failed`；validation 通过后的 B/C/D 质量问题使用新的 revision batch。当前 Quality Check 与 Final Review 均采用 A-only 严格生产 gate。
- 历史上 zh-CN 103 页由 ChatGPT 完成、提升并发布的事实不变；本次切换只改变后续正式 Translation Engine 和执行规范，不重写历史 evidence。

- GLM-5.2 单页完整 `present.Section` 翻译执行器已完成；request、response 和 validation 按 locale、page_id、source hash 与 attempt 编号保留。
- 正式模式对一个 source 最多执行 3 次完整页面 attempt；失败后进入 `blocked`，blocked 页面不能继续正式重试。
- `translate run -dev` 仅用于单页开发校准：每条命令只执行 1 次 attempt，可从 pending 或 blocked 继续且不受正式三次上限约束；失败回到 pending，成功进入 ready。正式批量翻译不得使用 dev 模式。
- 正式发布投影中，`welcome/1` 使用去除条件标记后的 `a remote server.` 分支，本地原始 Tour 仍使用 `your computer.` 分支；`welcome/4`、`welcome/5` 使用去除 `#appengine:` 标记后的完整 Section。candidate validator 会拒绝 projected source/candidate 中残留的 appengine 标记，并继续校验 present 结构、directive、链接 target、行内代码和预格式化代码。
- legacy present 程序字体语法已按官方语义保护：例如 `` `package`rand` `` 作为一个完整代码单元，语义内容为 `package rand`，token 恢复后仍逐字保持原始 present 源写法。
- 固定 zh-CN 翻译提示词与 glossary 控制说明已统一为简体中文；protected token 必须完整、唯一，且不得修改、删除、复制或伪造。为适应目标语言自然语序可以调整 token 位置，但不得破坏链接、代码、directive、预格式化等结构关系；这是面向后续多语言扩展的结构安全原则，不是中文特例。
- protected token 已记录 kind 元数据。恢复前严格检查总数、未知 token、缺失和重复，不再要求全部 token 保持一条全局原始顺序；随后按模型响应中的实际 token 顺序确定性规范化独立 inline-code token 的外部边界。`` `package`rand` `` 等 legacy 单一代码 span 继续作为完整单元保护，不会被拆分。
- 预格式化 Go 教学代码已支持安全翻译自然语言行注释：所有非注释代码逐字保护，注释内引用的 Go 标识符单独保护；candidate validator 会独立检查注释结构、非注释字节及标识符的数量、大小写和顺序。
- candidate validator 基于 present 解析结果校验斜体、粗体和 program span 的类型与结构。inline-code 校验内容和数量（多重集合）及 program span 安全，但不再根据跨语言出现顺序判断技术语义；legacy program span 继续完整保护。跨语言技术语义是否正确不伪装成静态结构校验能力，仍须由提示词与人工审核把关。
- link target 继续严格校验内容、数量和自身顺序；preformatted block 继续严格校验内容、块顺序和代码安全；directive 继续严格校验内容、数量、自身顺序和 Section 归属。Section 拓扑、预格式化块与 directive 的 Section 归属均受校验；源中位于 Section 尾部的 directive 在候选中仍必须位于尾部。
- zh-CN glossary 对强制链接 label 使用 protected token 确定性恢复，例如 `A Tour of Go` → `Go 语言之旅`、`previous` → `上一页`、`next` → `下一页`、`Run` → `运行`、`Format` → `格式化`；validator 仍执行防御性术语与禁用译法检查。
- 已支持 ready candidate 单页本地预览。运行 `go run ./cmd/tour-i18n preview -id welcome/1 -locale zh-CN` 会在 `/tmp` 创建临时 Tour 内容副本，只替换目标 Section，不修改仓库正式 `_content`。

### 2026-08-10 翻译输入架构实验

- 已提交 `62b1b1a feat: 增加翻译输入实验与最小保护能力`，新增 `--raw-input`、`--minimal-protect` 和受控同进程开发重试 `--dev-attempts`。
- `--raw-input` 将完整 production 页面直接发送给 GLM-5.2，不执行 protected token/restore；`flowcontrol/6` 首次真实 raw-input 请求即通过统一 validator，usage 为 1,403 tokens。该页旧默认 protected-token attempts 1～3 的失败，属于完整 inline-code 结构为适应中文语序换位后被旧保护/恢复逻辑误判。
- `methods/24` 两次纯 raw-input 均真实破坏 present 结构：模型会为 `.play methods/images.go` 追加参数，并将链接普通标签中的 `image`、`image/color` 自行改为新的 inline code；完全 raw 当前不足以作为默认生产方案。
- `--minimal-protect` 当前只保护完整 `.play` directive。`methods/24` 的唯一 `.play` token 被模型原样且恰好一次保留，并精确 restore，directive 问题消失；但链接普通标签新增 inline code 仍存在。随后已增加该失败的针对性 retry feedback 与同进程 `--dev-attempts`，后续实验又出现 font span count mismatch，说明当前 minimal-protect 尚不足以稳定替代默认完整保护流程。
- 当前结论是：大量 protected token 并非越多越好，会增加提示上下文、restore 复杂度，并可能妨碍符合中文习惯的自然语序调整；完全 raw 又不能稳定保护 present 机器结构。长期方向应为“原始页面优先 + 少量真正高风险机器结构保护 + 严格统一 validator + 针对性 retry feedback”。目前暂停继续扩大 minimal-protect，不主动为每种潜在结构增加保护规则；成熟的默认 protected-token 流程继续作为正式翻译路径。
- 正式翻译状态与 candidate 已恢复至实验前状态；真实实验 attempts 审计继续保留在 `data/translation-runs`。

### 2026-08-17 翻译质量与 Translation Engine 评估

- 对 7 个代表页的 Default protected-token 信息损失审计确认：157 个保护标记没有隐藏任何需要翻译的英文自然语言，hidden translatable English 为 0 项、0.00%。Default 与 minimal-v1 都保留完整页面级自然语言上下文。
- 5 个固定页面的重复实验确认，GLM-5.2 在实际 API Request 相同的情况下 response 仍存在运行间波动，且波动可能改变 validator pass / fail；项目不推测服务端原因。
- 5 页 × 2 模式 × 3 次的重复实验中，Static Context 未显示稳定优势：Default usable/planned 为 12/15，Static Context 为 9/15。Default protected-token 继续作为当前正式翻译执行路径，不再优先投入 minimal-protect / Static Context 的细微调优。
- 已完成 `methods/24`、`concurrency/7`、`concurrency/11` 三页，ChatGPT、Codex、GLM-5.2 三种候选来源，ChatGPT、DeepSeek、GLM-5.2、豆包四个 judge 的 12 次匿名评审。ChatGPT 获得 8/12 第一名；排除 ChatGPT self-judge 后仍获得 5/9 第一名，当前是下一阶段最值得优先验证的高质量翻译来源。
- 截至 2026-08-17 本轮实验结束时，zh-CN 103/103 ready、canonical candidates 与 production 均保持不变；本轮实验不意味着现有 103 页必须重翻，也不表示 ChatGPT 已接入正式 Translation Engine。
- 当时确定的下一阶段是设计尽可能自动化的 ChatGPT retranslation staging，先保存独立候选并与现有 canonical candidate 比较，达到替换标准后才考虑正式切换，不立即覆盖现有译文。
- 完整实验设计、排名、统计、质量观察和工程决策见 [TRANSLATION_QUALITY_EXPERIMENTS.md](TRANSLATION_QUALITY_EXPERIMENTS.md)。

### 2026-08-18 ChatGPT 全量重译、canonical promotion 与最终验收

- 已完成 zh-CN 全部 103 个正式页面的 ChatGPT 整页重译、批次 process/retry、统一 validator、独立语义审计和正式 canonical promotion。最终语义质量审计为 A=103、B/C/D=0；当前仓库 canonical candidates 103/103 均为本轮正式提升后的 ChatGPT 译文。
- 重译历史封存在 `data/retranslation-runs/zh-CN/chatgpt-zh-CN-001` 至 `011`，作为不可变 evidence。最终 revision 包括 Batch 009、010、011；`methods/17`、`methods/19` 最终来自 Batch 011。两个正式 retry 页面为 `moretypes/1` attempt-002 和 `concurrency/1` attempt-002。
- retranslation workflow 已完整实现 export/process/retry/promote。process 重新生成 Default protected input，并完成 raw response → restore → batch candidate → validator；retry 要求连续 raw attempt 和 validation history。promotion 默认 dry-run，只有显式 `--apply` 才修改 canonical/status；每页优先使用包含该页的最新批次，最新批次失败时禁止回退旧成功候选。
- promotion preflight 验证 Catalog 与 manifest metadata、source/input hash、saved input、当前 glossary 重建的 protected input、最终 raw attempt 的连续 provenance、restore 与历史 batch candidate 的字节一致性，以及 locale-aware canonical validator。历史 batch candidate 不会被修改；plan 分开记录原始 `source_candidate_sha256` 和最终 `candidate_sha256`。
- batch candidate → canonical candidate 边界采用唯一允许的确定性 EOF normalization：只移除多余尾部 LF，最终保证恰好一个 LF，不改变正文、中间空行、尾部空格或其他字节。正式 apply 结果为 103 pages、102 changed、1 unchanged、EOF normalized=80；apply 后 dry-run 为 0 changed、103 unchanged。
- promotion 后 103/103 状态均为 `ready`，原 `Attempts` 与 `SourceSHA256` 保留，canonical path 正确，note 记录来源 ChatGPT batch。promotion 数据一致性审计与 `status check` 均通过；正式 canonical promotion 数据提交为 `bfa9f929dd0f425adc97ba4b6ae390a8a50e071a`，验收测试基线提交为 `d2595f3d57cc2ea8c7a4df3034fb03e8c4c325b0`。
- `go test -mod=readonly -count=1 ./...` 全部通过。临时 production publish bundle 成功，结果为 `ready=103`、`pending=0`、`blocked=0`、`pages=103`、`articles=7`；`release.json` 正确，`SHA256SUMS` 全部通过，bundle 不含 status、candidates、retranslation-runs 等开发态数据。
- 临时 production binary HTTP 验收通过：103/103 正式课程路由与 7/7 article endpoint 均返回 HTTP 200；首页、`/tour/`、`/tour/list`、robots、sitemap、script 正常；`welcome/1` 使用 remote execution 分支；HTTP transport 正常且 SocketTransport 未启用；`/socket` 和保留路径按设计禁用。sitemap 仍为 104 个唯一 URL（首页 + 103 页），是否加入 `/tour/list` 使其变为 105 是独立 SEO 后续事项。
- 本轮 ChatGPT canonical 已生成正式 production release `/data/go-tour/releases/20260818-zh-CN-45f4cad`，对应仓库 commit `45f4cad98e67c01dac705559781e2311b75b0948`，`published_at=2026-08-18T13:22:48Z`。production deployment 脚本执行成功并以退出码 0 结束，`/data/go-tour/current` 已切换至该 release；service 为 active，origin 与 public acceptance 均为 HTTP 200，deploy lock 已清理。
- 最终线上验收全部通过：首页、`/tour/`、`/tour/list`、`robots.txt`、`sitemap.xml`、`tour/script.js` 均返回 HTTP 200；103/103 正式课程路由返回 HTTP 200；7/7 `/tour/lesson/<article>` 公网响应与本次正式 release 本地 binary 响应逐字节完全一致。新 canonical marker“每个 Go 程序都是由包组成的。”已在线，旧 marker“每个 Go 程序都由包构成。”已不再出现。
- production runtime 验收确认 `HTTPTransport=true`、`socketAddr` 为空、Playground base 为 `https://play.go-dev.shuijingwanwq.com:8443`，SocketTransport 未启用；`/socket`、`/socket/anything`、`/_/share` 均按设计返回 404。真实 Run 成功并输出 `ONLINE_RUN_OK`，真实 Format 成功且输出包含 `ONLINE_FMT_OK`。sitemap 当前仍为 104 个唯一 URL。

## zh-CN 课程正文完成状态

- 当前仓库／当前 production 的统一 TranslationUnit workflow 共 122 个 unit：103 个 Page 与 19 个 eligible Example；另保留 93 个 referenced Example source inventory 和 2 条条件源页面审计记录。当前状态为 `ready=122`、`pending=0`、`blocked=0`。
- 103 个正式发布页面均已完成翻译；当前 canonical 103/103 为 2026-08-18 正式提升后的 ChatGPT 整页译文，状态为 `ready`。最终语义质量审计为 A=103、B/C/D=0，缺失、重复和额外页面均为 0。
- 此前第一阶段译文及其全局质量审计仍是重要历史基线；本轮 ChatGPT 重译通过 Batch 001–011 保留完整 evidence，并经独立语义审核、全量 promotion preflight 与 production bundle/HTTP 验收后完成 canonical 切换。
- 特殊投影已纳入上述审计：`welcome/1` 使用 appengine remote 分支 `a remote server.`；`welcome/4`、`welcome/5` 使用完整 `#appengine:` 条件 Section 去前缀后的投影。
- 已形成的翻译执行结论继续有效：Protected Token 保护 payload 与结构角色，允许为目标语言自然语序调整位置但不得破坏 present 结构；静态校验持续覆盖链接、代码、directive、预格式化块及其拓扑关系。`flowcontrol/10`、`moretypes/1` 的校准和 raw-input/minimal-protect 实验均已形成结论，不是当前待办。

本阶段 glossary 已新增 preferred 术语：`standard library` → `标准库`、`iteration` → `迭代`、`loop condition` → `循环条件`。新增 mandatory 术语：`type switch`、`type switches` → `类型选择`，`type assertion`、`type assertions` → `类型断言`，`interface value` → `接口值`，`interface type` → `接口类型`，`concrete type` → `具体类型`。`square root`、`Newton's method`、`derivative` 等单页术语暂不加入。

`generics/1` attempt-004 与 `methods/20` attempt-001 共同证明，全局 protected token 顺序要求会误伤正确的跨语言自然语序；该调整是通用多语言恢复与候选校验修复，而非单页特例。

## 公共 UI 本地化完成状态

- 第一阶段公共 UI 本地化已经完成。UI 文案与课程 `.article` 翻译继续分开维护，且未将官方英文 UI 直接硬编码替换为中文。
- 统一 locale UI catalog 位于 `internal/tour/ui/`：`en.json` 是冻结 upstream UI source/canonical catalog，`zh-CN.json` 是第一阶段正式中文 UI catalog。实现使用 JSON locale 数据、Go `embed` 和标准库加载，并严格校验 key coverage 与 message kind；正式 locale 缺 key 会初始化失败，不提供英文 fallback。
- 当前为一个构建/进程选择一个 locale 的构建时单语言体系，不实现运行时动态语言切换；未引入数据库、Web 审校平台或第三方复杂 i18n 框架。新增语言原则上只需增加合法 locale JSON 并通过完整性校验，无需修改 catalog loader 注册代码。
- 已完成的 consumer 包括：
  - 页面框架：`html lang`、页面 title、顶栏“Go 语言之旅”及主题切换的 `aria-label` / `alt`。
  - JavaScript 公共 UI：TOC、`Waiting for remote server...`、feedback 入口及 feedback issue 模板。
  - 课程列表：`欢迎来到 Go 语言之旅`、5 个模块标题和 5 个模块说明。模块 title 为 plain message；description 保持冻结 upstream 的 rich HTML 语义并受严格受限 markup 校验，第一项保留 go.dev 链接。
  - 编辑器：`Syntax` → “语法高亮”、`Imports` → “导入”、`Run` → “运行”、`Kill` → “终止”、`Format` → “格式化”、`Reset` → “重置”。
  - 执行状态：`Program exited` → “程序已退出”。
- 已通过 `tour-i18n preview --locale zh-CN --id basics/6` 的实际浏览器检查：强制刷新后页面框架、`/tour/list`、模块标题和说明、go.dev 链接及编辑器已接入文案均正常；中文长度未造成明显布局问题，保留英文的 on/off 实际观感可接受。单页 preview 只替换目标 Section，因此列表页和 TOC 中其他课程 title / description 仍显示英文是该 preview 机制的既定行为，不是公共 UI 本地化遗漏。

### 第一阶段主动保留项与部署边界

- CSS 生成的 `on` / `off` 继续保持 upstream 原样；不为两个状态词修改 CSS、DOM 或 Angular。
- `404 page not found` 继续使用 Go 标准库 `http.NotFound`。这是低频异常路径，第一阶段不为本地化接管标准库 404 行为。
- playground 动态技术状态（例如 `killed`、`status N.`）继续保持 upstream / 技术状态原样。
- HTTPTransport 部署变体的 `Go vet failed.`、`Go build failed.`、`Error communicating with remote server.` 对应 message 继续保留在 catalog；production runtime 已确定使用 HTTPTransport，当前运行链路保持 upstream 的技术错误文案边界。

### Upstream 同步维护结论

- UI consumer 开始前的基准为 `519cc38`。经 SHA-256 / 文件核对，以下 upstream-facing 文件在该基准时与冻结 upstream `e11dacba76c5aae474746e9eedee19693f492803` 一致：`_content/tour/template/index.tmpl`、`_content/tour/static/js/app.js`、`_content/tour/static/js/controllers.js`、`_content/tour/static/js/directives.js`、`_content/tour/static/js/values.js`、`_content/tour/static/partials/list.html`、`_content/tour/static/partials/editor.html`。
- 当前 UI 本地化对 upstream 的改动集中于文案绑定、少量 i18n 接线和数据来源替换，未进行明显 UI 结构性重构。未来 upstream 同步时重点复核上述少量 consumer 文件；继续遵守“低收益 + 高侵入的边缘文案宁可保留 upstream，不为零英文扩大维护面”的原则。
- 已提供受限的人工 source 同步维护入口：先以 `upstream preview --source-root <website>` 只读检查，再人工复制已确认的 `_content/tour` 变化；当 Page route 与 conditional identity 保持不变时，使用 `catalog write --allow-source-change` 重建三个 source catalog。该命令不修改 locale status、candidate、review evidence 或 translation runs；Page 新增、删除、移动和无法识别的身份变化仍会拒绝。

#### 2026-08-20 upstream source 同步、stale retranslation 与生产验收

- active frozen upstream baseline 已从 `e11dacba76c5aae474746e9eedee19693f492803` 更新至 `645042eb697eaf69e33a9af00c6b5b3fffdead5a`。
- 本次官方 Tour source 的实际变化为：`welcome.article` 删除旧的简体中文社区翻译链接；`basics.article` 将 `var` 改为 inline code 标记；93 个 referenced Example source 均无内容变化。
- 同步流程为：`upstream preview --source-root ~/code/go-website-upstream` → 人工复制并确认 source 变化 → `catalog write --allow-source-change`。最终 preview 为 Page `unchanged=103`、`content_changed=0`；Example `unchanged=93`、`content_changed=0`；conditional `unchanged=2`；`safe for catalog write=true`、`manual mapping required=false`。
- catalog source hash 更新后，`welcome/2` 与 `basics/9` 的既有 `ready` status 因 source revision 变化进入 stale。`retranslation export` 已支持将 `ready + stale` 的当前 source revision 重新导出，而不会因历史 `unit_id` 已存在而永久跳过。
- 已生成 Batch `chatgpt-zh-CN-013`，包含 2 个 Page unit：`welcome/2`、`basics/9`。两页均完成 `export → ChatGPT raw response → process → restore → validation → Translation Quality Review → promotion`；process 结果为 `restore_passed=2`、`restore_failed=0`、`validation_passed=2`、`validation_failed=0`；review 为 2/2 approved，均为 A。
- promotion preflight 结果为：`unit_count=122`、`page_count=103`、`example_count=19`、`changed_count=2`、`unchanged_count=120`、`review_approved_count=122`、`can_apply=true`。apply 后状态为：`status OK: 122 translation units for zh-CN (103 pages, 19 examples)`。
- 本次真实 upstream revision 场景修复了两个 promotion 问题：历史 batch 属于旧 source revision 时，不应阻塞当前 Catalog，也绝不能 fallback 为当前 source 的 promotion evidence；stale canonical status 是“当前 source 需要重新 promotion”的合法过渡状态，只要当前 source revision 已有完整的 passed validation 和 approved review evidence，promotion 应允许原子更新 candidate/status。
- 修复后的规则为：历史 batch 保持不可变；promotion 仅在 source metadata 与当前 Catalog revision 匹配的 batch 中选择最高 batch number；旧 source revision evidence 被保留但不参与当前 revision promotion；当前 revision 没有 evidence 时进入 MissingEvidence；stale status 仅在当前 revision 的 preflighted promotion evidence 覆盖该 unit 时允许 apply；apply 后恢复严格 `CheckStatus`。对应正式提交为 `a1112043fd82234f2c2ba91e9a119719dbc7e7f7`（`fix: 支持 upstream 变更后的重译 promotion`）。
- 当前仓库／当前 production 的统一 workflow 状态为：translation units=122、Pages=103、eligible Examples=19、referenced Example source inventory=93、conditional source records=2、`ready=122`、`pending=0`、`blocked=0`、articles=7。93 个 Example 是被引用的 source inventory，不是 93 个需要翻译的 Example；只有当前 upstream source 含普通自然语言注释的 19 个 eligible Example 进入 locale translation/status/review/promotion workflow，其余 74 个直接继承 upstream source。
- 最终本地验收已完成：`go test ./...` 全部通过；`tour-i18n build --locale zh-CN` 输出 `ready=122 pending=0 blocked=0 pages=103 articles=7`；production publish bundle 在本地成功。其 `release.json` 为 `schema_version=2`、`upstream_commit=645042eb697eaf69e33a9af00c6b5b3fffdead5a`、`translation_units=122`、`pages=103`、`eligible_examples=19`、`articles=7`、`execution_transport=http-playground-proxy`、`execution_provider=play.golang.org`、`local_socket_enabled=false`。
- 本次 upstream revision 与 Batch 013 的 stale retranslation 已于 2026-08-20 正式部署。当前线上 production release 为 `/data/go-tour/releases/20260820-zh-CN-6ae139c`，`current` 已原子切换至该 release；`deploy-production.sh`、service health 与公网 acceptance 均通过。当前 production workflow 为 `ready=122`、`pending=0`、`blocked=0`、`pages=103`、`eligible_examples=19`、`articles=7`，upstream 为 `645042eb697eaf69e33a9af00c6b5b3fffdead5a`。

## 完整语言正式投影与本地预览完成状态

- 当前 upstream 为 `master@645042eb697eaf69e33a9af00c6b5b3fffdead5a`。
- 当前仓库／当前 production 的 zh-CN workflow 状态为 `ready=122`、`pending=0`、`blocked=0`；catalog 为 103 个 published pages、19 个 eligible Example 和 2 条 conditional source records（另有 93 个 referenced Example source inventory）。
- 已完成全部 103 页 canonical candidate、全局翻译质量审计、zh-CN 公共 UI 本地化、完整语言正式投影与完整语言本地预览。
- `tour-i18n build --locale <locale>` 从 catalog、locale status 与 canonical ready candidate 构建完整正式投影；它拒绝 pending、blocked、缺失、额外或非 canonical candidate，不回退到英文或旧译文，也不修改 candidate、status 或 catalog。
- `tour-i18n preview --locale <locale>` 直接复用完整投影能力启动本地预览；带 `--id <page_id>` 时继续是单页 candidate preview。
- zh-CN 完整投影验收结果为 `articles=7`、`pages=103`、HTTP `103/103`。全部 catalog page_id 的 `/tour/<page_id>` 页面返回正常 zh-CN Tour HTML，且对应 lesson Section 可渲染。
- `welcome/1` 的 remote server 分支、`welcome/4` 与 `welcome/5` 的完整条件 Section 去前缀投影均已在完整正式投影和 HTTP 验收中通过；zh-CN 公共 UI 与关键静态资源也已验证。

## Article/lesson metadata 多语言本地化完成状态

- 在 103/103 HTTP 验收后的人工浏览器截图中，发现 7 个 article 的课程导航主标题仍直接使用 upstream 英文。进一步审计确认同一文档层还存在 7 个 `present.Doc.Subtitle`。
- 根因是当前 103 个 `page_id` catalog 只覆盖顶层 `present.Section`，不覆盖 article 文档根级的 `present.Doc.Title` 与 `present.Doc.Subtitle`；因此 Section candidate 校验、状态以及 103/103 HTTP 页面状态验收均不会定义或校验这一层译文。
- 已新增独立的 locale article metadata 资源：`locales/<locale>/article-metadata.json`。该资源不属于 Section candidate、公共 UI 或 status；每个正式 article 单独维护 `title` 和 `subtitle`。
- metadata loader 由 catalog 动态推导正式 article 集合并严格校验：article 集合必须精确一致，title/subtitle 必须非空；缺失、额外、重复或无法解析的 metadata 都会失败，不允许 fallback 到 upstream 英文。
- 完整 projection 与单页 candidate preview 共用 metadata 应用路径。完整 projection 最终重新解析每个 article，并断言 `Doc.Title`、`Doc.Subtitle` 与 locale metadata 完全一致，同时继续校验全部 103 个 canonical Section。
- 当前仓库／当前 production 的 zh-CN 状态为：103/103 Section Page candidate ready、19/19 eligible Example ready（合计 `ready=122`、`pending=0`、`blocked=0`）；article metadata 7/7；title localized 7/7；subtitle localized 7/7；公共 UI 已本地化。
- 修复后完整预览确认课程导航 lesson title 与 lesson description 均已中文化；全部 103 页 HTTP 验收继续为 103/103，且未发现同类根级 upstream 英文 metadata 残留。article metadata 不计入正式页面数，catalog 仍为 103 个 published pages。

## Production runtime、publish bundle 与上线验收

- production runtime 已完成并正式部署：正常浏览器 Run / Format 使用 `HTTPTransport` 直连自建 ZgoCloud 固定代理的 `/compile` 和 `/fmt`，由其转发到官方 Playground `play.golang.org`；不再经过阿里云 Go 服务端。旧的 `/_/compile` 和 `/_/fmt` 服务端代理仍保留作为兼容/回滚路径，并按 go.dev 要求保留 form 编码；不注册本地 `SocketTransport` 或 `socket.NewHandler`。
- production runtime 的 `/socket` 普通请求和 WebSocket Upgrade 均返回 404；`/_/share` 当前未启用并返回 404；未知 `/_/` 路径不会落入 Tour SPA。production 主机不执行用户提交的 Go 程序，真实 Run / Format 已完成公网验收。
- production publish bundle 已在 commit `e4a8fe0 feat: 增加确定性的生产发布包生成` 中完成。正式命令为 `go run -mod=readonly ./cmd/tour-i18n publish --locale zh-CN --output <directory>`，输出包含 `bin/tour`、`_content/`、`release.json` 和 `SHA256SUMS`。
- publish 构建时将 locale 固定进 production binary；binary 从自身相邻的 `../_content` 定位内容，不依赖当前工作目录，不需要运行时 `--locale` 或 `--content`。bundle 不包含 candidate、status、translation-runs 等开发期数据。
- 2026-08-18 历史 release bundle 的验收结果为：`ready=103`、`pending=0`、`blocked=0`、`pages=103`、`articles=7`；103/103 课程页面、7/7 lesson JSON、103 个 Section、title/subtitle 和公共中文 UI 均通过验收。
- 两次 publish 在相同源码、Go 工具链和 GOOS/GOARCH 下逐文件一致；185 个 bundle 文件 SHA-256 校验全部通过。真实 Run / Format、`/socket` 404、WebSocket `/socket` 404、`/_/share` 404 及未知 `/_/` 404 均已验证。
- 当前正式上线状态：production release 为 `/data/go-tour/releases/20260820-zh-CN-6ae139c`，`current` 已指向该 release；release manifest 为 `schema_version=2`、`locale=zh-CN`、`published_at=2026-08-20T15:04:14Z`、`upstream_commit=645042eb697eaf69e33a9af00c6b5b3fffdead5a`、`upstream_commit_time=2026-08-20T05:56:11Z`、`translation_units=122`、`pages=103`、`eligible_examples=19`、`articles=7`、`execution_transport=http-playground-proxy`、`execution_provider=play.golang.org`、`local_socket_enabled=false`、`goos=linux`、`goarch=amd64`。`deploy-production.sh` 已完整成功执行：本地 bundle preflight、远端 permissions 与 SHA-256 verification、staging → final release、current 原子切换和 service restart 均通过；`go-tour.service` 为 active (running)，日志确认以 remote Playground execution 在 `http://127.0.0.1:3999` 服务 zh-CN，连续 3 次 localhost HTTP health check 均为 200，首页 `https://go-dev.shuijingwanwq.com/` 的 public acceptance 为 HTTP 200，且 `/`、`/tour/welcome/2`、`/tour/basics/9` 均为 HTTP 200。部署未发生自动回滚。页面访问链路为浏览器 → EdgeOne → 阿里云 Nginx → Go Tour；正常 Run / Format 链路分别为浏览器 → `https://play.go-dev.shuijingwanwq.com:8443/compile` 或 `/fmt` → ZgoCloud Nginx → `https://play.golang.org/compile` 或 `/fmt`。ZgoCloud 是本项目自建固定反向代理，不是 Go 官方服务；`play.golang.org` 才是最终官方 Playground provider。正常生产 Run / Format 不再经过阿里云 Go 服务端，旧 `/_/compile`、`/_/fmt` 仍保留为兼容/回滚路径。`go-tour.service` 监听 `127.0.0.1:3999`；remote Playground execution 保持，`local_socket_enabled=false`，production `/socket` 与 `/socket/` 仍为 404，生产服务器不执行用户提交的 Go 程序。
- ZgoCloud 正式接口已完成真实验收：`/compile` POST 返回 HTTP 200，`/fmt` POST 返回 HTTP 200，CORS OPTIONS 返回 204，错误 Origin 返回 403，GET `/compile` 返回 405，未知路径返回 404，旧 `/_/compile` 与 `/_/fmt` 返回 404；HTTP/2 正常。443 继续由 Wstunnel 使用，8443 由 Nginx 使用，WireGuard 与 Wstunnel 未受此次部署影响。
- 桌面 Firefox 已完成真实生产浏览器验收：Run 实际请求为 `POST https://play.go-dev.shuijingwanwq.com:8443/compile?backend=...` 并返回 HTTP 200，示例输出为 `Hello, 世界` 和 `程序已退出。`；Format 已切换至 `https://play.go-dev.shuijingwanwq.com:8443/fmt`，响应 CORS 允许来源为 `https://go-dev.shuijingwanwq.com`。
- 本次发布发现重要的 EdgeOne 缓存运维问题：新 release 切换后，缓存的 `/tour/script.js` 仍可能使浏览器继续请求旧的 `/_/compile`。定向刷新 `https://go-dev.shuijingwanwq.com/tour/script.js` 并重新加载页面后，Run / Format 立即切换到 ZgoCloud 新接口。今后涉及前端执行路径变化时，发布验收必须检查 Network 中的实际请求目标，必要时定向刷新该脚本缓存。
- 2026-08-13 手机端在未开启 VPN 的真实网络环境下完成生产 Run 验收：页面正常使用，点击“运行”后结果很快返回，用户体感明显优于迁移前阿里云 → Google/Playground 路线偶发卡顿的情况。该实测证明新浏览器 → ZgoCloud → `play.golang.org` 链路在该公网环境下可用且响应快速；结合此前 ZgoCloud → `play.golang.org` 连续测试和桌面浏览器验收，多层证据支持本次迁移方向有效，但不据此推断所有中国大陆运营商和所有时间均稳定。
- “生产 Playground 执行链路迁移至 ZgoCloud 固定代理”已完成生产部署和真实用户验收。后续仅进行运维观察（包括 ZgoCloud 执行稳定性）；是否删除阿里云旧 `/_/compile`、`/_/fmt` 兼容代理，留待后续评估，不作为当前必须执行事项。
- 旧 release `/data/go-tour/releases/20260811-zh-CN-925d59d` 继续保留用于回滚；临时 4005 smoke 实例已停止，服务器上传压缩包已清理。此次 smoke 中远程 Playground 曾返回上游 502，属于已知外部依赖波动，不视为本次发布失败。
- 2026-08-13 已完成 production 自动部署脚本首次真实生产验证。脚本提交为 `3852bc1c2b001ed1fa3c640c28aaa696f6ab9c80 feat: 添加生产发布自动部署脚本`；基于 release `/tmp/go-tour-release-20260813-zh-CN-3852bc1`（`locale=zh-CN`、`ready=103`、`pending=0`、`blocked=0`、`pages=103`、`articles=7`）完成上传、权限归一化、SHA-256 校验、权限验证、current 原子切换、服务重启、连续健康检查和公网验收。首次健康探测为 `active + HTTP 000`，随后连续 3 次 `active + HTTP 200` 后才判定成功，验证了连续 localhost 健康检查的必要性。最终 current 为 `/data/go-tour/releases/20260813-zh-CN-3852bc1`，deployment lock 已释放，service 与 localhost 均正常，正式域名返回 HTTP 200；未发生回滚、人工恢复或 staging/lock 残留。
- 2026-08-12 已完成生产入口域名迁移：正式生产域名为 <https://go-dev.shuijingwanwq.com/>，A Tour of Go 继续使用 `/tour/` 路径。迁移原因是避免 `go-tour.../tour/...` 中 Tour 语义重复，并为未来可能扩展 go.dev 的其他翻译内容保留更宽泛的站点入口；此次仅迁移生产入口，不是应用重新部署，既有 release、commit、`go-tour.service`、`/data/go-tour/` 和生产链路均不变。
- 旧域名 <https://go-tour.shuijingwanwq.com/> 继续保留 Cloudflare DNS 与 EdgeOne 接入，但仅作为 EdgeOne 永久 301 入口：`go-tour.shuijingwanwq.com/*` 重定向至 `go-dev.shuijingwanwq.com` 同路径并保留查询参数；不再回源。实测 `/tour/welcome/1` 返回 HTTP/2 301，Location 正确指向新域名。
- 新域名已完成 Cloudflare CNAME → EdgeOne、HTTPS 证书、HTTPS 回源及回源 Host 切换；源站为 `121.40.248.29`，正式回源 Host 为 `go-dev.shuijingwanwq.com`。Nginx 使用 `/usr/local/nginx/conf/vhost/go-dev.shuijingwanwq.com.conf` 和对应新证书，反向代理仍为 `http://127.0.0.1:3999`；旧 go-tour 虚拟主机、证书及备份配置已从源站清理。
- 服务器 OneinStack 正式维护目录统一为 `/root/oneinstack`；旧版已在完成验收后清理。新虚拟主机通过当前 OneinStack 的 `./vhost.sh --proxy --dnsapi` 创建，使用 Cloudflare DNS provider、Let's Encrypt、HTTP → HTTPS 和 `proxy_pass http://127.0.0.1:3999`。敏感 API 凭据不记录于本文档。acme.sh 当前仅保留 `go-dev.shuijingwanwq.com` 的 `ec-256` Let's Encrypt 记录，旧域名续期记录及证书已清理。
- OneinStack 自动生成配置中会截获 Tour 静态资源的两段 `location` 已删除；`/tour/static/css/app.css` 和 `/tour/static/lib/codemirror/lib/codemirror.css` 均已验证返回 200。当前环境使用 `nginx -t && service nginx reload` 使配置生效；`service nginx configtest` 不可用，仅支持基础 LSB 动作。
- 最终验收：新页面 `/tour/welcome/1` 返回 200，新静态资源返回 200，旧域名按预期 301 至新域名；浏览器中文页面、HTTPS、地址栏保持 go-dev 及 Go 示例远程运行均已验证。此前一次远程运行超时为偶发问题，随后运行成功，不属于本次域名迁移阻塞项。
- 部署期间已解决动态链接 glibc 兼容、release 目录权限和 Nginx systemd 接管问题。Cloudflare 仅负责 DNS，正式流量经过 EdgeOne，不采用 Cloudflare 双层代理。

## 第一阶段上线冻结

- Section：103/103 ready；article metadata：7/7；公共 UI：已完成；production publish：已实现并正式部署；production 自动部署脚本：首次真实生产验证通过；浏览器最终验收：通过；生产 Playground 执行链路迁移至 ZgoCloud 固定代理：已完成生产部署和真实用户验收。
- 本轮 ChatGPT 全量重译、canonical promotion、upstream revision stale retranslation、正式 production release、生产部署和线上最终验收均已完成。当前在线 release 为 `/data/go-tour/releases/20260820-zh-CN-6ae139c`，正式服务内容已切换至 upstream `645042eb697eaf69e33a9af00c6b5b3fffdead5a` 的 122-unit workflow（103 Pages、19 eligible Examples）。

## 下一阶段

- `/tour/list` 是否加入 sitemap（104 → 105）作为独立 SEO 完整性事项评估；当前 sitemap 仍为 104 个唯一 URL，不将该事项混入已完成的重译与上线状态。
- 按冻结 upstream 基线与现有同步规则继续评估后续 upstream 同步。
- 在复用现有 locale catalog、validator、projection、publish 与 deployment 能力的基础上，评估未来其他语言扩展。

## 站点首页、公共页脚与项目互链

- 根路径 `/` 已从课程自动跳转改为 locale 化的项目首页；`/tour/` 继续承载 A Tour of Go，当前没有 `/about` 页面。
- 首页以构建时选定的 UI locale 渲染项目介绍、当前翻译项目、项目状态、语言版本、项目链接及面向读者的翻译与校验说明；不实现运行时语言切换。
- `internal/tour/project.go` 是稳定项目配置的单一来源，包含正式站点、官方 Tour、GitHub、Issues、开发记录、upstream、备案和版权主体 URL/标识。
- publish 在 bundle 内 `_content/tour/site-metadata.json` 生成 locale、显式 production bundle 发布时间、upstream commit / UTC 时间、课程页数和 article 数；运行时首页读取该文件。发布时间由 `publish --published-at <RFC3339 UTC>` 显式提供，因此不是 Git commit、服务启动或请求时间。
- production 发布时间只属于生成出的 release metadata：源码 `_content/tour/site-metadata.json` 是 `go run ./tour` 与 preview 使用的开发态 metadata，不保存或回写真实 production `published_at`，开发态首页显示“开发环境”；publish 仍在 bundle `_content/tour/site-metadata.json` 和根 `release.json` 写入同一真实发布时间，production 首页继续显示该发布时间。
- 所有主模板页面都使用公共页脚：项目身份、首页、GitHub、开发记录、版权和 ICP 备案入口。页脚采用普通自然文档流，非 fixed、非 sticky：首页正常显示 footer；`/tour/list` 的 footer 位于课程列表自然末尾并横跨页面；`/tour/<page>` 的 footer 位于课程主体和翻页导航之后，并横跨整个 Tour 页面。footer 不属于左侧 `.slide-content`，不再存在 fixed footer 的永久底部预留、`--site-footer-height` 一类高度补偿或 JavaScript 滚动监听；课程内容超过一屏时，footer 会在自然页面滚动到末尾后出现。
- 右侧 editor 的底部越界已修复：`#explorer + div` 原本同时使用 `top: 32px` 和 `height: 100%`，导致 wrapper 下移后仍保持完整高度并进入 footer 区域；现调整为 `top: 32px` 与 `height: calc(100% - 32px)`。人工验收确认 editor 在 footer 上方自然结束、footer 全宽完整显示且无遮挡，editor 内部纵向滚动正常。
- 正式生产域名为 <https://go-dev.shuijingwanwq.com/>；公开页面不展示旧 go-tour 域名。博客 → go-dev / GitHub 的反向链接仍属于仓库外待办，需由博客侧另行维护。
