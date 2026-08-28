# Retranslation 执行手册

## 1. 当前默认生产流程

当前唯一默认的正式 TranslationUnit 翻译流程是：

```text
retranslation export
→ Codex（High）翻译并直接写入 raw-responses/
→ retranslation process
→ automatic validation
→ quality-check snapshot
→ ChatGPT Quality Check
→ Final Review
→ promote
```

模型与 reasoning 由用户在 Codex UI 中选择；当前推荐生产配置为 GPT-5.6 Sol + High。翻译输入、输出和 Codex 写入规则分别见 [翻译任务规范](TRANSLATION_TASK_SPEC.md) 与 [Codex 翻译执行规范](CODEX_TRANSLATION.md)。

## 2. Export 与首次翻译

新 locale 第一次 export 前必须已经完成统一状态表初始化：

```bash
go run -mod=readonly ./cmd/tour-i18n status init --locale <locale>
go run -mod=readonly ./cmd/tour-i18n status check --locale <locale>
```

只有 `status check` 通过后，才进入首次 `retranslation export`。`status init` 只创建缺失的初始 `status.tsv`，不得用于重置或同步已有 locale；已有 locale 的 source 更新与状态迁移继续走现有正式流程。

使用 `retranslation export` 创建 batch。开始翻译前，Codex 必须读取当前 batch 的：

1. `manifest.json`；
2. manifest 列出的全部 `inputs/*`；
3. `locales/<locale>/glossary.yaml`。

这三部分不可拆分。每个 TranslationUnit 独立翻译：Page 从 `inputs/*.article` 写入 `raw-responses/*.article`；Example 从 `inputs/*.txt` 写入 `raw-responses/*.txt`。TranslationUnit 是翻译、validation 和 review 的最小单位，batch 只是执行与归档容器。

## 3. Process 与 automatic validation

raw responses 完整后执行：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation process --locale <locale>
```

`process` 重新生成受保护输入、执行 restore、写入 batch candidate，并运行正式 automatic validation。结果保存在该 batch 的 `candidates/`、`validation/` 和 `result.json`，不会自动修改 canonical candidate 或 status。

Automatic validation 只负责结构、保护 token、代码、链接、source identity 等机器安全性，不能替代翻译质量检查。

### 提交前 whitespace 检查与不可变 artifact 例外

`git diff --check` 仍是所有正式提交前的必做检查，默认不得带 whitespace error 提交。源码、文档、手工编辑文件，以及普通 trailing whitespace，都必须正常修复；本节的例外不能用于它们。

极少数情况下，正式 exporter 生成的 immutable artifact 可能带有单一 `new blank line at EOF` warning。仅当**全部**满足以下条件时，允许将该 warning 作为局部、可审计的例外保留并继续提交：

- artifact 由正式 exporter 生成，且已由 manifest 与相关 SHA-256 固定；
- artifact 已参与该 batch 的正式 `retranslation process`；
- 该 batch 的 restore 和 automatic validation 都已通过；
- 修改该 artifact 会改变正式 input 或 hash identity；
- `git diff --check` 除这一项 EOF blank-line warning 外没有任何其他错误。

满足条件时，不得仅为了让 `git diff --check` 全绿而改写 immutable artifact，必须保留 exporter 的原始字节，并在提交记录中明确该例外的 artifact、warning 与通过的 process/validation。该规则源于已验证 artifact 的身份不可变性，不是通用 whitespace 豁免。

下列情况绝不适用本例外：尚未冻结或未完成 process 的 artifact、restore 或 validation 未通过的 artifact、任何普通 trailing whitespace、源码或文档、多项 whitespace error，以及任何可能掩盖实际内容损坏的 warning。

## 4. Retry：只处理 restore/validation failure

Retry 只用于：

- `restore_failed`；
- `validation_failed`。

它不用于翻译质量修改。Codex 读取失败 evidence 和原正式输入，生成下一份连续编号的原始译文：

```text
retries/<flattened-unit-id>/attempt-NNN.article
retries/<flattened-unit-id>/attempt-NNN.txt
```

文件已存在后再执行：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation retry \
  --locale <locale> \
  --batch-id <batch-id> \
  --unit-id <unit-id>
```

`retranslation retry` 本身不调用模型、不生成或改写译文；它只处理已经存在的 retry raw response，归档前一份 validation，并更新目标 unit 的 candidate、validation 与 batch 汇总。Attempt 必须连续且不得覆盖。

## 5. Candidate Snapshot

完整 locale workflow 的所有 TranslationUnit 都有通过 automatic validation 的最终 candidate 后，生成本轮审核的 Candidate Snapshot：

```bash
go run -mod=readonly ./cmd/tour-i18n quality-check snapshot \
  --locale <locale> \
  --snapshot-id <snapshot-id>
```

产物只有：

```text
data/quality-check-snapshots/<locale>/<snapshot-id>/manifest.json
```

