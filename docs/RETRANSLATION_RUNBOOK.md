# Retranslation 执行手册

Learn/Docs current `learn-docs-v1` = 51 Page；initial Generation与 initial QC均为30+21两次独立 invocation。`docs-entry-v1`仅作历史读取，不是新 locale的第二阶段。旧49+2 finalized历史无需重新 Generation/QC；先执行 readonly [package composition preflight](LEARN_DOCS_MEMBERSHIP.md)，之后另行授权统一 closure。历史命令示例中的原 Snapshot/Unit identity不可改写。

全项目执行共同遵守[共享翻译单元生命周期与五角色协作](SHARED_TRANSLATION_UNIT_LIFECYCLE.md)：所有内容表面复用同一生命周期；页面差异限于适配器；传输安全与逐单元机器校验分层；先恢复、先复用，中文说明，交接只给一个执行者、动作和停止条件。本文中 Tour 的 present / Example 细节属于 Tour 适配边界，不另建 Learn/Docs 状态机。Learn/Docs 两批生成和机器校验全部完成后才初审，两批初审全部完成后集中修订。角色与工具分工不替代任何正式 machine authority 或语言质量 gate。

## 1. 当前正式生产流程

当前正式 TranslationUnit 翻译流程支持两个语言生成执行环境：普通 ChatGPT 使用 GPT-5.6 Sol + High；Codex 新启动的正式 Generation 默认使用 GPT-5.6 Sol + High（`gpt-5.6-sol-high`）。每个 locale 在开始正式 generation 前依 [多语言翻译流程](TRANSLATION_WORKFLOW.md) 选择并原则上持续使用其中一个 Generation provider；两者共用同一 TranslationUnit workflow 与全部质量 gate：

```text
current Glossary Review coverage
→ retranslation export
→ provider-neutral Generation Bundle → 本地 staging 目录 import（跨环境回退时 Result Bundle import）
→ 完整 raw-responses/
→ retranslation process
→ automatic validation
→ quality-check snapshot
→ 独立 ChatGPT Quality Check session
→ quality-check finalize
→ promote
```

模型与 reasoning 由用户在对应 UI 中选择；ChatGPT Generation 使用 GPT-5.6 Sol + High，Codex 新启动的正式 Generation 默认使用 GPT-5.6 Sol + High（`gpt-5.6-sol-high`）。provider-independent 输入/输出见 [翻译任务规范](TRANSLATION_TASK_SPEC.md)；provider 为 `chatgpt` 时的 staging 与会话隔离见 [ChatGPT 正式语言生成执行规范](CHATGPT_LANGUAGE_GENERATION.md)；provider 为 `codex` 时见 [Codex 正式语言生成执行规范](CODEX_TRANSLATION.md)。

## 2. Export 与首次翻译

新 locale 第一次 export 前必须已经完成统一状态表初始化，并按 [Glossary Review 规范](GLOSSARY_REVIEW.md) 取得 current passed gate：

```bash
go run -mod=readonly ./cmd/tour-i18n status init --locale <locale>
go run -mod=readonly ./cmd/tour-i18n status check --locale <locale>
go run -mod=readonly ./cmd/tour-i18n glossary-review check --locale <locale>
```

只有 `status check` 与 Glossary Review check 都通过后，才进入首次 `retranslation export`。`status init` 只创建缺失的初始 `status.tsv`，不得用于重置或同步已有 locale；已有 locale 的 source 更新与状态迁移继续走现有正式流程。

使用 `retranslation export` 创建 batch。provider 为 ChatGPT 时必须显式传 `--generator chatgpt`；provider 为 Codex 时可显式传 `--generator codex`，未指定 generator 仍是向后兼容的 `codex`。下例展示 ChatGPT 路径：

Exporter 对第一次及后续正式 export 都执行同一个 current Glossary Review gate，并在创建 batch 前 fail closed。既有 live locale 仅通过固定、SHA-bound 的一次性 legacy migration coverage 保持兼容；未来 locale 不能凭任意 matching Locale Surface Review A-gate 动态绕过。

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation export \
  --locale <locale> --generator chatgpt \
  --unit-kind page --limit 60
```

开始翻译前，当前生成执行者必须读取当前 batch 的：

新增 locale 首次 Page batch 的推荐生产基线、正式顺序和未来调整条件，以 [Codex 正式语言生成执行规范](CODEX_TRANSLATION.md#新增-locale-的首次-page-batch) 为唯一执行规则。首次 Page export 仍显式传 `--unit-kind page --limit 60`；初次自动 Page / Example、显式 `--id`、正常 Quality Check B/C/D revision 与经 `quality-check surface-reopen` 授权的精确 revision export 均使用默认且硬性上限 60。Page 与 Example 永不混合，也不做均衡分片；当前正式课程仍只有 19 个 eligible Example，不为凑满批次扩大 scope。retry scope 不因本上限改变。

未传 `--batch-id` 时，batch 根据 generator 自动命名为 `codex-<locale>-NNN` 或 `chatgpt-<locale>-NNN`。两种 prefix 共享同一 numeric namespace；自动编号取实际最大序号后递增，重复 numeric suffix fail closed。同一 TranslationUnit 的 latest export/source revision 也按 numeric suffix 选择，不依赖 prefix 字典序。显式 `--batch-id` 的既有兼容行为、历史 batch 名称、manifest 和 evidence 均保持不变；manifest 不新增 provider/model/reasoning provenance。

1. `manifest.json`；
2. manifest 列出的全部 `inputs/*`；
3. `locales/<locale>/glossary.yaml`。

这三部分不可拆分。initial Page / Example bulk Generation 仍由维护者 Local terminal 导出 `generation-bundle export` ZIP 并 handoff 给 ChatGPT 或 Codex；不得改用逐文件输入或模型自行扫描工作树。生成者将完整 TranslationUnit 成组保存到按 locale + batch 隔离的隐藏 staging；具备本机访问能力时默认由 `generation-bundle import --input-dir` 直接安装，不生成或下载 outputs ZIP。确需跨环境传输、目录不可访问时，兼容使用 outputs ZIP → `result-pack` → Result ZIP import。若 Generation/transport 在 import 前中断，先核对 formal state，再对仍 CURRENT 的 Generation Bundle 使用 `generation-bundle recover` 从 partial staging / interrupted tar/tar.gz 中逐 Unit 回收可验证完整成果，只重新生成其 `missing` scope；recovery staging 补齐 exact set 后仍走普通 import。Reviewer ZIP 继续由维护者上传给独立 Reviewer。目录与 ZIP importer、recovery preflight 都保留现有机器安全与 provenance gate，随后仍须执行 process / validation。完整命令、current/exact-set/no-overwrite/recovery 规则见 [翻译任务规范](TRANSLATION_TASK_SPEC.md)。TranslationUnit 是翻译、validation 和 review 的最小单位；Generation Bundle 仅运输输入，不是 batch authority 或审核 evidence。

## 3. Process 与 automatic validation

raw responses 完整后执行：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation process --locale <locale>
```

`process` 重新生成受保护输入、执行 restore、写入 batch candidate，并运行正式 automatic validation。结果保存在该 batch 的 `candidates/`、`validation/` 和 `result.json`，不会自动修改 canonical candidate 或 status。

Automatic validation 只负责结构、保护 token、代码、链接、source identity 等机器安全性，不能替代翻译质量检查。

只要任一 Unit 为 `restore_failed` 或 `validation_failed`，`retranslation process` 在完整写出合法的 partial candidates、validation evidence 和 `result.json` 后返回 non-zero；human summary 仍显示 `FAILED`，`--json` 仍先输出完整结果再以 non-zero 结束。调用方必须同时保留输出 evidence 并按 shell status 判定失败，不得把已保存的 partial evidence 当成整批 PASS。

`retranslation export`、`retranslation process`、`retranslation revalidate`、`retranslation retry`、`quality-check scope` 与 `retranslation review scope` 默认输出适合复制的人类摘要；成功 Unit、reusable Unit 和 carry-forward Unit 不逐条展开，失败 Unit 保留原因与 evidence path。`quality-check scope` 的 human summary 最多显示本轮 60 个 pending Unit，并在 Page / Example 边界截断；legacy `retranslation review scope` 仍最多显示 30 个。已有机器调用方应显式传 `--json` 获取完整稳定 JSON；scope 的 JSON 始终包含完整 pending 列表，JSON 写 stdout，错误与诊断写 stderr。

### 提交前 whitespace 检查与 EOF 契约

`git diff --check` 是所有正式提交前的必做检查，不得带 whitespace error 提交。Retranslation 文本 artifact 的统一 EOF 契约是：文件以**恰好一个 LF**结束，不得在 EOF 保留额外空行。不为 `data/retranslation-runs/` 设置 Git whitespace 豁免。

新导出的 batch 在 manifest 中声明 `artifact_eof: single_lf`，并按以下边界执行：

- exporter 在写入 Page `.article` 和 Example `.txt` input 前规范化 EOF；`source_sha256` 仍绑定原始 TranslationUnit source 字节，`input_sha256` 绑定规范化后的 protected input；
- ChatGPT 或 Codex 生成的 `raw-responses/*` 与 `retries/*/attempt-NNN.*` 必须以恰好一个 LF 结束；
- `retranslation process` 和 `retranslation retry` 在 restore 前检查对应 raw artifact；EOF 不合规时直接失败，不自动改写 raw response，也不产生部分 candidate 或 validation evidence；
- restore 成功后，process/retry 在 validation 与写入前将 Page `.article` 和 Example `.go` candidate 规范化为恰好一个结尾 LF。

历史 manifest 没有 `artifact_eof` 字段时，流程仍按历史字节进行兼容验证；不得为追加新字段或消除历史 warning 而改写已提交的 batch artifact。该兼容边界不是新 batch 的 whitespace 例外；新 batch 必须满足上述 EOF 契约并通过 `git diff --check`。

## 4. Revalidate：validator 修复后重验或恢复同一 attempt

当已处理 Unit 的 restore 已成功、candidate 已存在，而旧 `validation_failed` 已确认来自 validator 规则错误或规则更新时，使用：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation revalidate \
  --locale <locale> \
  --batch-id <batch-id> \
  --unit-id <unit-id>
```

