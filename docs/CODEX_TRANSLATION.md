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

Page 输出为 `raw-responses/*.article`，Example 输出为 `raw-responses/*.txt`。每个文件只能包含对应 TranslationUnit 的完整翻译结果，禁止包含：

- JSON wrapper；
- ZIP；
- Markdown code fence；
- 解释、前言或后记；
- ChatGPT artifact 或其他对话产品 artifact。

## 完成翻译后的检查

每次写完 raw responses 后，Codex 必须检查：

- manifest 中的 unit 数量；
- raw response 数量；
- 是否缺少任何 unit；
- protected token 是否完整、唯一且未被改写；
- glossary 的 `mandatory`、`forbidden` 和 `keep` 是否满足；
- 是否残留明显未翻译的自然语言；
- `git status --short`。

本翻译阶段不自动执行 `process`、Quality Check、Final Review 或 `promote`。只有用户明确要求继续下一阶段时，才进入相应步骤。

## Retry 与 revision

首次 raw response 写入 `raw-responses/`。若 `process` 得到 `restore_failed` 或 `validation_failed`，Codex 根据失败 evidence 生成连续编号的 `retries/<unit>/attempt-NNN.*`；现有 `retranslation retry` 命令只处理该文件，不调用模型、不生成译文。

翻译质量问题不使用 retry。Automatic validation 已通过后，Quality Check 或 Final Review 得到 B、C、D 时，必须创建 revision batch，并按 [Retranslation 执行手册](RETRANSLATION_RUNBOOK.md) 重新导出和翻译。
