# 新增 Locale 执行手册

本文档是新增一门社区语言的唯一高层入口。它把 locale 准备、翻译、语言表层审核、首次生产部署和上线验收串成一条发布路径，但不复制各阶段的正式细则。执行具体阶段时，必须进入本文引用的对应规范。

本文适用于新增 locale；已有 locale 的 TranslationUnit 修订仍从 [多语言翻译流程](TRANSLATION_WORKFLOW.md) 开始，日常生产发布直接使用 [生产运维手册](PRODUCTION_RUNBOOK.md) 的维护部署流程。

如果用户没有明确指定 locale，先从 [Locale 长期路线图](LOCALE_ROADMAP.md) 选择排名最高且尚未完成的 Standard locale。路线图只确定商业优先级与 candidate identity，不替代本手册要求的正式 identity freeze。

## 总体顺序与五条边界

```text
locale / domain / CDN 决策
→ locale init
→ locale glossary
→ 独立 Glossary Review → current passed gate
→ locale 配置与公共 UI catalog
→ 首页、导航、语言选择器与 article metadata
→ TranslationUnit 翻译与 automatic validation
→ Candidate Snapshot → Quality Check → machine finalization
→ promotion
→ 确认 canonical English source-description asset 与人工 review gate current
→ 以 canonical English description 固定范围，结合当前 Page 完整 source/ready target 与 glossary 离线生成 schema v2 description
→ assemble 并验证课程页 SEO metadata
→ build
→ surface-review reviewer-bundle（standalone export 仅用于诊断）
→ Locale Surface Review A
→ record current A gate
→ 完整 locale preview
→ automated rendered acceptance
→ visual HUMAN gate
→ production publish
→ 首次 production 基础设施、部署 profile 与广告接入
→ 最终源站、公网和浏览器上线验收
→ first-production finalize
→ production_state=live（同时自动更新中英文 README 的 live locale 投影）
→ search-engine submission closeout（Google → Bing → locale-specific → IndexNow 全站 bootstrap）
```

必须区分五类工作：

1. **Glossary Review**完整审核术语决策本身，规则见 [Glossary Review 规范](GLOSSARY_REVIEW.md)。它是下游 generation 的独立前置 gate，不审核 TranslationUnit candidate 或组合后的页面。
2. **TranslationUnit 质量审核**只审核进入 translation workflow 的 Page 和 eligible Example candidate，规则见 [Translation Quality Review](TRANSLATION_QUALITY_REVIEW.md)。它保持逐 TranslationUnit、Quality Check 全 A 与 machine finalization promotion gate。
3. **Locale Surface Review**审核 TranslationUnit 之外及组合后页面上的语言表层，规则见 [Locale Surface Review](LOCALE_SURFACE_REVIEW.md)。它是独立的 locale release gate，不生成 TranslationUnit review evidence，也不允许替代或弱化 A-only gate。
4. **首次生产部署**为新 locale 建立 hostname、CDN、service、port、TLS、vhost、DNS/CDN、Playground Origin、部署 profile，以及对既有 AdSense 能力的 production 接入；它不是一次普通 release 切换。课程页手动广告、Auto Ads、Angular SPA 生命周期和局部布局保护均为共享实现，第三门及后续 locale 不重新开发广告功能。
5. **日常维护部署**只对已完成上述基线、并标记为 `production_state=live` 的 locale 执行 `scripts/maintenance-production.sh <release-dir>`，不重新探测或设计服务器环境。

## 执行成本与协作

新增 locale 始终以质量 gate 为先。每个 locale 固定维护 **1 个长期 Generation role/session + 1 个独立长期 Reviewer role/session + 维护者 Local terminal**：Generation session 只产生或修订语言内容；Reviewer session 执行 Glossary Review、TranslationUnit Quality Check、revision re-QC 与 Locale Surface Review。同一个从未参与该 locale 语言 generation 的 Reviewer session 可以连续承担这些审核，不要求拆分更多 reviewer conversation/session；各 gate 范围和 evidence 独立，reviewer finding 必须回到 Generation session 产生 replacement。

新增 locale 的 initial Page / Example 大批量 TranslationUnit Generation 一律使用维护者 Local terminal 导出的 provider-neutral ZIP handoff。进入 Generation / Reviewer 往返后，具备仓库终端能力的当前 AI execution environment 默认自动完成闭环所需的短机械步骤，包括小批 revision lifecycle、审核结果 current-check / record / finalize、locale-level replacement 落盘、Surface Reviewer bundle / record-a，以及 Course SEO 结果落盘和正式 assemble / refresh / revise；然后把新材料交给独立 Reviewer 复审。闭环结束后的 promotion、build、preview / publish / Production、search closeout 与最终 Git commit / push 由维护者 Local terminal 成组执行。该分工只改变操作主体，不改变任何现有 gate、角色隔离或 provenance。

