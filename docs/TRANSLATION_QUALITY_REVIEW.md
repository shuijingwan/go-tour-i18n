# Translation Quality Review 规范

Translation Quality Review 是所有语言翻译工作流在 promotion 前的正式必经步骤。这套规范面向长期维护和多语言复用，不绑定 `zh-CN`、ChatGPT、Page、Example 或任何特定模型。

本规范只审核 TranslationUnit candidate。公共 UI、首页、`/tour/list`、导航、语言选择器、runtime message、article metadata、SEO 与桌面/移动端组合页面由独立的 [Locale Surface Review](LOCALE_SURFACE_REVIEW.md) 审核。Surface Review 是 promotion 后、publish / production 前的 locale release gate，不生成本规范的 review evidence，也不改变全 A、逐 TranslationUnit 和 Final Review A-only 规则。

## 正式流程

所有进入 locale translation workflow 并产生翻译结果的 TranslationUnit 必须遵循：

```text
TranslationUnit
→ export
→ model translation
→ process
→ automatic validation
→ Candidate Snapshot
→ ChatGPT Quality Check
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

当前完整 locale workflow 包含：

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

## Candidate Snapshot

Candidate Snapshot 是 automatic validation 与 ChatGPT Quality Check 之间的正式轻量边界。它回答本轮完整语言审核使用哪 122 个 candidate，不执行审核本身。

正式命令为：

```bash
go run -mod=readonly ./cmd/tour-i18n quality-check snapshot \
  --locale <locale> \
  --snapshot-id <snapshot-id>
```

产物固定为：

```text
data/quality-check-snapshots/<locale>/<snapshot-id>/manifest.json
```

Snapshot 对每个当前 workflow TranslationUnit 复用 promotion 的 latest-batch 选择与自动证据校验边界：

- 按 numeric batch number 选择最新的一份；
- 最新结果必须匹配当前 source revision identity 且状态为 `passed`；
- 最新结果失败或 identity 不匹配时绝不 fallback 到旧 batch；
- retry 后 validation 必须指向连续 provenance 的最终 attempt；
- manifest、source、input、candidate、validation identity 和相关 SHA-256 必须一致；
- 顺序固定为 Catalog 中的 Page 顺序，其后为 eligible Example inventory 顺序。

Snapshot manifest 顶层记录 `schema_version`、`snapshot_id`、`locale`、`glossary_path`、`glossary_sha256`、`unit_count`、`page_count`、`example_count` 和 `units`。每个 unit 记录稳定 `index`、`unit_id`、`unit_kind`、`selected_batch_id`、source/candidate/validation 的 path 与 SHA-256，以及 `attempt`；Page 另以 `page_section` 记录 `article`、`section_number`、`source_title` 和 `route`，从完整 article source path 唯一定位该 `present.Section`。

所有 path 都是对仓库已有文件的引用。Snapshot 不复制 source、candidate、validation 或 `_content`，不生成 ZIP，不修改 `locales/<locale>/status.tsv`，不 promotion，也不生成 Quality Check 或 Final Review evidence。Snapshot 目录只包含 `manifest.json`。

## Full Snapshot 与 review scope

Candidate Snapshot 始终是 **full Snapshot**：它冻结完整 locale workflow 的全部 TranslationUnit。当前 `zh-CN` 与 `ja-JP` 都是 103 Page + 19 eligible Example = 122 Unit。不得为了维护批次创建只含变化 Unit 的 Snapshot。

在同一份 full Snapshot 上，以下命令只读地建立 **review scope**：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation review scope \
  --locale <locale> \
  --snapshot-id <snapshot-id>
```

- **reusable reviewed unit**：存在有效且完全匹配当前 Snapshot identity 的 Final Review evidence，且 `rating=A`、`decision=approved`、rubric 为当前 `translation-quality/v1`。匹配 batch、locale、unit、source SHA-256、candidate/validation path 与 SHA-256，以及最终 attempt。
- **pending review unit**：scope 为每个 pending Unit 输出机器可读的 `reason` 和 `required_action`。`missing_review`、`identity_changed` 与 `rubric_mismatch` 的 action 是 `review_required`；`non_a_review`（B/C/D）是 `revision_required`；`rejected_review` 保守地也是 `revision_required`。前两类需要实际重新审核，后两类不得借重新记录 review 绕过 revision batch。