普通路径重新生成并核对 protected input，确认当前 validation、raw response、restore 结果、candidate、manifest 与 `result.json` 的 identity 一致，然后只用当前 canonical validator 重验现有 candidate。它不生成或修改 raw response/candidate，不增加 translation attempt；当前 validation 的 `attempt` 与 `raw_response_path` 原样保留。

对 `current_validation` feedback 的普通 successor（不含 transport recovery、representation bridge，且不含 installed approved-interval selection/proof），若已有 restore 成功的 candidate，revalidate 使用现有 sealed reopen 验证 exact Unit、source、package/scope、previous Snapshot 与 finding/lineage，核对 manifest/result/validation 和 saved input SHA，然后用当前 canonical validator 重验现有 candidate，不为此重建 historical Generation input 或 decoder。已有 candidate SHA binding 必须先 exact 匹配，不能借 revalidate 接受被替换的已绑定 candidate。旧 evidence 无 binding 时，新重验只证明本次实际验证的 bytes，不补称旧 invocation 曾绑定该 SHA，也不重建或改写旧 archive。`restore_failed` 的恢复及其它 feedback 路径仍执行原严格 restore/provenance 校验。若 initial imported Generation Bundle 含 approved-interval selection/proof，revalidate 仍按 [approved-interval migration contract](CURRENT_VALIDATION_APPROVED_INTERVAL_MIGRATION_DESIGN.md) 执行原 strict decoder/provenance preflight，不能刷新或绕过 stale installed proof；malformed selection、未选中的 extra proof 或 import/archive 读取错误均 fail closed。

每次更新前，旧 validation evidence 按原始字节归档到 `revalidation-history/<flattened-unit-id>/revalidation-NNN-validation.json`。这个连续编号只表示 validator revalidation 次数，不是 translation attempt。随后命令原子更新当前 `validation/<unit>.json` 和 `result.json`。默认 human summary 显示 attempt、revalidation 序号、前后状态和 history/current evidence path；`--json` 输出完整 revalidation result。

对于已确认的 adapter/validator 误报，`restore_failed` 且尚无 candidate 的 Unit 也使用上述共享命令恢复。原 manifest、source、protected input、完整 glossary、原 invocation context 与 raw response 必须仍 current；原 imported Generation Bundle archive 与 import provenance 必须精确绑定同一 Unit、raw bytes、provider/model 和 attempt（历史正式表示沿用已有 GenerationOrigin verifier）。仅允许 validator authority 代码变化，不重新执行语言 Generation，也不增加 attempt。pending import、stale input/glossary/provenance、现存不明 candidate 或再次 restore failure 均在 mutation 前停止。

恢复时从同一 raw response重新 restore，以既有 EOF 契约写 candidate，并执行当前 canonical validation。旧 validation **原始字节**保存在既有 `revalidation-history/<flattened-unit-id>/revalidation-NNN-validation.json`；旧 `result.json` 原始字节同时保存在同目录的 `revalidation-NNN-result.json`。candidate 与 current validation/result 使用既有 file-update transaction 一起落盘，历史 require-missing，不覆盖旧 evidence。真实 candidate defect 保留为新的 `validation_failed` evidence，CLI 在输出 evidence 后返回 non-zero，停止并决定 exact retry；不自动修改语言。

Candidate Snapshot 仍读取当前 `validation/` evidence，并绑定其 SHA-256；revalidation 后的 evidence 是最终 evidence，但 attempt 仍由未变的 raw-response provenance 决定。非 `restore_failed` Unit 缺 candidate 仍禁止恢复；真正需要新译文时使用 Retry，而不是借 revalidate 伪造 Generation。

Automatic validation schema v2 显式增加可选 `candidate_sha256`：process/retry/revalidate 在当前 validator PASS 后写入实际 candidate bytes（含既有 EOF 规则）的 SHA-256；新失败 evidence 不授予 PASS binding。严格 reader 拒绝 malformed SHA 或没有 candidate path 的 binding。它属于原 validation JSON，进入现有 validation bytes hash / Snapshot evidence identity，不引入新 authority、receipt 或 CLI。旧 v1/v2 evidence 没有该字段时不推断、不自动升级；历史 bytes 原样保留，显式 revalidate 仍按既有 transaction 将旧 validation 原字节归档后发布新 evidence。

Candidate Snapshot 仅对具有该 binding 的普通 `current_validation` successor 使用 current candidate eligibility 路径：candidate path/bytes/SHA、Unit/result/status/attempt、当前 source、active validator、完整 glossary Review、membership、exact reopen finding/scope/lineage 全部必须通过，commit 前重新检查。候选被替换成另一份同样通过 validator 的译文也会因 SHA mismatch 失败。此路径不为证明 candidate 未变而重放 historical raw/decoder/approved-interval implementation；历史 Generation/provenance 留作 audit/recovery 的独立 evidence。旧 unbound evidence、其它 feedback 与 transport-recovery/representation-bridge invocation 不取得该例外，继续原严格 reader；新 candidate 不继承旧 Reviewer A，后续 Independent QC、Surface 与 Production gate 不变。command currentness/consumer 语义已变化，首次真实 revalidation/Snapshot 前仍须完成 current docs → implementation → direct contract tests 的 first-real-use gate。

## 5. Retry：只处理 restore/validation failure

Retry 只用于：

- `restore_failed`；
- `validation_failed`。

它不用于翻译质量修改。ChatGPT 或 Codex 读取失败 evidence 和原正式输入，生成下一份连续编号的原始译文：

```text
retries/<flattened-unit-id>/attempt-NNN.article
retries/<flattened-unit-id>/attempt-NNN.txt
```

`raw-responses/<unit>.*` 是正式 attempt 1，其初始 validation evidence 也记录 `attempt: 1`。因此首次 retry 必须写入 `attempt-002.*`，不得创建 `attempt-001.*`；之后从当前 validation 的 attempt 依次加一，且不得覆盖已有 attempt。`attempt-001-validation.json` 是 retry 命令归档的初始 validation evidence，不是需要模型生成的 retry raw response。ChatGPT retry 先写同目录隐藏 staging file，完整核对后再原子 rename 为正式 attempt 文件。

文件已存在后再执行：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation retry \
  --locale <locale> \
  --batch-id <batch-id> \
  --unit-id <unit-id>
```

`retranslation retry` 本身不调用模型、不生成或改写译文；它只处理已经存在的 retry raw response，归档前一份 validation，并更新目标 unit 的 candidate、validation 与 batch 汇总。

`retranslation retry` 默认只输出目标 Unit 的 PASS/FAILED、当前 attempt/status 和整个 batch 的 restore/validation 汇总，不展开 batch 中的全部成功 Unit；失败时同时显示当前失败原因及 candidate/validation evidence path。需要完整 `RetranslationProcessResult` JSON（包括 `units` 数组）的机器调用方必须显式传 `--json`。

## 6. Candidate Snapshot

Candidate Snapshot 的 scope 必须由下述显式 contract 确定，不从 Snapshot ID、batch 名称或命令经验推导。省略 `--content-package` 的正式 CLI 固定选择 native Tour workflow（当前 103 Page + 19 eligible Example）；`site-v2-shell-*` 这样的 Snapshot ID **不会**选择 shell package。完整所选 scope 的所有 TranslationUnit 都有通过 automatic validation 的最终 candidate 后，才能生成 Snapshot。

approved-interval 的完整 historical PASS 按 [Snapshot admission 合同](TRANSLATION_TASK_SPEC.md)
核验：历史 result/validation/raw/import/archive/proof integrity 与当前 decoder/proof reproduction
分开。implementation SHA/producer/decoder drift 本身不撤销历史 PASS；仅在全部 typed semantic
evidence 和非 implementation dependencies exact、原批完整 PASS/最新选择无冲突、raw exact
还原 candidate 且当前 active validator/source/membership/eligibility 全通过时，允许 Snapshot
继续。tampered/missing evidence、semantic invalidation 或当前 validation failure 仍 fail closed。
不改历史 evidence、不刷新旧 invocation、不自动继承 A，且仍须实现后的 first-real-use gate。

Tour legacy/default 命令保持：

```bash
go run -mod=readonly ./cmd/tour-i18n quality-check snapshot \
  --locale <locale> \
  --snapshot-id <snapshot-id>