在开始该 locale 的正式 generation 前，根据当时真实的 ChatGPT 可用额度、Codex 5 小时额度与周额度、Remote Desktop Commander 月额度、历史实测成本和并发计划，选择 `chatgpt` 或 `codex` 作为 Generation provider；两者都使用 **GPT-5.6 Sol + High**。一旦开始正式 generation，原则上 glossary generation / revision、UI catalog、article metadata、其他 locale-level 文案、TranslationUnit initial / revision / 需要新译文的 retry、schema v2 Course SEO localization / refresh / revise replacement，以及 Locale Surface Review finding replacement generation 全程沿用同一 provider。不得仅为临时节省额度随意切换；只有真实额度耗尽、provider/tool failure 或仓库明确支持的恢复条件出现时才允许改变执行环境，并必须保留真实 provenance，不新增虚假 generation 记录，不弱化任何 gate。该选择是协作决定，不新增 schema、receipt、machine gate 或 locale state 字段。

正式 Reviewer 路径继续使用独立的普通 ChatGPT **GPT-5.6 Sol + High** session；Reviewer 从未参与该 locale language generation 时才可连续承担上述审核，且不得生成 replacement 后批准自己的输出。provider 为 `chatgpt` 时的执行规范见 [ChatGPT 正式语言生成执行规范](CHATGPT_LANGUAGE_GENERATION.md)，provider 为 `codex` 时见 [Codex 正式语言生成执行规范](CODEX_TRANSLATION.md)。仓库已有正式工具与已确认事实直接复用，不为新增 locale 建立硬 wall-clock 时间目标。canonical English source-description extraction / review 是跨 locale 共享 authority，不属于上述单 locale 固定配对；其 current/stale 规则仍只按 [课程页正式 SEO Metadata 规范](COURSE_SEO_METADATA.md) 执行，不因每个新增 locale 默认重做。

### 新增 locale 默认调度路径（效率优先）

下表是首次上线的唯一默认调度路径。各阶段 CLI 仍保留 retry、中途 revision、stale bundle 与已有 evidence 的合法恢复能力；这些能力用于恢复已经发生的状态，不表示默认应提前导包、逐组立即 revision 或重放已完成步骤。

| 阶段 | 默认动作与停止点 |
| --- | --- |
| 并行启动 | 在任何正式 generation 前，一次确认本轮全部 locale 的 locale identity、hostname、CDN、loopback port、service、data/release/current/lock path、public URL、registry 顺序与 URL，以及 Generation / Reviewer session；冻结每个 locale 的 Production identity 唯一值，缺失或歧义时不开始翻译。核对拟公开链接与各目标的实际 `production_state`、可达时点和上线顺序兼容；不得为通过计划检查提前把 `first-production` 改成 `live`。 |
| 共享输入落定 | 完成 init、glossary gate、locale assets，并在**任一并行 locale 首次导出 Surface Reviewer Bundle 前**完成本轮双方需要的 `internal/tour/languages.go` 与 `production/identity.json` 修改。进入任一 Stage A 后保持共享输入稳定；若正式输入变化，依 current-check / receipt schema fail closed，只重导、重审受影响的 current package，不把旧 ZIP 或聊天记忆继续当输入。 |
| 初始 TranslationUnit Generation | 全部 provider 都按 stable 顺序执行 `60 Page → 43 Page → 19 Example`，分别对应三个 initial batch。一次 model invocation 只处理一个 batch；该批正式落地并取得 automatic validation evidence 后停下，只有维护者明确“继续”才开始下一次 invocation。首次 122 Unit 仍逐批使用 provider-neutral Generation ZIP handoff。 |
| 结果落地 | 模型/UI 返回的只是**尚待 `result-pack` 的原始输出传输包**时，先取得 exact output directory，再恰好执行一次 `generation-bundle result-pack` 生成正式 **Result ZIP**，随后 `import`、`process` / `retry`。若下载物已经是 `kind=go-tour-i18n/generation-result-bundle` 的正式 Result ZIP，直接交给 `import` 的 current / identity preflight，不重复打包。bundle inventory/hash/current 检查与正式 automatic validation evidence 是不同层次，不得混称。恢复执行前先核对 `raw-responses/` 或 retry attempt、`result.json`、`validation/`、Snapshot/QC evidence；已导入或已验证的步骤不得覆盖或盲目重放。 |
| 首次 QC | 122 Unit 都有 passed automatic validation 后创建一个 full Snapshot。只导出当前下一组 Reviewer ZIP：`1-60` Page → current-check / 独立 Reviewer invocation / record；完成后才导出 `61-103` Page；完成后才导出 `104-122` Example。三组各用新的用户请求/model invocation。默认先完成三组首审并汇总全部 B/C/D，不在第一组后立即 revision。 |
| 集中 revision / re-QC | 按 Page / Example 分开、每个 revision batch 最多 30 Unit，集中处理汇总的 B/C/D。生成新的 full Snapshot；只有旧 A 且 rubric、source、selected batch、candidate、validation、attempt 与 glossary lineage 完全符合现有规则时才 carry-forward，其余只按 pending scope re-QC，直到全 A 后 finalization。 |
| 闭环收口与上线 | `promote`、build、preview / browser verifier、publish、Production / deploy / verifier、search closeout、最终 commit / push 继续由维护者 Local terminal 成组执行；`first-production` / `live`、Production HUMAN gate 与 visual HUMAN gate 的现有规则不变。 |

