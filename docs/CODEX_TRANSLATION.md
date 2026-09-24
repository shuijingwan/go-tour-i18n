# Codex 正式语言生成执行规范

本文档定义某个 locale 在开始正式 generation 前选定 `provider=codex` 后，Codex **GPT-5.6 Sol + High** 执行正式语言生成时的仓库内规范，不是需要用户在每次会话中复制的 Prompt 模板。普通 ChatGPT + Remote Desktop Commander 的并列执行规范见 [ChatGPT 正式语言生成执行规范](CHATGPT_LANGUAGE_GENERATION.md)。两种 provider 共用相同的 TranslationUnit contract、validation、Quality Check、revision、A-only、promotion、Course SEO semantic scope 和 Production gate；TranslationUnit 的定义和结构规则以 [翻译任务规范](TRANSLATION_TASK_SPEC.md) 为准。

## Generation role 与职责边界

Codex 被选为某个 locale 的 Generation provider 后，同一个长期 Generation role/session 负责该 locale 的 glossary generation / revision、UI catalog、article metadata、其他 locale-level 文案、TranslationUnit initial / revision / 需要新译文的 retry、schema v2 Course SEO localization / refresh / revise replacement，以及 Locale Surface Review finding replacement generation。原则上整个 locale 持续使用 Codex；只有真实额度耗尽、provider/tool failure 或正式文档明确支持的恢复条件出现时才改变执行环境。不得仅为临时节省额度随意切换，不得新增虚假 generation 记录，Course SEO 必须记录真实 provenance，也不得新增 provider selection schema、receipt、machine gate 或 locale state 字段。

Codex 另有独立的 repository-level code/docs/config/schema/workflow/tooling 与复杂 failure diagnosis 职责；该职责不等于任何 locale 自动选择 Codex 作为 Generation provider。provider-neutral 的选择规则见 [多语言翻译流程](TRANSLATION_WORKFLOW.md) 与 [新增 Locale 执行手册](NEW_LOCALE_RUNBOOK.md)。

正式 Reviewer 路径继续使用从未参与该 locale language generation 的独立 ChatGPT **GPT-5.6 Sol + High** session。它可以连续承担 Glossary Review、TranslationUnit Quality Check、revision re-QC 与 Locale Surface Review；Codex Generation role 不得审核并批准自己的 glossary、candidate 或 replacement，Reviewer 也不得生成 replacement 后批准自己的输出。

## 首次正式翻译

Codex 支持的正式翻译阶段为：

```text
retranslation export（先验证 current Glossary Review coverage）
→ Local terminal 导出 provider-neutral Generation Bundle 并完成 ZIP handoff
→ Codex 完整读取 ZIP 内 manifest、全部 inputs、glossary 与 authority
→ Codex 将完整 TranslationUnit 写入隔离本地 staging
→ 目录 import（跨环境回退时 result-pack → Result ZIP import）
```

manifest、全部 inputs 与 locale glossary 是不可拆分的正式模型输入。Codex 必须在翻译前完整读取 glossary，并遵守其中的 `mandatory`、`preferred`、`forbidden` 和 `keep`；glossary 不是仅供 validator 后置检查的材料。

正式输入 transport 与 ChatGPT 完全相同：新增 locale 的 initial Page / Example bulk batch 必须由维护者 Local terminal 执行 `generation-bundle export` 并把 ZIP handoff 给 Codex；Codex 不以重新扫描工作树代替这次 handoff。Codex 将 exact expected outputs 写入按 locale + batch 隔离的本地隐藏 staging，默认用 `generation-bundle import --input-dir` 直接导入；跨环境传输或目录不可用时才使用 `result-pack` / Result ZIP import。revision / retry 同样复用目录 import。Codex 先将每个完整 TranslationUnit 写入临时文件，再 rename 成 staging 成品；导入及正式 process / automatic validation 通过后，每批停下等待维护者明确“继续”。命令示例：

