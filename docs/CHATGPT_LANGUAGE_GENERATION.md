# ChatGPT 正式语言生成执行规范

本文定义某个 locale 在开始正式 generation 前选定 `provider=chatgpt` 后，普通 ChatGPT **GPT-5.6 Sol + High** 配合 Remote Desktop Commander 执行正式语言生成时的仓库边界。它与 [Codex 正式语言生成执行规范](CODEX_TRANSLATION.md) 并列；不新增 Translation Engine，也不改变任何 validation、Quality Check、Locale Surface Review 或 Production gate。

## 单 locale 固定会话角色

一个 locale 默认维护一个长期 **Generation session**、一个与之独立的长期 **Reviewer session**，再由维护者 **Local terminal** 执行闭环外的成组 deterministic lifecycle；闭环内短机械步骤按下文由具备仓库终端能力的 AI execution environment 自动完成。这是协作职责规则，不是机器身份系统；不得为此新增 session 字段、receipt、schema、CLI flag 或 machine gate。

### Generation session

Generation session 负责所有产生或修订该 locale 语言内容的模型工作：

- locale glossary 的语言判断与制定；
- 公共 UI catalog、article metadata 与其他 locale-level 文案的生成或修改；
- TranslationUnit initial translation、revision，以及确实需要新译文的 `restore_failed` / `validation_failed` retry；
- schema v2 Course SEO localization，以及 Surface Review finding 所需的 refresh / revise description；
- Glossary Review、TranslationUnit Quality Check 或 Locale Surface Review finding 回流后的 replacement generation。

同一个 Generation session 可以按 `glossary → Glossary Review PASS → UI / metadata → TranslationUnit initial / revision → Course SEO localization → 后续 replacement generation` 长期连续工作，只要它始终承担 generation。它可以读取完整 source、target、glossary 与 reviewer finding；Course SEO generation 可以读取当前完整 Page source 与 ready target。读取这些材料不会破坏 generation 独立性，真正禁止的是同一 conversation/session 同时承担 generation 与 formal review。

Generation session 不得执行正式 TranslationUnit Quality Check、revision 后正式 re-QC、Locale Surface Review，或批准自己生成的输出。

Generation session 制定完整 glossary 后必须停在 Glossary Review 边界；只有 generation-independent Reviewer PASS 且当前 AI execution environment 记录并检查 current gate 后，才正式生成 UI / article metadata 并进入 TranslationUnit generation。若 Reviewer 返回 failed finding，Generation session 负责修订 glossary，并在继续下游 generation 前交回同一独立 Reviewer session 完整复审。

### Reviewer session

Reviewer session 负责该 locale 的正式语言审核：

- Glossary Review；
- TranslationUnit Quality Check；
- revision 后 re-QC；
- Locale Surface Review。

同一个 Reviewer session 可以先承担 Glossary Review，再承担 TranslationUnit Quality Check，之后继续承担同一 locale 的 Locale Surface Review。这三个 gate 的范围和 evidence 保持独立，但仓库不要求它们使用不同 reviewer conversation/session。前提是该 Reviewer session 从未参与该 locale 的 glossary、UI catalog、article metadata 或其他 locale-level language generation，从未参与 TranslationUnit initial / revision / retry generation，也从未参与 schema v2 Course SEO localization / revision generation。

Reviewer 发现问题时必须遵循：

```text
reviewer finding
→ Generation session 产生 replacement
→ AI execution environment 执行闭环内 deterministic lifecycle
→ Reviewer session re-review
```

Reviewer session 不得直接修改 candidate、生成 replacement translation 或 Course SEO replacement description 后再批准自己的结果，也不得因为自己审核过上一轮而自动批准 revision。只要它没有参与 replacement generation，就可以继续审核 Generation session 修订后的新输出。Validation passed、旧 A、Surface Review 结论或模型历史表现都不能替代当前正式审核。

TranslationUnit Quality Check 与 revision re-QC 的单次 Reviewer model invocation / response 最多审核 60 TranslationUnits；Page / Example 分开。60 是实际审核质量边界，不得通过在同一次 response 内串联多个 `<=60` working set 绕过；下一组需要新的用户请求和新的 model invocation。每组必须逐 TranslationUnit 审核，不得抽样。首次 122-Unit locale 的 full QC 推荐使用 Page stable index `1-60`、Page stable index `61-103`、Example stable index `104-122` 三组；revision re-QC 按实际 pending scope 分组，每组最多仍为 60。

