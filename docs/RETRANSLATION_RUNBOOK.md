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

自动 export 默认且最多选择 30 个 TranslationUnit，始终先用 `--unit-kind page` 按正式 Page 顺序逐 30 个导出，再用 `--unit-kind example` 按 eligible Example 顺序逐 30 个导出；不得混合两种 kind，也不做均衡分片。当前 103 Page + 19 Example 自然形成 `30 + 30 + 30 + 13` 与 `19` 五批。

未传 `--batch-id` 时，新 batch 自动命名为 `codex-<locale>-NNN`。自动编号同时扫描保留的 `chatgpt-<locale>-NNN` 与 `codex-<locale>-NNN` 历史目录，取实际最大序号后递增；历史 batch 名称、manifest 和 evidence 均保持不变。

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

`retranslation export`、`retranslation process`、`quality-check scope` 与 `retranslation review scope` 默认输出适合复制的人类摘要；成功 Unit、reusable Unit 和 carry-forward Unit 不逐条展开，失败 Unit 保留原因与 evidence path，scope 最多显示本轮 30 个 pending Unit。已有机器调用方应显式传 `--json` 获取完整稳定 JSON；JSON 写 stdout，错误与诊断写 stderr。

### 提交前 whitespace 检查与 EOF 契约

`git diff --check` 是所有正式提交前的必做检查，不得带 whitespace error 提交。Retranslation 文本 artifact 的统一 EOF 契约是：文件以**恰好一个 LF**结束，不得在 EOF 保留额外空行。不为 `data/retranslation-runs/` 设置 Git whitespace 豁免。

新导出的 batch 在 manifest 中声明 `artifact_eof: single_lf`，并按以下边界执行：

- exporter 在写入 Page `.article` 和 Example `.txt` input 前规范化 EOF；`source_sha256` 仍绑定原始 TranslationUnit source 字节，`input_sha256` 绑定规范化后的 protected input；
- Codex 生成的 `raw-responses/*` 与 `retries/*/attempt-NNN.*` 必须以恰好一个 LF 结束；
- `retranslation process` 和 `retranslation retry` 在 restore 前检查对应 raw artifact；EOF 不合规时直接失败，不自动改写 raw response，也不产生部分 candidate 或 validation evidence；
- restore 成功后，process/retry 在 validation 与写入前将 Page `.article` 和 Example `.go` candidate 规范化为恰好一个结尾 LF。

历史 manifest 没有 `artifact_eof` 字段时，流程仍按历史字节进行兼容验证；不得为追加新字段或消除历史 warning 而改写已提交的 batch artifact。该兼容边界不是新 batch 的 whitespace 例外；新 batch 必须满足上述 EOF 契约并通过 `git diff --check`。

## 4. Retry：只处理 restore/validation failure

Retry 只用于：

- `restore_failed`；
- `validation_failed`。

它不用于翻译质量修改。Codex 读取失败 evidence 和原正式输入，生成下一份连续编号的原始译文：

```text
retries/<flattened-unit-id>/attempt-NNN.article
retries/<flattened-unit-id>/attempt-NNN.txt
```

`raw-responses/<unit>.*` 是正式 attempt 1，其初始 validation evidence 也记录 `attempt: 1`。因此首次 retry 必须写入 `attempt-002.*`，不得创建 `attempt-001.*`；之后从当前 validation 的 attempt 依次加一，且不得覆盖已有 attempt。`attempt-001-validation.json` 是 retry 命令归档的初始 validation evidence，不是需要 Codex 生成的 retry raw response。

文件已存在后再执行：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation retry \
  --locale <locale> \
  --batch-id <batch-id> \
  --unit-id <unit-id>
```

`retranslation retry` 本身不调用模型、不生成或改写译文；它只处理已经存在的 retry raw response，归档前一份 validation，并更新目标 unit 的 candidate、validation 与 batch 汇总。

## 5. Candidate Snapshot

完整 locale workflow 的所有 TranslationUnit 都有通过 automatic validation 的最终 candidate 后，生成本轮审核的 Candidate Snapshot：

```bash
go run -mod=readonly ./cmd/tour-i18n quality-check snapshot \
  --locale <locale> \
  --snapshot-id <snapshot-id>