Snapshot 按 Catalog 的 Page 顺序、再按 eligible Example inventory 顺序冻结完整 workflow。当前基线是 103 Page + 19 eligible Example = 122 TranslationUnit。每个 unit 先按 batch number 选择最新的一份结果，再要求它匹配当前 source revision 且状态为 `passed`；最新结果失败或 identity 不匹配时都禁止回退旧 batch。Snapshot 同时校验 manifest/source/input/candidate/validation identity、相关 SHA-256、restore 绑定和 retry 最终 attempt。

Manifest 只引用仓库中已有的 glossary、source、candidate 和 validation 文件，不复制这些文件，不创建 `_content`、ZIP 或 review artifact。该命令不修改 `locales/<locale>/status.tsv`，不执行 Quality Check、Final Review 或 promotion。

Snapshot 后先运行 `retranslation review scope --locale <locale> --snapshot-id <snapshot-id>`。scope 在完整 Snapshot 内区分可复用的有效 A + approved Final Review evidence 与 pending review unit，并对每项输出 `reason`、`required_action`：`review_required` 才进入实际复审，`revision_required` 必须先建 revision batch。首次 locale 的 pending 等于全部 Snapshot，后续 revision 只审核 identity 已变化或 evidence 无效的 Unit。glossary snapshot mismatch 是 scope 的整体 blocker，不是 unit pending。默认每轮 20 个 reviewer chunk 针对可复审的 pending unit，Snapshot 仍必须一次冻结完整 locale workflow，不得按 chunk 创建局部 snapshot。

## 6. Quality Check 与 revision batch

ChatGPT Quality Check 必须以同一份 Candidate Snapshot manifest 为审核范围，不得自行从各 batch 中重新挑选 candidate。当前严格生产策略只接受 A：

- A：通过 Quality Check；
- B、C、D：未通过质量 gate，必须进入 revision batch。

Quality Check 默认每轮审核 20 个连续 snapshot index，但必须逐 TranslationUnit 给出判断，并在多轮结束后覆盖同一 snapshot 的全部 unit。分片不是抽样，不改变 `A = 全部 TranslationUnit` 才能进入 Final Review 的条件。

Quality Check 的质量修改不得使用 retry。Revision 流程为：

```text
创建新的 revision batch
→ re-export 对应 TranslationUnit
→ Codex 重新读取完整正式输入并重新翻译
→ process
→ automatic validation
→ 生成新的完整 locale Candidate Snapshot
→ ChatGPT Quality Check
```

重复以上流程直到 Quality Check 为 A。旧 batch 及其 evidence 保持不可变。

## 7. Final Review 与 promotion

只有完整语言满足以下条件，才能进入 Final Review：

```text
A = 全部 TranslationUnit
B = 0
C = 0
D = 0
```

Final Review 重新审核最终 candidate 并生成正式 review evidence。当前严格策略下，Final Review A 才允许 `approved`；Final Review B、C、D 不得 promotion，必须创建新的 revision batch，随后重新执行 process、automatic validation、ChatGPT Quality Check 和 Final Review。

Final Review 也默认每轮逐一审核 20 个连续 `review_required` unit；最后一轮处理全部剩余可记录 unit。完成一轮后，默认用以下命令记录本轮 evidence：

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

`--limit` 默认 20，`--start-index` 是普通 record 可处理列表的 1-based 起点，`--issue` 可以重复。命令按 Candidate Snapshot 自动使用每个 unit 的 `selected_batch_id`，即使一轮跨越多个 retranslation batch，也不要求调用者人工拆分或判断 batch。它对整轮先完成单 unit review 记录所用的 schema、identity、attempt 与 hash preflight，并与 snapshot evidence 对齐；全部通过后才写文件。任一 unit 失败时本轮不产生部分 review evidence，已有 review 不覆盖。相同 rating/decision/summary 等参数必须真实适用于本轮每个 unit；若审核结论不同，应按适用的连续范围分别记录。rubric 仅过期而 identity 未变时，实际复审后必须使用 Quality Review 规范中的显式 `review supersede`；这不重新翻译 candidate，也不新建 batch。

Review evidence 与 promotion gate 的完整规则见 [Translation Quality Review 规范](TRANSLATION_QUALITY_REVIEW.md)。`retranslation review record-batch` 与保留的单 unit `retranslation review record` 都只记录已完成的 Final Review，不执行审核，也不改变 promotion gate。

Promotion 默认 dry-run：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation promote --locale <locale>
```

只有用户明确要求应用时才使用 `--apply`。Promotion 会验证最新 batch 的 manifest/source/input、glossary 重建结果、retry provenance、candidate、validation 和 approved Final Review evidence；最新结果失败时不得回退到旧 batch。

## 8. 阶段边界

Codex 完成首次翻译或 retry 译文生成后，不自动继续执行 process、Quality Check、Final Review 或 promote，除非用户明确要求继续。UI catalog 翻译是独立流程，不属于本手册。
