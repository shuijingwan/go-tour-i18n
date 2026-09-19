# ChatGPT 正式语言生成执行规范

本文定义普通 ChatGPT **GPT-5.6 Sol + High** 配合 Remote Desktop Commander 执行正式语言生成时的仓库边界。它与 [Codex TranslationUnit 翻译执行规范](CODEX_TRANSLATION.md) 并列；不新增 Translation Engine，也不改变任何 validation、Quality Check、Locale Surface Review 或 Production gate。

## 单 locale 固定会话角色

一个 locale 默认维护一个长期 **Generation session**、一个与之独立的长期 **Reviewer session**，再由维护者 **Local terminal** 执行 deterministic lifecycle。这是协作职责规则，不是机器身份系统；不得为此新增 session 字段、receipt、schema、CLI flag 或 machine gate。

### Generation session

Generation session 负责所有产生或修订该 locale 语言内容的模型工作：

- locale glossary 的语言判断与制定；
- 公共 UI catalog、article metadata 与其他 locale-level 文案的生成或修改；
- TranslationUnit initial translation、revision，以及确实需要新译文的 `restore_failed` / `validation_failed` retry；
- schema v2 Course SEO localization，以及 Surface Review finding 所需的 refresh / revise description；
- Glossary Review、TranslationUnit Quality Check 或 Locale Surface Review finding 回流后的 replacement generation。

同一个 Generation session 可以按 `glossary → Glossary Review PASS → UI / metadata → TranslationUnit initial / revision → Course SEO localization → 后续 replacement generation` 长期连续工作，只要它始终承担 generation。它可以读取完整 source、target、glossary 与 reviewer finding；Course SEO generation 可以读取当前完整 Page source 与 ready target。读取这些材料不会破坏 generation 独立性，真正禁止的是同一 conversation/session 同时承担 generation 与 formal review。

Generation session 不得执行正式 TranslationUnit Quality Check、revision 后正式 re-QC、Locale Surface Review，或批准自己生成的输出。