### 闭环机械操作与 Local terminal

新增 locale 的首次大批量 TranslationUnit Generation 固定使用 ZIP handoff：维护者 Local terminal 完成 initial Page / Example batch export、生成 current provider-neutral Generation Bundle，并把 ZIP 上传到 Generation session。ChatGPT 不通过 Remote Desktop Commander 逐文件读取这批输入，也不以重新扫描仓库替代附件。生成结果继续使用正式 Result Bundle / deterministic import contract。

首次 bulk handoff 后，Generation / Reviewer 闭环内部的短 deterministic 操作默认由具备仓库访问能力的 ChatGPT + Remote Desktop Commander 自动连续完成，包括首次 bulk 原始输出附件的定位 / hash 核对 / result-pack / import / process / validation，小批 revision / retry 的 export、bundle、result-pack / import、process / validation、Snapshot / scope / reviewer-bundle、审核结果 current-check / record / finalize、locale-level replacement 落盘、Surface Reviewer bundle / record-a，以及 Course SEO description 结果落盘和正式 assemble / refresh / revise。机械执行不得让 Generation session 承担 Reviewer 的语言判断，也不得让 Reviewer 生成 replacement；每个 current-check、exact-set、A-only、schema 与 provenance gate 都照常执行。终端能力不可用、需要新的附件 handoff 或出现真实 failure / mutation-unknown 时才停下交回维护者。

维护者 Local terminal 负责闭环外或闭环收口后的成组步骤：locale init、首次 bulk export / ZIP handoff、promotion、build、preview、browser verifier、publish、shared assets、Production、deploy、verifier、checksum / curl、search closeout，以及按正式顺序需要的最终 Git commit / push。ChatGPT 给维护者提供可直接粘贴的终端命令时，不得在当前交互 shell 顶层启用 `set -e` / `set -u` / `set -o pipefail` 或组合形式；需要 fail-fast 时必须用独立 subshell 或独立脚本进程，避免失败退出或改变维护者当前 shell。

对 `preview` + browser verifier、`publish`、`shared-assets-production.sh`、`first-production.sh`、`indexnow-closeout.sh` 等可能持续数十秒到数分钟的成组确定性命令，普通 ChatGPT + Remote Desktop Commander 只给出一条完整可粘贴命令，由维护者在本地终端执行并把终态输出回传；不要为了等待完成而反复调用远程 process-output 轮询。脚本内部的 machine gate、receipt、bounded retry 和 HUMAN gate 均保持不变。

Locale Surface Review 的 Reviewer 输入默认由当前 AI execution environment 使用 `surface-review reviewer-bundle` 生成并完成 current-check；产品界面需要人工附件上传时，只把实际 ZIP handoff 留给维护者。只要该 bundle 来自未变化的正式输入且 `manifest.json` hash 校验成立，Reviewer 对附件内 `surface-review.json` 与 `authority/` 的完整读取即满足本轮正式输入读取，不应再通过 Remote Desktop Commander 重复扫描这些仓库文件。任何正式输入或 bundle 内 authority 发生变化，都必须重新导出 bundle；附件模式不允许抽样、跳过 Course SEO full-context coverage，亦不改变独立 Reviewer 与 finding → Generation 回流边界。

canonical English source-description extraction / review 是所有 locale 共享的 authority，不属于单个 locale 固定的 Generation / Reviewer 配对。只有其 authority 确实 stale 时才按 [课程页正式 SEO Metadata 规范](COURSE_SEO_METADATA.md) 执行，不因新增每个 locale 重复，也不默认交给该 locale 的 Reviewer session。

本规范只适用于已经选定 ChatGPT 的 locale Generation role。provider-neutral 的选择、持续使用和例外切换规则由 [多语言翻译流程](TRANSLATION_WORKFLOW.md) 与 [新增 Locale 执行手册](NEW_LOCALE_RUNBOOK.md) 规定；Remote Desktop Commander 的闭环自动化边界不扩展到维护者的 promotion / build / Production / Git 成组操作，也不接管 Codex 的 repository-level code/docs/config、tooling 与复杂诊断职责。无论 Generation provider 为 ChatGPT 还是 Codex，generation 与 formal review 的会话隔离规则不变，Reviewer 继续使用独立 ChatGPT GPT-5.6 Sol + High session。