```sh
go run -mod=readonly ./cmd/tour-i18n generation-bundle import \
  --bundle /tmp/<locale>-<batch-id>-generation.zip \
  --input-dir /tmp/.go-tour-i18n-generation/<locale>/<batch-id> \
  --provider codex --model gpt-5.6-sol-high
```
所有路径都重验 current identity、路径、exact set、UTF-8/no-BOM/single-LF、protected restore、glossary、machine candidate 与 no-overwrite；导入持久保存既有 Result Bundle manifest 作为 provenance 记录，不新增 schema；记录真实 provider/model、Generation Bundle SHA-256、input identity、attempt 与输出 hashes。bundle 不改变 TranslationUnit 边界，也不是 authority 或质量 gate；导入后仍执行正式 process/validation 与独立 A-only QC。

Glossary 的制定、独立审核与 machine gate 以 [Glossary Review 规范](GLOSSARY_REVIEW.md) 为准。Codex Generation role 不得用自己的 generation 上下文审核并批准同一 glossary。

## Codex 额度边界

新增 locale 的运营目标是单个 Codex 5 小时额度窗口使用不超过 100%。开始正式 generation 前，应把当前 5 小时额度、周额度、历史实测成本和并发计划纳入 provider selection。这些只是成本/运营约束，不是 TranslationUnit quality gate：不得为此降低 A-only Quality Check、跳过 QC、减少必要 revision，或将正式翻译模型从 **GPT-5.6 Sol + High** 自动降级。暂时不设置新增 locale 的硬 wall-clock 时间目标。

在主要 model-intensive Codex 阶段（TranslationUnit translation / revision、需重新生成译文的 retry、较大的代码理解任务）前后观察当前 5 小时额度与周额度。当前 5 小时窗口已使用约 80% 时，不再启动新的非必要 Codex 重任务；对新的大型 translation、revision 或 code-understanding 任务，若剩余额度明显不足，应留到下一额度窗口，而不是硬顶到超过 100%。额度真正耗尽时可以依 provider-neutral 恢复规则改变执行环境，但必须保留真实 provenance 与全部质量 gate。闭环内不会调用模型的短机械步骤不因 Codex quota 停止，默认由 Codex 连续完成；闭环收口后的 promotion、build、preview / publish、Production / deploy / verifier、checksum / curl、assets，以及最终 Git commit / push 由维护者 Local terminal 成组执行。

## 非 TranslationUnit 与 Course SEO generation

Codex 生成或修订 glossary、UI catalog、article metadata 和其他 locale-level 文案时，必须读取对应完整 source/context 与当前完整 locale glossary，并遵守 [Glossary Review 规范](GLOSSARY_REVIEW.md) 和 [Locale Surface Review](LOCALE_SURFACE_REVIEW.md) 的独立审核边界。正式输入优先使用 `generation-bundle locale-export --task glossary|locale-assets|surface-replacement`，并在使用前以 `generation-bundle locale-check` 确认 current；该 ZIP 只运输上下文，不直接写正式资产。

首次 schema v2 Course SEO localization 使用与 ChatGPT 相同的 `course-metadata localization-bundle`；后续 refresh / revise replacement 使用 `course-metadata generation-bundle --task refresh|revise`。bundle 必须完整包含 canonical source descriptions、每个 Page 的完整 English source、完整 ready canonical target、完整 locale glossary、locale identity 和 current authority。canonical English description 仍是唯一 semantic-scope authority；Codex 不直接手写正式 `course-metadata.json`，而是默认把 strict descriptions JSON 落盘并立即通过 assemble / refresh / revise CLI 机械写入正式资产，记录真实 `provider=codex`、`model=gpt-5.6-sol-high` provenance。

## 新增 locale 的首次 Page batch

新增 locale 的首次 Page 翻译当前推荐生产基线是：每批目标且最多 `60` 个完整 Page TranslationUnit。TranslationUnit 仍绝对不可拆分或合并；Page 必须按正式 Catalog/source 顺序连续组织。当前 A Tour of Go 有 103 个 Page 时，典型安排为：`60 Pages → 剩余 43 Pages → Examples 独立 batch`。执行首次 Page export 时显式使用 `--unit-kind page --limit 60`，不依赖命令的默认 limit。