```

Snapshot 命令本身只创建：

```text
data/quality-check-snapshots/<locale>/<snapshot-id>/manifest.json
```

Snapshot 按 Catalog 的 Page 顺序、再按 eligible Example inventory 顺序冻结完整 workflow。当前基线是 103 Page + 19 eligible Example = 122 TranslationUnit。每个 unit 先按 batch number 选择最新的一份结果，再要求它匹配当前 source revision 且状态为 `passed`；最新结果失败或 identity 不匹配时都禁止回退旧 batch。Snapshot 同时校验 manifest/source/input/candidate/validation identity、相关 SHA-256、restore 绑定和 retry 最终 attempt。

Manifest 只引用仓库中已有的 glossary、source、candidate 和 validation 文件，不复制这些文件，不创建 `_content`、ZIP 或 review artifact。该命令不修改 `locales/<locale>/status.tsv`，不执行 Quality Check、Final Review 或 promotion。后续 `quality-check record` 可以在同一 Snapshot 目录新增独立的 `quality-check-results.json`，但不改写 manifest。

Snapshot 后的两个 scope 必须分开执行。ChatGPT Quality Check 使用 `quality-check scope --locale <locale> --snapshot-id <snapshot-id>`；revision 后另加 `--previous-snapshot-id <previous-snapshot-id>`，只 carry-forward 上一轮 A 且 source/candidate/validation/attempt identity 完全相同的 Unit。Final Review 使用 `retranslation review scope --locale <locale> --snapshot-id <snapshot-id>`，只复用有效 A + approved Final Review evidence。两种 scope 都基于完整 Snapshot，但 evidence 与 pending 列表互不混用。glossary snapshot mismatch 是 scope 的整体 blocker，不是 unit pending。

## 6. Quality Check 与 revision batch

ChatGPT Quality Check 必须以同一份 Candidate Snapshot manifest 为审核范围，不得自行从各 batch 中重新挑选 candidate。首次 locale 没有历史 QC result 时，`quality-check scope` 必须返回全部 Unit pending。完成实际审核后用 `quality-check record` 或 `quality-check record-batch` 写入该 Snapshot 的 `quality-check-results.json`。该轻量结果只服务于多轮 revision carry-forward，不是 Final Review evidence，也不参与 promotion。当前严格生产策略只接受 A：

- A：通过 Quality Check；
- B、C、D：未通过质量 gate，必须进入 revision batch。

Quality Check 每轮最多审核 `quality-check scope` 中 30 个 pending Unit，Page 与 Example 分开处理，但必须逐 TranslationUnit 给出判断，并在直接结果与 carry-forward 合计后覆盖同一 full Snapshot 的全部 Unit。分片不是抽样，不改变 `A = 全部 TranslationUnit` 才能进入 Final Review 的条件。revision 后按实际 pending scope 审核，剩几个就审核几个。

Quality Check 的质量修改不得使用 retry。Revision 流程为：

```text
创建新的 revision batch
→ re-export 对应 TranslationUnit
→ Codex 重新读取完整正式输入并重新翻译
→ process
→ automatic validation
→ 生成新的完整 locale Candidate Snapshot
→ `quality-check scope --previous-snapshot-id ...`
→ 只对 identity 变化、无有效 A 结果或旧结果非 A 的 Unit 执行 ChatGPT Quality Check
→ `quality-check record` 记录新结果
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

Quality Check carry-forward 不缩小首次 Final Review。首次 locale 没有正式 Final Review evidence 时，`retranslation review scope` 仍必须返回全部 TranslationUnit 为 `missing_review`。只有后续已有 identity 完全匹配的 Final Review A + approved evidence 时，当前 Final Review incremental scope 才能复用它。

`retranslation review scope` 中 `required_action=review_required` 的 Unit 决定实际需要 Final Review 的工作集。`record-batch` 始终按 Candidate Snapshot stable index 写入固定连续范围：`--start-index` 从该 stable index 起算，`--limit` 选择最多 30 个连续 Snapshot Unit。Page 与 Example 分开审核。首次 full Final Review 可按 Page `1-30`、`31-60`、`61-90`、`91-103`，再按 Example `104-122` 分片。完成一轮后，用以下命令记录该固定范围的 evidence：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation review record-batch \
  --locale <locale> \
  --snapshot-id <snapshot-id> \
  --start-index <Candidate-Snapshot-stable-index> \
  --rating A \
  --decision approved \
  --summary <summary> \
  --reviewer <reviewer> \
  --rubric translation-quality/v1
```

`--start-index N` 始终表示 Candidate Snapshot manifest 中固定的 `index=N`，不是当前 pending/processable 列表中的位置。`--limit M` 表示从 stable index N 开始连续最多 M 个 Snapshot Unit；仅当该固定范围越过 Snapshot 尾部时才截断，绝不会因为 pending gap 自动截断或跳过 Unit。incremental Final Review 的 pending Unit 可以稀疏，例如 index 17、37、94；不得以 `--start-index 17 --limit 30` 跨过其中的 reusable Unit。应将 `review_required` Unit 按 stable index 划分为连续、同一种 Unit kind、且范围内全部可普通 record 的 range，每个 range 最多 30 个；在 reusable、supersede、revision 或 Page/Example 边界前结束。稀疏 pending 可使用多个短 range，必要时使用 `--limit 1`。已有 evidence 不会被静默跳过，范围也不会向后漂移：请求范围内任何 Unit 已有有效 review、需要 supersede/revision，或因其他状态不可由普通 record 写入时，命令都会在写文件前失败，并报告具体 Snapshot index 与 `unit_id`。这是预期的安全行为，不要求程序自动截断或跳过。`--issue` 可以重复。命令按 Candidate Snapshot 自动使用每个 unit 的 `selected_batch_id`，即使一轮跨越多个 retranslation batch，也不要求调用者人工拆分或判断 batch。它对固定范围内的全部 Unit 先完成单 unit review schema、identity、attempt 与 hash preflight，并与 Snapshot evidence 对齐；全部通过后才写 evidence。任一 Unit 失败时本轮不产生部分 evidence，已有 review 不覆盖。相同 rating/decision/summary 等参数必须真实适用于本轮每个 Unit；若审核结论不同，应按适用的 stable index 连续范围分别记录。rubric 仅过期而 identity 未变时，实际复审后必须使用 Quality Review 规范中的显式 `review supersede`；这不重新翻译 candidate，也不新建 batch。

Review evidence 与 promotion gate 的完整规则见 [Translation Quality Review 规范](TRANSLATION_QUALITY_REVIEW.md)。`retranslation review record-batch` 与保留的单 unit `retranslation review record` 都只记录已完成的 Final Review，不执行审核，也不改变 promotion gate。

Promotion 默认 dry-run：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation promote --locale <locale>
```

只有用户明确要求应用时才使用 `--apply`。Promotion 会验证最新 batch 的 manifest/source/input、glossary 重建结果、retry provenance、candidate、validation 和 approved Final Review evidence；最新结果失败时不得回退到旧 batch。

## 8. 阶段边界

Codex 完成首次翻译或 retry 译文生成后，不自动继续执行 process、Quality Check、Final Review 或 promote，除非用户明确要求继续。UI catalog 翻译是独立流程，不属于本手册。
