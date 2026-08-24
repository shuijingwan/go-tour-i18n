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

## 6. Quality Check 与 revision batch

ChatGPT Quality Check 必须以同一份 Candidate Snapshot manifest 为审核范围，不得自行从各 batch 中重新挑选 candidate。当前严格生产策略只接受 A：

- A：通过 Quality Check；
- B、C、D：未通过质量 gate，必须进入 revision batch。

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

Review evidence 与 promotion gate 的完整规则见 [Translation Quality Review 规范](TRANSLATION_QUALITY_REVIEW.md)。`retranslation review record` 只记录已完成的 Final Review，不执行审核。

Promotion 默认 dry-run：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation promote --locale <locale>
```

只有用户明确要求应用时才使用 `--apply`。Promotion 会验证最新 batch 的 manifest/source/input、glossary 重建结果、retry provenance、candidate、validation 和 approved Final Review evidence；最新结果失败时不得回退到旧 batch。

## 8. 阶段边界

Codex 完成首次翻译或 retry 译文生成后，不自动继续执行 process、Quality Check、Final Review 或 promote，除非用户明确要求继续。UI catalog 翻译是独立流程，不属于本手册。