```

non-Tour 必须显式选择完整 registered package，先 readonly preflight：

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check snapshot \
  --locale <locale> --snapshot-id <new-unused-id> \
  --content-package <registered-package> --preflight
```

`site-v2-shell` 选择当前 package authority 的完整 physical Page scope，Homepage 始终是一个 Page TU；protected slots 不是多个 Unit。preflight stdout 的 `manifest.content_package` 给出 `id`、`authority_sha256`、`scope_identity_sha256`、完整 ordered `unit_ids` 与当前 source/representation authority references；顶层 `recorded=false, mutation=none`。它不创建 Snapshot、QC result、finalization 或 authorization，也不预留 ID。

只有维护者另行授权后，才用同一 scope/new ID **去掉 `--preflight`** 创建；不是增加 `--apply`/`--record`。创建会重新计算全部当前 proof，不信任先前的 preflight stdout。package omission 不能继承 hidden caller scope；Snapshot 自己解析/传递 `QualityCheckSnapshotOptions.ContentPackage`，不会让 generic dispatcher 静默吞掉选择。原内部 API 的显式 catalog-scoped legacy callers 保持兼容，但不作为 formal CLI contract。

### Candidate Snapshot command contract

修改后的 CLI 第一次真实使用前，必须按 [Formal Command Contract v3](WORKFLOW_HANDOFF.md#formal-command-contract-v3) 在实现完成后顺序核对 current canonical docs、implementation、direct CLI tests；不沿用旧 contract verification。

| 字段 | Contract |
| --- | --- |
| COMMAND | `quality-check snapshot` |
| OPERATION | 默认 `create`；`--preflight` 为 readonly selection/currentness |
| INPUT_KIND | 当前 native Tour 或显式完整 registered package 的 validated workflow candidate set；不消费 plan/receipt 文件 |
| INPUT_PATH_CONTRACT | none；`--locale`/`--snapshot-id`/`--content-package` 是 identifiers，不是 paths；路径型 Snapshot ID 如 `../x` 拒绝 |
| REQUIRED_FLAGS | `--locale`、新的 `--snapshot-id`；non-Tour 必须有单个非空 `--content-package` |
| FORBIDDEN_FLAGS | `--id`、`--unit-id`、`--unit-kind`、`--start-index`、`--limit`、`--apply`、`--record`；不支持 partial/mixed scope、重复 package flag 或 positional args |
| STDOUT | create 保留原 path/count 文本，显式 package 另打印 package/authority/scope identity；preflight 为 `{manifest,recorded:false,mutation:"none"}` JSON |
| MUTATION_EXPECTATION | create=known，仅新建 manifest；preflight=none |
| IDEMPOTENCE | write-once，不覆盖、不把重复 ID 当 success/no-op；已有 ID（包括空目录）preflight/create 均拒绝 |
| CURRENTNESS_CHECK | registry/package authority与完整ordered scope、source/adapter/representation authority、当前glossary review、latest candidate/input/raw/validation SHA及active validation；提交前重读selected evidence、source/context与mode，不复用旧PASS |
| PRIOR_OR_CAS | 无 predecessor 参数；preflight 不授权 CAS。destination 必须 absent，提交沿用 no-replace directory publication（当前 Linux amd64；不支持的平台 fail closed） |
| SUCCESS_STOP | preflight 后停止审阅scope；create后停止核对manifest，不自动执行QC |
| FAILURE_STOP | unknown/unregistered/missing-authority package、scope/identity/source/validation drift、existing ID或I/O失败立即停止，不回退Tour、不缩减范围、不覆盖 |
| RECOVERY / REPLAY | 保留既有manifest/evidence。create失败先只读核对是否已发布或存在staging；不盲重试、不复用已存在ID、不自动delete/repair。新的scope需新unused ID |

新 explicit package manifest 在 schema v1 中增加 optional `content_package`；旧 Tour 与无此字段的旧 package Snapshot 保持原 reader。新字段必须与当前/显式历史 registered descriptor、完整 Unit/source membership 对齐；current QC reader 还验证 current authority refs。后续 package 的 `quality-check scope` / Reviewer / record / finalize 仍须显式选同一 `--content-package`，不能让 package Snapshot 在默认 Tour catalog 下继续。历史 projection 只解析 exact bound package identity，不用路径相似性冒充 current authority。

错误 scope invocation 已创建的 Snapshot 也是 immutable artifact。例如 `site-v2-shell-qc-003` 实际保存 Tour scope，必须保留，不能删除/重写/复用该 ID；修复 contract 后需另用未使用的新 ID（如经核对未占用的 `site-v2-shell-qc-004`）。Snapshot 名称不能反向改变它已冻结的 scope。

Snapshot 命令本身只创建：

```text
data/quality-check-snapshots/<locale>/<snapshot-id>/manifest.json
```

Snapshot 按 Catalog 的 Page 顺序、再按 eligible Example inventory 顺序冻结完整 workflow。当前基线是 103 Page + 19 eligible Example = 122 TranslationUnit。每个 unit 先按 batch number 选择最新的一份结果，再要求它匹配当前 source revision 且状态为 `passed`；最新结果失败或 identity 不匹配时都禁止回退旧 batch。Snapshot 同时校验 manifest/source/input/candidate/validation identity、相关 SHA-256、restore 绑定和 retry 最终 attempt。

Manifest 只引用仓库中已有的 glossary、source、candidate 和 validation 文件，不复制这些文件，不创建 `_content`、ZIP 或 review artifact。该命令不修改 `locales/<locale>/status.tsv`，不执行 Quality Check、Final Review 或 promotion。后续 `quality-check record` 可以在同一 Snapshot 目录新增独立的 `quality-check-results.json`，但不改写 manifest。

新流程只使用 `quality-check scope --locale <locale> --snapshot-id <snapshot-id>`；revision 后另加 `--previous-snapshot-id <previous-snapshot-id>`，只 carry-forward 上一轮 A 且 source/candidate/validation/attempt identity 完全相同的 Unit。`retranslation review scope` 仅为 legacy Final Review evidence 的读取和验证保留，不属于新流程。glossary snapshot mismatch 是 scope 的整体 blocker，不是 unit pending。

实际 Reviewer 输入优先由同一 scope 导出 deterministic ZIP；首次 122 Unit 的三次独立请求分别导出 `1/60`、`61/60`（在 Page 边界得到 43）与 `104/60`（在 Example 尾部得到 19）：

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check reviewer-bundle \
  --locale <locale> --snapshot-id <snapshot-id> \
  [--previous-snapshot-id <snapshot-id>] \
  --start-index <stable-index> --limit 60 \
  --output /tmp/<locale>-<snapshot-id>-qc-<stable-index>.zip

go run -mod=readonly ./cmd/tour-i18n quality-check reviewer-bundle-check \
  --bundle /tmp/<locale>-<snapshot-id>-qc-<stable-index>.zip
```

bundle 完整包含该 working set 的 source/current target、完整 glossary、relevant manifest/input、validation、Snapshot、carry-forward/pending scope、rubric/authority 与 hashes。`reviewer-bundle-check` 必须在记录结论前确认 ZIP 与当前 working tree 逐字节一致。Reviewer 仍只返回 A/B/C/D + findings；现有 `quality-check record` / `record-batch` 是唯一记录入口，不新增第二套结果 import 或 QC state machine。

## 7. Quality Check 与 revision batch

ChatGPT Quality Check 必须以同一份 Candidate Snapshot manifest 为审核范围，不得自行从各 batch 中重新挑选 candidate。首次 locale 没有历史 QC result 时，`quality-check scope` 必须返回全部 Unit pending。完成实际审核后用 `quality-check record` 或 `quality-check record-batch` 写入该 Snapshot 的 `quality-check-results.json`。全 A 后由 `quality-check finalize` 生成唯一 promotion authority。当前严格生产策略只接受 A：

- A：通过 Quality Check；
- B、C、D：未通过质量 gate，必须进入 revision batch。

单次 Reviewer model invocation / response 最多审核 `quality-check scope` 中 60 个 pending TranslationUnits；Page / Example 分开。60 是实际审核质量边界，不得通过在同一次 response 内串联多个 `<=60` working set 绕过；下一组需要新的用户请求和新的 model invocation。每组必须逐 TranslationUnit 给出判断，分片不是抽样，并在直接结果与 carry-forward 合计后覆盖同一 full Snapshot 的全部 Unit。首次 122-Unit locale 的 full QC 推荐拆为 Page stable index `1-60`、Page stable index `61-103`、Example stable index `104-122`，最多三次独立 Reviewer 请求。revision re-QC 按实际 pending scope 审核，每次最多仍为 60，且 Page / Example 分开；`A = 全部 TranslationUnit` 后进入 machine finalization。

Quality Check 的质量修改不得使用 retry。Revision 流程为：

```text
创建新的 revision batch
→ re-export 对应 TranslationUnit
→ 选定的生成执行者重新读取完整正式输入并重新翻译
→ process
→ automatic validation
→ 生成新的完整 locale Candidate Snapshot
→ `quality-check scope --previous-snapshot-id ...`
→ 只对 identity 变化、无有效 A 结果或旧结果非 A 的 Unit 执行 ChatGPT Quality Check
→ `quality-check record` 记录新结果
```

重复以上流程直到 Quality Check 为 A。旧 batch 及其 evidence 保持不可变。

正式 revision export 必须显式给出产生 revision evidence 的 previous Candidate Snapshot：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation export \
  --locale <locale> --generator <chatgpt|codex> --allow-reexport \
  --previous-snapshot-id <snapshot-id> \
  --id <unit-id> [--id <unit-id> ...]
```

命令使用该 locale 当前选定的 Generation provider；ChatGPT 必须显式使用 `chatgpt`，Codex 使用 `codex`。两种路径的 revision eligibility 与 manifest feedback 字段完全相同。

新正式 revision 模式只允许以下 Unit：

- 同一 Snapshot 中 Quality Check 为 B/C/D 且 finding 非空；

Exporter 验证 Snapshot、Unit、result 与当前 source/candidate/validation/attempt identity；Quality Check A、缺失或 identity 不匹配均不能创建新 revision。manifest 固化 `previous_snapshot_id`、`revision_feedback_source=quality_check`、`previous_rating` 与 `previous_finding`。历史 `revision_feedback_source=final_review` 字段及 parser 仅为已有 manifest/provenance 解释保留，不能创建新 batch。每批默认且最多 60 个，Page / Example 不混合，也不得为凑满批次扩大 revision scope；Revision 的正式翻译输入仍是 manifest、manifest 全部 `inputs/*` 与 locale glossary 三者不可拆分。

### Integrated Surface historical predecessor revision export

已记录的 Integrated Surface exact TU reopen 使用同一个 `retranslation export`，增加 `--surface-reopen-id <reopen-id>`，并显式传 `--content-package <current-registered-package>`、`--previous-snapshot-id <reviewed-finalized-predecessor-id>`、`--allow-reexport` 和完整 exact `--id` 集合；需要固定 batch 时传 `--batch-id <provider-locale-number>`。Snapshot/reopen/batch 参数都是 stable identity，不是 JSON 或目录路径。Page/Example 不混合，最多60，subset/superset/duplicate、structured finding 与错误 predecessor 均拒绝。

对[正式 membership composition](LEARN_DOCS_MEMBERSHIP.md) 的 package，current Catalog 仍是完整 current membership。先经 shared reopen reader 完整重验 FAILED Surface receipt/current scope、原 closure/membership composition、所有 sibling predecessor 的 current authorization、current glossary/full Unified Review、source/protector/validator、exact candidate/validation/attempt 和 latest selection。随后仅以指定 predecessor 的正式 historical descriptor/effective original A 验证本次 feedback，不能要求单个 historical Snapshot 等于 current package 全集。该历史 scope 只用于 predecessor evidence evaluation，不替换 current scheduling Catalog，也不授权 sibling Units；缺失/错误/tampered membership、closure、sibling finalization 或任一 affected/stale/unknown 都停止。普通 `AssessCurrentTranslation` 的 single-complete-scope invariant 不放宽。

合法 sibling predecessor 的历史 package ID 可以与显式 current owner 不同。该关系只从已验证的 current Surface closure/composition 与完整 historical scope 的 exact current-membership 子集取得；current Surface context 必须绑定该显式 owner 的 canonical closure，不能按名称、route 或 hash 猜 package。各 predecessor 使用自己的 Snapshot/reopen，A 的 export-only batch 不使不相交 B 的 reopen/export 失效。

export Unit 集必须 exact-match reopen receipt；manifest 保留原 `previous_snapshot_id`，并逐 Unit 固化 `revision_feedback_source=surface_review`、`revision_authorization_id=<reopen-id>`、`previous_rating=A`、原 exact finding 和 immutable Surface evidence。其余 predecessor Units 与 sibling Units 不导出；不会合成 Snapshot/QC、产生新 A 或修改已有 reopen/Surface/closure。已存在的 successor export（即使尚无 processed result）表示该 Unit 的 reopen 已消费或存在冲突，必须继续既有 lifecycle，不能换 batch ID 再次导出。

成功只通过现有 staging + absence guard/rename 发布一个新 batch 的 manifest 与 exact protected inputs，human summary/`--json` 输出沿用原 contract。已有 batch ID 拒绝，不覆盖、不视为 no-op；所有 validation 在 publication 前完成，失败不发布 batch。I/O 或并发导致终态未知时先只读核对，不删除、不重放。此命令不执行 Generation/import/process/QC；历史 A/finalization、reopen 和 Surface receipt 保持 byte-identical。

### Glossary-induced protected-input stale recovery

candidate 已生成后若 glossary 合法变化，修改前先按 [Glossary Compatibility](GLOSSARY_COMPATIBILITY.md) 保存原 glossary bytes，再对完整当前 glossary 重新取得独立 Review CURRENT PASS。旧 Snapshot 不修改；没有逐 Unit compatibility proof 时 glossary mismatch 仍 stale。不得把 glossary 变化伪造成 Quality Check B/C/D、Surface Review finding、retry 或 revalidation。

只对当前 canonical source + 当前 glossary 经正式 protector 重建后，与该 Unit 最新已处理 numeric batch 中保存的 protected input 字节不同的 Unit 使用：

```sh
go run -mod=readonly ./cmd/tour-i18n retranslation export \
  --locale <locale> --generator <chatgpt|codex> \
  --allow-reexport --glossary-stale \
  --id <unit-id> [--id <unit-id> ...]
```

该模式不接受 `--previous-snapshot-id` 或 `--surface-reopen-id`。调用者显式列出本批 Unit；每个请求 Unit 都必须有 current-source、最新已处理且 `passed` 的 result/validation，旧 input 字节与 manifest hash 必须一致，并且当前 protected input 必须真实发生 drift，否则整批在创建前失败。Page / Example 仍不得混合，默认且硬性上限仍为 60。manifest 以 `reexport_reason=glossary_input_stale` 绑定 previous batch/input identity；Generation Bundle 使用独立 `translation-unit-glossary-stale-recovery` task kind，因此 audit 时不会与 initial、QC revision 或 Surface reopen 混淆。

新 batch 后仍走普通 `generation-bundle export → import → retranslation process → automatic validation`。未发生 protected-input drift 的可信 candidate 不重译；处理完成后创建当前 glossary 的 full Candidate Snapshot。verified compatibility + unchanged exact Unit identity 允许旧 A carry-forward；affected、identity changed 或 missing/stale lineage 仍须正式 QC，必要时由真实 finding 进入 revision。新 Generation transport 始终使用完整当前 glossary。

## 8. Machine finalization 与 promotion

只有完整语言满足以下条件，才能 machine finalization：

```text
A = 全部 TranslationUnit
B = 0
C = 0
D = 0
```

使用 `quality-check finalize --locale <locale> --snapshot-id <snapshot-id>`。该命令不再进行第二次语言审核；它 fail-closed 验证 full Snapshot 的 QC A-only lineage、glossary、rubric 与 source/candidate/validation/attempt identity，并写入 `finalization.json`。

以下 `retranslation review` 内容为 **legacy compatibility only**：它保留既有 Final Review evidence 的读取、验证与历史维护；不得用于新 locale、新 revision 或新 promotion：

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

Legacy Review evidence 的 schema/validation 规则见 [Translation Quality Review 规范](TRANSLATION_QUALITY_REVIEW.md)。它们不改变新 promotion gate。

Promotion 默认 dry-run，必须指定已 finalization 的 Snapshot：

```bash
go run -mod=readonly ./cmd/tour-i18n retranslation promote --locale <locale> --snapshot-id <snapshot-id>
```

默认输出简洁的 human summary，包括 locale、dry-run/apply mode、Page/Example 与 review/changed/unchanged/EOF normalization 统计和 `can_apply`；blocked dry-run 只展开非空的 `missing_evidence`、`missing_review`、`rejected_review`、`invalid_review` 集合，不展开正常 Unit。机器调用方必须显式传 `--json` 获取完整且保持既有 schema 的 `RetranslationPromotionPlan` JSON（包括 `units` 数组）。

只有用户明确要求应用时才使用 `--apply`。成功 apply 默认同样输出简洁 human summary 并明确标记已应用；需要完整 JSON 时可同时传 `--json`。Promotion 会验证 finalization、最新 batch 的 manifest/source/input、glossary 重建结果、retry provenance、candidate 与 validation；最新结果失败时不得回退到旧 batch。

## 9. 阶段边界

### Zero-review incremental finalization

新 Snapshot 的 preflight/scope 若为 100% predecessor A carry-forward、0 Reviewer Unit、零 pending，可直接显式绑定 finalized predecessor 完成 finalization，不创建新 QC response：

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check scope --locale zh-CN --snapshot-id qc-008 --previous-snapshot-id qc-007
go run -mod=readonly ./cmd/tour-i18n quality-check finalize --locale zh-CN --snapshot-id qc-008 --previous-snapshot-id qc-007
go run -mod=readonly ./cmd/tour-i18n quality-check authorization --locale zh-CN --snapshot-id qc-007
go run -mod=readonly ./cmd/tour-i18n site-content campaign-plan --locale zh-CN
```

只有 scope 为 complete A-only 且当前完整 Review、source/candidate/validation、逐 TU compatibility 与 predecessor finalization 全部合法时才保存新 finalization。`source_snapshot_id` 继续指向真实历史 Reviewer 来源，optional `previous_snapshot_id` 固定零 response 路径，输入只绑定真实历史 result 文件。已有 current result 与显式 predecessor 不一致必须停止，不能手改 results。历史 promotion/recovery 按原 glossary context 验证；当前语言授权另由 current compatibility + 新 finalized Snapshot 建立。遇到下一真实 derived-consumer stale 时停止，不重跑 promotion/recovery，不生成新的 Reviewer A。

### Native receipt 与缺失历史 receipt 的恢复

未来正式 apply 保存 `go-learning/native-promotion-receipt/v1`：完整 finalized A-only/source/glossary/selected-candidate/validation preflight → exclusive mkdir `.promotion-staging-transaction` → staged candidates/status/完整 receipt → 复核 old inputs/targets/status → 备份并安装 canonical/status → 复核 exact installed outputs、inputs 与 sealed receipt → exclusive link 发布 immutable receipt → 清理本次 staging。receipt 发布是 commit point，staging JSON 不是原始 apply success artifact。安装或发布失败时用 recoverable rename 恢复原 candidates/status；rollback 失败保留所有 backups/staging、报告 `mutation=unknown`，禁止盲目重放。mutation 开始后 panic/进程崩溃也保留 staging；任何 `.promotion-staging-*` 残留都会使 current authorization/campaign-plan/下一写操作 fail closed，必须先检查并恢复该事务。这是现有两处路径的 fail-safe 两阶段流程，不宣称跨路径 crash-atomic 或文件系统断电 durability。

receipt 内 `resulting_status_bytes` 与 SHA 绑定成功安装时的完整 status，exact membership/source/candidate/attempt/promotion note 校验仍保留。后续合法 `status.tsv` 更新不影响 TU successor authority，verifier 不再拿它的当前 SHA 与 receipt 永久比较；current canonical 仍必须 exact 等于 finalized selected candidate 的正式 promotion 输出。status 自身及其他消费者的 currentness 独立验证。

已存在 native receipt 的 Snapshot 不得再次 apply，也不得改用 historical recovery 重造事件。commit 后 staging cleanup 失败会显式报告 `mutation=known; committed=true` 并保留 guard，先检查/收口事务，不盲目重放已完成的 promotion。

旧实现未保存原始 receipt 时不得补造 apply event。`historical-promotion-recovery` 以稳定 reason `missing-original-promotion-receipt`，由 immutable closure-bound old bytes → finalized A-only Snapshot/carry-forward → selected batch/candidate → source/glossary/validation → exact current promotion output 构成历史 successor authority。Page output 沿用正式 EOF normalization，其原始 selected SHA 与输出 SHA 均绑定。完整 membership preflight，只记录 changed transitions，不给 unchanged Unit 编造事件。旧 closure、compatibility、QC/finalization 一律不改写。

```sh
go run -mod=readonly ./cmd/tour-i18n retranslation recover-promotion --locale zh-CN --snapshot-id qc-007
# 只有维护者显式授权并确认上述 preflight 后执行：
go run -mod=readonly ./cmd/tour-i18n retranslation recover-promotion --locale zh-CN --snapshot-id qc-007 --apply
go run -mod=readonly ./cmd/tour-i18n quality-check authorization --locale zh-CN --snapshot-id qc-007
go run -mod=readonly ./cmd/tour-i18n site-content campaign-plan --locale zh-CN
```

Recovery 默认只读，apply 只新增 content-addressed artifact、重复 apply 拒绝覆盖；不是重跑 promotion，也不是重审核。`quality-check authorization` 的历史 A 数量与 current canonical authorization 分开报告。即使 TU current，Course SEO / Surface 等 derived consumer 仍可 stale；下一个真实 blocker 暴露后停止，不用 successor 绕过其独立 gate。

阶段成功、FAILED、stale、额度耗尽或 mutation-unknown 的最终交接必须遵守 [正式阶段交接合同](WORKFLOW_HANDOFF.md)，包含真实数量/evidence、唯一下一步、执行者、停止条件，以及已验证附件或未执行的完整导出命令。

首次 bulk TranslationUnit Generation 仍以 Generation ZIP handoff 为输入阶段边界，不自动越过到语言审核。具备本地访问能力时，initial / revision / retry 结果均默认通过隐藏 staging 目录 import；进入 Generation / Reviewer 闭环后，export、bundle、目录 import（跨环境时 Result ZIP landing/import）、process / validation、Snapshot / scope / reviewer-bundle，以及 Reviewer 结论的 current-check / record / finalize，默认均由维护者 Local terminal 执行；AI 按[Call-Zero-First](WORKFLOW_HANDOFF.md#call-zero-first-交接合同)提供完整可复制命令并消费合法 terminal evidence，不以任务 duration 授权 remote call；正式 Quality Check 仍必须使用独立于本轮生成的 ChatGPT conversation/session，所有 A-only 与 identity gate 保持不变。`promote` 是闭环收口后的维护者 Local terminal 操作，不由 AI 自动执行。UI catalog 翻译是独立流程，不属于本手册。

## 10. QC lineage 与 Surface finding revision

普通 QC B/C/D revision 继续使用 `--previous-snapshot-id`。已 finalization 的 A 若由正式 Surface Review finding 定向纠错，必须使用 [Translation Quality Review](TRANSLATION_QUALITY_REVIEW.md) 定义的 `quality-check surface-reopen` + `retranslation export --surface-reopen-id`，不得伪造 B/C/D、修改旧结果或丢弃其余旧 A。新 QC lineage、第一次 record 和完整重新审核先运行 `quality-check preflight`；完整重审必须显式 `--full-rereview`。

同一个 FAILED Integrated Surface receipt 的多个独立 TU-language exact finding，可通过 repeatable `--finding` 建立一个 aggregate reopen，再用同一 reopen ID 一次导出集中 revision / Generation Bundle。条件为同 finalized predecessor、同 UnitKind、最多60、显式 UnitIDs exact-match 所选 findings 的 artifact union；逐 Unit 保留自己的 exact finding，不继承 initial partition。具体兼容表示与全部 fail-closed 校验见上述质量规范。

### QC B/C/D representation reconciliation → successor exact revision export

旧 protected representation 无法合法表达正式 B/C/D finding 时，先按 [Page replacement 合同](TRANSLATION_UNIT_REPLACEMENT.md#qc-bcd-的-representation-successor-authority)建立 `authority_source=quality-check` 的 immutable successor reopen。receipt 必须绑定原 Snapshot/results、原 candidate/validation/input、原 rating/finding、完整 frozen representations 和必要性 engineering proof。`surface-reopen` 是共享 receipt transport 名称；此分支不需要 Surface finding 或 finalized A，也不新增一次 initial language QC。

receipt 建立后，维护者 Local terminal 使用以下完整 export 模板。六个变量分别替换 locale、package、原 Snapshot、reconciliation/reopen ID、current successor Unit ID 和新 batch ID；generator 沿用该 locale 已选 provider（下例为 `chatgpt`）。不得传旧 Unit ID，不得改写旧 batch。多 Unit scope 必须显式重复 `--id`，exact-match receipt 全集，Page / Example 不混合且最多60。

```sh
(
  REVISION_LOCALE='<locale>'
  REVISION_PACKAGE='<package>'
  REVISION_SNAPSHOT='<predecessor-snapshot>'
  REVISION_REOPEN='<reconciliation-receipt-id>'
  REVISION_UNIT='<current-successor-unit-id>'
  REVISION_BATCH='<new-batch-id>'
  go run -mod=readonly ./cmd/tour-i18n retranslation export \
    --locale "$REVISION_LOCALE" --content-package "$REVISION_PACKAGE" \
    --generator chatgpt --batch-id "$REVISION_BATCH" --unit-kind page \
    --previous-snapshot-id "$REVISION_SNAPSHOT" \
    --surface-reopen-id "$REVISION_REOPEN" --allow-reexport \
    --id "$REVISION_UNIT"
)
```

`--surface-reopen-id`、`--previous-snapshot-id`、`--allow-reexport` 必须同时提供；`--content-package` 显式保留完整 package QC scope，`--id` 只限定此次 revision working set。共享 reader 对未指定 package 的 inventory 也会从已验证 replacement package 恢复完整 historical scope，不修改调用者、不扩大输出集，不覆盖显式 partial/wrong scope。随后仍执行完整初审覆盖检查，包含正确 package namespace 下已有 sealed A carry-forward；缺少初审的 ordinary Unit 仍被 gate 拦截。

export 成功后继续既有 Generation Bundle → exact finding revision → import/process/validation → 新 Snapshot → independent re-QC。successor 不继承旧 B/C/D 或 A 为新语言结论。若 export 失败，先只读检查目标 batch 目录；任何已落盘 artifact 均不得删除、覆盖或重放以掩盖失败。本模板不授权自动执行后续步骤。

qc-014 的 `chatgpt-zh-CN-046` 永久保留 immutable、0 output，不重放。已建立的 `learn-docs-qc014-representation-successor` receipt 消费同一个 qc-014 B finding；当前047已由维护者完成 import、restore PASS、validation PASS，不再执行047 export/Generation。本模板保持通用合同，实际下一步受下节 UNKNOWN 与独立 glossary gate 约束。

### Current-validation machine finding 的 shared reopen

current re-attestation 的 canonical `current_validation_failed` 使用独立 authority source `current-validation`；不是 Surface/Preview、原 QC B/C/D 或 representation change。它复用同一个 `data/quality-check-reopens` receipt 和现有 revision exporter，默认只读：

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check current-validation-reopen \
  --content-package '<package>' --locale '<locale>' \
  --reopen-id '<stable-id>' \
  --previous-snapshot-id '<finalized-predecessor>' \
  --previous-snapshot-sha256 '<exact-manifest-sha256>' \
  --current-reattestation '<complete-assessment.json>' \
  --current-reattestation-sha256 '<exact-evidence-sha256>' \
  --current-reattestation-identity '<identity_sha256>'
```

`--content-package` 必填且只能提供一次：显式 `tour-v1` 选择 canonical Tour-native default complete scope（不是 registered descriptor），其它 package 必须选择完整 registered scope。Tour selector 要求原 Catalog 没有 explicit `TranslationScope`，保留完整 native Page + eligible Example ordered inventory；不得清空已有 partial/different scope 后隐式 widening，也不伪造 Tour registration。registered packages 继续通过 `UseTranslationPackage()` exact 选择，已有 explicit scope 必须一致；未知 package fail closed。完整 selector contract 见 [Current Translation Authorization](CURRENT_TRANSLATION_AUTHORIZATION.md#current-validation-的-affected-only-reopen-authority)。

所选 package 必须与 frozen assessment 的 `package` 及 predecessor Snapshot 的完整 ordered Unit set 一致，partial/wrong package fail closed。输出 `recorded=false / mutation=none`、planned receipt/path/identity 和精确逐 Unit machine finding。完整 assessment 必须由 current source/candidate、原 A、glossary/full Review 和当前 validator/adapter/policy 在同一 package scope 重算匹配；stale/unknown、非 validation 的 affected、缺失/冲突 successor、wrong predecessor、hash/identity/scope drift 均 fail closed。Unit scope 自动取完整 affected set；可重复的 `--unit-id` 只作 exact-set/order 约束，不提供额外 authority。同 kind、最多60，不复制 initial batching，也不 whole-package invalidation。

只有 Coordinator 另行明确授权后，维护者才可在同一命令增加 `--record`；重复 ID 拒绝。它冻结原 evidence bytes，不改旧 A、Snapshot、finalization、authorization 或 target。之后仍用上文现有 `retranslation export --content-package <package> --previous-snapshot-id ... --surface-reopen-id ... --allow-reexport --id ... --unit-kind page`，exact-match receipt 全集。此 current-validation export 的正式调用须显式、只提供一次 package：`tour-v1` 保留 canonical Tour-native default complete Catalog，不伪造 registration；registered packages 仍取完整 descriptor，任何已有 partial/different explicit scope 都不可 widening。shared lifecycle dispatcher 消费 `--content-package`，`parseRetranslationExportOptions` 不拥有此 flag；选择后仍由原 current receipt reader 验证 package/predecessor/完整 affected set 与当前 authority。后续 Generation Bundle export/current-check 使用同一 complete selector，且是独立 invocation；详见 [Generation Bundle transport](TRANSLATION_TASK_SPEC.md#deterministic-generation-bundle-transport)。

manifest 的 `revision_feedback_source=current_validation` 保留 machine finding 语义；Generation 只修该 Unit 的 exact affected comment bodies/其他列明失败和局部必要上下文，禁止整页重译。active validation、独立 re-QC、A-only 和 successor gates 保持不变。完整历史/current reader 边界见 [Current Translation Authorization](CURRENT_TRANSLATION_AUTHORIZATION.md#current-validation-的-affected-only-reopen-authority)。

对于 exact teaching-comment carry，Generation Bundle 在核验历史 raw/candidate 后生成 finding-authorized、candidate-specific transport 投影，令授权 body 在 `formal/protected-candidates` 中可编辑；其它区域仍 protected，原 raw 另存 `formal/historical-protected-candidates`，mapping 位于 `formal/comment-revision-transport`。import/process/retry 和后续 candidate evidence reader 共用同一 response decoder，source/input/Unit ID/reopen/batch manifest 不变；active validator 与独立 re-QC 不省略。若旧 ZIP 无法表达 finding 且没有正式 output，保留旧 ZIP、修复 transport contract 后重导同 batch，不重新 reopen/export revision、不新建 batch；见 [Generation Bundle contract](TRANSLATION_TASK_SPEC.md#deterministic-generation-bundle-transport)。

### reviewed A 的 representation-only successor reconciliation

这一路径消费旧正式 QC A，复用同一 replacement resolver、source/input identity、frozen-origin/attempt、current validator、Candidate selector 和 QC lineage。它不是 language revision、retry、Surface reopen 或 glossary-stale recovery。显式 package 必须是完整 registered scope；formal impact 只提供工程 mapping eligibility，不能单独授权 A 或绕过其它 gate。

维护者先执行只读 preflight；下例为 qc-014 的完整参数合同（不同 locale 替换 locale/package/ID/path/SHA，不改变 flags）：

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check representation-reconcile \
  --content-package learn-docs-v1 --locale zh-CN \
  --reconciliation-id learn-docs-qc014-machine-successors \
  --previous-snapshot-id learn-docs-v1-qc-014 \
  --previous-snapshot-sha256 234a37330a348590ab3b46e30d720ad42718dc798cb3002e212b86ce47acb619 \
  --formal-impact docs/PROTECTED_REPRESENTATION_FORMAL_IMPACT.json \
  --formal-impact-sha256 8fb332e4ba2b8be55de91c057a3f174ac4bb922536c90e9d4e927e21e6b74e0e \
  --historical-documents docs/protected-representation-evidence/qc014-historical-representations.json \
  --historical-documents-sha256 e119897a3822d9abc42360c46889ac08c850c870f00ad51f250e3b71e91162e4
```

默认无写入，输出逐 Unit `ready / excluded-current-passed / blocked` 与完整 planned receipt。CLI 在 blocked 非零时退出失败；plan 不是正式 receipt，也不授予新 candidate。命令不接受手选 `--id` 来绕过 package exact-set。已有 current passed successor 自动排除；已有 failed/stale/current export 则拒绝覆盖。非 representation impact 的 Unit 不进入集合；尚未 revision 的 language defect 拒绝。`unknown=true` 只有在 adapter 重算唯一、字节不变的机器 proof 后才可在新证据中标记 resolved，原 impact 文件不被改写。缺失 literal、额外重复 literal、source/provenance/URL/binding drift、glossary affected/unknown 一律拒绝。

仅在全部 pending proof PASS、无 unresolved UNKNOWN、scope 已确认且维护者授权时，同一命令才可启用 `--apply`。apply 原子发布 `data/translation-unit-reconciliations/<locale>/<id>/receipt.json` 及其 bound raw/candidate/machine-validation；不写 `retranslation-runs`，不新增 batch、model provenance 或 translation attempt，不覆盖已有目录。Snapshot 中逐 Unit 增加显式 `representation_reconciliation` path/SHA，并保留真正旧 batch/attempt；selector 不按路径猜 candidate，有任何 current processed result 时优先保留它，包括失败结果，禁止退回机器副本。

current reader 重算 old/current protected input、原 raw restore、原 attempt/provenance、逐字节相同的 target、当前结构/机器 identities 与 glossary validation。QC 使用 receipt 绑定的原 Snapshot、old/current ID mapping 和真实独立 A 来源 carry-forward；不写新语言 A。已有真实 B/C/D revision successor 仍需 independent re-QC。后续 Snapshot 仍走既有 `quality-check snapshot --content-package <package> --locale <locale> --snapshot-id <new-id>`；新的 machine evidence 不能豁免未参与本 scope 的 glossary-stale Unit。

Shell Page 使用同一流程与 receipt schema。`--historical-documents` 也可直接绑定现有 `go-learning/shell-source-successor/v1` foundation 的 path/SHA；reader 必须验证 exact current adoption，并从该 immutable authority 读取完整 old Page，而非手工生成旧表示。`ShellPageAdapter` 保留 frozen slot encoding 与自己的 restore/结构校验，复用当前 source-owned slot/inline binding reader；原 raw 与 current raw 必须还原为逐字节相同的 reviewed candidate，且经过当前 glossary/validator。任何 source、binding、package/adoption 或历史 A/provenance 不匹配仍拒绝。

当前 qc-014 preflight 为44 impact：047排除1、ready41、blocked2；43 pending 中 UNKNOWN 只关闭 `managing-dependencies` 的唯一 `404` literal，json 与 vuln/database 仍 unresolved。**当前不提供 bulk apply 命令，不创建 qc-015，不启动 glossary-stale recovery。** 下一步先取得 json 的正式语言 authority 决定，并关闭 vuln/database 的唯一 binding proof 缺口，再重跑同一只读 preflight。完整 disposition 和 targeted proof 见 [replacement 合同](TRANSLATION_UNIT_REPLACEMENT.md#representation-only-machine-successor-contract)。

### representation reconciliation 的 Independent Reviewer semantic transport

仅在确定性 proof 不足且能绑定一个完整旧 prose 区间时使用；不复用普通 QC rating，也不启动 Generation。机器已能唯一证明的 mapping 不送语言审核；重复 target-site 的任意选择仍 fail closed。详细 authority / receipt 边界见 [replacement 合同](TRANSLATION_UNIT_REPLACEMENT.md#unknown确定性机器证据或独立语义-authority)。

以下为真实 json 问题已执行的完整 Bundle export 合同。其它 locale 只替换 identity/path/SHA 值；flags 不变。`--output` 必须是不存在的文件，禁止覆盖旧 transport。**当前该 ZIP 已存在，不重放此 export。**

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check representation-review-bundle \
  --content-package learn-docs-v1 --locale zh-CN \
  --reconciliation-id learn-docs-qc014-machine-successors \
  --previous-snapshot-id learn-docs-v1-qc-014 \
  --previous-snapshot-sha256 234a37330a348590ab3b46e30d720ad42718dc798cb3002e212b86ce47acb619 \
  --formal-impact docs/PROTECTED_REPRESENTATION_FORMAL_IMPACT.json \
  --formal-impact-sha256 8fb332e4ba2b8be55de91c057a3f174ac4bb922536c90e9d4e927e21e6b74e0e \
  --historical-documents docs/protected-representation-evidence/qc014-historical-representations.json \
  --historical-documents-sha256 e119897a3822d9abc42360c46889ac08c850c870f00ad51f250e3b71e91162e4 \
  --id b49d8e873a0f66f6803077eb534583a155d2bd209a032d7c00fba8fea856da5c \
  --review-id 20261016-zh-CN-learn-docs-json-reconciliation-001 \
  --output /tmp/zh-CN-learn-docs-qc014-json-representation-review-v2.zip

go run -mod=readonly ./cmd/tour-i18n quality-check representation-review-current-check \
  --content-package learn-docs-v1 \
  --bundle /tmp/zh-CN-learn-docs-qc014-json-representation-review-v2.zip
```

把该 ZIP、命令输出的 identity 与 SHA 交给独立普通 ChatGPT Reviewer（GPT-5.6 Sol + High）。Reviewer 仅凭 Bundle 的 `REVIEWER.md` / `question.json` / 全部 inventory inputs 判断。json 唯一问题是：旧 reviewed `自定义编组与解组` 能否在当前 source 的 Marshal / Unmarshal 实际上下文中保持可信对应、全部字节不变；不得混同 API identity 与普通概念。输出 `decision-template.json` 对应的严格 JSON：从 manifest 复制 Bundle identity、从 terminal handoff 复制 ZIP SHA，填写真实 session、独立性声明、唯一决定与 exact finding/reason。不得输出译文或 A/B/C/D。

**本次停止在 Reviewer invocation 前。** 以下是获得真实独立决定后、维护者另行执行的记录模板，当前禁止执行：

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check representation-review-record \
  --content-package '<package>' \
  --bundle '<current-checked-reviewer-bundle.zip>' \
  --evidence '<exact-independent-reviewer-response.json>'
```

record 只写独立语义 authority，不做 reconciliation apply；重复 record 拒绝。随后在本节前面的完整 `representation-reconcile` **只读** 命令中增加 `--language-authority-review-id '<review-id>'`，显式消费 exact immutable receipt。批准仅解除 json 的 unknown，不代表其它 blocked Unit 已关闭，更不绕过 glossary compatibility。要求 revision 则继续阻塞 machine apply，保留 exact finding，后续另行走正式 revision authority，不把它改写为 B/C/D。vuln/database 尚无唯一 machine mapping，当前无 bulk apply 命令；module-conclusion 的独立 glossary-stale 仍不属于本 scope。

### representation-unbindable 的 machine-authorized exact revision

前节记录的是 transport 初次交接。当前 json semantic authority 已真实记录，不重放其 review/record，不修改 receipt。最新完整 preflight 为 **42 ready / 1 excluded / 1 blocked**；唯一 blocked vuln/database 的重复 A/B 不符合 semantic Reviewer 例外。其不同 binding 虽然 restore 字节一致，但尚不能证明固定 target sites 的 protection / source-role assignment 与后续 exact revision locality 完全等价；保持 `unknown=true`，不进行任意 canonical selection。

对于这类“旧 A 无语言错误结论、current representation 无法唯一绑定”的情况，以下命令**默认只读**，在同一 replacement/reopen 生命周期重算 exact machine revision authority。所有 locale 替换相同 identity/path/SHA 参数，不新建另一种 batch 或 language rating。

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check representation-revision \
  --content-package learn-docs-v1 --locale zh-CN \
  --reconciliation-id learn-docs-qc014-machine-successors \
  --previous-snapshot-id learn-docs-v1-qc-014 \
  --previous-snapshot-sha256 234a37330a348590ab3b46e30d720ad42718dc798cb3002e212b86ce47acb619 \
  --formal-impact docs/PROTECTED_REPRESENTATION_FORMAL_IMPACT.json \
  --formal-impact-sha256 8fb332e4ba2b8be55de91c057a3f174ac4bb922536c90e9d4e927e21e6b74e0e \
  --historical-documents docs/protected-representation-evidence/qc014-historical-representations.json \
  --historical-documents-sha256 e119897a3822d9abc42360c46889ac08c850c870f00ad51f250e3b71e91162e4 \
  --id a69694fb9c89d47440249dfd760d315586297f634d199b13532ac95cbced201f \
  --reopen-id learn-docs-qc014-vuln-database-representation-revision
```

输出 `recorded=false / mutation=none`，含 exact old/current replacement、Snapshot/A lineage、raw/validation/provenance/hash、machine finding、scope 与 source-only scaffold SHA；没有 candidate/model output 或新语言判断。该命令不消费 json 的 semantic receipt 来授权 vuln/database，也不接受 `--apply`、Surface/Preview/semantic finding 或自填 rating。

**当前停止在 formal revision 前。** 未来只有维护者确认这份 inspection 并明确授权时，才在同一命令末尾增加 `--record`。它仅写 `data/quality-check-reopens/<locale>/<reopen-id>.json`，`authority_source=representation-machine`；重复记录、缺 proof/hash drift、wrong Unit/source/Snapshot/A 或已有 successor result 均拒绝。随后使用上文同一个 successor exact revision export 模板：`--previous-snapshot-id`、`--surface-reopen-id`、`--allow-reexport`、exact current `--id`、`--unit-kind page` 和新的未使用 batch ID。不得编造 QC B/C/D，不插入 initial language QC；原 A 是 predecessor lineage，不是新语言结论。

新的 Generation Bundle 明确把 `formal/protected-candidates` 标记为 **INCOMPLETE revision scaffold**：只有 exact ambiguity interval 使用 verbatim source，区间外保留旧译文。真实旧 reviewed candidate / raw 独立冻结供对照；Generation 必须在 finding locality 内保留完整原意并消除绑定歧义，使用 normal current tokens 输出。scaffold 没有 Generation provenance 或 accepted candidate 资格，不能用机械复制冒充完成结果，也不得由工程端补中文。之后仍需真实 Generation provenance、restore、current validation、independent re-QC；新增候选不继承旧 A。细节与 fixture 见 [replacement 合同](TRANSLATION_UNIT_REPLACEMENT.md#representation-unbindable-的机器-finding-与-exact-revision)。

vuln/database 完成该正式 revision 后，bulk machine reconciliation 将因其已有 current passed candidate 自动排除它；其它42 ready Unit仍未 apply，json authority继续保留，module-conclusion glossary-stale依然走独立流程。当前不创建 qc-015、不执行以上 record/export/Generation 或 bulk apply。

本轮已测试的只读命令也保存为 `bash docs/protected-representation-evidence/vuln-database-next-canonical-command.sh`。真实 inspection、exact authority 与冻结历史 observation 的关联见 [engineering report](protected-representation-evidence/vuln-database-revision-closure.json)；该报告及 planned receipt 不能作为已 record 的 authority。

## Same-unit restricted transport → ordinary QC bridge

支持两种 exact predecessor：既有 restricted current-validation comment response，以及合法 imported/passed Surface-bounded response。后者只借用 immutable representation mapping，新的普通 QC B/C/D 与独立 Reviewer evidence 才授权 revision；不扩大旧 Surface regions。机器构造独立 ordinary template 并证明 exact reviewed candidate restore，完整合同见 [Surface→QC bridge](TRANSLATION_UNIT_REPLACEMENT.md#surface-bounded-predecessor--independent-ordinary-qc-bridge)。缺 capability 的既有零输出 batch 不变；bridge record 后使用新 batch，不能 retrofit/replay。此次 currentness/construction semantics 扩展使旧 first-real-use verification 失效；下一次真实 preflight 前须核对 current docs、implementation、direct CLI/contract tests，preflight PASS 不自动授权 record。

formal Unit/source/input 均未变化时，不使用 replacement reconciliation。先由 Maintainer Local terminal 执行默认 readonly preflight，explicit package 必须与 predecessor 完整 registered scope 一致；`--unit-id` 列出同 kind、最多60的 exact B/C/D affected scope，不按 route 选择，不合并逐 Unit finding：

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check representation-bridge \
  --content-package <registered-package> --locale <locale> \
  --bridge-id <immutable-bridge-id> \
  --previous-snapshot-id <reviewed-snapshot> \
  --previous-snapshot-sha256 <exact-manifest-sha256> \
  --reviewer-evidence <repository-review-evidence-path> \
  --reviewer-evidence-sha256 <exact-evidence-sha256> \
  --unit-id <same-formal-unit-id> [--unit-id <another-id> ...]
```

输出 `recorded=false, mutation=none`，包含 planned receipt path/SHA、完整 source scope、逐 Unit old/new representation identity、同一 candidate/validation/attempt 和 exact rating/finding。preflight 不创建 receipt/batch/candidate，不授权自动 record 或继续 Generation。preflight PASS 后，只有单独明确授权才使用**同一组参数追加 `--record`**；实现重新证明全部 authority，write-once 保存到既有 quality-check-reopens，重复 ID 拒绝，无 `--apply` 或隐式覆盖。

record 以后新 ordinary revision 的原 export 模板额外增加 `--representation-bridge-id <bridge-id>`；仍须 `--allow-reexport --previous-snapshot-id <same-snapshot>`、完整 package 和 receipt 全集的显式 `--id`，不能混用 `--surface-reopen-id` 或 `--glossary-stale`。batch ID 由届时正式 authority 确定；不改写失败的旧零输出 batch。feedback source 仍为 `quality_check`。后续普通 Bundle/current-check → Generation → import/process → 新 Snapshot/独立 re-QC 不变；retry 继续冻结相同 bridge authority/template/history。完整 invariant 与不支持场景见[bridge contract](TRANSLATION_UNIT_REPLACEMENT.md#同一-formal-unit-的受限-transport-bridge)。

### Current-validation predecessor representation readiness

独立 [`generation-bundle legacy-provenance` authority](LEGACY_GENERATION_PROVENANCE_AUTHORITY_DESIGN.md#正式-implementation-command-contract) 已实现；维护者已完成 json exact authority 的 first-real-use、inspection 和 write-once record，正式 path/SHA/identity 见 [approved-interval contract v2](CURRENT_VALIDATION_APPROVED_INTERVAL_MIGRATION_DESIGN.md#已记录-authority-与当前-case)。直接消费该交接，不重复 record，也不把下一阶段退回 legacy first-real-use。正常 lifecycle 执行者仍为 Maintainer Local terminal；`.pending` 或 `mutation=unknown` 继续 preserve/diagnose，禁止盲重放。

approved-interval representation transport 的 [contract v2 已实现](CURRENT_VALIDATION_APPROVED_INTERVAL_MIGRATION_DESIGN.md#derived-transport-command-contract-v2已实现真实使用待-gate)，真实使用仍待独立 gate：historical provenance 按 `NORMAL_ARCHIVE > EXPLICIT_RECORDED_LEGACY_PROVENANCE`；缺 archive 才可显式提供 exact recorded path/SHA、调用正式 verifier 并传 exact requested options，不扫描 latest 或按 Unit 自动发现。只消费 `historical-response/v1` allowlist；需要 explicit unknown facts 时停止。archive 后来出现必须 fresh compare covered facts，冲突 fail closed、匹配优先 normal，不删除 legacy receipt。legacy PASS 不授权 opaque parent、editable regions 或 response codec。

显式 selection、legacy refs 与 approved-interval proof 全部冻结在 derived Bundle graph，不修改064 manifest；`SAME_BATCH_064_REFRESH_POSSIBLE=yes` 是安装前的架构可能性，仍不代表 lifecycle authorization。whole parent 不可编辑、不拆 target atoms，typed finding nonoverlap、active validation/current glossary、独立 re-QC/A-only 均保留。原 migration 在 `current-validation-approved-interval-first-real-use-gate` 前不授权真实 selection、064 export/current-check 或 Generation；不因此重做后来已经完成的 invocation。current-check 不依赖 external selection 文件继续存在；后续 process/retry/Snapshot 沿真实 initial import archive 的 frozen request 重构 proof。retry 必须继承同一 request，不允许换 authority 或省略 opaque protection。

上述same-batch可能性只覆盖尚未installed的transport刷新。installed approved proof出现
implementation/dependency drift时，Snapshot 可仅按第6节的完整 historical-PASS admission
重新证明 eligibility；旧 proof 不因此变成 CURRENT。现有re-export、retry、revalidate和换
reopen ID的拒绝边界不变。确需新的 invocation 时，后续工程遵循
[Installed derived-transport stale recovery设计v1](INSTALLED_DERIVED_TRANSPORT_STALE_RECOVERY_DESIGN.md)：
恢复整个原batch decoder context，重新进行该完整scope的Generation，不carry其中未直接
drift的candidate；batch外仍走普通currentness/A-carry。实现的synthetic verification不能据此授权
真实record/export、创建successor或重放import；必须先独立first-real-use gate。

找到 exact historical implementation 副本不等于恢复资格已成立。外部 bytes 的未来正式采用
以 [Historical construction bytes formalization](HISTORICAL_CONSTRUCTION_BYTES_FORMALIZATION_DESIGN.md)
为独立合同：先 implementation/first-real-use，再 preflight/record/check supplemental evidence；
当前不得直接从 `/tmp` fallback。采用只解决 byte availability，后续 transport recovery 仍须
独立 fresh preflight；不得改旧 archive/receipt/proof 或据此创建 Snapshot。

已 record 的 current-validation revision batch 遇到 predecessor raw restore failure 时，先按完整 batch stable order 做只读 representation readiness diagnosis；impact report 的 labels 只是工程 evidence，不是 recovery authority。普通 restore PASS 不需新 authority；sealed Surface/inherited bridge 可按 [predecessor transport](TRANSLATION_UNIT_REPLACEMENT.md#current-validation-predecessor-representation-transport) 派生 Bundle-only representation capability，保留原 reopen/A/finding 与 batch bytes，不建立 QC bridge 或另一 lifecycle。exact raw/attempt/import/codec、current formal input/source/Unit 或 candidate 无法证明时停止；不得从 target prose 猜新增 token。新 transport acceptance/currentness semantics 的第一次真实使用必须重新经过 docs → implementation → direct contract tests；本工程阶段不授权真实 export/current-check/Generation，也不自动 record recovery。
