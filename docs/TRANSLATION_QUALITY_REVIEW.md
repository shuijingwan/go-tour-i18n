# Translation Quality Review 规范

Translation Quality Review 是所有语言翻译工作流在 promotion 前的正式必经步骤。这套规范面向长期维护和多语言复用，不绑定 `zh-CN`、ChatGPT、Page、Example 或任何特定模型。

## 正式流程

所有进入 locale translation workflow 并产生翻译结果的 TranslationUnit 必须遵循：

```text
TranslationUnit
→ export
→ model translation
→ process
→ automatic validation
→ Final Review
→ promotion
→ ready
→ projection / publish
```

Automatic validation 不等于 Translation Quality Review。自动 validator 负责结构、代码、链接、保护 token、source identity 等机器可验证的安全性；Translation Quality Review 独立判断：

- 语义是否准确、完整；
- 技术含义是否正确；
- 术语是否一致、恰当；
- 表达是否自然；
- 是否符合教程语气；
- 是否存在漏译或误译；
- 是否达到正式发布质量。

Validator 通过不能替代质量审核。

## 适用范围

审核范围不是 Catalog 中的全部 source，而是所有进入 locale translation workflow 并产生 translation candidate 的 TranslationUnit。

当前 `zh-CN` workflow 包含：

```text
103 Page + 19 eligible Example = 122 TranslationUnit
```

另外 74 个没有可翻译自然语言注释的 Example：

- 不进入 locale translation workflow；
- 不产生 candidate；
- 不产生 validation；
- 不产生 review；
- projection 继续直接使用 upstream source。

它们不会被复制为语言目录中的独立翻译 candidate。

## 正式质量评级

`A`、`B`、`C`、`D` 是长期正式质量 rubric，不是某个模型盲评实验的专用机制。

### A

- 达到正式发布质量；
- 技术准确；
- 语义完整；
- 术语正确；
- 表达自然；
- 无需修改。

### B

- 基本达到发布质量；
- 只有轻微润色空间或非常小的问题；
- 不影响技术理解和正式发布。

### C

- 不应直接发布；
- 存在需要修改的翻译质量问题；
- 修订后必须重新 Final Review。

### D

- 不可接受；
- 存在严重误译、漏译、技术含义错误或明显不符合翻译要求；
- 必须重新翻译或大幅修订，再重新 automatic validation 和 Final Review。

## Rating 与 decision 分离

Review evidence 分别记录：

```text
rating = A | B | C | D
decision = approved | rejected
```

程序不得只根据 rating 推导 decision。正式推荐规则是：

- A：通常 `approved`；
- B：允许 `approved`，reviewer 也可以根据具体问题决定 `rejected`；
- C：原则上 `rejected`；
- D：`rejected`。

Promotion gate 的机器判断只接受 `decision == approved`。这种边界保留了质量评级与 workflow 决策的独立性。

## Quality Check 与 Final Review

Quality Check 用于发现翻译质量问题，可以在 candidate 形成后的修订过程中由质量检查执行者多轮执行。它不生成正式 review evidence，也不作为 promotion 的直接依据。

Final Review 是针对最终 candidate 的正式 Translation Quality Review。只有 Final Review 生成 review evidence；该 evidence 必须对应最终 candidate 及其 validation evidence，作为 promotion 的审核依据。

## 必须逐 TranslationUnit 审核

每一个最终准备进入 promotion 的 TranslationUnit candidate，都必须有其 Final Review 生成、且与其字节内容绑定的独立 review evidence。不采用以下规则：

- 只审核首批，后续抽样；
- validator 通过即免审；
- 某个模型表现稳定后免审；
- 用批次级结论代替逐 TranslationUnit review。

抽样评审可以用于模型实验、prompt 调优和质量趋势分析，但不能替代 promotion 前的逐 TranslationUnit Review。

## Candidate 变化与重新审核

Review evidence 同时绑定：

- `source_sha256`；
- `candidate_sha256`；
- `validation_sha256`；
- `attempt`。

因此，candidate 字节发生变化、retry 产生新 candidate，或 validation evidence 发生变化时，旧 review 自动失效。新的最终 candidate 必须重新经过：

```text
automatic validation → Final Review
```

