# 多语言翻译流程入口

本文档是本项目多语言翻译流程的唯一导航入口。它不重复各规范的具体规则；执行每个阶段时，请以对应链接文档为准。

## 总流程

```text
术语准备
→ 翻译执行
→ 自动校验与质量审核
→ promotion
→ 发布部署
→ 项目状态确认
```

## 1. 术语准备

开始某个 locale 的翻译或调整术语前，阅读：

- [术语治理政策](TRANSLATION_TERMINOLOGY.md)：技术身份保护、glossary、protector 与 validator 的职责边界。
- [术语制定指南](TERMINOLOGY_GUIDE.md)：locale 独立的术语评估与 glossary 维护流程。
- [ja-JP 术语草案](ja-JP-TERMINOLOGY-DRAFT.md)：仅在准备 ja-JP glossary 时阅读。

产物为目标 locale 的 `locales/<locale>/glossary.yaml`；不要将一个 locale 的 glossary 机器翻译为另一个 locale 的 glossary。

## 2. 翻译执行

`AGENTS.md` 是 Codex 自动导航入口。执行 Page 或 Example 翻译、导出重译批次、处理模型返回和重试时，阅读：

- [翻译任务规范](TRANSLATION_TASK_SPEC.md)：TranslationUnit、模型输入/输出契约及结构约束。
- [Codex 翻译执行规范](CODEX_TRANSLATION.md)：当前默认 Codex TranslationUnit 翻译与直接写入规则。
- [Retranslation 执行手册](RETRANSLATION_RUNBOOK.md)：batch、retry、revision、质量检查与提升的执行顺序。

翻译任务输出为 raw response；后续 restore、validation、review 与 promotion 由下一阶段规范约束。

## 3. 质量审核

candidate 完成 automatic validation 后、promotion 前，阅读：

- [Translation Quality Review 规范](TRANSLATION_QUALITY_REVIEW.md)：A/B/C/D rubric、逐 TranslationUnit review evidence、`approved` 决策与 promotion gate。

当前严格生产策略要求所有 TranslationUnit 的 ChatGPT Quality Check 均为 A 后才能进入 Final Review；Final Review 也只有 A 才能 `approved` 并进入 promotion。

## 4. 发布部署

promotion 后构建 production bundle、部署和验收时，阅读：

- [生产运维手册](PRODUCTION_RUNBOOK.md)：production bundle、自动部署、健康检查、回滚与线上验收边界。

## 5. 项目状态

在开始工作、判断当前完成度，或核对历史 batch / upstream retranslation / production 结果时，阅读：

- [项目状态](PROJECT_STATE.md)：当前基线、已实现能力、实际状态和历史决策记录。
- [翻译质量实验](TRANSLATION_QUALITY_EXPERIMENTS.md)：历史实验、候选比较与当时的工程决策；它不是当前正式 review 或 promotion 规范。

## 文档优先级

发生冲突时，以阶段对应的正式规范为准：术语以 `TRANSLATION_TERMINOLOGY.md` 为准；模型无关输入/输出契约以 `TRANSLATION_TASK_SPEC.md` 为准；当前默认 Codex 执行规则以 `CODEX_TRANSLATION.md` 为准；batch、retry 与 revision 顺序以 `RETRANSLATION_RUNBOOK.md` 为准；质量审核和 promotion gate 以 `TRANSLATION_QUALITY_REVIEW.md` 为准；发布部署以 `PRODUCTION_RUNBOOK.md` 为准。`AGENTS.md` 只提供 Codex 自动导航，`PROJECT_STATE.md` 与 `TRANSLATION_QUALITY_EXPERIMENTS.md` 只记录状态和历史。