首次 locale 没有可复用 evidence，因此 review scope 等于 full Snapshot，是 **full review**。后续 upstream sync 或 revision 中，full Snapshot 仍保持完整；只审核 identity 已变化或 evidence 失效的 pending Unit，是 **incremental review**。

Snapshot 的 glossary SHA-256 会在读取 review scope 时与当前 glossary 重新核验。glossary 已变化时，scope 以 **snapshot-level blocker** 失败，必须先创建新的 full Snapshot；不得把它伪装为普通 unit pending。rubric 使用明确版本 identifier，当前有效 evidence 必须是 `translation-quality/v1`；升级该 identifier 会使旧 evidence 以 `rubric_mismatch/review_required` 进入 pending。上述 guard 不能缩小 promotion gate。

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
- 在通用 rubric 中与 A 保持区分；在本项目当前严格生产策略下仍须修订。

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

程序不得只根据 rating 推导 decision，通用 schema 继续保留二者分离。当前项目采用更严格的 A-only 生产策略：

- A：可以 `approved`；
- B、C、D：必须 `rejected` 并进入 revision batch。

Promotion gate 的机器判断只接受 `decision == approved`。这种边界保留了质量评级与 workflow 决策的独立性。

## Quality Check 与 Final Review

Quality Check 用于发现翻译质量问题，可以在 candidate 形成后的修订过程中多轮执行。当前由 ChatGPT 统一执行，且必须覆盖同一份 Candidate Snapshot 中的所有 TranslationUnit。它不生成正式 review evidence，也不作为 promotion 的直接依据。

ChatGPT Quality Check 与 Final Review 的默认审核执行分片均为每轮 20 个 **pending review unit**。pending 列表保持 Snapshot 的稳定 index 顺序；最后一轮不足 20 个时审核全部剩余 pending Unit。20 只是 reviewer 每轮读取和处理的 chunk size，不缩小或重新生成 Candidate Snapshot，不改变该 snapshot 的完整 locale workflow 范围，也不把逐 TranslationUnit 审核改成批次级审核或抽样。每个 pending TranslationUnit 在 Quality Check 和 Final Review 两个阶段仍分别接受独立判断。

当前严格生产 gate 为：A 通过；B、C、D 均不通过并进入新的 revision batch。只有完整语言达到 `A = 全部 TranslationUnit，B = 0，C = 0，D = 0`，才能进入 Final Review。

Final Review 是针对最终 candidate 的正式 Translation Quality Review，会重新审核最终 candidate。只有 Final Review 生成 review evidence；该 evidence 必须对应最终 candidate 及其 validation evidence，作为 promotion 的审核依据。当前严格策略下只有 Final Review A 可以 `approved`；B、C、D 不得 promotion，必须进入新的 revision batch，修订后重新走 process、automatic validation、ChatGPT Quality Check 和 Final Review。

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
automatic validation → Candidate Snapshot → ChatGPT Quality Check → Final Review
```

不得把旧 review evidence 套用到新 candidate 或新的 validation evidence 上。

翻译质量修订不覆盖旧 candidate。需要修订时，必须通过新的 revision batch 产生新的 candidate 和 validation evidence，并重新生成完整 locale Candidate Snapshot；旧 snapshot 仍是历史审核范围，不会自动改指新 candidate，旧 batch 的 review evidence 也不适用于新的 candidate。

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

当前正式实现要求 `rubric` 精确为上述 identifier。未来 rubric 演进必须使用新的明确版本，而不是静默改变同一 identifier 的含义。

## 批量记录 Final Review evidence

Final Review 完成一个默认 20-unit 执行分片后，使用 Candidate Snapshot 的稳定 index 批量记录该轮 evidence：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation review record-batch \
  --locale <locale> \
  --snapshot-id <snapshot-id> \
  --start-index <1-based-index> \
  --rating A \
  --decision approved \
  --summary <summary> \
  --reviewer <reviewer> \
  --rubric translation-quality/v1
```

