# 多语言翻译流程入口

本文档是本项目多语言翻译流程的唯一导航入口。它不重复各规范的具体规则；执行每个阶段时，请以对应链接文档为准。

## 总流程

```text
术语准备
→ 翻译执行
→ automatic validation
→ Candidate Snapshot
→ ChatGPT Quality Check 与 Final Review
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

新 locale 在第一次 `retranslation export` 前，必须从当前正式 TranslationUnit catalog 初始化统一状态表并通过完整性检查：

```bash
go run -mod=readonly ./cmd/tour-i18n status init --locale <locale>
go run -mod=readonly ./cmd/tour-i18n status check --locale <locale>
```

初始化只允许在 `locales/<locale>/status.tsv` 尚不存在时执行；它不会覆盖、修复或同步已有状态。已有 locale 的 source 更新与状态迁移继续使用对应正式流程，不得通过重新初始化清除 candidate、ready 或 published 状态。

翻译任务输出为 raw response；后续 restore、validation、review 与 promotion 由下一阶段规范约束。

## 3. 质量审核

candidate 完成 automatic validation 后，先生成覆盖完整 locale workflow 的 Candidate Snapshot，再进入 ChatGPT Quality Check；promotion 前还必须完成 Final Review。阅读：

- [Translation Quality Review 规范](TRANSLATION_QUALITY_REVIEW.md)：A/B/C/D rubric、逐 TranslationUnit review evidence、`approved` 决策与 promotion gate。

Candidate Snapshot 只冻结本轮审核使用的唯一完整 candidate 集合，manifest 引用现有 source、candidate 和 validation evidence，不复制文件、不修改 status，也不产生审核结果。随后 review scope 在同一 full Snapshot 中识别可复用的有效 A + approved Final Review evidence 与带 `reason` / `required_action` 的 pending unit；20-unit reviewer chunk 只处理 `review_required` unit，`revision_required` 先走 revision batch。rubric 过期但 candidate identity 未变时，必须实际重新 Final Review 并通过显式 supersede 续审，不重新翻译 candidate。promotion 仍要求完整 workflow 的每个 Unit 都有与当前最终 candidate 匹配的 A + approved Final Review evidence。

## 4. 发布部署

promotion 后构建 production bundle、部署和验收时，阅读：

- [生产运维手册](PRODUCTION_RUNBOOK.md)：production bundle、自动部署、健康检查、回滚与线上验收边界。

## 5. 项目状态

在开始工作、判断当前完成度，或核对历史 batch / upstream retranslation / production 结果时，阅读：

- [项目状态](PROJECT_STATE.md)：当前基线、已实现能力、实际状态和历史决策记录。
- [翻译质量实验](TRANSLATION_QUALITY_EXPERIMENTS.md)：历史实验、候选比较与当时的工程决策；它不是当前正式 review 或 promotion 规范。

## 文档优先级

发生冲突时，以阶段对应的正式规范为准：术语以 `TRANSLATION_TERMINOLOGY.md` 为准；模型无关输入/输出契约以 `TRANSLATION_TASK_SPEC.md` 为准；当前默认 Codex 执行规则以 `CODEX_TRANSLATION.md` 为准；batch、retry 与 revision 顺序以 `RETRANSLATION_RUNBOOK.md` 为准；质量审核和 promotion gate 以 `TRANSLATION_QUALITY_REVIEW.md` 为准；发布部署以 `PRODUCTION_RUNBOOK.md` 为准。`AGENTS.md` 只提供 Codex 自动导航，`PROJECT_STATE.md` 与 `TRANSLATION_QUALITY_EXPERIMENTS.md` 只记录状态和历史。
