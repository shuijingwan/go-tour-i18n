# Codex TranslationUnit 翻译执行规范

本文档是 Codex 执行正式 TranslationUnit 翻译时的仓库内规范，不是需要用户在每次会话中复制的 Prompt 模板。TranslationUnit 的定义和结构规则以 [翻译任务规范](TRANSLATION_TASK_SPEC.md) 为准。

## 首次正式翻译

当前默认生产流程的翻译阶段为：

```text
retranslation export
→ Codex 读取 manifest.json
→ Codex 读取 manifest 列出的全部 inputs/*
→ Codex 读取 locales/<locale>/glossary.yaml
→ Codex 完整翻译每个 TranslationUnit
→ Codex 直接写入 raw-responses/
```

manifest、全部 inputs 与 locale glossary 是不可拆分的正式模型输入。Codex 必须在翻译前完整读取 glossary，并遵守其中的 `mandatory`、`preferred`、`forbidden` 和 `keep`；glossary 不是仅供 validator 后置检查的材料。

## Codex 5 小时额度与执行分工

新增 locale 的运营目标是单个 Codex 5 小时额度窗口使用不超过 100%。这是成本/运营目标，不是 TranslationUnit quality gate：不得为此降低 A-only Quality Check、跳过 QC、减少必要 revision，或将正式翻译模型从 **GPT-5.6 Sol + High** 自动降级。暂时不设置新增 locale 的硬 wall-clock 时间目标。

在主要 model-intensive 阶段（TranslationUnit translation / revision、需重新生成译文的 retry、较大的代码理解任务）前后观察当前 5 小时额度。当前窗口已使用约 80% 时，不再启动新的非必要 Codex 重任务；对新的大型 translation、revision 或 code-understanding 任务，若剩余额度明显不足，应留到下一额度窗口，而不是硬顶到超过 100%。build、不会调用模型的 process / revalidate、status、validation、publish、deploy、verifier、checksum、curl、Git、assets 等确定性终端工作可以继续，不因 Codex quota 停止。

## 新增 locale 的首次 Page batch

新增 locale 的首次 Page 翻译当前推荐生产基线是：每批目标且最多 `60` 个完整 Page TranslationUnit。TranslationUnit 仍绝对不可拆分或合并；Page 必须按正式 Catalog/source 顺序连续组织。当前 A Tour of Go 有 103 个 Page 时，典型安排为：`60 Pages → 剩余 43 Pages → Examples 独立 batch`。执行首次 Page export 时显式使用 `--unit-kind page --limit 60`，不依赖命令的默认 limit。

Example 必须始终独立于 Page batch，不得混合。revision batch 只包含确实需要 revision 的 TranslationUnit，不得为了凑满 60 扩大范围；retry 规则不因 Page batch 大小而改变。

`60` 是当前推荐生产基线，不是已证明的理论最优值。es-ES 的实际执行显示小 batch 有明显固定执行成本；it-IT 已完成 60-Page batch，未暴露需要回退该规模的质量或 automatic validation 问题。因此为减少 batch 数量和人工操作采用此基线，但不从粗略额度数字推断其更省额度。新增 locale 应按本文件的窗口观察规则将 Codex 总额度作为显式运营约束；只有未来真实 evidence 表明 60 Page 导致模型超时或执行不稳定、automatic validation failure 增加、QC B/C/D 或 revision 成本增加、或 Codex 总额度/总耗时异常时，才重新调整该基线。

Page 输出为 `raw-responses/*.article`，Example 输出为 `raw-responses/*.txt`。每个文件只能包含对应 TranslationUnit 的完整翻译结果，禁止包含：

- JSON wrapper；
- ZIP；
- Markdown code fence；
- 解释、前言或后记；
- ChatGPT artifact 或其他对话产品 artifact。

每个 Page `.article` 和 Example `.txt` raw response 必须以恰好一个 LF 结束；最后一行之后不得再有空行。对声明 `artifact_eof: single_lf` 的 batch，`retranslation process` 会在 restore 前拒绝不合规的 raw response，不会代替 Codex 改写该原始译文。

## 完成翻译后的检查

每次写完 raw responses 后，Codex 必须检查：

- manifest 中的 unit 数量；
- raw response 数量；
- 是否缺少任何 unit；
- protected token 是否完整、唯一且未被改写；
- 每个 raw response 是否以恰好一个 LF 结束且 EOF 无额外空行；
- glossary 的 `mandatory`、`forbidden` 和 `keep` 是否满足；
- 是否残留明显未翻译的自然语言；
- `git status --short`。

本翻译阶段不自动执行 `process`、Quality Check、`quality-check finalize` 或 `promote`。只有用户明确要求继续下一阶段时，才进入相应步骤。

## Retry 与 revision

首次 raw response 写入 `raw-responses/`，并被记为 attempt 1。若 `process` 得到 `restore_failed` 或 `validation_failed`，Codex 根据失败 evidence 生成下一份连续编号的 `retries/<unit>/attempt-NNN.*`；因此首次 retry 必须是 `attempt-002.*`，不得写 `attempt-001.*`。Retry raw response 同样必须以恰好一个 LF 结束且 EOF 无额外空行。现有 `retranslation retry` 命令只处理该文件，不调用模型、不生成或自动改写译文。

若 failure 已确认只来自 validator 规则修正，且 restore 成功、原 candidate 保持有效，不得伪造 retry。使用 `retranslation revalidate --locale ... --batch-id ... --unit-id ...` 以当前 canonical validator 重验同一 candidate；命令归档旧 validation evidence，更新当前 validation/result，但保持 raw response、candidate 和 translation attempt 不变。

翻译质量问题不使用 retry。Automatic validation 已通过后，Quality Check 得到 B、C、D 时，必须创建 revision batch，并按 [Retranslation 执行手册](RETRANSLATION_RUNBOOK.md) 重新导出和翻译。
