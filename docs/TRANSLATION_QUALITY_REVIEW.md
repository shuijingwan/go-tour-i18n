# Translation Quality Review 规范

Translation Quality Review 是所有语言翻译工作流在 promotion 前的正式必经步骤。这套规范面向长期维护和多语言复用，不绑定 `zh-CN`、ChatGPT、Page、Example 或任何特定模型。

本规范只审核 TranslationUnit candidate。公共 UI、首页、`/tour/list`、导航、语言选择器、runtime message、article metadata、SEO 与桌面/移动端组合页面由独立的 [Locale Surface Review](LOCALE_SURFACE_REVIEW.md) 审核。Surface Review 是 promotion 后、publish / production 前的 locale release gate，不生成本规范的 QC/finalization evidence，也不改变全 A、逐 TranslationUnit规则。

前置 [Glossary Review](GLOSSARY_REVIEW.md) 只审核 glossary 的术语决策本身，同样不替代本规范。Glossary 任意字节变化后，旧 Snapshot / QC carry-forward 仍按现有 glossary identity 规则 stale；新的 Glossary Review PASS 不会恢复旧 A。

若 glossary 变化只使部分 TranslationUnit 的正式 protected input 发生字节变化，先重新完成完整 Glossary Review，再按 [Retranslation 执行手册](RETRANSLATION_RUNBOOK.md#glossary-induced-protected-input-stale-recovery) 只恢复这些真实 stale Unit。protected input identity 未变化的可信 candidate 不重译。恢复只解决 input/candidate identity；新的 full Snapshot 仍因 glossary identity 改变而要求完整 QC，旧 A 一律不 carry-forward。

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
→ machine finalization
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

所有 path 都是对仓库已有文件的引用。Snapshot 不复制 source、candidate、validation 或 `_content`，不生成 ZIP，不修改 `locales/<locale>/status.tsv`，不 promotion，也不生成 Quality Check 结果或 Final Review evidence。Snapshot 创建时目录只包含 `manifest.json`；完成实际 Quality Check 后，`quality-check record` 才会在同一目录新增独立的 `quality-check-results.json`，但不会改写 immutable Snapshot manifest。

## Full Snapshot 与两种独立 scope

Candidate Snapshot 始终是 **full Snapshot**：它冻结完整 locale workflow 的全部 TranslationUnit。当前 `zh-CN` 与 `ja-JP` 都是 103 Page + 19 eligible Example = 122 Unit。不得为了维护批次创建只含变化 Unit 的 Snapshot。

Quality Check incremental scope 是新流程唯一的语言审核 scope。下文保留的 Final Review incremental scope 仅用于读取和验证历史 evidence，不能作为新 promotion authority。

### Quality Check incremental scope 与 carry-forward

首次 Quality Check 使用：

```bash
go run -mod=readonly ./cmd/tour-i18n quality-check scope \
  --locale <locale> \
  --snapshot-id <snapshot-id>
```

没有历史 Quality Check 结果时，`pending_count` 必须等于 full Snapshot 的 `unit_count`。实际逐 TranslationUnit 完成 Quality Check 后，用独立命名空间记录结果：

独立 Reviewer 的正式输入优先按本轮 stable working set 导出：

```bash
go run -mod=readonly ./cmd/tour-i18n quality-check reviewer-bundle \
  --locale <locale> --snapshot-id <snapshot-id> \
  [--previous-snapshot-id <previous-snapshot-id>] \
  --start-index <stable-index> --limit <1..60> \
  --output /tmp/<locale>-<snapshot-id>-qc-<stable-index>.zip

go run -mod=readonly ./cmd/tour-i18n quality-check reviewer-bundle-check \
  --bundle /tmp/<locale>-<snapshot-id>-qc-<stable-index>.zip
```

ZIP 绑定 full scope/carry-forward/pending context，并完整包含本 working set 的 stable index、Unit id/kind、source/current target、完整 glossary、Snapshot、relevant manifest/input、validation、rubric/authority 与 exact hashes。export 会在 Page/Example 边界停止，`--limit` 最大 60；每份 ZIP 仍必须由一次新的用户请求/model invocation 独立审核，不能把多份 ZIP 放入一次 response 绕过限制。Reviewer 只给 A/B/C/D 与 findings；`reviewer-bundle-check` 在记录前要求旧 ZIP 与 current working tree 逐字节一致。bundle 不生成结果、不改变 Snapshot，也不替代 validation；以下既有 record 入口仍是唯一 QC state machine：

```bash
go run -mod=readonly ./cmd/tour-i18n quality-check record \
  --locale <locale> --snapshot-id <snapshot-id> \
  --unit-id <unit-id> --rating <A|B|C|D> \
  [--finding '<specific finding>']

go run -mod=readonly ./cmd/tour-i18n quality-check record-batch \
  --locale <locale> --snapshot-id <snapshot-id> \
  --start-index <1-based-snapshot-index> --limit <count> \
  --rating <A|B|C|D> [--finding '<finding shared by this group>']
```

`record` 的 `--unit-id` 可以重复，用于一次原子记录多个同 rating、同 finding Unit；同一次 Reviewer working set 的异构结论使用下面的 `--judgments-file` 模式，不必逐 Unit 重跑 recording。`record-batch` 默认且最大 `limit=60`，按 immutable Snapshot 的稳定 index 记录连续范围。A 可以没有 finding；B/C/D 必须提供非空 finding。两个旧入口的正常语义不变，均拒绝覆盖已有 Unit 结果。

#### 同一 Reviewer working set 的异构结论记录

```bash
go run -mod=readonly ./cmd/tour-i18n quality-check record \
  --locale <locale> --snapshot-id <snapshot-id> \
  --content-package <registered-package> \
  --reviewer-session <independent-reviewer-session> \
  [--previous-snapshot-id <previous-snapshot-id> | --full-rereview] \
  --judgments-file /tmp/<working-set>-judgments.json
```

JSON 文件必须是明确的数组，例如：

```json
[
  {"unit_id": "<unit-a>", "rating": "A"},
  {"unit_id": "<unit-b>", "rating": "B", "finding": "该 Unit 的具体问题。"},
  {"unit_id": "<unit-c>", "rating": "C", "finding": "另一个 Unit 的具体问题。"}
]
```

这是 command transport/input，不是新 lifecycle authority、receipt 或持久化 schema；不会复制输入文件到正式目录。由 caller 明确提供同一次独立 Reviewer response 的结论，不解析或猜测自然语言 Markdown。文件路径可为当前工作目录下的相对路径或绝对路径（例如 `/tmp`）。严格拒绝 unknown fields（含客户端 `index`）、重复 JSON key、trailing JSON、空数组/null、重复/空 Unit ID、非 A/B/C/D、B/C/D 空 finding。finding 仅按既有 `record` 合同 trim 首尾空白，内部文字与换行保留；stable index 从 Snapshot 取得。

`--judgments-file` 只可出现一次，且与 `--unit-id`、`--rating`、`--finding` 互斥，显式传空值也不例外。bulk 模式必须提供非空 `--reviewer-session`，校验独立角色并与已开始 results 的 session exact-match。仍限同 kind、最多 60 Unit；不合并多次 Reviewer invocation，也不豁免 initial working-set partition。registered package 沿用共享 `--content-package` catalog selection 与 Snapshot binding；native Tour 沿用原 scope。predecessor/full-rereview 的 first-write 与持久化 lineage 规则完全不变。

一次调用只建立一次共享 Snapshot/currentness/lineage recording context，逐 Unit 仍执行完整 repository identity/evidence 校验，相关 reconciliation receipt context 仅在本 invocation 内复用。所有 judgment 全部通过后才 merge/sort，复用现有 writer 经临时文件和单次 rename 发布 `quality-check-results.json`；任何 judgment、session、lineage、scope 或 identity 校验失败均不写部分结果，也不覆盖已有 Unit。最终 schema/bytes 与同一 Snapshot 上逐 Unit 使用旧 `record` 得到的结果一致。没有新缓存、恢复协议或并发写入合同；维护者仍串行记录，写入后才出现的 stdout/I/O failure 不能当作“未写入”，须先只读核对 canonical results。

stdout 继续使用 `QualityCheckRecordResult` JSON：`recorded_count` 是本次新增数，`result_count` 是总数，`unit_ids` 为请求顺序，`path` 为 canonical results 路径；bulk 模式的共享 `rating` 为空字符串，各 Unit 的真实 rating/finding 以 results 文件为准。此 CLI input/currentness contract 已修改，旧 first-real-use verification 失效；真实使用前由 Maintainer Local 按 [Formal Command Contract v3](WORKFLOW_HANDOFF.md#formal-command-contract-v3) 重新核对 current docs → implementation → direct CLI tests。该能力不改变 revision、finalization、Independent QC/A-only、Surface 或 Production gate。


单次 Reviewer model invocation / response 最多审核 60 TranslationUnits；Page / Example 分开。60 是实际审核质量边界，不得通过在同一次 response 内串联多个 `<=60` working set 绕过。下一组必须由新的用户请求触发新的 model invocation；每组仍须逐 TranslationUnit 审核，不得抽样。当前首次 122-Unit locale 的 full QC 推荐依次审核 Page stable index `1-60`、Page stable index `61-103`、Example stable index `104-122`，最多三次独立 Reviewer 请求。revision re-QC 按实际 pending scope 分组，每次仍最多 60，且 Page / Example 分开。

新增 locale 的效率优先默认调度以[新增 Locale 执行手册](NEW_LOCALE_RUNBOOK.md#新增-locale-默认调度路径效率优先)为唯一高层入口：不得提前导出后续两组 Reviewer ZIP；每组完成 `reviewer-bundle-check`、独立审核与 record 后才导出下一组，并默认先完成三组首审、汇总 B/C/D，再集中 revision。record 会改变同一 Snapshot 的 current scope，使先前预导出的后续 ZIP stale；中途 revision 又必须切换到新的 full Snapshot，旧 Snapshot 的预导出 ZIP 不再是当前审核目标。CLI 仍允许已有执行在中途 revision；恢复时不倒退或改写历史结果，而是从新的 full Snapshot、`--previous-snapshot-id` 与 current pending scope 继续，旧 A 只按下述 exact identity 规则 carry-forward。

Learn/Docs 的固定 working set 只约束 initial Generation / initial QC；完整初审后，正式 B/C/D 与 Surface reopen revision 可跨 initial group。successor Snapshot 的 re-QC 采用上述 current pending、stable index、同 kind、最多60规则，不继承 initial partition。Reviewer Bundle 与 record 根据解析后的 predecessor lineage 使用相同阶段约束；多组继续 JIT。具体共享调度见[共享生命周期](SHARED_TRANSLATION_UNIT_LIFECYCLE.md#固定调度与独立审核)。

旧 evidence 中没有 finding 的 B/C/D 继续兼容读取，但在 revision export 前必须通过 CLI 补录，禁止人工编辑 JSON：

```bash
go run -mod=readonly ./cmd/tour-i18n quality-check backfill-finding \
  --locale <locale> --snapshot-id <snapshot-id> \
  --unit-id <unit-id> --finding '<specific finding>'
```

该命令验证 Snapshot manifest、stable index、unit id、candidate/validation repository identity 和已有 result，只允许给已有 B/C/D 的空 finding 补值，不允许改变 rating，也不静默覆盖已有 finding。

轻量结果位于：

```text
data/quality-check-snapshots/<locale>/<snapshot-id>/quality-check-results.json
```

它以 `evidence_type=quality_check_results`、Snapshot manifest SHA-256、locale、snapshot、rubric、stable index、unit id、rating 和 finding 绑定实际 Quality Check 结论。它不直接作为 promotion authority；`quality-check finalize` 读取完整 lineage 后生成唯一 authority。

revision 后仍创建新的 full Snapshot。新一轮 scope 显式指定上一轮：

```bash
go run -mod=readonly ./cmd/tour-i18n quality-check scope \
  --locale <locale> \
  --snapshot-id <new-snapshot-id> \
  --previous-snapshot-id <previous-snapshot-id>
```

只有上一轮有效结果为 A，且 rubric、source、selected batch、candidate path/SHA-256、validation path/SHA-256 和最终 attempt 与新 Snapshot 完全相同的 Unit 才能 **QC carry-forward**。B/C/D 不得 carry-forward；没有结果、identity 变化、glossary 变化或 rubric 变化的 Unit 必须重新进入 Quality Check。identity 未变化但上一轮为 B/C/D 时，scope 输出 `non_a_quality_check/revision_required`；revision 后 identity 已变化时输出 `identity_changed/quality_check_required`。

`quality-check-results.json` 的 lineage identity 在该文件第一次成功写入时固定。revision Snapshot 的第一次 `quality-check record` 或 `record-batch` 必须使用与 prospective scope 相同的 `--previous-snapshot-id`；后续 record 可以省略该参数并沿用已经持久化的 predecessor。results 文件已经存在时，scope 或 record 显式传入的 `--previous-snapshot-id` 只能与 persisted `previous_snapshot_id` 做 exact-match 一致性确认，不能把 empty lineage 补成 revision lineage，也不能改为另一个 predecessor；lineage 错误应回到正确的 Snapshot/QC recording workflow evidence 处理，禁止人工编辑 JSON。后续 scope 可以从结果文件继续解析多轮 carry-forward。`a_count == unit_count`、B/C/D 均为 0 且 `pending_count == 0` 时，`ready_for_finalization=true`，才可以执行 finalization；JSON 同时保留 deprecated `ready_for_final_review` alias 供旧消费者读取。

### 已记录 Quality Check judgment 的 source-mismatch correction

独立 Reviewer 确认自己已记录的 B/C/D finding 与 exact canonical source 不符、明确撤回该 finding 并纠正为 A 时，使用 `quality-check correct`。这是 admission C 的 bounded recorded-judgment correction；只支持 `non-A → A` 与 `finding_disposition=withdrawn-source-mismatch`，不支持其它 rating 调整、A reopen、译文生成或新的审核历史平台。不得手改原 results/candidate/protected token，也不得用 legacy `retranslation review supersede` 替代。

维护者将原独立 Reviewer 的真实 correction response 明确转录为 JSON，不得让机器自行判断或补造 Reviewer 结论。例如：

```json
{
  "locale": "<locale>",
  "snapshot_id": "<snapshot-id>",
  "original_results": {"path": "data/quality-check-snapshots/<locale>/<snapshot-id>/quality-check-results.json", "sha256": "<exact-original-file-sha256>"},
  "reviewer_session": "<original-independent-reviewer-session>",
  "unit_id": "<exact-unit-id>",
  "index": 7,
  "original_rating": "B",
  "original_finding": "<exact-original-finding>",
  "corrected_rating": "A",
  "finding_disposition": "withdrawn-source-mismatch",
  "reason": "<Reviewer 对 exact source occurrence 的纠错理由>",
  "correction_required": true,
  "replacement_text_generated": false,
  "repository_mutation": "none"
}
```

```bash
go run -mod=readonly ./cmd/tour-i18n quality-check correct \
  --content-package <registered-package> \
  --locale <locale> --snapshot-id <snapshot-id> \
  --reviewer-session <original-independent-reviewer-session> \
  --correction-file /tmp/reviewer-correction.json \
  --correction-sha256 <exact-input-file-sha256>
```

| Contract | 语义 |
| --- | --- |
| COMMAND / OPERATION | `quality-check correct` / immutable recorded-judgment correction；调用本身即 record，无 `--apply` 或 dry-run |
| INPUT_KIND / INPUT_PATH_CONTRACT | 一个独立 Reviewer correction JSON；`--correction-file` 接受绝对路径或相对当前工作目录的路径；内含 `original_results.path` 必须 exact canonical repository-relative path，绝对路径即使位于 repo 内也拒绝 |
| REQUIRED_FLAGS / FORBIDDEN_FLAGS | locale、snapshot-id、reviewer-session、correction-file、correction-sha256 各恰好一次且非空；registered lifecycle 显式携带原 package selector，native/default Tour 可不带；拒绝 positional、unknown flags、rating/unit-id 覆盖参数及重复 flags |
| STDOUT | `QualityCheckCorrectionRecordResult` JSON：`locale,snapshot_id,unit_id,index,path,sha256`；path/SHA 指向新 artifact，不是原 results 或输入 JSON |
| MUTATION_EXPECTATION | known：仅新增 `data/quality-check-snapshots/<locale>/<snapshot-id>/judgment-corrections/<sha256(unit-id UTF-8 bytes)>.json` 一个 immutable artifact；不改原 results、Snapshot、candidate/raw/validation、batch、finalization 或 status |
| IDEMPOTENCE | no-replace；同 Unit 重复（包括 exact replay）及 conflicting correction 一律失败，不覆盖、不自动合并；每个 Unit 最多一份 correction |
| CURRENTNESS_CHECK | Snapshot/current glossary/package scope、latest processed selected candidate identity、磁盘 source、candidate/validation evidence；仅 unfinalized Snapshot 可 record。输入 SHA、原 results path/SHA、原 rating/finding、Snapshot stable index/full Unit identity 必须 exact；Reviewer 必须与原 results session 相同且通过现有独立角色检查。legacy 无明确 Reviewer identity 的 results 不可据此获得新 A |
| PRIOR_OR_CAS | prior 是 JSON 内显式 original results path/SHA/rating/finding 与 input SHA；发布前再核对 results/manifest bytes；目标必须 absent，完整临时文件经 atomic link no-replace 发布。维护者串行记录；没有跨文件锁或针对外部并发编辑的事务/CAS 保证 |
| SUCCESS_STOP | 新 artifact 与 stdout path/SHA 一致后，回既有 readonly scope/preflight 核对 exact pending；不自动生成 replacement、revision、Snapshot 或 finalization |
| FAILURE_STOP | prior/currentness/role/JSON/duplicate/tamper 失败不得继续；发布前失败不写 correction。stdout/I/O failure 或外部并发编辑时先只读核对目标 path/SHA，不假设未写、不盲目 replay |
| RECOVERY / REPLAY | 保留原结果和原 Reviewer correction bytes；禁止修补或删除 artifact、重放历史 invocation、批量迁移旧 QC。缺真正独立 correction evidence 时停止 |

artifact schema 为 additive `go-learning/quality-check-judgment-correction/v1`，保留 exact Reviewer response bytes（JSON 中 base64）、其 SHA、原 results 引用、Snapshot manifest SHA、完整 frozen Unit、原 judgment、纠正后 A/empty finding、reason/disposition 和 self identity。严格拒绝 unknown/duplicate JSON keys、trailing JSON、缺失显式 false、wrong identity 和非 canonical artifact encoding。correction 不自授 Reviewer authority；原独立 Reviewer identity 与现有角色隔离检查不可由输入声明替代。

共享 effective reader 先验证原 results，再逐份验证 correction，仅替换 exact Unit 的 rating/finding；其它 judgments、lineage、rubric 与 carry identity 不变。没有 correction 时旧 reader 行为不变，不做 implicit migration。首份 correction 冻结原 results bytes，后续 record/backfill 不得再改该文件；其它 Unit 的独立 correction 可继续绑定同一原 SHA。原 results 永久可审计。scope、revision authorization、carry 和 finalization 读取同一 effective judgment；finalization 沿用既有 QC input reference 列表额外 hash-bind correction，historical verifier 检查 exact correction inventory/bytes，不将今日临时 overlay 冒充历史 A。旧 results/finalization schema 不改、无 correction 的旧 evidence bytes 不变。

真实首次使用前，由 Maintainer Local 按 [Formal Command Contract v3](WORKFLOW_HANDOFF.md#formal-command-contract-v3) 重新核对 current docs → parser/dispatcher/reader/publisher → direct CLI tests；工程测试不授权真实 record。

### Legacy Final Review incremental scope

以下命令只读取正式 Final Review evidence：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation review scope \
  --locale <locale> \
  --snapshot-id <snapshot-id>
```

- **reusable reviewed unit**：存在有效且完全匹配当前 Snapshot identity 的 Final Review evidence，且 `rating=A`、`decision=approved`、rubric 为当前 `translation-quality/v1`。匹配 batch、locale、unit、source SHA-256、candidate/validation path 与 SHA-256，以及最终 attempt。Quality Check result 不参与该判断。
- **pending review unit**：scope 为每个 pending Unit 输出机器可读的 `reason` 和 `required_action`。`missing_review`、`identity_changed` 与 `rubric_mismatch` 的 action 是 `review_required`；`non_a_review`（B/C/D）是 `revision_required`；`rejected_review` 保守地也是 `revision_required`。前两类需要实际重新审核，后两类不得借重新记录 review 绕过 revision batch。

首次 locale 没有可复用 Final Review evidence，因此即使 Quality Check 已通过或发生过 QC carry-forward，Final Review scope 仍等于 full Snapshot，是 **full Final Review**。后续 locale 已有有效 Final Review A + approved evidence 时，full Snapshot 仍保持完整；只审核 identity 已变化或 evidence 失效的 pending Unit，是 **incremental Final Review**。

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
- 修订后必须重新 automatic validation、Candidate Snapshot、Quality Check，并在全 A 后重新 finalization。

### D

- 不可接受；
- 存在严重误译、漏译、技术含义错误或明显不符合翻译要求；
- 必须重新翻译或大幅修订，再重新 automatic validation、Quality Check 和 finalization。

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

## Quality Check 与 machine finalization

Quality Check 用于发现翻译质量问题，可以在 candidate 形成后的修订过程中多轮执行。它由 ChatGPT 逐 Unit 执行，直接结果与有效 carry-forward 必须覆盖同一份 full Candidate Snapshot。全 A 后执行 `quality-check finalize`；该命令不作语言判断，而是机械生成独立、versioned 的 finalization evidence。

Quality Check 的实际审核工作集由 `pending_quality_check_units` 决定，保持 Snapshot 稳定 index、每次最多 60 Unit、Page/Example 分开，并遵守上述单次 model invocation / response 质量边界。下文 Final Review commands 和 schema 仅为历史 evidence compatibility 保留；新流程不运行它们。

当前严格生产 gate 为：A 通过；B、C、D 均不通过并进入新的 revision batch。只有完整语言达到 `A = 全部 TranslationUnit，B = 0，C = 0，D = 0`，才能 finalization。

finalization 绑定 Snapshot manifest、glossary、rubric、QC-result lineage、完整解析后的 A 结果及每个 Unit 的 source/candidate/validation/attempt identity。新 promotion 必须显式指定并验证该 finalization，且仍独立重验 manifest/input、glossary、restore、validator 与 retry provenance。

### finalize 的历史 representation successor 合同

`quality-check finalize --content-package <package> --locale <locale> --snapshot-id <current> --previous-snapshot-id <previous>` 中，历史 predecessor 的 representation reconciliation 按该事件自己的冻结 descriptor world 验证。例如 A→B 的 immutable receipt 必须仍精确证明 A→B；今天的 canonical representation 为 C 时，还必须由 source provider 独立验证 B→C 的完整正式 successor lineage、当前 pointer 和必要 dependency re-attestation。不能用相同 path/raw SHA 代替 descriptor、protected input 或 successor authority，也不能把 receipt 中的 B 改解成 C。缺链、歧义、循环、错误 package/source/identity 或 stale authority 均拒绝。

这只保持历史 A 的可审计性，不授予 C carry/A。current Snapshot 的完整 active validation、currentness 和独立 A-only QC 必须另外通过。

| Contract | 当前语义 |
| --- | --- |
| COMMAND / OPERATION | `quality-check finalize` / `finalize` |
| INPUT_KIND | 完整 QC lineage，含显式 package scope 和 persisted historical predecessor chain |
| INPUT_PATH_CONTRACT | 无文件输入；仅 locale/package/Snapshot identifiers |
| MUTATION_EXPECTATION | 成功 publication 为 `known`；publication 前 validation failure 为 `none` |
| IDEMPOTENCE | write-once/no-overwrite；已存在 finalization 拒绝，不 replay/覆盖 |
| CURRENTNESS_CHECK | current Snapshot + frozen predecessor event worlds + exact canonical successor chain；dependency drift 必须有正式 current re-attestation |
| PRIOR_OR_CAS | 显式 persisted predecessor 及其 exact finalization SHA；目标不存在 |
| SUCCESS_STOP | immutable finalization publication 后停止 |
| FAILURE_STOP / RECOVERY | 保留已有 evidence；无 fallback、predecessor rewrite 或 receipt mutation，不盲目重跑；publication 状态未知先诊断 |

此 historical-currentness contract 修改后，首次真实 finalize 前须重新读本合同、current implementation 和 direct CLI/contract tests，完成 [first-real-use gate](WORKFLOW_HANDOFF.md#formal-command-contract-v3)。engineering proof 只在隔离副本执行，不记录真实 Snapshot finalization。

新 revision export 只接受 Quality Check B/C/D feedback。Final Review B/C/D feedback 的 parser 与历史 manifest fields 仅为历史兼容保留，不得用于新正式流程。

## 必须逐 TranslationUnit 审核

每一个最终准备进入 promotion 的 TranslationUnit candidate，都必须有一次与其字节内容绑定的 ChatGPT Quality Check A；machine finalization 不能替代这次语言审核。不采用以下规则：

- 只审核首批，后续抽样；
- validator 通过即免审；
- 某个模型表现稳定后免审；
- 用批次级结论代替逐 TranslationUnit review。

抽样评审可以用于模型实验、prompt 调优和质量趋势分析，但不能替代 promotion 前的逐 TranslationUnit Review。

### B/C/D finding 完整性与 successor re-QC

Reviewer 必须对完整 current Unit 做 fresh judgment，不能只核对上一轮 examples。一旦决定 B/C/D 并识别某个 **finding category**，在提交最终 finding 前，必须对当前完整 reviewed Unit 做一次该 category 的 **same-class scan**，尽可能一次收全已经观察到的同类实例。这里要求 reasonable complete scan，不要求数学意义的穷举证明；不能明知是重复型 defect 却只列几个代表实例就结束。

finding 必须明确 defect category 与 bounded scope。已观察实例应尽可能完整列出，并用 exact phrase/token adjacency、paragraph/section/table row 或其他稳定局部坐标定位。实例很多时，可以用 **exact predicate / defect pattern + bounded scope** 表达，例如“当前 Page 全文中，按该目标语言规范应分隔、却与紧邻正文不当粘连的 inline Go API identifier”；不能仅以“例如 / e.g. / 等”留下模糊开放边界。scope 可以是当前完整 Page、指定 section、表格列或明确的 teaching-comment 区域，不能跨出正式授权的 Unit/slot 集合。

Reviewer examples 是 finding evidence，**不默认构成 exhaustive revision boundary**。只有 finding 明确标为 **exact-instance-only**，才只授权列出的具体实例；Generation 不得自行把 examples 当作该限制，也不得从含糊 examples 猜出一个全文 scope。category、predicate 或 scope 不清楚时，按[精确修订任务语义](TRANSLATION_TASK_SPEC.md#精确修订任务语义) fail closed，报告 ambiguity，不回写历史 finding。

same-class scan 不得混入无关措辞偏好、一般润色或另一 defect category。fresh judgment 若发现多个独立问题，必须分别说明各自类别、实例/predicate 和修订边界，不把不同类别并成“改好全文”这类无限 scope finding。以下只是可能的重复类别，不是一律判错的语言规则：

- inline code / API / identifier 与相邻目标语言正文不当粘连；合法词缀和 protected identity 仍按 source、目标语言规范及既有结构合同判断。
- 指定范围内同一类中文标点前后的异常空格。
- 同一 bounded section 内某个术语反复出现的同型错误。
- 同一表格列说明中重复的同型语义错误。
- 授权 teaching-comment 区域内同一种局部问题。

可在现有自然语言 finding 中使用以下写法；这是推荐 evidence 组织方式，不新增 machine 字段、schema、CLI 或 receipt：

```text
Finding category:
<precise defect class>

Observed instances:
<尽可能完整的已观察实例及定位，或 exact predicate + bounded scope>

Revision scope:
<同一 current Unit 内的 same-class complete sweep 边界；若仅限具体实例，明确 exact-instance-only>

Out of scope:
<其他问题类别、一般性润色及必须保持的 protected/structure 内容>
```

successor re-QC 仍须对完整 current Unit 做独立 fresh judgment；除检查原 finding 明列实例外，还必须检查 **persisted finding category 在授权 scope 内是否完整关闭**。若仍有同类残留，可以继续判 B/C/D，但 finding 必须明确标为 **same-class residual**，尽可能完整列出剩余实例或给出 exact bounded predicate，不能将残留伪装成全新类别。新发现的独立类别另行表达，不能借 closure 扩大原 revision scope。旧 finding/evidence 不回写、不迁移；上述规则不改变评级定义、独立 Reviewer、A-only 或既有调度与 gate。

## Candidate 变化与重新审核

Review evidence 同时绑定：

- `source_sha256`；
- `candidate_sha256`；
- `validation_sha256`；
- `attempt`。

因此，candidate 字节发生变化、retry 产生新 candidate，或 validation evidence 发生变化时，旧 review 自动失效。新的最终 candidate 必须重新经过：

```text
automatic validation → Candidate Snapshot → ChatGPT Quality Check → machine finalization
```

不得把旧 review evidence 套用到新 candidate 或新的 validation evidence 上。

翻译质量修订不覆盖旧 candidate。需要修订时，必须通过新的 revision batch 产生新的 candidate 和 validation evidence，并重新生成完整 locale Candidate Snapshot；旧 snapshot 仍是历史审核范围，不会自动改指新 candidate，旧 batch 的 review evidence 也不适用于新的 candidate。

## Review evidence schema v1

以下 `review/*.json` 是 **legacy Final Review evidence**，仅用于历史读取/验证，不是新流程的正式 evidence，位于：

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

Final Review 完成一个最多 30-unit、且 Page / Example 分开的执行分片后，使用 Candidate Snapshot 的稳定 index 批量记录该轮 evidence：

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

`--start-index N` 精确表示 Candidate Snapshot manifest 的 stable `index=N`；它不是动态 pending/processable 列表的 1-based 位置。`--limit M` 精确选择从 N 开始连续最多 M 个 Snapshot Unit，只在该固定范围越过 Snapshot 尾部时截断；它不会在 pending gap 自动截断，也不会跳过已有 evidence 后继续选择更靠后的 Unit。首次 full Final Review 可依序记录 Page `1-30`、`31-60`、`61-90`、`91-103`，再记录 Example `104-122`。incremental Final Review 必须先以 scope 的 `review_required` Unit 划分 stable-index 连续、同一种 Unit kind 且全部可普通 record 的 range，每个 range 最多 30 个；遇到 reusable、`supersede`、`revision_required` 或 Page/Example 边界就在 gap 前结束。若 pending 为 17、37、94，则分别使用短 range（必要时 `--limit 1`），不得使用 `--start-index 17 --limit 30`。固定范围内若有 Unit 已有有效 Final Review evidence、需要 `supersede`、必须 revision，或因 identity/state 不可由普通 record 写入，整批会在写文件前失败，错误包含该 Unit 的 Snapshot index 和 `unit_id`；这是预期的安全行为，不要求程序自动截断或跳过。`rubric_mismatch` 必须使用下述显式 supersede；B/C/D 和 rejected review 必须走 revision。命令只记录已经完成的 Final Review，不执行或推导审核；传入的 `rating`、`decision`、`summary`、`reviewer`、`rubric` 和 `issue` 应真实适用于本轮固定范围内的每个 TranslationUnit。

命令从 Snapshot unit 自动读取 `selected_batch_id`，调用者不得也无需人工判断 candidate 属于哪个 batch。它先对 stable index 固定范围内的全部 Unit 完成单 Unit `RecordRetranslationReview` 所用的 schema、identity 和 hash preflight，并核对 Snapshot 中的 source、candidate、validation、attempt 与 path；全部通过后才写 evidence。任一 Unit preflight 失败时本轮不写任何新 review，已有 review 也不会被覆盖。成功输出中的 `indexes=N-E` 就是实际 Candidate Snapshot stable index 范围，`units` 是该范围实际写入数。原有 `retranslation review record --batch-id ... --unit-id ...` 继续保留，用于明确的单 TranslationUnit 记录。

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

`reviewer` 是审核主体标识，不是账号或权限系统。每个 locale 的 Generation provider 可以在正式 generation 开始前选为普通 ChatGPT 或 Codex，均使用 GPT-5.6 Sol + High；独立 ChatGPT GPT-5.6 Sol + High Reviewer session 统一承担 Quality Check。Quality Check 必须来自与该 locale Generation session 独立的 Reviewer session：该 session 不得参与该 locale 的 glossary、UI / metadata、TranslationUnit initial / revision / retry、Course SEO language generation 或任何 replacement generation。Legacy Final Review 的审核主体同样必须独立重新读取 source、candidate 和相关 evidence 后执行审核。

同一个 generation-independent Reviewer session 可以先完成 [Glossary Review](GLOSSARY_REVIEW.md)，再承担 TranslationUnit Quality Check、revision re-QC 与 [Locale Surface Review](LOCALE_SURFACE_REVIEW.md)；三个 gate 仍各自使用自己的范围和 evidence，不要求再建立其他 reviewer session。B/C/D finding 必须回到 Generation session 产生 replacement；维护者 Local terminal 默认执行 revision export / bundle / result landing / process / validation、新 Snapshot / scope、Reviewer bundle current-check 与结果 record / finalize；AI 按[Call-Zero-First](WORKFLOW_HANDOFF.md#call-zero-first-交接合同)提供完整命令并消费合法 terminal evidence，与小批规模或预计耗时无关。只要 Reviewer session 没有参与 replacement generation，它可以继续 re-QC；不得直接修改 candidate、生成 replacement 后批准自己的结果，或因为上一轮已经审核过就自动批准 revision。全 A finalization 后的 promotion 仍由维护者 Local terminal 执行。

审核必须重新判断译文质量，不得用 validation passed、旧 A、Surface Review 结论、reviewer 身份或模型历史表现自动批准。未来可以使用其他模型或人工 reviewer，但都必须遵循同一正式 rubric 和 evidence schema。本项目当前不构建 reviewer 账号系统；完整会话职责见 [ChatGPT 正式语言生成执行规范](CHATGPT_LANGUAGE_GENERATION.md)。

## Issues

`issues` 是简单数组，用于记录需要说明的具体质量问题。即使没有问题，也应保留为空数组。Review evidence 不引入 Web discussion、assignee、workflow queue、comment thread 或 database ID。

## Promotion gate

新 promotion 使用 `retranslation promote --snapshot-id <snapshot-id>`，只接受与当前完整 candidate 集合精确匹配的 `data/quality-check-snapshots/<locale>/<snapshot-id>/finalization.json`。它不直接把 `quality-check-results.json` 当 authority；finalization 解析并绑定完整 QC lineage。

以下任何情况都必须令 `can_apply=false`：

- missing review；
- rejected review；
- invalid review 或 schema mismatch；
- candidate 或 validation hash mismatch；
- batch、locale、unit、source、attempt 或 path identity mismatch。

只有完整 finalization 的所有 Unit 为有效 A 才允许 promotion。promotion preflight 始终检查完整 locale workflow，并重新验证底层 evidence。

## 历史质量记录

历史文档中记录过的 A/B/C/D 统计不自动视为 schema v1 的结构化 review evidence。正式 Review gate 启用后，任何现有 candidate 如果要进入新的 promotion，都必须为当前 candidate 建立 `review/*.json`，并通过当前 identity、attempt 和 hash 校验。

不得因为历史报告曾将某个翻译评为 A，就自动生成或推导 `decision: approved`。

## 多语言复用原则

Review 机制不绑定 `zh-CN`、ChatGPT、Page 或 Example。未来任何 locale、任何翻译模型、任何 TranslationUnit kind，只要产生翻译 candidate 并准备 promotion，都必须对最终 candidate 执行：

```text
automatic validation → Candidate Snapshot → ChatGPT Quality Check → machine finalization → promotion
```

新增语言无需重新决定是否审核，也无需重新定义 A/B/C/D 的基本含义。语言特定的术语和表达要求可以在同一正式 rubric 下补充，但不能取消逐 TranslationUnit review 或弱化 promotion gate。

## QC lineage preflight、完整重审与 Surface 定向纠错

创建新 QC lineage、第一次 `record`，或执行会显著改变 pending scope 的操作前，必须先运行只读预检查：

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check preflight \
  --locale <locale> --snapshot-id <snapshot-id> \
  [--previous-snapshot-id <finalized-snapshot>|--full-rereview]
```

输出列出已有 finalized Snapshot、结果是否已经开始、persisted/selected predecessor、可 carry-forward A、需要 QC 或 revision 的 Unit，以及预计 Reviewer Unit 数。存在既有 finalization 而新 Snapshot 未选择 predecessor 时，mode 为 `predecessor-required`；第一次 `record` 会 fail closed。维护者确实决定完整重审时，先以 `--full-rereview` 检查预计重复范围，并且只在第一次 `record` 或 `record-batch` 显式传 `--full-rereview`；结果文件建立后继续记录不得重复传该授权。结果文件一旦存在，`previous_snapshot_id` 仍不可补写或更换。

Surface Review 已明确发现某个旧 A TranslationUnit 缺陷时，不把它伪装成完整重审，也不覆盖旧 A/finalization。Integrated Surface 使用既有 immutable receipt 的 exact finding、explicit affected artifacts 和 current scope 授权；`class=translation-unit-language` 的 finding 才能进入 TU reopen。canonical candidate artifact 直接绑定 Snapshot 的 UnitID/kind、source path/SHA、candidate path/SHA、validation path/SHA 和 attempt，表示由 TU/adapter 决定，不限定 Markdown。历史独立 Surface Markdown finding 的直接 UnitID 合同保留兼容。

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check surface-reopen \
  --locale <locale> --reopen-id <id> \
  --previous-snapshot-id <exact-finalized-predecessor-snapshot> \
  --surface-review-id <review-id> --finding "<specific finding>" \
  [--finding "<another exact finding>" ...] \
  [--integrated-surface] \
  --unit-id <unit-id> [--unit-id <unit-id> ...]

go run -mod=readonly ./cmd/tour-i18n retranslation export \
  --locale <locale> --generator <provider> --allow-reexport \
  --previous-snapshot-id <current-finalized-snapshot> \
  --surface-reopen-id <id> --id <unit-id> [--id <unit-id> ...]
```

两种 reopen 路径均绑定 exact finalized predecessor、正式 Surface evidence 的 path/SHA-256、精确同 kind TU identity 和 finding。普通/legacy 直接 Markdown 路径继续要求当前 finalization 验证通过；它要求 `--finding` 原样存在于 evidence，每个 Unit ID 作为完整标识出现，不能用子串冒充。revision unit set 必须 exact-match，默认且硬性上限为 60，Page / Example 不得混合。manifest 使用 `revision_feedback_source=surface_review` 与显式 authorization ID。revision 后的新 full Snapshot 仍以旧 finalized Snapshot 为 predecessor：未变化 A 正常 carry-forward，变化 Unit 进入 re-QC。任何 evidence、finalization 或 TU identity 变化都 fail closed。

上段 Markdown 文本匹配仅适用于历史直接 evidence 路径。`--integrated-surface` 使用 `data/locale-language/<locale>/surface/<review-id>.json` 原 receipt，经独立 session、receipt/scope identity、evidence SHA、closure/runtime/current scope 检查，再通过共享 `ExactSourceTranslationUnit` 将 source path + UnitKind + frozen source SHA 映射为恰好一个 UnitID；0/多个匹配拒绝，不用 route、latest 或近似名称。finding 的 `artifacts` 必须原样属于 Surface scope 的 `candidate_artifacts`，且 Snapshot/validation/candidate bytes 全部一致。多 Unit finding 必须在 receipt 中列出完整 affected scope；未列出的 A 正常 carry。已有 carry TU 只有 scope 绑定其 exact artifact、receipt 明确记录语言 finding 才可定向 reopen，这不产生一次新的 TU A。

Integrated Surface predecessor 可以是 glossary/package membership 已演进的 immutable historical finalization。必须先完整验证 FAILED Surface receipt、独立 Reviewer/session、Markdown SHA、current scope/closure 与 exact artifacts，再按指定 Snapshot 的完整 ordered coordinates 经共享 `SelectFinalizedTranslationPackage` 选择其正式 historical package scope；显式 `--content-package` 仍必须是 Surface 实际绑定的 current owner，历史 predecessor 按下述 composition 关系解析；不能把今天新增的 Page 当作历史 QC，也不能由旧 scope 授权新增 Page。随后通过共享 `RequireCurrentTranslationAuthorization` 验证 sealed checkpoint 或执行等价 readonly assessment：完整 historical A/finalization integrity、exact current source/input/candidate、current glossary、full Unified Review、semantic assessment 与 active validator 全部通过才可 reopen；affected/stale/unknown 一律拒绝，不因没有 checkpoint 而建立新的 authorization。历史 Snapshot/glossary 不要求等于今天 mutable glossary，也不改写原 bytes。exact artifact 另由既有 historical reader 验证 Snapshot membership、candidate/validation path/SHA、attempt 和 reconciliation lineage；当前 source path + UnitKind + source SHA 仍必须唯一映射到同一 UnitID。当前选中 candidate 必须仍为该 predecessor artifact，不能用 latest 替换 reviewed artifact；conflicting successor 拒绝。此分域仅适用于 `--integrated-surface`，不放宽普通 `VerifyQualityCheckFinalization`。

membership composition 中某个 sibling 的 historical package ID 可以不同于显式 current package ID；这不是跨 owner 的任意授权。共享 reader 必须已验证 current Surface scope/closure 的完整 composition，Surface context 必须绑定显式 current owner 的 canonical closure，且该 predecessor 的完整 ordered Unit ID/kind/source path/SHA 必须精确包含于 current membership；所选 finding artifact 仍必须属于指定 historical Snapshot。只给同 bytes、相同 route 或另一个 package 名称不能成立此关系。各 predecessor 分别记录 reopen 并 exact export；A 的 export-only state 不消费 B 的不相交 scope，同一 reopen 或冲突 Unit 的 successor export 仍拒绝。

`--previous-snapshot-id`、`--surface-review-id` 和 `--reopen-id` 都是 stable identity，不是文件路径。成功只以 exclusive create 写入一个新的 immutable `data/quality-check-reopens/<locale>/<reopen-id>.json`；同 ID 重复执行拒绝，不覆盖、不伪装为 idempotent success。validation 失败不发布 reopen；旧 A/finalization、Surface receipt 和已有 authorization 全部保留。record 与首次消费重新检查 current authority，已冻结 invocation 的历史读取仍验证原 integrity 与 exact artifacts，不重放今天的语言资格。

同一个 FAILED Integrated Surface receipt 的多个独立 `translation-unit-language` exact finding，可以用 repeatable `--finding` 形成一个 aggregate reopen：所有 affected Unit 必须属于同一个已通过下述 historical integrity 与 current authorization 检查的 finalized predecessor Snapshot、同一种 UnitKind，总数最多60。每个 finding 必须非空、trimmed、在原 receipt 中 exact match，并具有已审核的 exact artifacts。授权从所选 findings 的 artifact union 建立 Unit→finding 映射；显式 `--unit-id` 集合必须与 union 完全相等，不按 flags 顺序或位置配对。重复 finding、重复 Unit、同一 Unit 被两个 finding 授权、missing/extra Unit、未匹配或 non-TU finding 均拒绝。revision 不继承 initial `1–30` / `31–51` partition；一个 aggregate reopen ID 可一次导出全部授权 Unit，形成一次 concentrated revision 和一个 Generation Bundle，不为独立 finding 拆分 invocation。

reopen schema v1 additive 保存 `findings` 与每条 `units[].finding`；aggregate 的全局 `finding` 为空，不用第一个 finding 代替其他 Unit。读取时从原 authority 重新验证 selected findings 的 exact-set 及逐 Unit finding。旧 single-`finding` v1 receipt 继续原样读取、校验与 export，无需迁移或 schema bump；新单 finding reopen 仍使用原表示。每个 revision manifest Unit 固化自己的 exact `previous_finding`，同时绑定相同的 `revision_authorization_id`、`revision_feedback_source=surface_review`、`previous_rating=A`、`previous_review_decision=reopened` 与原 immutable receipt path/SHA。Generation Bundle 沿用现有 per-unit feedback 和 predecessor 输入；所有 currentness/identity/SHA gate 保持不变，不修改原 receipt、Reviewer evidence、旧 QC/finalization 或 closure。

UI、结构化资产、navigation、sitemap、canonical、runtime、ads、coverage、package integration 等 finding 保持原 Surface/package blocker，不填写伪造 UnitID，也不走 TU revision。record reopen 与首次 revision export 都检查 receipt 当前性；export 后 authority 作为原 invocation 的冻结证据，由既有 revision/retry currentness 接管。原 receipt、reviewer evidence、context references、source 和 predecessor artifact 仍不可变化；已成功 sibling 不把其余原 retry Bundle 失效化。新 scope/closure 完成后必须重新取得独立 Integrated Surface PASS，不能用旧 failed receipt 或 TU A-only 代替。

独立 Surface PASS 后由 preview acceptance 发现的可机械证明 TU 缺陷，使用[Preview TU failure authority](PREVIEW_TU_FAILURE.md) 的独立 immutable machine receipt，通过 `quality-check surface-reopen --preview-failure` 复用同一 revision lifecycle。不能修改旧 PASS、补造 Reviewer finding 或滥用 `--historical-documents`；machine authorization 不产生语言 rating，successor 仍需 independent re-QC。

## Present 可见文本审核

Reviewer 判断 `.article` 的下划线、强调范围或词间空格前，必须以仓库锁定的 `golang.org/x/tools/present` 实际解析结果为准。legacy Present 用同一个 marker 分隔 span 内的词：`_two_words_` 渲染为一个 italic span，用户可见文本是 `two words`，内部下划线不是缺陷；`_two words_` 因真实空格分词而不会形成同一 emphasis span，用户会看到下划线，才可能构成 finding。仓库测试直接调用实际 parser/`present.Style` 固定该正反例。不得仅搜索原始下划线就提出 finding，也不得为绕过审核放宽 protected token、restore 或 importer。

### 受限 predecessor 的同 Unit representation recovery

正式 B/C/D finding 与 reviewed candidate 保持有效，但 passed predecessor raw 不具备 ordinary prose editing capability 时，先走[同 Unit machine bridge](TRANSLATION_UNIT_REPLACEMENT.md#同一-formal-unit-的受限-transport-bridge)只读 preflight；只在明确授权后 record，再建立新的 exact ordinary QC revision。此 authority 不给语言 rating，不修改原结果，不继承 A，不伪造 formal replacement。per-Unit B/C/D 与 finding 原样进入后续 revision，修订完成仍需 validation、新 Snapshot、独立 re-QC 和 A-only 收口。没有 current bridge 时 early capability guard 继续 fail closed。

## Deterministic language audit 与 Reviewer attention

current TU machine validation 先要求合法 [locale language policy](LANGUAGE_DEFECT_EXHAUSTION.md#正式接入合同首次真实使用前仍须-command-contract-gate)。只有机器可证明的 policy violation 阻断；source-prose/protected-language/paragraph 等 review candidates 不改变 validation/QC decision。QC Reviewer Bundle 保留完整 source/target/glossary，并通过既有 inventory 携带 `machine/language-review-candidates.json` 和 exact policy metadata。Reviewer 必须阅读完整材料，候选不是 finding、不是评分、不是 reopen authority。旧 A 不改写，当前 eligibility 不 grandfather；失败沿既有 current-validation/reopen/recovery 处理。
