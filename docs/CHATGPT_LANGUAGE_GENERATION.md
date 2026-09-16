# ChatGPT 正式语言生成执行规范

本文定义普通 ChatGPT **GPT-5.6 Sol + High** 配合 Remote Desktop Commander 执行正式语言生成时的仓库边界。它与 [Codex TranslationUnit 翻译执行规范](CODEX_TRANSLATION.md) 并列；不新增 Translation Engine，也不改变任何 validation、Quality Check、Locale Surface Review 或 Production gate。

## 职责范围

ChatGPT 可以承担以下正式语言工作：

- locale glossary 的语言判断与制定；
- 公共 UI catalog 与 article metadata 的生成或修改；
- TranslationUnit initial translation、revision，以及确实需要新译文的 `restore_failed` / `validation_failed` retry；
- schema v2 Course SEO localized description；
- 与生成会话分离的逐 TranslationUnit Quality Check、canonical English description review（仅在其正式 authority 确实 stale 时）和 Locale Surface Review。

Codex GPT-5.6 Sol + High 继续作为正式语言生成 fallback，并主要负责 repository-level code/docs/config 修改、需要完整仓库理解的变更和复杂 failure evidence 诊断。维护者本地终端负责 export、process、revalidate、status、validation、assemble、check、refresh、finalize、promote、build、publish、deploy、verifier 与 Git 等确定性生命周期。

生成与正式审核必须分离。同一 ChatGPT conversation/session 不得同时生成本轮译文并充当其正式 Quality Check reviewer；locale-level 语言资产生成与 Locale Surface Review 也必须使用独立审核 session。分离上下文不改变现有逐 TranslationUnit A-only、carry-forward、machine finalization 或 Surface Review gate。

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

不新增 importer。`retranslation process` / `retranslation retry` 继续是 restore、validation 与 attempt provenance 的正式 fail-closed authority。ChatGPT 即使能操作本地终端，也不默认继续执行 process、finalize、promote 或其他确定性生命周期；这些步骤由维护者本地终端执行，除非维护者明确要求扩大当前操作范围。

## 非 TranslationUnit 语言资产

这些资产按以下输入边界生成或修改：

- 首次制定 locale glossary：先遵循 [术语治理政策](TRANSLATION_TERMINOLOGY.md) 与 [术语制定指南](TERMINOLOGY_GUIDE.md)，并读取制定术语所需的正式 English/source context；尚未建立完成的完整 locale glossary 不是其自身的前置输入。
- 修改已有 glossary：必须读取完整当前 glossary 与相关正式 source/context。
- 生成或修改 UI catalog、article metadata 等其他 locale-level 语言资产：必须读取对应完整 source/context 和已经建立的完整 locale glossary。

所有资产仍须保持现有 key、kind、placeholder、markup、schema 和技术 identity。不得给 `glossary.yaml`、`internal/tour/ui/<locale>.json` 或 `article-metadata.json` 增加 provider/model/generation 字段。Glossary 继续是该 locale 的正式术语 authority，这些资产也继续由现有 validator 与 Locale Surface Review 审核实际内容。

schema v2 Course SEO localization 只把当前 canonical English description、完整 locale glossary、locale identity 与 `course-seo-localization-v2` constraints 交给生成 session；不得提供完整 English/target Page body。普通 ChatGPT 的真实 provenance 固定记录为：

```text
provider=chatgpt
model=gpt-5.6-sol-high
```

生成 session 只产出 `page_id → localized description` 输入；正式 `course-metadata.json` 必须由现有 `course-metadata assemble` / `refresh` CLI 机械生成并通过现有 gate，不得由 ChatGPT 直接编辑。canonical English source-description 使用当前共享 authority；普通 locale 生成不重新生成它，也不改变其 review gate。