Example 必须始终独立于 Page batch，不得混合。revision batch 只包含确实需要 revision 的 TranslationUnit，不得为了凑满 60 扩大范围；retry 规则不因 Page batch 大小而改变。

`60` 是当前推荐生产基线，不是已证明的理论最优值。es-ES 的实际执行显示小 batch 有明显固定执行成本；it-IT 已完成 60-Page batch，未暴露需要回退该规模的质量或 automatic validation 问题。因此为减少 batch 数量和人工操作采用此基线，但不从粗略额度数字推断其更省额度。新增 locale 应按本文件的窗口观察规则将 Codex 总额度作为显式运营约束；只有未来真实 evidence 表明 60 Page 导致模型超时或执行不稳定、automatic validation failure 增加、QC B/C/D 或 revision 成本增加、或 Codex 总额度/总耗时异常时，才重新调整该基线。

Page staging 输出为与 bundle `expected_outputs` 对应的 `.article`，Example 为 `.txt`；正式路径只由 deterministic import 安装。每个文件只能包含对应 TranslationUnit 的完整翻译结果，禁止包含：

- JSON wrapper；
- ZIP；
- Markdown code fence；
- 解释、前言或后记；
- ChatGPT artifact 或其他对话产品 artifact。

每个 Page `.article` 和 Example `.txt` raw response 必须以恰好一个 LF 结束；最后一行之后不得再有空行。对声明 `artifact_eof: single_lf` 的 batch，`retranslation process` 会在 restore 前拒绝不合规的 raw response，不会代替 Codex 改写该原始译文。

## 完成翻译后的检查

每次写完 staging outputs 后，Codex 必须检查：

- manifest 中的 unit 数量；
- raw response 数量；
- 是否缺少任何 unit；
- protected token 是否完整、唯一且未被改写；
- 每个 raw response 是否以恰好一个 LF 结束且 EOF 无额外空行；
- glossary 的 `mandatory`、`forbidden` 和 `keep` 是否满足；
- 是否残留明显未翻译的自然语言；
- exact output file set。

initial bulk ZIP handoff 不自行跨越到后续语言审核。进入 Generation / Reviewer 闭环后，Codex 默认自动执行当前小批工作所需的 `process`、validation、Snapshot / scope、reviewer-bundle 和审核结果落盘 / `quality-check finalize`；正式 Quality Check 仍必须由独立 ChatGPT conversation/session 完成。`promote` 属于闭环收口后的维护者 Local terminal 操作，Codex 不自动执行。

Codex 生成本轮 TranslationUnit 时，后续正式 Quality Check 必须由独立 ChatGPT conversation/session 执行；生成上下文不得同时充当正式 reviewer。

## Retry 与 revision

首次 output 由 deterministic import 安装到 `raw-responses/`，同一目录下持久化的 Result Bundle manifest 绑定 attempt 1 与 generation provenance。若 `process` 得到 `restore_failed` 或 `validation_failed`，Codex 根据 retry Generation Bundle 与失败 evidence 生成下一份连续编号输出；import 将其安装到 `retries/<unit>/attempt-NNN.*`，因此首次 retry 必须是 `attempt-002.*`，不得创建 `attempt-001.*`。Retry raw response 同样必须以恰好一个 LF 结束且 EOF 无额外空行。现有 `retranslation retry` 命令只处理已导入文件，不调用模型、不生成或自动改写译文。

若 failure 已确认只来自 validator 规则修正，且 restore 成功、原 candidate 保持有效，不得伪造 retry。使用 `retranslation revalidate --locale ... --batch-id ... --unit-id ...` 以当前 canonical validator 重验同一 candidate；命令归档旧 validation evidence，更新当前 validation/result，但保持 raw response、candidate 和 translation attempt 不变。

翻译质量问题不使用 retry。Automatic validation 已通过后，Quality Check 得到 B、C、D 时，必须创建 revision batch，并按 [Retranslation 执行手册](RETRANSLATION_RUNBOOK.md) 重新导出和翻译。
