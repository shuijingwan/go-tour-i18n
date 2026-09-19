# 多语言翻译流程入口

本文档是本项目多语言翻译流程的唯一导航入口。它不重复各规范的具体规则；执行每个阶段时，请以对应链接文档为准。

## 总流程

```text
术语准备
→ 翻译执行
→ automatic validation
→ Candidate Snapshot
→ ChatGPT Quality Check → machine finalization
→ promotion
→ 发布部署
→ 项目状态确认
```

## 单 locale 协作职责

一个 locale 默认由一个长期 Generation session、一个与之独立的长期 Reviewer session，加上维护者 Local terminal 协作。下图只说明职责流，不改变任何 stage-specific authority 规定的 CLI 顺序：

```text
Generation session
  glossary / UI / metadata / TranslationUnit generation
        ↓
Local terminal
  export / process / validation / Candidate Snapshot
        ↓
Reviewer session
  TranslationUnit Quality Check
        ↓ B/C/D finding
Generation session
  revision generation
        ↓
Local terminal
  process / new Candidate Snapshot
        ↓
Reviewer session
  re-QC
        ↓
Local terminal
  finalize / promotion
        ↓
Generation session
  schema v2 Course SEO localization
        ↓
Local terminal
  assemble / build / surface-review export
        ↓
Reviewer session
  Locale Surface Review
        ↓
language defect → Generation session
PASS → Local terminal preview / publish / Production
```

Generation session 与 Reviewer session 不得是同一 conversation/session；同一个未参与该 locale 语言 generation 的 Reviewer session 可以连续承担 TranslationUnit Quality Check、revision 后 re-QC 与 Locale Surface Review。详细角色边界见 [ChatGPT 正式语言生成执行规范](CHATGPT_LANGUAGE_GENERATION.md)。canonical English source-description extraction / review 是跨 locale 共享 authority，不属于这个单 locale 固定配对，仍以 [课程页正式 SEO Metadata 规范](COURSE_SEO_METADATA.md) 为准。

## 1. 术语准备

开始某个 locale 的翻译或调整术语前，阅读：

- [术语治理政策](TRANSLATION_TERMINOLOGY.md)：技术身份保护、glossary、protector 与 validator 的职责边界。
- [术语制定指南](TERMINOLOGY_GUIDE.md)：locale 独立的术语评估与 glossary 维护流程。
- [ja-JP 术语草案](ja-JP-TERMINOLOGY-DRAFT.md)：仅在准备 ja-JP glossary 时阅读。

产物为目标 locale 的 `locales/<locale>/glossary.yaml`；不要将一个 locale 的 glossary 机器翻译为另一个 locale 的 glossary。

## 2. 翻译执行

`AGENTS.md` 是自动执行者进入仓库流程的导航入口，不替代各 stage-specific authority。执行 Page 或 Example 翻译、导出重译批次、处理模型返回和重试时，阅读：

- [翻译任务规范](TRANSLATION_TASK_SPEC.md)：TranslationUnit、模型输入/输出契约及结构约束。
- [ChatGPT 正式语言生成执行规范](CHATGPT_LANGUAGE_GENERATION.md)：普通 ChatGPT + Remote Desktop Commander 的支持范围、原子 staging 写入和生成/审核隔离。
- [Codex 翻译执行规范](CODEX_TRANSLATION.md)：Codex TranslationUnit 翻译 fallback、首次 Page batch 与直接写入规则。
- [Retranslation 执行手册](RETRANSLATION_RUNBOOK.md)：export/retry/revision、质量检查与提升的执行顺序。

新 locale 在第一次 `retranslation export` 前，必须从当前正式 TranslationUnit catalog 初始化统一状态表并通过完整性检查：

```bash
go run -mod=readonly ./cmd/tour-i18n status init --locale <locale>
go run -mod=readonly ./cmd/tour-i18n status check --locale <locale>
```

初始化只允许在 `locales/<locale>/status.tsv` 尚不存在时执行；它不会覆盖、修复或同步已有状态。已有 locale 的 source 更新与状态迁移继续使用对应正式流程，不得通过重新初始化清除 candidate、ready 或 published 状态。

翻译任务输出为 raw response；新 retranslation batch 的 input、raw response、retry raw response 与 candidate 统一使用恰好一个结尾 LF，详细生成与拒绝边界见对应执行规范。后续 restore、validation、review 与 promotion 由下一阶段规范约束。

## 3. 质量审核

candidate 完成 automatic validation 后，先生成覆盖完整 locale workflow 的 Candidate Snapshot，再进入 ChatGPT Quality Check；全 A 后执行 machine finalization，再 promotion。阅读：

- [Translation Quality Review 规范](TRANSLATION_QUALITY_REVIEW.md)：A/B/C/D rubric、逐 TranslationUnit review evidence、`approved` 决策与 promotion gate。

Candidate Snapshot 只冻结本轮审核使用的唯一完整 candidate 集合。Quality Check 使用 `quality-check scope` 与 `quality-check-results.json`，revision 后只 carry-forward identity 完全未变的 A。全 Snapshot A 后，`quality-check finalize` 机械验证完整 lineage 与 identity 并生成独立 `finalization.json`；新 promotion 只接受匹配当前 Snapshot 的 finalization，且仍重验底层 source/input/glossary/candidate/validation/retry evidence。历史 Final Review evidence 原样保留，仅用于历史解释与验证。

## 4. 发布部署

promotion 后构建 production bundle、部署和验收时，阅读：

- [生产运维手册](PRODUCTION_RUNBOOK.md)：production bundle、自动部署、健康检查、回滚与线上验收边界。

## 5. 项目状态

在开始工作、判断当前完成度，或核对历史 batch / upstream retranslation / production 结果时，阅读：

- [项目状态](PROJECT_STATE.md)：当前基线、已实现能力、实际状态和历史决策记录。
- [翻译质量实验](TRANSLATION_QUALITY_EXPERIMENTS.md)：历史实验、候选比较与当时的工程决策；它不是当前正式 review 或 promotion 规范。

## 文档优先级

发生冲突时，以阶段对应的正式规范为准：术语以 `TRANSLATION_TERMINOLOGY.md` 为准；模型无关输入/输出契约以 `TRANSLATION_TASK_SPEC.md` 为准；ChatGPT 执行与安全 staging 以 `CHATGPT_LANGUAGE_GENERATION.md` 为准；Codex fallback 与新增 locale 首次 Page batch 以 `CODEX_TRANSLATION.md` 为准；export、retry 与 revision 顺序以 `RETRANSLATION_RUNBOOK.md` 为准；质量审核和 promotion gate 以 `TRANSLATION_QUALITY_REVIEW.md` 为准；发布部署以 `PRODUCTION_RUNBOOK.md` 为准。`AGENTS.md` 只提供自动导航，`PROJECT_STATE.md` 与 `TRANSLATION_QUALITY_EXPERIMENTS.md` 只记录状态和历史。