## TranslationUnit 正式输入与 export

ChatGPT TranslationUnit 执行仍以 [翻译任务规范](TRANSLATION_TASK_SPEC.md) 为 provider-independent contract。每个 batch 的以下三部分是不可拆分的正式输入：

1. `manifest.json`；
2. manifest 列出的全部 `inputs/*`；
3. `locales/<locale>/glossary.yaml`。

开始写任何 raw response 前，生成 session 必须完整读取三部分；不得用聊天上下文、上一批输入或 glossary 摘要替代仓库当前字节。

ChatGPT batch 必须显式导出为：

```sh
go run -mod=readonly ./cmd/tour-i18n retranslation export \
  --locale <locale> \
  --generator chatgpt \
  [--unit-kind page|example] [--limit <count>] [--id <unit-id> ...]
```

未提供 `--batch-id` 时，`--generator chatgpt` 选择 `chatgpt-<locale>-NNN`；Codex 是兼容默认并使用 `codex-<locale>-NNN`。两种 prefix 共享同一 numeric namespace，latest export、Candidate Snapshot 与 promotion 都按 numeric suffix 判断，不按 prefix 字典序判断。显式 `--batch-id` 保持既有兼容语义；generator 不写入 manifest，也不构成可由机器证明的模型 provenance。

新增 locale 的首次 Page batch 仍使用 60-Page 基线，Examples 独立；revision 与 retry 范围不变。

initial Page / Example bulk export 后由 Local terminal 生成 provider-neutral input ZIP 并 handoff；ChatGPT 完整读取一个附件即可取得 manifest、全部 inputs、完整 glossary 与当前 authority，无需再通过 Remote Desktop Commander 逐文件读取：

```sh
go run -mod=readonly ./cmd/tour-i18n generation-bundle export \
  --locale <locale> --batch-id <batch-id> \
  --output /tmp/<locale>-<batch-id>-generation.zip
```