`--limit` 默认是 20；可以显式设置，最后一轮会自动截断到可由普通 record 写入的 `missing_review/identity_changed` 列表末尾。`--start-index` 是该列表的 1-based 起点；`--issue` 可以重复。`rubric_mismatch` 必须使用下述显式 supersede；B/C/D 和 rejected review 必须走 revision。命令只记录已经完成的 Final Review，不执行或推导审核；传入的 `rating`、`decision`、`summary`、`reviewer`、`rubric` 和 `issue` 应真实适用于本轮选中的每个 TranslationUnit。

命令从 snapshot unit 自动读取 `selected_batch_id`，调用者不得也无需人工判断 candidate 属于哪个 batch。它先对选中范围内的全部 unit 完成单 unit `RecordRetranslationReview` 所用的 schema、identity 和 hash preflight，并核对 snapshot 中的 source、candidate、validation、attempt 与 path；全部通过后才写 evidence。任一 unit preflight 失败时本轮不写任何新 review，已有 review 也不会被覆盖。原有 `retranslation review record --batch-id ... --unit-id ...` 继续保留，用于明确的单 TranslationUnit 记录。

`record-batch` 只是 Final Review evidence 的写入工具。它不会改变 Candidate Snapshot、Quality Check/Final Review 的逐 TranslationUnit 语义、A-only 生产策略或 promotion gate。

## Rubric renewal / supersede

rubric 升级不等于 TranslationUnit 重译：candidate、source、validation、attempt、batch 与 path identity 完全不变时，reviewer 仍须实际重新读取并完成 Final Review，但不得重新翻译 candidate，也不得伪造新的 retranslation batch。

仅当 scope 明确给出 `rubric_mismatch` 和 `review_required`，且旧 canonical review 本身是 identity 完全匹配的 A + approved 时，才能显式执行：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation review supersede \
  --locale <locale> --snapshot-id <snapshot-id> --unit-id <unit-id> \
  --rating A --decision approved --summary <summary> \
  --reviewer <reviewer> --rubric translation-quality/v1
```

该命令先逐字段核对 identity，再将旧 canonical JSON 原样保存到同一 batch 的 `review/history/`，随后才原子替换 canonical `review/*.json`。普通 `record` 和 `record-batch` 永不覆盖已有 review。B/C/D、rejected、缺失或 identity 不匹配的 evidence 不可 supersede；它们必须按 scope 指示重审或创建 revision batch。promotion 只读取 canonical review，history 不参与 gate，但始终保留可审计性。

## Reviewer

`reviewer` 是审核主体标识，不是账号或权限系统。历史正式翻译可以由 ChatGPT 生成；当前默认翻译由 Codex 执行，ChatGPT 统一承担 Quality Check。Final Review 的审核主体必须独立重新读取 source、candidate 和相关 evidence 后执行审核。

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

只有 rating A 且 `decision == approved` 的有效 evidence 才允许对应 TranslationUnit 继续 promotion。promotion preflight 始终检查完整 locale workflow，而不是当前 review scope；最新 batch 的 review 缺失、非 A、rejected 或无效时，不允许回退到旧 batch 的 approved review。

## 历史质量记录

历史文档中记录过的 A/B/C/D 统计不自动视为 schema v1 的结构化 review evidence。正式 Review gate 启用后，任何现有 candidate 如果要进入新的 promotion，都必须为当前 candidate 建立 `review/*.json`，并通过当前 identity、attempt 和 hash 校验。

不得因为历史报告曾将某个翻译评为 A，就自动生成或推导 `decision: approved`。

## 多语言复用原则

Review 机制不绑定 `zh-CN`、ChatGPT、Page 或 Example。未来任何 locale、任何翻译模型、任何 TranslationUnit kind，只要产生翻译 candidate 并准备 promotion，都必须对最终 candidate 执行：

```text
automatic validation → Candidate Snapshot → ChatGPT Quality Check → Final Review → promotion
```

新增语言无需重新决定是否审核，也无需重新定义 A/B/C/D 的基本含义。语言特定的术语和表达要求可以在同一正式 rubric 下补充，但不能取消逐 TranslationUnit review 或弱化 promotion gate。