Generation session 制定完整 glossary 后必须停在 Glossary Review 边界；只有 generation-independent Reviewer PASS 且 Local terminal 记录 current gate 后，才正式生成 UI / article metadata 并进入 TranslationUnit generation。若 Reviewer 返回 failed finding，Generation session 负责修订 glossary，并在继续下游 generation 前交回同一独立 Reviewer session 完整复审。

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
→ Local terminal 执行 deterministic lifecycle
→ Reviewer session re-review
```

Reviewer session 不得直接修改 candidate、生成 replacement translation 或 Course SEO replacement description 后再批准自己的结果，也不得因为自己审核过上一轮而自动批准 revision。只要它没有参与 replacement generation，就可以继续审核 Generation session 修订后的新输出。Validation passed、旧 A、Surface Review 结论或模型历史表现都不能替代当前正式审核。

TranslationUnit Quality Check 与 revision re-QC 的单次 Reviewer model invocation / response 最多审核 60 TranslationUnits；Page / Example 分开。60 是实际审核质量边界，不得通过在同一次 response 内串联多个 `<=60` working set 绕过；下一组需要新的用户请求和新的 model invocation。每组必须逐 TranslationUnit 审核，不得抽样。首次 122-Unit locale 的 full QC 推荐使用 Page stable index `1-60`、Page stable index `61-103`、Example stable index `104-122` 三组；revision re-QC 按实际 pending scope 分组，每组最多仍为 60。

### Local terminal

维护者本地终端负责所有确定性步骤，包括 locale init；glossary-review record / check；retranslation export、process、retry process、revalidate；status / validation；Candidate Snapshot；quality-check scope、record / record-batch、finalize；promotion；Course SEO assemble，以及 refresh / revise 的机械 CLI；canonical/source/current checks；build；surface-review export 与 record-a；preview、browser verifier、publish、Production、deploy、verifier、checksum / curl、Git、assets、search closeout，以及现有 CLI/script 覆盖的其他机械步骤。 ChatGPT 给维护者提供可直接粘贴的终端命令时，不得在当前交互 shell 顶层启用 `set -e` / `set -u` / `set -o pipefail` 或组合形式；需要 fail-fast 时必须用独立 subshell 或独立脚本进程，避免失败退出或改变维护者当前 shell。

Generation session 只产生 Course SEO refresh / revise 所需的新 description 文本；正式 `course-metadata.json` 的 mutation 由 Local terminal 执行 `course-metadata refresh` / `revise`。ChatGPT 即使能操作本地终端，也不默认接管这些确定性步骤，除非维护者明确扩大当前操作范围。

canonical English source-description extraction / review 是所有 locale 共享的 authority，不属于单个 locale 固定的 Generation / Reviewer 配对。只有其 authority 确实 stale 时才按 [课程页正式 SEO Metadata 规范](COURSE_SEO_METADATA.md) 执行，不因新增每个 locale 重复，也不默认交给该 locale 的 Reviewer session。

Codex GPT-5.6 Sol + High 继续作为正式语言生成 fallback，并主要负责 repository-level code/docs/config 修改、需要完整仓库理解的变更和复杂 failure evidence 诊断。无论使用普通 ChatGPT 还是 Codex fallback，generation 与 formal review 的会话隔离规则不变。

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

## Remote Desktop Commander 安全写入

首次翻译或 revision 不得逐个文件直接建立正式 `raw-responses/`。在同一 batch filesystem 内执行以下顺序：

1. 确认正式 `raw-responses/` 尚不存在，在 batch 内创建隐藏 staging directory，例如 `.raw-responses.staging/`。
2. 将整个 batch 的每个 Unit 完整输出写入 staging；Page 使用对应 `.article`，Example 使用对应 `.txt`。
3. 对照 manifest 核对 expected filename exact set 与 count，并逐文件检查 protected token、glossary、明显未翻译自然语言、无 Markdown/code fence/解释，以及恰好一个结尾 LF。
4. 只有全部检查通过后，才在同一 batch 内将整个 staging directory rename/move 为 `raw-responses/`。

任何检查失败或 Remote Desktop Commander 中断时，隐藏 staging 不是正式 raw response，必须保留为未完成工作或安全清理；不得留下部分 `raw-responses/`。正式目录已存在时不得覆盖或合并，必须先查明当前 batch 状态。

Retry 使用同一原子提交思想：先将完整内容写入目标 Unit retry 目录内的隐藏 staging file，核对它满足当前连续 attempt 编号、文件名、protected token、glossary 和 single-LF contract，再 rename 为正式 `attempt-NNN.article` 或 `attempt-NNN.txt`。不得覆盖既有 attempt、跳号或伪造 provenance。

不新增 importer。`retranslation process` / `retranslation retry` 继续是 restore、validation 与 attempt provenance 的正式 fail-closed authority。

## 非 TranslationUnit 语言资产

这些资产按以下输入边界生成或修改：

- 首次制定 locale glossary：先遵循 [术语治理政策](TRANSLATION_TERMINOLOGY.md) 与 [术语制定指南](TERMINOLOGY_GUIDE.md)，并读取制定术语所需的正式 English/source context；尚未建立完成的完整 locale glossary 不是其自身的前置输入。
- 修改已有 glossary：必须读取完整当前 glossary 与相关正式 source/context。
- 生成或修改 UI catalog、article metadata 等其他 locale-level 语言资产：必须读取对应完整 source/context 和已经建立的完整 locale glossary。

首次 glossary 制定或任何使当前 Glossary Review coverage stale 的修改，都必须按 [Glossary Review 规范](GLOSSARY_REVIEW.md) 由 Reviewer session 完整审核。UI / article metadata 的正式 generation 排在 PASS 之后；Glossary Review 不因这些资产已经存在而自动批准。

所有资产仍须保持现有 key、kind、placeholder、markup、schema 和技术 identity。不得给 `glossary.yaml`、`internal/tour/ui/<locale>.json` 或 `article-metadata.json` 增加 provider/model/generation 字段。Glossary 继续是该 locale 的正式术语 authority，这些资产也继续由现有 validator 与 Locale Surface Review 审核实际内容。

schema v2 Course SEO localization 允许并推荐为每个当前 Page 提供 canonical English description、完整 English source、完整最终 ready canonical locale target、完整 locale glossary、locale identity 与 `course-seo-localization-v2` constraints。canonical description 是唯一 semantic-scope authority；source/target 只用于技术语义核对、正文术语一致性、自然表达和实际内容对齐，不授权重新摘要、增删语义或重选重点。同一 session/batch 可处理多个 Page，但每页必须只用自己的 canonical description 决定 semantic scope，不得跨页补充、混合或推断语义。普通 ChatGPT 的真实 provenance 固定记录为：

```text
provider=chatgpt
model=gpt-5.6-sol-high
```

生成 session 只产出 `page_id → localized description` 输入；正式 `course-metadata.json` 必须由现有 `course-metadata assemble` / `refresh` / `revise` CLI 机械生成并通过现有 gate，不得由 ChatGPT 直接编辑。canonical English source-description 使用当前共享 authority；普通 locale 生成不重新生成它，也不改变其 review gate。

Course SEO revise 对明确 subset 可额外向 generation session 提供 current localized description 与独立 reviewer finding，两者只用于定位和修复语言质量问题，不能取代 canonical description 的 semantic scope，也不得把 finding 当作增加新语义的依据。生成 replacement 后仍由 `course-metadata revise` 机械更新正式资产与真实 provenance，并由独立 Locale Surface Review session 复审。