不得把旧 review evidence 套用到新 candidate 或新的 validation evidence 上。

翻译质量修订不覆盖旧 candidate。需要修订时，必须通过新的 revision batch 产生新的 candidate 和 validation evidence；旧 batch 的 review evidence 不适用于新的 candidate。

## Review evidence schema v1

每个 TranslationUnit 的正式 evidence 是 Final Review 的产物，位于：

```text
data/retranslation-runs/<locale>/<batch-id>/review/<unit-name>.json
```

Schema v1 的字段及含义如下：

| 字段 | 含义 |
| --- | --- |
| `schema_version` | Review schema 版本，当前为 `1` |
| `batch_id` | candidate 所属 retranslation batch |
| `locale` | 目标语言 |
| `unit_id` | TranslationUnit identity |
| `unit_kind` | TranslationUnit 类型 |
| `source_sha256` | 被翻译 source 的字节哈希 |
| `attempt` | validation 指向的最终翻译 attempt |
| `candidate_path` | batch candidate 路径 |
| `candidate_sha256` | batch candidate 的字节哈希 |
| `validation_path` | automatic validation evidence 路径 |
| `validation_sha256` | validation evidence 的字节哈希 |
| `decision` | workflow 决策：`approved` 或 `rejected` |
| `reviewer` | 审核主体标识 |
| `reviewed_at` | 审核时间 |
| `rubric` | 本次审核采用的 rubric identifier |
| `rating` | 正式质量评级：A、B、C 或 D |
| `summary` | 审核结论摘要 |
| `issues` | 需要说明的具体质量问题数组 |

项目当前正式推荐的 rubric identifier 是：

```text
translation-quality/v1
```

当前 schema 实现要求 `rubric` 非空，但没有在程序中把它限制为某个固定枚举。Reviewer 应使用上述 identifier；未来 rubric 演进应使用新的明确版本，而不是静默改变同一 identifier 的含义。

## Reviewer

`reviewer` 是审核主体标识，不是账号或权限系统。当前正式翻译可以由 ChatGPT 生成，Review 也可以由 ChatGPT 在独立重新读取 source、candidate 和相关 evidence 后执行。

审核必须重新判断译文质量，不得仅因为 reviewer 或同一模型生成了译文就自动批准。未来可以使用其他模型或人工 reviewer，但都必须遵循同一正式 rubric 和 evidence schema。本项目当前不构建 reviewer 账号系统。

## Issues

`issues` 是简单数组，用于记录需要说明的具体质量问题。即使没有问题，也应保留为空数组。Review evidence 不引入 Web discussion、assignee、workflow queue、comment thread 或 database ID。

## Promotion gate

Promotion preflight 对 locale workflow 的全部 TranslationUnit 检查 review。当前 `zh-CN` 必须具有 122/122 份与最终 candidate 匹配且有效的 review evidence。

以下任何情况都必须令 `can_apply=false`：

- missing review；
- rejected review；
- invalid review 或 schema mismatch；
- candidate 或 validation hash mismatch；
- batch、locale、unit、source、attempt 或 path identity mismatch。

只有 `decision == approved` 的有效 evidence 才允许对应 TranslationUnit 继续 promotion。最新 batch 的 review 缺失、rejected 或无效时，不允许回退到旧 batch 的 approved review。

## 历史质量记录

历史文档中记录过的 A/B/C/D 统计不自动视为 schema v1 的结构化 review evidence。正式 Review gate 启用后，任何现有 candidate 如果要进入新的 promotion，都必须为当前 candidate 建立 `review/*.json`，并通过当前 identity、attempt 和 hash 校验。

不得因为历史报告曾将某个翻译评为 A，就自动生成或推导 `decision: approved`。

## 多语言复用原则

Review 机制不绑定 `zh-CN`、ChatGPT、Page 或 Example。未来任何 locale、任何翻译模型、任何 TranslationUnit kind，只要产生翻译 candidate 并准备 promotion，都必须对最终 candidate 执行：

```text
automatic validation → Final Review → promotion
```

新增语言无需重新决定是否审核，也无需重新定义 A/B/C/D 的基本含义。语言特定的术语和表达要求可以在同一正式 rubric 下补充，但不能取消逐 TranslationUnit review 或弱化 promotion gate。