首次 122 Unit 的默认调度固定为 60 Page、43 Page、19 Example 三个 initial batch；每个 batch 使用一次 model invocation，正式落地并取得 automatic validation evidence 后停止，等待维护者明确“继续”。不得在一次 response 串联多个 initial batch。完整顺序与合法恢复边界见[新增 Locale 执行手册](NEW_LOCALE_RUNBOOK.md#新增-locale-默认调度路径效率优先)。

ChatGPT 仍在 hidden staging 中生成 exact expected files，并自行完成 protected token、glossary、明显未翻译文本、无解释/fence 与 single-LF 检查。必须区分两种 ZIP：

- ChatGPT/UI 为跨会话或跨 Ubuntu 边界返回的 outputs ZIP 是**原始输出传输包**，尚未经过 `generation-bundle result-pack`，不是正式 Result ZIP；
- `generation-bundle result-pack` 以 original Generation ZIP、exact output directory 与真实 provider/model 生成 `kind=go-tour-i18n/generation-result-bundle` 的正式 **Result ZIP**。正式 Result ZIP 直接交给 `generation-bundle import` 的 current / identity preflight，不得再次 `result-pack`。

首次 bulk handoff 的返回结果按上述边界恰好执行一次 `result-pack` 和一次 `import`；后续小批 revision / retry 默认由当前 AI execution environment 直接完成这两步及紧随其后的 `process` / `retry`。`result-pack` / `import` 会重验 current bundle、exact set、路径/encoding/EOF、受保护内容、candidate preflight 与 no-overwrite，但这些 transport / preflight 检查不等于 `retranslation process` / `retry` 写出的正式 automatic validation evidence，更不等于 Quality Check。Result ZIP 保存真实 provider/model，但不成为语言质量 evidence。

产品界面下载附件后，ChatGPT + Remote Desktop Commander 应先在该环境的默认下载目录中定位本轮唯一候选，计算 SHA-256 并与交接值核对，再自动解包到临时目录并完成短机械步骤；不得要求维护者手工 `cp` 到 `/tmp`。候选不唯一、hash 不符、附件不可读、terminal 不可用或 mutation-unknown 时才停下交回维护者。当前仍没有无需用户操作即可跨 ChatGPT / Ubuntu 或跨 session 传递 ZIP 的正式通道，维护者只承担这些确实必要的附件上传/下载。

恢复执行前先核对 `raw-responses/` 或 retry attempt、`result.json`、`validation/`、Snapshot 与 QC evidence。若正式输出已经 import，直接从缺失的 process / retry 或后续 evidence 继续；若 automatic validation evidence 已 current，则不得覆盖输出、重复 import 或重放 validation。CLI 的 stale、exact-set 与 no-overwrite failure 是保护信号，不是要求清空既有状态后重来。

## Remote Desktop Commander 安全写入

首次翻译或 revision 不得逐个文件直接建立正式 `raw-responses/`。在同一 batch filesystem 内执行以下顺序：

1. 确认正式 `raw-responses/` 尚不存在，在 batch 内创建隐藏 staging directory，例如 `.raw-responses.staging/`。
2. 将整个 batch 的每个 Unit 完整输出写入 staging；Page 使用对应 `.article`，Example 使用对应 `.txt`。
3. 对照 manifest 核对 expected filename exact set 与 count，并逐文件检查 protected token、glossary、明显未翻译自然语言、无 Markdown/code fence/解释，以及恰好一个结尾 LF。
4. 只有全部检查通过后，才在同一 batch 内将整个 staging directory rename/move 为 `raw-responses/`。

任何检查失败或 Remote Desktop Commander 中断时，隐藏 staging 不是正式 raw response，必须保留为未完成工作或安全清理；不得留下部分 `raw-responses/`。正式目录已存在时不得覆盖或合并，必须先查明当前 batch 状态。

Retry 使用同一原子提交思想：先将完整内容写入目标 Unit retry 目录内的隐藏 staging file，核对它满足当前连续 attempt 编号、文件名、protected token、glossary 和 single-LF contract，再 rename 为正式 `attempt-NNN.article` 或 `attempt-NNN.txt`。不得覆盖既有 attempt、跳号或伪造 provenance。

手工 Remote Desktop Commander staging 路径只作为 bundle transport 不可用时的兼容恢复路径；不得与同一 attempt 的 `generation-bundle import` 混用或覆盖既有文件。无论哪种 transport，`retranslation process` / `retranslation retry` 继续是 candidate、validation evidence 与 attempt lifecycle 的正式 fail-closed authority。

## 非 TranslationUnit 语言资产

这些资产按以下输入边界生成或修改：

- 首次制定 locale glossary：先遵循 [术语治理政策](TRANSLATION_TERMINOLOGY.md) 与 [术语制定指南](TERMINOLOGY_GUIDE.md)，并读取制定术语所需的正式 English/source context；尚未建立完成的完整 locale glossary 不是其自身的前置输入。
- 修改已有 glossary：必须读取完整当前 glossary 与相关正式 source/context。
- 生成或修改 UI catalog、article metadata 等其他 locale-level 语言资产：必须读取对应完整 source/context 和已经建立的完整 locale glossary。

首次 glossary 制定或任何使当前 Glossary Review coverage stale 的修改，都必须按 [Glossary Review 规范](GLOSSARY_REVIEW.md) 由 Reviewer session 完整审核。UI / article metadata 的正式 generation 排在 PASS 之后；Glossary Review 不因这些资产已经存在而自动批准。

所有资产仍须保持现有 key、kind、placeholder、markup、schema 和技术 identity。不得给 `glossary.yaml`、`internal/tour/ui/<locale>.json` 或 `article-metadata.json` 增加 provider/model/generation 字段。Glossary 继续是该 locale 的正式术语 authority，这些资产也继续由现有 validator 与 Locale Surface Review 审核实际内容。

这些 generation 输入优先一次性导出：

```sh
go run -mod=readonly ./cmd/tour-i18n generation-bundle locale-export \
  --locale <locale> --task glossary \
  --output /tmp/<locale>-glossary-generation.zip

go run -mod=readonly ./cmd/tour-i18n generation-bundle locale-export \
  --locale <locale> --task locale-assets \
  --output /tmp/<locale>-locale-assets-generation.zip

go run -mod=readonly ./cmd/tour-i18n generation-bundle locale-export \
  --locale <locale> --task surface-replacement --review-id <review-id> \
  --output /tmp/<locale>-surface-replacement-<review-id>.zip
```

`glossary` 包含完整 locale identity、English/source corpus 与 terminology authority；`locale-assets` 在 current Glossary Review 后包含完整 UI/article source、当前 target 与 glossary；`surface-replacement` 还绑定指定 reviewer evidence 和 current full Surface Review package。开始或记录结果前用 `generation-bundle locale-check --bundle <zip>` 确认 current。此 contract 只运输生成上下文和预期 deliverable，不直接覆盖正式 locale 资产；闭环内由当前 AI execution environment 按现有 validator/lifecycle 正式落盘，Reviewer finding 仍须返回 Generation session 后再独立复审。

首次 schema v2 Course SEO localization bundle 默认由当前 AI execution environment 从正式 working tree 生成并检查；产品界面需要人工附件上传时，只把实际 ZIP handoff 留给维护者：

```sh
go run -mod=readonly ./cmd/tour-i18n course-metadata localization-bundle \
  --locale <locale> \
  --output /tmp/<locale>-course-seo-localization-generation.zip
go run -mod=readonly ./cmd/tour-i18n course-metadata localization-bundle-check \
  --locale <locale> \
  --bundle /tmp/<locale>-course-seo-localization-generation.zip
```

ZIP 的 `course-seo-localization.json` 对 Catalog 全部当前 Page 逐页序列化 canonical English description、完整 English Page source、完整最终 ready target、source/source-description/target identity，并绑定 current canonical source-review authority、完整 glossary 与 `locale.json` identity；`formal/source-descriptions.json` 同时保留 canonical asset 的原始正式字节，`formal/` 还包含 glossary 与 locale identity；`authority/` 封装 ChatGPT/Codex generation、Course SEO、workflow 与 `AGENTS.md`。`manifest.json` 为全部文件提供 SHA-256。只要 bundle 来自未变化的正式 working tree 且 manifest/hash 完整，Generation session 直接完整读取附件，不再通过 Remote Desktop Commander 分批读取同一 103-Page context。ZIP 只优化传输，不新增 Course SEO schema、receipt 或 stale identity，也不允许跳过任一 Page；Codex 使用完全相同的 bundle contract。

schema v2 refresh/revise 使用独立 maintenance bundle，仍由既有 CLI 写正式 asset：

```sh
go run -mod=readonly ./cmd/tour-i18n course-metadata generation-bundle \
  --locale <locale> --task refresh \
  --output /tmp/<locale>-course-seo-refresh.zip

go run -mod=readonly ./cmd/tour-i18n course-metadata generation-bundle \
  --locale <locale> --task revise \
  --page-id <page-id> [--page-id <page-id> ...] \
  --finding data/locale-surface-reviews/<locale>/<review-id>.md \
  --output /tmp/<locale>-course-seo-revise-<review-id>.zip
```

refresh 自动选择 exact stale subset；revise 要求 current schema-v2 base、显式 Page subset 与 repository 内 reviewer finding。下载的 strict `page_id`/`description` JSON 仍交给 `course-metadata refresh` / `revise`，真实 provenance 由调用者明确记录。

schema v2 Course SEO localization 允许并推荐为每个当前 Page 提供 canonical English description、完整 English source、完整最终 ready canonical locale target、完整 locale glossary、locale identity 与 `course-seo-localization-v2` constraints。canonical description 是唯一 semantic-scope authority；source/target 只用于技术语义核对、正文术语一致性、自然表达和实际内容对齐，不授权重新摘要、增删语义或重选重点。同一 session/batch 可处理多个 Page，但每页必须只用自己的 canonical description 决定 semantic scope，不得跨页补充、混合或推断语义。普通 ChatGPT 的真实 provenance 固定记录为：

```text
provider=chatgpt
model=gpt-5.6-sol-high
```

生成 session 只产出 `page_id → localized description` 输入，不得直接手写正式 `course-metadata.json`。具备仓库终端能力的当前 AI execution environment 默认把 strict descriptions JSON 正式落盘，并立即运行现有 `course-metadata assemble` / `refresh` / `revise` CLI 机械生成资产、记录真实 provenance 并通过现有 gate。canonical English source-description 使用当前共享 authority；普通 locale 生成不重新生成它，也不改变其 review gate。

Course SEO revise 对明确 subset 可额外向 generation session 提供 current localized description 与独立 reviewer finding，两者只用于定位和修复语言质量问题，不能取代 canonical description 的 semantic scope，也不得把 finding 当作增加新语义的依据。生成 replacement 后由当前 AI execution environment 自动运行 `course-metadata revise`，机械更新正式资产与真实 provenance，再交由独立 Locale Surface Review session 复审。