若历史或当前执行已经在首审中途合法 revision，不倒退状态、不删除或伪造 evidence，也不强行补完旧 Snapshot 的三组。应从当前 full Snapshot / scope 和已有结果恢复：重新导出下一份 current Reviewer ZIP，按 identity 规则 carry-forward 旧 A，并只审核当前 pending Unit。只有真实缺少 terminal 能力、确需跨 ChatGPT / Ubuntu 或跨 session 上传附件，或发生真实 failure / mutation-unknown 时才交回维护者；不能要求维护者把已下载 ZIP 手工 `cp` 到 `/tmp`，也不能声称当前 ChatGPT / Remote Desktop Commander 已有无需人工附件传递的跨环境通道。

## 1. 冻结语言与生产身份

在创建翻译资产前记录并确认：

- 规范 locale、`html_lang`、本地显示名、英文名和域名使用的 lowercase language code；
- production hostname 与 CDN；除 zh-CN 既有站外，社区语言遵循 [LANGUAGES.md](../LANGUAGES.md) 的 `<language-code>-go-dev.shuijingwanwq.com` 与 Cloudflare 约定；
- 独立 data root、release/current/lock 路径、systemd service、未占用的 loopback port、public URL；
- 是否使用非中文共享静态资源基线；
- Playground 代理需要新增的精确 HTTPS Origin；
- 首页语言 registry 中的显示顺序与目标 URL。

这些决定应先形成明确记录，再修改 locale 资源或生产环境。不得根据其他 locale 的目录名、端口或译法自动推导新 locale。

首页 language registry 是 build-time registry：新 locale 的 release 会包含构建时的完整 registry，但既有 production locale 会继续运行各自已部署 release 中的 registry，直到其下一次正常 publish/deploy。正式采用 **existing locale language list = eventual consistency**。因此，新 locale 首次 production gate 只要求验证新 locale 自己的语言选择器：current identity、当前正式 registry、指向已有 locale 的链接，以及 English 指向官方 Tour。已有 locale → 新 locale 的反向链接不属于首次上线 gate；不得仅为即时出现新语言而批量重跑旧 locale 的 Quality Check、finalization、Surface Review、publish、deploy、CDN purge 或 production final。未来只有明确要求全部 locale 即时同步语言列表时，才重新评估 runtime registry 解耦。

## 2. 建立 locale 术语权威来源

先运行正式初始化命令生成机械骨架；locale 目录或 UI catalog 已存在时命令 fail closed，绝不覆盖：

```sh
go run -mod=readonly ./cmd/tour-i18n locale init \
  --locale <locale> \
  --language-name <autonym> \
  --english-name <English-name> \
  --html-lang <html-lang>
```

命令生成 `locale.json`、显式 TODO glossary、保持英文 source 的 UI key/kind/占位符/markup identity 的 TODO catalog、article metadata、`course-metadata.todo.json` Page inventory，以及按 Page 后 Example 正式顺序初始化的 `status.tsv`。同时创建 `.locale-init-incomplete`；该标记存在时，完整 build、完整 preview 和 publish 均 fail closed。

随后阅读 [术语治理政策](TRANSLATION_TERMINOLOGY.md) 和 [术语制定指南](TERMINOLOGY_GUIDE.md)，由 Generation session 建立完整 `locales/<locale>/glossary.yaml`。不得机器翻译 zh-CN、ja-JP 或其他 locale 的 glossary。

当前 AI execution environment 默认先导出 provider-neutral generation ZIP，ChatGPT 或 Codex 完整读取同一 contract；修订前以 `locale-check` 确认 current。产品界面需要人工附件上传时，只把实际 ZIP handoff 留给维护者：

```sh
go run -mod=readonly ./cmd/tour-i18n generation-bundle locale-export \
  --locale <locale> --task glossary \
  --output /tmp/<locale>-glossary-generation.zip
go run -mod=readonly ./cmd/tour-i18n generation-bundle locale-check \
  --bundle /tmp/<locale>-glossary-generation.zip
```

Glossary 同时承担两项正式职责：

- 它是 TranslationUnit 模型执行时与 manifest、全部 inputs 不可拆分的正式输入；
- 它是该 locale 全站的正式术语权威来源，公共 UI、首页、`/tour/`、`/tour/list`、导航、语言选择器、编辑器、runtime message、article metadata 和 SEO 可见文案均必须遵守。

对 `Go Playground` 这类可能翻译、部分本地化或 keep 的名称，必须在该 locale 的 glossary 中形成显式决定。不同 locale 可以做不同决定，但同一 locale 不得在不同表层混用。此步骤只建立新 locale 的规则，不顺带修改 zh-CN 或 ja-JP 的现有译文。

完整 glossary 制定后，必须按 [Glossary Review 规范](GLOSSARY_REVIEW.md) 由未参与该 locale generation 的 Reviewer session 完整审核，并由当前 AI execution environment 默认记录、检查 current passed receipt：

```sh
go run -mod=readonly ./cmd/tour-i18n glossary-review reviewer-bundle \
  --locale <locale> --output /tmp/<locale>-glossary-reviewer.zip
go run -mod=readonly ./cmd/tour-i18n glossary-review reviewer-bundle-check \
  --locale <locale> --bundle /tmp/<locale>-glossary-reviewer.zip
go run -mod=readonly ./cmd/tour-i18n glossary-review record \
  --locale <locale> --review-id <review-id> \
  --reviewer <reviewer> --decision passed
go run -mod=readonly ./cmd/tour-i18n glossary-review check --locale <locale>
```

只有 PASS 后，Generation session 才正式生成 UI catalog 与 article metadata，并继续 TranslationUnit generation。Reviewer 若返回 failed finding，只由 Generation session 修订 glossary；修订后重新完整审核。Glossary Review 不替代最终 Locale Surface Review。

## 3. 建立非 TranslationUnit 语言资产

初始化生成的 TODO 只是不可发布的工作标记，不是译文；进入 export 前必须完成并通过 glossary review，进入 Surface Review 前必须完成全部 UI 与 metadata 语言内容。

生成后按以下边界补充语言内容，不复制其他 locale 的语言内容。选定的 Generation session 先制定 glossary；Glossary Review PASS 后才生成 UI catalog 与 article metadata。正式 Glossary Review 和 Locale Surface Review 均必须由不同于 Generation session 的独立 ChatGPT Reviewer session 完成：

- `locales/<locale>/locale.json`：locale 身份；
- `internal/tour/ui/<locale>.json`：完整公共 UI catalog，key 与 `plain` / `rich` kind 必须匹配英文 source `internal/tour/ui/en.json`，正式 locale 不使用英文 fallback；
- `locales/<locale>/article-metadata.json`：全部正式 article 的本地化 `title` 与 `subtitle`；
- `locales/<locale>/course-metadata.todo.json`：初始化阶段的 Page inventory，不是正式 SEO metadata；
- `locales/<locale>/course-metadata.json`：全部 TranslationUnit promotion 后，先确认 [课程页正式 SEO Metadata 规范](COURSE_SEO_METADATA.md) 定义的 canonical English source-description asset 与人工 review gate current；再以每页 canonical English description 作为唯一 semantic-scope authority，结合该页完整 English source、完整最终 ready canonical target、完整 locale glossary、目标 locale identity 和 v2 contract 生成目标语言 description；由 `course-metadata assemble --schema-version 2` 自动绑定完整 English source、canonical description、ready target 与 glossary identity；
- 首页、导航、语言选择器与语言 registry 所需的 locale 条目。

UI catalog、首页和 metadata 不属于 TranslationUnit candidate、status、Quality Check、machine finalization 或 promotion。它们必须在后续 Surface Review 中单独验收。

Glossary Review PASS 后，UI/article generation 使用 `generation-bundle locale-export --task locale-assets`，并在使用结果前执行 `generation-bundle locale-check`。bundle 完整绑定 English UI/article source、当前 target、reviewed glossary、locale identity 与 authority，但不直接覆盖 skeleton 文件；Generation / Reviewer 闭环中的当前 AI execution environment 默认完成结构校验与正式落盘。

`status.tsv` 不是语言资产，也不得从其他 locale 复制。`locale init` 已调用与 `status init` 相同的正式 catalog 初始化逻辑；不要再次运行会因文件已存在而 fail closed 的 `status init`。第一次进入 TranslationUnit retranslation export 前立即校验：

```sh
go run -mod=readonly ./cmd/tour-i18n status check --locale <locale>
```

底层正式初始化逻辑只负责首次创建缺失的 `locales/<locale>/status.tsv`：按 Catalog Page 顺序、再按 eligible Example inventory 顺序写入当前 workflow 的全部 TranslationUnit，初始状态均为 `pending`。它不写当前时间、不覆盖已有文件，也不承担已有状态的修复、同步或 source 更新迁移。只有 `status check` 与 current Glossary Review gate 都通过后，才能执行首次 retranslation export。正式 export path 会机械执行 Glossary Review check；missing、failed 或 stale 时不会创建 batch。

## 4. 执行 TranslationUnit workflow

TranslationUnit 工作从 [多语言翻译流程](TRANSLATION_WORKFLOW.md) 进入。正式翻译前还必须读取 [翻译任务规范](TRANSLATION_TASK_SPEC.md)、[Retranslation 执行手册](RETRANSLATION_RUNBOOK.md)、当前执行环境规范（[ChatGPT 正式语言生成执行规范](CHATGPT_LANGUAGE_GENERATION.md) 或 [Codex 正式语言生成执行规范](CODEX_TRANSLATION.md)）、当前 batch manifest、manifest 列出的全部 inputs，以及目标 locale glossary。

首次 Page batch 使用当前推荐的 60-Page 基线；Page 顺序、Examples 独立、revision/retry 边界和调整条件只以 [Codex 正式语言生成执行规范](CODEX_TRANSLATION.md#新增-locale-的首次-page-batch) 为准，本手册不重复细则。ChatGPT 路径 export 必须显式使用 `--generator chatgpt`；Codex 路径可使用兼容默认值或显式 `--generator codex`。initial Page 与 Example bulk batch 的 Generation input 必须由维护者 Local terminal 生成 ZIP 并 handoff 给 Generation session，不使用逐文件 transport。batch prefix 只是现有执行路径与归档命名，不新增 provider provenance 字段。

保持既有顺序：

```text
export
→ generation-bundle export
→ 选定的 ChatGPT 或 Codex Generation provider
→ result-pack / import
→ process
→ automatic validation
→ Candidate Snapshot
→ 独立 ChatGPT session Quality Check（全 A）
→ machine finalization（完整 QC A）
→ promotion
→ ready
```

不要把 UI catalog 或 metadata 塞入 TranslationUnit batch；不要用 Surface Review 结论生成 review evidence；不要在 promotion 前用完整站点观感替代逐 TranslationUnit 审核。

首次 QC 的独立 Reviewer input 依次用 `quality-check reviewer-bundle` 导出 Page stable index `1-60`、Page `61-103`、Example `104-122`，每份由新的用户请求/model invocation 审核；默认调度、集中 revision 与中途 revision 的兼容恢复边界见[新增 locale 默认调度路径](#新增-locale-默认调度路径效率优先)。只在上一组 current-check、审核与 record 完成后导出下一组，记录前运行 `quality-check reviewer-bundle-check`。Reviewer 只返回逐 Unit A/B/C/D 与 findings，现有 `record` / `record-batch` / `finalize` state machine 不变。

## 5. 完整投影、预览与 Surface Review

promotion 完成后，先执行 canonical English source-description 的 current 检查与人工 review gate 检查：

```sh
go run -mod=readonly ./cmd/tour-i18n course-metadata source check
go run -mod=readonly ./cmd/tour-i18n course-metadata source review-check
```

再生成 ChatGPT 与 Codex 共用的 provider-neutral deterministic ZIP；该准备步骤可由当前 AI execution environment 自动完成，产品界面需要附件上传时只把实际 ZIP handoff 留给维护者：

```sh
go run -mod=readonly ./cmd/tour-i18n course-metadata localization-bundle \
  --locale <locale> \
  --output /tmp/<locale>-course-seo-localization-generation.zip
go run -mod=readonly ./cmd/tour-i18n course-metadata localization-bundle-check \
  --locale <locale> \
  --bundle /tmp/<locale>-course-seo-localization-generation.zip
```

把该 ZIP 交给长期 Generation session。只要它来自未变化的正式 working tree 且 `manifest.json`/SHA-256 完整，ChatGPT 或 Codex 都直接读取附件内 103/103 Page full context、完整 glossary、locale identity 与当前 authority，不再重复扫描同一批输入；正式输入变化后必须重新导出 ZIP。ZIP 只是 transport container，不是 Course SEO authority 或 gate。

随后由选定的 **GPT-5.6 Sol + High** Generation provider 为每个 Page 输出完整 `page_id → localized description`。canonical description 仍是唯一 semantic-scope authority；source/target 只用于技术语义、正文术语、自然度与实际内容对齐，不授权重新摘要。同一 generation session/batch 可处理多页，但不得跨 Page 补充、混合或推断语义。Generation 结果由具备仓库终端能力的当前 AI execution environment 默认正式落盘，并立即使用离线命令组装 v2 asset；target body 由命令机械读取并计算 `target_sha256` freshness，与其作为生成上下文的职责互不替代。真实 provenance 必须与本 locale 实际 Generation provider 一致：ChatGPT 使用 `--provider chatgpt --model gpt-5.6-sol-high`，Codex 使用 `--provider codex --model gpt-5.6-sol-high`：

```sh
go run -mod=readonly ./cmd/tour-i18n course-metadata assemble \
  --schema-version 2 \
  --locale <locale> \
  --descriptions <localized-descriptions.json> \
  --provider <provider> \
  --model <model> \
  --generated-at <RFC3339-UTC> \
  --output locales/<locale>/course-metadata.json
```

只有全部 workflow TranslationUnit 为 canonical `ready`，并且 locale 配置、UI catalog、article metadata 与 strict-valid schema v2 `course-metadata.json` 完整后，才删除 `.locale-init-incomplete`，并先构建完整 projection。不得仅为绕过 gate 提前删除标记：

```sh
go run -mod=readonly ./cmd/tour-i18n build --locale <locale>
```

并行 locale 必须先完成[默认调度路径](#新增-locale-默认调度路径效率优先)中的共享输入落定检查。随后由当前 AI execution environment 默认从 working tree 导出完整、确定性且自包含的审核输入；自包含包括实际 first-party Playground/Tour runtime JavaScript、首页/Tour shell/footer Go template 与 Tour list/editor/navigation partial 的完整 source context，并明确排除 `static/lib` vendored third-party library，而不只是包含负责加载它们的 Go 文件。该步骤不调用模型、不生成 evidence 或 gate，不能替代 ChatGPT 对完整 source ↔ target 的语言审核。普通 ChatGPT Reviewer 优先使用可直接上传的 deterministic ZIP；产品界面需要人工附件上传时，只把实际 ZIP handoff 留给维护者：

```sh
go run -mod=readonly ./cmd/tour-i18n surface-review reviewer-bundle \
  --locale <locale> --output /tmp/<locale>-surface-review-reviewer.zip
```

ZIP 内的 `surface-review.json` 与 standalone `surface-review export` 使用同一正式 exporter；`manifest.json` 绑定 package 与 Stage A authority 的 SHA-256。需要单独 JSON 诊断时仍可运行：

```sh
go run -mod=readonly ./cmd/tour-i18n surface-review export \
  --locale <locale> --output /tmp/<locale>-surface-review.json
```

当 ZIP 来自未变化的正式 working tree 且 manifest 完整时，Reviewer 直接读取上传附件内 package + authority，不再使用 Remote Desktop Commander 重复扫描同一 repository 输入；输入变化后必须重新生成 ZIP。ChatGPT 在与 locale-level generation 分离的 session 完成 Locale Surface Review A；schema v2 package 会让审核者逐页同时看到完整 English source、canonical English description、完整最终 target、完整 glossary、localized description 和对应 identity。此 full-context review 始终是独立的最终语言 gate；允许 generation session 读取完整当前 Page 上下文不允许它批准自己的输出。若审核中修复 UI、metadata 或其他表层资产，必须重新导出当前 package 并复审受影响范围。若发现 TranslationUnit candidate 问题，仍须回 revision batch、validation、QC A、finalization、promotion，再刷新受影响 course metadata 和 package。目标 locale 为 `production_state=first-production` 时，先在同一 review-id 的 Markdown evidence 写入完整、未改写的 first-production finalization placeholder；`record-a` 会在写 receipt 前检查它。A 通过后记录当前正式输入的 machine-readable gate（Markdown evidence 仍照 [Locale Surface Review](LOCALE_SURFACE_REVIEW.md) 保留）：

```sh
go run -mod=readonly ./cmd/tour-i18n surface-review evidence-scaffold \
  --locale <locale> --review-id <review-id> --reviewer <reviewer> \
  --date <YYYY-MM-DD> \
  --bundle /tmp/<locale>-surface-review-reviewer.zip
go run -mod=readonly ./cmd/tour-i18n surface-review record-a \
  --locale <locale> --review-id <review-id> --reviewer <reviewer>
```

只有 current A gate 存在时才可启动完整 locale preview：

```sh
go run -mod=readonly ./cmd/tour-i18n preview \
  --locale <locale> \
  --http 127.0.0.1:0
```

命令打印实际 loopback URL 后，运行正式自动验收入口：

```sh
scripts/verify-preview-browser.py http://127.0.0.1:<port>/ <locale>
```

Remote Desktop Commander 是 ChatGPT 自动完成闭环内短机械操作和 failure diagnosis 的正式辅助工具，但不接管闭环后的维护者批量操作。对于上述 browser verifier，以及后续 `publish`、shared-assets Production、`first-production.sh`、IndexNow closeout 等可能持续数十秒到数分钟的确定性长任务，普通 ChatGPT + Remote Desktop Commander 只提供一条完整可粘贴命令，由维护者在本地终端执行并回传终态；不要仅为等待完成而反复远程轮询，也不要自行执行最终 Git commit / push。该协作规则不改变脚本内部任何 gate、receipt、retry 或 HUMAN gate。

执行 [Locale Surface Review](LOCALE_SURFACE_REVIEW.md) 时，先以 exporter 提供的英文/source、目标资产和 glossary 为正式输入，完整审核 TranslationUnit 之外的 UI catalog、article metadata、首页及其他 locale-level 文案；不得用浏览器抽查替代。严格顺序为 promotion → course metadata → build → surface-review reviewer-bundle（standalone export 仅用于诊断）→ ChatGPT Locale Surface Review A → 必要 revision/fix + 重新生成 reviewer bundle → 写 Markdown evidence → record current A gate → full locale preview → automated rendered acceptance → visual HUMAN gate → publish。机器已经覆盖的 canonical、sitemap、language selector URL、Run / Format / Reset、SPA、`/socket` 和 desktop/mobile overflow 不由人工重复。

正式审核记录写入 `data/locale-surface-reviews/<locale>/<review-id>.md`。发现 TranslationUnit 内容问题时，回到新的 revision batch 和完整 A-only 审核链；发现表层资产问题时，修正对应 locale 资产并重新执行受影响的 Surface Review。语言质量审核或 preview acceptance 未通过，不得 publish production bundle。

## 6. Publish 与首次生产部署

Surface Review 通过并完成其中所有修复后，使用 `assets-go-dev.shuijingwanwq.com` 的非中文 locale 必须在首次 production release 激活前通过 shared-assets current-state freshness gate：以当前仓库最终状态执行正式 `assets export` 与 `assets validate`，然后运行 [生产运维手册](PRODUCTION_RUNBOOK.md#shared-assets-production-发布状态机) 的 `scripts/shared-assets-production.sh <export-dir>`。状态机正式比较 current export 与 production origin；`NO_CHANGES` 完全不调用 purge 但仍执行 public verifier，`DEPLOYED` 则只根据 receipt 的安全 changed paths 自动发出一次 Cloudflare exact-URL purge，明确 PASS 后才执行 verifier。验证继续覆盖 changed URL 缓存验收（如有）、完整 14 个 allowlist 公网 SHA-256 与当前 export 的对照及 boundary 404；不再要求维护者进入 Cloudflare Dashboard。purge 明确失败可保留原 changed paths 重试，purge 已 PASS 而 verification 失败时只重试 verification，mutation 不确定则 fail closed 且禁止盲目重复 POST。仅当最终为 14/14 一致且全部验证通过时，本 gate 才通过。不得以历史“11/11 已部署”记录、Git 历史、上次部署时间或人工判断文件是否变化替代此流程；receipt 只是本次 execution evidence，不能替代未来 locale 的 freshness gate。本 gate 只在首次 production 激活前检查当前最终状态，不要求每次 Surface Review 都部署 production。

随后按 [生产运维手册](PRODUCTION_RUNBOOK.md) 生成 Linux/amd64 production bundle，并核对 `release.json`、bundle 内 `site-metadata.json`、文件集合和 `SHA256SUMS`。`publish` 只生成 release，不创建 hostname、service、TLS、vhost 或部署脚本 profile。

随后执行该手册的“新 locale 首次生产部署”：先完成并记录 production profile、基础设施和 AdSense production 接入，再部署已验收 bundle。首次 deployment 的 `current` 可以尚不存在；脚本首次原子创建它后，如新 release 健康检查失败，没有旧 release 可回滚，必须保留现场并人工检查。已有 locale 的日常 deployment 继续要求 `current` 为指向 release root 内既有 release 的合法 symlink，并沿用既有 rollback 流程。首次激活的页面必须已是最终的启用广告形态，不安排“先无广告上线、再接广告”的两次发布。当前 `scripts/deploy-production.sh` 与 `scripts/verify-production.sh` 都对 locale fail closed；新 locale 未经明确 deployment/verification profile 实现和测试前，不能假定通用命令已经支持它。

## 7. 正式上线验收与移交

首次上线至少完成三层验收：

- **源站层**：deployment 已完成 service 与 loopback 连续健康；FIRST_DEPLOYMENT 先用 production hostname + `--resolve` 做外部 direct-origin acceptance，EXISTING_DEPLOYMENT 则在 hostname purge 后运行 `scripts/verify-production.sh <release-dir>`；
- **公网层**：同一 machine acceptance 命令确认 HTTPS 关键路由、首页、`/tour/`、`/tour/list`、课程页、静态资源、`robots.txt`、sitemap 全量 URL、canonical/locale identity、`/socket` 404，并记录 CDN cache status；cache observation 要求 HTTP 200、对应 header 存在且状态属于正式 allowlist，但不以固定 `MISS → HIT` 时序或固定次数内出现 `HIT` 作为上线 gate；
- **真实浏览器层**：桌面与移动端页面、导航、语言选择器、Run / Format / Reset、runtime message，以及 Network 中真实 Playground endpoint 和允许的 Origin；并按生产运维手册对最终课程页做轻量广告确认。

FIRST_DEPLOYMENT 的正式入口为 `scripts/first-production.sh <release-dir>`。执行前，该 locale 的正式 production identity 必须显式设置 `production_state=first-production`；已经上线并标记为 `live` 的 locale 即使 `current` 或 receipt 缺失也会 fail closed。它从正式 production identity 执行全量 preflight、基础设施、Playground Origin、既有 `deploy-production.sh`、zgocloud direct-origin、Cloudflare proxied DNS、zgocloud public readiness、既有 `verify-production.sh` 与 Chrome automated browser acceptance；尚无正式公网 DNS/cache 时不要求 hostname purge。全部自动 gate 通过后脚本只写 passed receipt，输出 `READY FOR FINALIZATION`，并打印唯一下一条、已绑定 release 与 current review-id 的 `go run -mod=readonly ./cmd/tour-i18n first-production finalize ...` 命令；它不会自行 finalize。维护者显式执行该命令后，finalizer 才在校验 receipt、当前 A gate 和唯一 evidence placeholder 后记录 machine-finalizable production conclusion，并将 lifecycle 转为 `live`。这两个阶段之间不新增 HUMAN gate，Production visual review 仍不是 blocking gate。EXISTING_DEPLOYMENT 使用 `scripts/maintenance-production.sh <release-dir>` 编排 deploy → automatic exact-hostname CDN purge → machine → browser → PASS；多 locale 可使用 `scripts/maintenance-production-batch.sh <release-dir>...` 严格串行执行。

一批上线完成后的人工视觉工作仅为非阻塞 spot check：抽样中文、非中文带广告、非中文不带广告各一个站点，不写 receipt、不阻止 finalize/live，也不要求逐 locale 执行。发现问题走正常修复 → publish → maintenance deploy。

## 8. Search-engine submission closeout

在首次 production machine/browser acceptance 和正式 first-production finalize 全部通过，且目标 profile 已是 `production_state=live` 后，维护者完成以下外部运营 closeout：

1. 在 Google Search Console 为该 production hostname/property 完成必要接入，并提交该站的正式 `/sitemap.xml`。
2. 在 Bing Webmaster Tools 为该 production hostname/property 完成必要接入，并提交同一正式 `/sitemap.xml`。
3. 完成适用于目标 locale/市场的 locale-specific search engine 提交后，执行一次 IndexNow 全站 bootstrap；它是 closeout 的最后一步，不替代 Google、Bing 或 locale-specific search engine 的 sitemap 提交。

不要按 locale 猜测 hostname 或手写 sitemap origin。先从唯一 machine authority 查询已 live profile：

```sh
python3 scripts/production-identity.py list --state live
```

从目标 locale 对应行的 `public URL` 取得正式 origin，并将其末尾 `/` 替换为 `/sitemap.xml` 后提交；不得拼出双斜杠或按 locale 猜测 host。若 platform 已通过 DNS/domain ownership 或其他共享方式验证 property，应复用实际可用的 verification 状态，不要求每个 locale 重复固定 verification 方法。不得把 Google、Bing 或 locale-specific 平台的 token、verification secret 写入仓库、命令行或新脚本。

Google Search Console 和 Bing Webmaster Tools 是所有新增 production locale 的标准 closeout。Naver 不属于全 locale 强制项；只有目标 locale/市场确实适合时，才补充 locale/market-specific search engine。当前明确例子是 `ko-KR` 的 Naver Search Advisor 与 sitemap submission。

IndexNow 是 `first-production finalize` 之后的 search-engine closeout，不是另一种 Production deployment。它使用正式 `production/identity.json`，且只接受 `production_state=live` 的目标 locale。Google、Bing 与适用 locale-specific search engine 的 UI sitemap submission 完成后，正式入口为：

```sh
scripts/indexnow-closeout.sh --locale <locale>
```

首次运行会在仓库外的 `${XDG_DATA_HOME:-$HOME/.local/share}/go-tour-indexnow/<locale>/` 生成 locale-specific key；store 和 locale directory 为 `0700`，`<key>.txt` 为 `0600`。生成的 key 是 64 个 lowercase hex 字符，符合 8–128 个 `[A-Za-z0-9-]` 字符、仅可选一个末尾 LF/CRLF 的正式 contract。重跑自动复用该目录中唯一的有效 key；多个 candidate、symlink、非 regular file、非法 key 或其他造成 identity 不明确的条目都会 fail closed，绝不生成替代 key。若维护者已有受保护 key，可显式使用 `--key-file /secure/path/<key>.txt`；此时不会访问或生成默认 store key。closeout 仅从唯一 production identity 取得 Aliyun origin、目标 data root、Nginx vhost 与正式 Nginx test/reload command，部署 root verification key、配置精确 Nginx location 并执行 test/reload。配置 test 或 reload 明确失败时会恢复本轮 vhost/key 变更；已存在完全一致 location 时幂等，不一致则 fail closed。本流程不声称 vhost 写入具备 crash-safe atomic transaction 语义。随后 Go primitive 使用调用机的正常直连网络验证公网 HTTPS key 并提交。没有实际 failure evidence 时，它不建立 zgocloud/SOCKS tunnel；若将来需要稳定境外公网 runner，应复用既有 first-production / verify-production 的 direct-runner 基线，而不是为 IndexNow 新建代理栈。第三方 submission 失败不回滚已成功 provisioning 的 key/vhost。probe 的 HTTP 202 只在同一次 Go 调用中复用相同 key 与完全相同的一 URL payload，最多 3 次、backoff 1 秒与 2 秒；不重新 provisioning，不重试 transport 或其他 HTTP semantic failure，耗尽仍 non-zero。bulk HTTP 202 也不自动重试。明确失败后人工重跑仍复用同一 key。成功后不周期性重复全站 bootstrap。不要把 key 写入 Git、production identity 或 evidence。

底层 submission primitive 仍为下列 Go 命令；closeout 自动调用它，维护者不应为新增 locale 手工部署 key 或修改 vhost。该命令会验证公网 root key、正式 HTTPS `/sitemap.xml`、hostname、无重复 URL，以及动态 sitemap URL 数量必须在 `1..10000`（IndexNow 单请求上限）内；固定 probe URL 为正式 production origin（homepage），它先单独提交以验证 key。probe 返回 HTTP `202` 时按上述 bounded policy 重试，期间绝不 bulk；probe 返回 `200` 后，命令从 sitemap URL 集合排除 probe，再 bulk 提交剩余 `N-1` 个 URL。只有 bulk 最终 HTTP `200` 才输出动态 URL 数量的 PASS，其中 `submitted_urls=N` 包含已成功提交的 homepage probe；若 sitemap 仅含 homepage，则 probe 的 HTTP `200` 即为最终成功且 `submitted_urls=1`：

```sh
go run -mod=readonly ./cmd/tour-i18n indexnow bootstrap \
  --locale <locale> \
  --key-file /secure/path/<key>.txt
```

该命令固定提交到 `https://api.indexnow.org/indexnow`。Bing Webmaster Tools 的 IndexNow Dashboard 不属于 gate；不等待 indexing/coverage 状态，不新增 receipt、schema 或必须 current 的第三方 evidence。

此 closeout 不属于 production availability/security gate、Locale Surface Review A、rendered surface acceptance 或广告 gate；它不阻止 first-production finalize，也不阻止 `production_state` 从 `first-production` 转为 `live`。提交后 Google/Bing/Naver/IndexNow 的异步 indexing、coverage 或收录状态不要求在首次上线当天成功，不能等待其完成才认定 production live。

如需保留执行痕迹，可在现有 locale Surface Review evidence 或项目状态记录中轻量记录 `submitted`、日期和平台；不新增 receipt、schema 或 machine gate，也不把第三方异步状态维护为必须 current 的证据。
