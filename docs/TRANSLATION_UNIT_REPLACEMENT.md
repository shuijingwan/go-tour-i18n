# 正式 Page replacement lineage

本合同复用[共享生命周期](SHARED_TRANSLATION_UNIT_LIFECYCLE.md)和既有 Surface reopen、revision、Generation Bundle、process、Snapshot、QC 与 A-only finalization。Page ID 算法保持不变；protector representation 真实变化仍产生新 ID，旧 Bundle 自然 stale。

## 必要性与最小扩展

历史 multi-segment lineage 明确区分 old A、reconciliation-event current B 与 today-current C。historical machine reader 仅在 source provider 已验证完整 canonical B→C authority 后采用事件 B 的冻结 catalog，继续精确验证原 A→B receipt、package mapping、protected input、candidate/validation 与 glossary world；普通 current reconciliation 不采用此 historical override。当前 Shell pointer 选择唯一正式 chain；未被 pointer 选中的 orphan receipt 不授予分支 authority。所有被采用的 successor/re-attestation 引用进入历史 audit proof，每次独立 invocation 重新验证，无跨 invocation world 复用。finalize 的完整命令/currentness/恢复合同见 [Quality Check](TRANSLATION_QUALITY_REVIEW.md#finalize-的历史-representation-successor-合同)。

既有 exact-source mapping 只能证明物理 source，不能证明旧 protected ID 与当前 ID 的关系；旧 reopen 和 batch 又只能绑定 predecessor Snapshot 的原 ID。仅忽略 ID mismatch、按 path 选择 latest candidate 或改写旧 batch 都会失去授权证据。因此在现有 schema-v1 reopen 上增加可选 `replacement_context` 和逐 Unit `replacement`，作为新的 immutable successor reopen；不新增独立 receipt/schema 或状态机。旧 single-finding、aggregate 和无 replacement revision 原样读取。

`replacement_context` 绑定原 reopen 的 repository path/SHA 和显式 frozen Page representations 的 path/SHA。冻结文件为 `HistoricalPageRepresentations` JSON，包含 `package`、`package_sha256`、完整有序 `documents`（含原 formal Unit、internal slots、protected raw/role/pair）。必须提供真实保存的旧表示；不得从 candidate、path、stable index 或最新文件猜测旧 token mapping。缺少完整旧表示时 fail closed。该输入进入 successor 和 Bundle identity，文件改动会使 current-check 失败。

`ReconcileHistoricalPages` 使用既有 `ReconcileDocuments` 的同一确定性 matching 实现。历史表示通过 frozen slots 重组、protected ID、source hash 和源坐标验证，不用当前 protector 要求它重新变为 current 表示。当前表示仍通过当前 parser 重建。双方必须为同 package authority、同 kind、同 physical path/source bytes/SHA、同 route、同 registry index、同 dependency identity，各一个 formal Page；完整 membership 和顺序一致。重复、歧义、移动、源字节变化或伪造 slots 均拒绝。旧 ID 与 current ID 不同时必须有真实 protected-input drift 和唯一 reconciliation replacement；不授予旧译文或 A 的自动继承。

## successor authorization

维护者之后明确执行真实恢复时，使用既有 `quality-check surface-reopen`，另外指定 `--previous-reopen-id` 和 repository-relative `--historical-documents`。必须使用新的 reopen ID，并显式给出 current UnitIDs、原 Snapshot、原 Surface review 和完全相同的 selected findings。此文档不代表已经创建任何真实 successor。

reader 先使用显式 historical inventory 验证原 finalized Snapshot、原 reopen、独立 Surface receipt/session/context、candidate、validation 和原 UnitID，再重算 reconciliation。历史 artifact 验证与 current artifact 验证分开；`ValidateSurfaceCandidateArtifact` 仍拒绝旧 ID 冒充 current。每个 replacement 绑定 old/current ID、package authority、path/source SHA、registry index/route、old/current input SHA 和 reconciliation SHA。旧 input 还须匹配原 Snapshot 所选 batch 的正式 input hash。混合 unchanged/replacement scope 沿用同 predecessor、同 Page kind、最多60、exact selected findings 和一个集中 revision batch。新授权不覆盖旧 evidence；已有新正式结果时不能再次用旧授权建立 batch。

历史 finalization 校验 Surface replacement event 时，`HistoricalCatalog` 是 context 绑定的 frozen old side，`resolution.Units` 则是 old → 调用者当前 Catalog，不能拿后者要求历史 unchanged 永久不变。事件本身按原 predecessor reopen、Snapshot、source 和 saved input 验证 unchanged 或显式 replacement；若事件之后又有 representation 变化，必须由当前 finalization 已完整验证的 reconciliation successor 精确衔接 event-time ID/input 与 current ID/input。缺 authority、错误映射、source/input 或 candidate/validation 篡改仍拒绝；不改写历史，不猜 mapping，不新增 schema 或 receipt。

## revision 与 QC

新 batch 只记录 current ID，immutable successor reopen 显式解析其 historical predecessor ID。Generation Bundle 冻结 current source/input、old Snapshot/candidate/validation、原 Surface authority、原 reopen、successor reopen 和 frozen representations。对 replacement，`formal/historical-protected-candidates` 保存原 passed raw response；`formal/protected-candidates` 是通过源坐标确定性转换的当前 token baseline。转换仅允许把旧 source atom 中的内容重新表示，不能从译文推断 token；转换后必须恢复为逐字节相同的 reviewed candidate，否则失败。未变化 Unit 的 baseline 原 bytes 保持不变。

current-check 重新验证整个 bridge 和 derived baseline。Generation 编辑 current baseline，仍提交完整 protected raw response；process/validation/新 Snapshot 使用 current ID。replacement 不继承旧 A，必须独立 re-QC；finalization 通过所选 batch 的显式 successor lineage 验证历史 finalized predecessor，旧 finalization 不改写。

## 已生成输出恢复

复用既有 `generation-bundle recover`。successor batch ID/聚合 identity 改变不要求重做 exact compatible 的 unchanged Unit；同 ID/input、同 predecessor candidate 和原 finding semantics 才构成该授权的复用范围。恢复仍逐 expected filename、protected restore、机器校验，并拒绝不同有效 bytes、覆盖和已有正式导入状态；provenance 仍只由 import 建立。旧 replacement ID 的输出不会映射到新文件名，必须 missing。partial ZIP 与既有 staging/tar 路径共用同一逐 Unit preflight；ZIP 成员完整性错误保留为诊断，不改写原 ZIP。

## QC B/C/D 的 representation successor authority

同一个 schema-v1 replacement reopen 增加可选 `authority_source=quality-check`。它的 `replacement_context.quality_check` 精确绑定原 Snapshot manifest、QC results 及工程必要性 proof 的 path/SHA；不填 Surface、Preview、旧 reopen 或 finalization authority。原 Snapshot 可以尚未 finalized，但初审必须完整，所选项必须是 identity-matching 的 B/C/D，并有原始 finding。逐 Unit 保存原 rating、finding、完整 Snapshot candidate/validation/input identity 及原有 replacement mapping。工程 proof 的 `reason=protected-natural-language-unrepresentable`、rating、finding SHA 和完整 replacement 必须一致；没有真实 protected-input/Unit identity drift、scope 不符或已有相同 revision output 时拒绝。

CLI 复用 `quality-check surface-reopen`，新增 `--authority-source quality-check --previous-rating B|C|D --representation-proof <repository-path>`，同时要求 `--historical-documents`、原 `--previous-snapshot-id`、exact `--finding` 和 current `--unit-id`。不同 findings 按 stable index 重复传入。工程 proof 是工程必要性的声明及机器 mapping 证明，不提供任何译文或新语言结论；不得仅凭实现文件变化建立它。

普通 Page 的 heading navigation label 使用 source tokenizer/heading role 判断可见正文，不能仅因前一个词也可作 symbol 引导词就把后续普通 label 变成 opaque atom。tag、href、id、非语言 attributes、URL、显式 code/API 与 link binding 保持原保护。修正这类 source-owned classification 若产生真实 input/UnitID drift，继续使用本节 QC B/C/D successor；不能借用 restricted-comment same-unit bridge，也不能让模型重建 mapping。新的 baseline 由原 source coordinates 对旧 passed raw 做确定性 rebase，必须精确 restore 到原 reviewed candidate；原 finding/rating 不变，零输出失败批次保留。

工程只读 proof 可调用 `InspectQualityCheckRepresentationReopen`，它与正式 record 共用完整 QC/replacement、candidate/validation、exact finding 和 unconsumed-output 验证，但不写 receipt；已存在的 receipt ID 拒绝。CLI `quality-check surface-reopen --authority-source quality-check ...` 仍是正式 write-once record，不能把它当 preflight 执行。initial adoption 的 frozen Page inventory 可作为完整 historical documents 来源，须保持其原 hash/identity/source coordinates；当前 representation 另由 current parser 重建，不改写原 source adoption。

reader、Bundle、retry、revision predecessor 与 QC 都复用已有 replacement 路径；新的 manifest feedback source 为 `quality_check_representation`。Generation Bundle 同时携带原 QC authority、工程 proof 和 frozen representations。新 Unit 的 revision 必须消费同一个原 B/C/D finding；不得继承旧 A/B 作为新结论，仍需新 validation 与独立 re-QC。旧 Surface / Preview reader 和无 replacement 的普通 B/C/D revision 保持原分支。

此工程实现不自动执行真实 reconciliation 或 export。SYS-01～08 已收口；当前 `chatgpt-zh-CN-046` 永久保持原 manifest/input、0 output、无 import/process。合法链为：046 immutable preserved → representation reconciliation → successor Unit representation → 消费同一 qc-014 B finding 的新 revision batch → validation → independent re-QC。不存在 same-batch refresh。receipt 建立后的完整命令统一维护于 [Retranslation Runbook 的 successor export 模板](RETRANSLATION_RUNBOOK.md#qc-bcd-representation-reconciliation--successor-exact-revision-export)。

### Restricted transport 不等于 formal representation replacement

ordinary QC revision 的 editing template 必须能由该新 response 的既有 decoder 还原为 reviewed candidate（仅沿用现有 artifact EOF framing）。Generation Bundle export/current-check 在历史 raw/candidate 验证后执行此机器检查；不分析 finding 的自然语言，不根据关键词选择 locality，不重新保护 candidate，也不替模型建立 token mapping。restricted current-validation comment response 不能仅因曾通过自己的专用 decoder，就被当作普通 QC 的可执行 baseline。拒绝只阻止不可执行 transport，不产生 reopen、replacement、rating 或语言结果。

comment revision projection 是 internal transport，不是 formal Page representation。若 source-owned formal Unit ID 和 protected input 均未变化，现有 `authority_source=quality-check` replacement 分支要求的 nonidentity mapping 就不成立；不得从 projection token 或 candidate 拼造历史 Page 表示、伪造新 Unit ID、放宽 exact-set 或把 source-only scaffold 的其它 authority 借给普通 B/C/D。此时必须停止，先明确建立共享的 transport reconciliation authority/binding contract；没有该 authority 时不提供 same-batch refresh 或 successor export。历史 candidate、QC、finalization、失败的零输出 invocation 与 ZIP 保持原样。

既有 receipt schema 在逐 Unit 字段保留独立 rating/finding，但当前 QC replacement record/CLI 要求单一 `--previous-rating`，可按同 rating 聚合多个 exact findings。混合 rating 不能强行塞入同一次 record；按现有合法分组处理，不能为减少 receipt 数而改写 finding 或 rating。上述分组仍须先满足真实 formal replacement 的全部必要性条件。

### qc-014 successor export failure diagnosis

2026-10-16 的真实失败发生在 batch 写入之前；`data/retranslation-runs/zh-CN/chatgpt-zh-CN-047` 不存在，两次失败 attempt 均无 artifact mutation。receipt 的 quality-check authority 本身有效。未选择 package 的 CLI inventory 被原 replacement resolver 原样带入 historical Catalog，导致 QC reader 使用默认 Tour authorization namespace，遗漏 Learn/Docs 已封存的 A carry-forward。只读复现得到 effective=14、expected=49、historical_scope=0；显式选择 `learn-docs-v1` 后，同一 receipt 通过完整 gate。不能将该 14/49 解读为真正缺少 initial independent QC。

修复位于共享 `replacementResolution`：仅对 quality-check authority 且调用者无 scope 的分支，先验证 frozen source/package/registry/input 全部 replacement bindings，再从唯一 package descriptor 恢复完整 historical scope。export、Bundle 和 retry 使用同一个解析结果。显式 scope 不被覆盖，完整初审计数、原 finding/rating、engineering proof、candidate/validation/input 和 unconsumed-output gate 全部保留。没有新增 machine materialization step，也没有绕过初审或引入第二种 receipt。

修复后的真实只读 probe 在 unselected / `learn-docs-v1` 两种 inventory 均通过，原 receipt bytes 不变、047 仍不存在（两次 preflight 合计45.60s）。正式 export 仅在 `t.TempDir` synthetic frozen fixture 执行：完整 package 的五个 A 来自 sealed checkpoint，另一个 candidate 经过 revision 后分别得到 B/C/D，随后在显式 package 和 unselected inventory 导出 successor，并检查 Bundle 与 manifest 的 exact finding/rating/current ID。旧 ID、错误 Snapshot/receipt/finding、缺 engineering proof、A authority 和显式 partial scope 均 fail closed；真正缺初审的 ordinary revision 仍被初审 gate 拦截。CLI canonical command 通过与 `run` 相同的 global/subcommand parser 验证。本轮未进行真实 export、Generation 或任何正式语言 artifact mutation。

`docs/protected-representation-evidence/qc014-next-canonical-command.sh` 保留为上一阶段建立 receipt 的原命令证据；该动作已由维护者完成，不再次执行。当前唯一下一步是上述 Runbook 模板的 successor revision export，执行者仍为维护者 Local terminal。

本轮 targeted tests exact set（全部 PASS，不运行 package-wide suite）：

```text
internal/sitecontent:
  TestQCRepresentationSuccessorExportWithSealedCarry
  TestQualityCheckRepresentationReconciliation
  TestQCRepresentationExportReadonlyProbe (GO_TOUR_QC_RECONCILIATION_PROBE=1; read-only)
internal/i18n:
  TestOrdinaryRevisionStillRequiresCompleteInitialQC
  TestRevisionGenerationBundleOrdinaryRatings
  TestSurfaceReviewFindingCanExplicitlyReopenFinalizedAWithoutChangingHistory
  TestPreviewRevisionGenerationBundleInfersFrozenPackageScope
cmd/tour-i18n:
  TestQCRepresentationSuccessorCanonicalCommand
  TestParseRetranslationExportOptionsGenerator
```

B/C/D sealed-carry export fixture 最终27.58s；原 reconciliation 完整 fixture7.77s；四个 i18n regressions 合计1.430s；两个 CLI parser tests 合计0.017s。以上均为 focused 验证实测，不是完整 lifecycle 性能基准。

## representation-only machine successor contract

047 后的真实 Snapshot failure 证明既有 replacement 路径只为有 finding 的 revision 提供 baseline；formal impact mapping 本身没有发布 current candidate。原 selector 只扫描 processed retranslation batches，43个 reviewed A 的 representation successor 因此显示 missing。不能用43次 Generation 或简单改 Unit ID 复制候选来填补这个 machine evidence 缺口。

新增共享入口 `quality-check representation-reconcile`，规范命令唯一维护于 [Runbook](RETRANSLATION_RUNBOOK.md#reviewed-a-的-representation-only-successor-reconciliation)。它依次绑定 package、Snapshot manifest SHA、formal impact path/SHA、完整 frozen old representations、逐 Unit old/current source/package/registry/input mapping、真实原 candidate/validation/final attempt/raw、原 batch manifest/result/provenance、实际原 QC A、派生 current protected raw、当前 machine validation 与 glossary。新 candidate 必须与 reviewed predecessor **逐字节相同**；没有翻译、语义改写或自动 A record。current passed successor 排除，current failed/stale/exported successor 拒绝覆盖，不能绕过更晚 attempt 回退。

成功 apply 只在现有生命周期新增一种明确的 machine evidence。独立目录原子发布 receipt 与 artifacts，selector/validation/Snapshot/QC 显式消费其 path/SHA；translation batch、provider/model/session 和 attempt 仍来自原 invocation。原 Snapshot/raw/candidate bytes 不改写，旧 reader 的无该字段路径不变。current selector、progress、glossary context、Reviewer/revision transport 与历史 A lineage 都区分 machine evidence 和普通 processed attempt。重复 apply、冲突 successor、未完成 transaction、任意 dependency/hash drift 拒绝。

原正式 A integrity 与当前语言输入资格分开：从已有 sealed history 读取原 A，不因本次无关 Unit 的 glossary stale 抹去旧判断；每个 machine successor 仍独立检查当前 glossary compatibility、input 和 validator。历史 A 来源保留原 SourceSnapshotID/old Unit，只有显式验证的 replacement mapping 可将它 carry 到字节相同的新 identity。新语言 revision（包括047）不继承原 B 或 A 为新结论。

### 三个 UNKNOWN 的 exact disposition

原 formal impact 文件 SHA `8fb332e4ba2b8be55de91c057a3f174ac4bb922536c90e9d4e927e21e6b74e0e` 保持不变。新的只读 machine proof 不改写旧分类，不制造语言 finding。

| Route；old → current | 实际证据与 disposition |
| --- | --- |
| `/doc/modules/managing-dependencies`；`2a2116ee48edd06f9ddac08c7b54b05422d8a34dbae7387166ea1f9de4686bf0` → `82a1f9b16b20d60a538c1873daf9afb5c446aded1d2bff16e3b5650ed6bafb1e` | 原 source prose ` 404 or 410.` 对应旧 target prose ` 404 或 410 时才会继续尝试。`。在两个已验证旧 token anchor 之间，404 literal 唯一；新 token 可确定性绑定，完整 restore 与 reviewed candidate bytes 相等，current validation PASS。**RESOLVED_MACHINE**；只读计划尚未成为正式 evidence。 |
| `/doc/tutorial/json`；`99824b13c26ffe219230ad0171e93d850003d16aa9f780c2f06c41bc46845cb3` → `b49d8e873a0f66f6803077eb534583a155d2bd209a032d7c00fba8fea856da5c` | 新 protected `Marshal` 位于 source heading `Custom Marshal and Unmarshal`；对应旧 target 为 `自定义编组与解组`，该 prose 区间中 Marshal literal=0。机器不能在保持 reviewed bytes 时插入此 identity，也不能决定改译。**UNRESOLVED，requires language authority decision**；不是工程新建的语言 defect/rating。 |
| `/doc/security/vuln/database`；`2fdb32dbf7afa3b472911994cca882ce4b57e9432a19e731b294306dd9ff405b` → `a69694fb9c89d47440249dfd760d315586297f634d199b13532ac95cbced201f` | 新 source interval 中 A/B 各一个，旧译文同一区间为 `漏洞，包 B 依赖包 A，而且包 A 和 B 分别有不同的 `，A/B 各两个。字面值仍在，但 source-site 到 target-site 对应不唯一。**UNRESOLVED machine binding proof gap**；不选择“第一个/最近一个”，不声称已证明语言错误。 |

`UNKNOWN_TOTAL=3`、`UNKNOWN_RESOLVED_MACHINE=1`、`UNKNOWN_REQUIRES_LANGUAGE=1`、`UNKNOWN_REMAINING=2`；requires_language 是 remaining 的子集，不作互斥相加。真实43 pending 为41 ready +2 blocked；fips140 的047 current passed candidate 自动排除，module-conclusion 没有 representation impact，未进入计划。任何 blocked 都阻止整个 bulk apply，不能悄悄落盘41个子集。

### 定向 regression proof

`TestRepresentationOnlySuccessorPackageLifecycle` 使用隔离49 Page frozen fixture，覆盖44 impact、047真实 fixture import/process后排除1、机器派生43、原A carry与047 pending independent re-QC、immutable/no-overwrite、错误 Snapshot/impact/frozen/package/language/unknown mapping、raw/candidate/validation/QC evidence 篡改拒绝。另一个冻结副本从5 passed /43 missing /1 glossary-stale开始；机器apply后48 passed /0 missing /1 stale，Snapshot仍因该 glossary-stale fail closed，目录不落盘。全部实际调用都发生在 TempDir，没有真实 apply、Snapshot 或 Reviewer invocation。

新 proof 只在旧 prose 的明确 token anchors 之间保护唯一 exact technical literal，支持 identifier boundary，额外或缺失 occurrence 均拒绝，且整页必须 byte-exact；不翻译、不相似匹配、不重排 target。`TestRepresentationNewAtomsDeterministicProof` 覆盖404、Marshal缺失和A/B重复；`TestRepresentationHistoricalAIsSeparateFromCurrentGlossary` 证明 sealed old A 与 current affected 判定分别保留；`TestRepresentationReconciliationCanonicalCommand` 验证完整 CLI、默认只读、必需 path/SHA 和拒绝手选id。保留的直接回归：普通 B/C/D、initial-QC gate、Surface reopen、Preview reopen、原 QC representation bridge、sealed A carry export、glossary byte-only lineage 与 missing/stale/ambiguous evidence。没有重跑 SYS-01～08 taxonomy/corpus audit。

本轮最终工程结论 **FAILED（2个 UNKNOWN 尚未关闭）**。共享能力和 synthetic contract 可用不等于真实 package 已可 apply/Snapshot-ready；当前不提供 bulk apply 命令。047、qc-014、formal impact 和其它正式 artifacts 均不修改；module-conclusion 后续独立 glossary recovery 不属于本轮动作。

真实 exact-set 只读结果保存在 [readonly plan](protected-representation-evidence/representation-only-readonly-plan.json)，`applied=false`，不位于 canonical receipt 路径，不能由正式 reader 当作已 apply evidence。最终 focused timing：preflight本体13.595s（含 registered reader 的 test总18.498s）；49 Unit synthetic contract49.23s，包含独立的后续 revision transport fixture，非单次 workflow timing。7个 i18n targeted regressions 合计4.389s、3个 CLI parser tests0.031s。没有全 corpus、全 package 或全 repository 性能/测试扫描。

本轮测试 exact set：

```text
internal/sitecontent:
  TestRepresentationOnlySuccessorPackageLifecycle
  TestRepresentationNewAtomsDeterministicProof
  TestRepresentationMaterializationReadonlyPlan (explicit read-only probe)
  TestRepresentationUnknownReadonlyProbe (exact 3-route failure diagnosis)
  TestQCRepresentationSuccessorExportWithSealedCarry
  TestQualityCheckRepresentationReconciliation
internal/i18n:
  TestRepresentationHistoricalAIsSeparateFromCurrentGlossary
  TestOrdinaryRevisionStillRequiresCompleteInitialQC
  TestRevisionGenerationBundleOrdinaryRatings
  TestSurfaceReviewFindingCanExplicitlyReopenFinalizedAWithoutChangingHistory
  TestPreviewRevisionGenerationBundleInfersFrozenPackageScope
  TestGlossaryCompatibilityByteOnlyLineagePreservesHistory
  TestGlossaryCompatibilityLineageMissingStaleAndAmbiguous
cmd/tour-i18n:
  TestRepresentationReconciliationCanonicalCommand
  TestQCRepresentationSuccessorCanonicalCommand
  TestParseRetranslationExportOptionsGenerator
```

本轮实际修改文件（保留所有既有 WIP，以下不代表整个 worktree diff）：

```text
cmd/tour-i18n/main.go
cmd/tour-i18n/representation_reconciliation.go
cmd/tour-i18n/representation_reconciliation_test.go
internal/i18n/representation_reconciliation.go
internal/i18n/representation_reconciliation_test.go
internal/i18n/candidate_snapshot.go
internal/i18n/retranslation_latest.go
internal/i18n/retranslation_review.go
internal/i18n/translation_progress.go
internal/i18n/quality_check_results.go
internal/i18n/quality_check_bundle.go
internal/i18n/generation_revision_inputs.go
internal/i18n/glossary_context.go
internal/i18n/glossary_compatibility.go
internal/i18n/retranslation_export.go
internal/i18n/historical_finalization.go
internal/i18n/translation_authorization.go
internal/sitecontent/page_replacement.go
internal/sitecontent/page_replacement_test.go
internal/sitecontent/representation_materialization_test.go
internal/sitecontent/representation_materialization_probe_test.go
docs/RETRANSLATION_RUNBOOK.md
docs/TRANSLATION_UNIT_REPLACEMENT.md
docs/protected-representation-evidence/representation-only-readonly-plan.json
```

### UNKNOWN：确定性机器证据或独立语义 authority

同一 replacement 生命周期允许两种显式 proof；不修改旧 impact、旧 A 或任何 candidate。第一种仍是唯一 source-coordinate / protected-anchor / exact-literal mapping，完整 restore 必须与 reviewed bytes 相同。第二种仅适用于机器不能决定的 **一个完整冻结 prose 区间**：用 `quality-check representation-review-bundle` 提交 Independent Reviewer semantic question。重复 literal 的位置选择不属于这个非 literal 语义例外，不能借 Reviewer receipt 让机器猜第一个或最近一个。

Bundle 复用共享 deterministic ZIP inventory / SHA / strict JSON / current-check 与旧 Snapshot、replacement、原 A reader。它包含 locale/package/route/source、old/current Unit、原 Snapshot/实际 QC A chain、原 source/candidate/validation/final attempt/raw/manifest/result、old/current representation、全部 source-coordinate inventory 与 scoped changed atoms、当前 source/glossary、完整 frozen impact及其 SHA、单项 impact、已建立的 machine evidence、唯一语义问题和决定模板。原历史 batch 没有记录 session 时不补造；独立性由真实 Reviewer session 声明并受已有 locale role collision gate 校验。

Reviewer 仅回答：冻结 reviewed target 能否在当前 source 实际上下文中继续作为可信语义对应，并保持全部 reviewed candidate bytes 不变。必须区分 Go identifier/API identity 和普通自然语言概念。只允许 `APPROVE_RECONCILIATION` 或 `REQUIRE_LANGUAGE_REVISION`，两者均须附 exact finding/reason；不是 QC，不给 A/B/C/D，不生成 replacement。

Local terminal 的 `representation-review-record` 在 current-check、exact identity/provenance/decision/schema 校验后原子发布 `data/representation-reconciliation-reviews/<locale>/<review-id>/{bundle.zip,decision.json,receipt.json}`。receipt 绑定 Bundle identity/SHA、原始 response evidence SHA、locale/package、old/current Unit、Snapshot、原 A/QC input chain、impact SHA、review_id/session/provider/model；已有目录或残留 staging 拒绝覆盖。未知字段（包括 replacement/rating）和非法决定拒绝。不能编辑 JSON flag 或 impact 解锁。

未来只读 reconciliation 显式传 `--language-authority-review-id <review-id>`，不扫描 latest 决定。reader 重读 exact archived Bundle/response/receipt、重算当前性和原 A chain，再派生整个批准区间的 **machine-only semantic projection**；全部区间外 bindings 仍需确定性证明，最终 candidate 必须逐字节不变且通过 current structural/glossary validation。该 projection 及 receipt 是显式 machine evidence，不是 current canonical Generation input 或一次新模型 response。canonical source/input 和一般 protected-token exact/unique 规则保持原义；不能把 scoped semantic correspondence 扩大到 code/link/URL/其它区间。新 receipt 单独标记 `semantic_authority/semantic_inputs`，不会冒充 `unknown_resolved_machine`，所有 downstream machine authority transport 同时携带这些 frozen inputs。

`APPROVE_RECONCILIATION` 只解除这一 Unit 的 unknown proof gate，不执行 apply、不生成 candidate/model provenance，也不创建语言 A。原 A carry 仍依赖已有 exact predecessor/replacement reader。`REQUIRE_LANGUAGE_REVISION` 保持 representation-only blocked，并保留 exact finding 供随后正式 finding-anchored revision；不能伪造 QC B、冒用 Surface finding 或用此决定直接做 Generation。本 transport 不授权自动开启后续 revision。

2026-10-16 的真实结果：json Bundle 已导出/current-check，尚无决定/receipt。vuln/database 的单一 source prose 区间内 A/B 各一个，旧 target 同一区间各两个且无独立 protected child；允许的 role/anchor/enclosing structure 均不能区分两次出现。source span 唯一不证明 target span 唯一，不能用中文意义或 occurrence 次序补足。因此保持 `unknown=true`；精确证据见 [vuln/database machine proof](protected-representation-evidence/vuln-database-binding-unknown.json)。原 formal impact SHA 不变；真实 package 仍为 **41 ready / 1 excluded / 2 blocked**，无 apply。json 的独立语义决定与此 machine binding blocker 完全独立。

### representation-unbindable 的机器 finding 与 exact revision

上述是 semantic transport 初次交接的历史状态。当前 json 的真实 `APPROVE_RECONCILIATION` 已由 Local terminal 记录，receipt SHA 为 `8b952e7d9c80f5283239f4910fa47f0f31f56f91f5f4ff47743a297da4e5d5c9`，保持 immutable。当前完整只读预检 **42 ready / 1 excluded / 1 blocked / applied=false**；唯一 blocker 为 vuln/database，不是 json authority 缺失。

重复 literal 的四种 A/B 放置都能恢复同一旧 candidate，不能单凭这个事实采用 interchangeable binding。当前合同中 `atom` token 必须 exact/unique，其它旧 prose occurrence 仍可编辑；两个固定 target site 的保护角色并不相同。fixture 同时验证四种 raw 均 restore/validate PASS，以及固定第一个 A 的相同编辑在两种绑定下分别被拒绝/允许。token 可以由 Generation 按目标语序移动，但这不能反过来授权机器先替 Generation 选择历史 binding；这个反例也不声称经过重新绑定后可达的最终文本集合不同。旧 target 没有新增 source-site 对应的 semantic-role evidence，因而不能证明所有 site assignments 在 source/target binding semantics 与后续 locality 上完全等价。既不放宽唯一性，也不把此情况交给 semantic Reviewer 选择 occurrence。

最小 additive 路径仍使用原 `quality-check-reopens`、replacement resolver、exact revision export / Generation Bundle / retry / validation / QC readers；新增 `authority_source=representation-machine`，而非另一套 lifecycle。`quality-check representation-revision` 默认只读重算机器 finding；未来维护者显式 `--record` 才写现有目录下的 immutable reopen。它要求完整 registered package、同一 source bytes、原正式 A / Snapshot / candidate / validation / final attempt / 原 raw / manifest / result / QC inputs、旧/current protected input 和 frozen impact SHA 全部绑定，且 current successor 无结果。只接受 exact prose interval 中**新增 unpaired literal atom、单 source site、多 target sites**的可证明 binding failure；缺失 literal、不同 source sites、code/URL/link/pair/directive/placeholder 等不符合条件时拒绝，不造通用语言 finding。

finding kind 为 `duplicate-prose-literal`，绑定 exact intervals/anchors/atoms/sites；不是语言错误诊断，不把原 A 改成 B/C/D。新 receipt 的 `replacement_context.machine` 冻结全部 authority 与 `scaffold_sha256`，current reader 每次重算并 exact compare，拒绝 scope/finding/proof/hash 篡改、混用 semantic/Surface/Preview authority、重复 record 或 successor fallback。

这里无法提供恢复旧 candidate 的 current protected baseline，因此 transport 必须明确区别两类输入：`formal/candidates` 与 `formal/historical-protected-candidates` 保留真正旧 reviewed bytes / raw；`formal/protected-candidates` 是**尚未完成的 revision scaffold**。机器只把整个已证明不可绑定的 prose interval 替换成 verbatim canonical source，按正常 current protector 生成合法 token set；不选择任何旧 target occurrence、不翻译、不写 candidate，不将 scaffold 当作受信语言结果。所有区间外的 prose 与 bindings 经原 deterministic rebase 保持不变。Bundle 用 `formal/representation-revision-scope.json` 与显式 instruction 标明这个唯一例外；普通 B/C/D、Surface/Preview 与一般 representation rebase 不变。

后续 Generation 才负责在 exact finding locality 内给出语义完整的 revision，保持其它旧译文，输出 normal current protected raw。新 candidate 必须重新 restore/validate，并进入 independent re-QC；即使恢复后的字节恰巧与旧译文一致，也不能从此机器授权获得新语言 A。synthetic fixture 验证了同一 shared export → Bundle/current-check/import/process → successor Snapshot scope，结果为 unchanged A carry 2、revised Unit pending 1，旧 A JSON 逐字节不变。所有这些写操作仅在 TempDir，真实 authority 尚未 record，真实 export/Generation/apply 均未执行。

原 [UNKNOWN 观察证据](protected-representation-evidence/vuln-database-binding-unknown.json) 保持 SHA `27cc66928c3e7958993f070f6083904160ee19ef0586b80a1b4c804623abad02`，不静默覆盖历史。新增 [最终只读计划](protected-representation-evidence/vuln-database-final-readonly-plan.json) 仅是 engineering report，不是 canonical receipt。完整可复用命令见 [Runbook](RETRANSLATION_RUNBOOK.md#representation-unbindable-的-machine-authorized-exact-revision)。当前只允许只读 inspection；先由维护者决定是否正式记录 machine authority，然后才另行授权 exact revision export。本轮不提供 bulk apply，不修改 json receipt、047、glossary-stale 或 qc-015。

## 同一 formal Unit 的受限 transport bridge

### Surface-bounded predecessor → independent ordinary QC bridge

本节也允许 passed Surface-bounded same-ID predecessor 在后续独立 QC 得到 B/C/D 后建立**新的 ordinary QC bridge template**。Unit ID/kind、source path/SHA、formal protected input path/SHA 必须不变；previous Snapshot、所选 Surface batch、candidate/validation/attempt、原 import/archive/raw、Surface reopen/transport 与完整 frozen dependencies 必须精确验证。当前完整 package、glossary/full Review、active validator、QC results/rating/finding 和显式独立 Reviewer evidence path/SHA 仍单独重验。preflight 的临时 revision record 从已验证 predecessor record 取得 input path/SHA，再保留原 exact comparison；不以空 SHA 或可选比较通过。

Surface authority 仅为 historical representation dependency，不是新 QC 的 semantic authority。机器读取原 import archive 封存的 exact Surface codec，以原 source-owned `Origin` mapping 和该 invocation 的 passed protected raw 按既有 boundary/atom coordinates 确定性组合 ordinary response；绝不 Protect(restored candidate)、猜 token mapping、搜索译文位置或重放语言生成。已被合法 Surface revision 修改的 opaque payload 只作为新的 private ordinary decoder 中**仍 opaque 的固定值**，code/API/link/directive/literal bindings 不开放；普通 prose 的编辑能力来自 ordinary source mapping，不来自 Surface exposed regions。独立 QC finding 继续约束 exact Unit 的 finding-anchored、locally complete revision；工程层不推断或扩大 finding locality。旧 Surface codec/receipt/regions 不变，也不能直接当作 generic QC editor。

旧 decoder 与新 ordinary decoder 必须分别将相同 passed raw / 新 template byte-exact restore 为当前 reviewed candidate；当前 canonical validator 还必须 PASS。缺失、歧义、不同 Unit/source/input、不可执行 ordinary mapping、非零 target delta 或任意 dependency drift 均 fail closed，不 fallback。新 template/decoder 只是 revision-only response projection，不注册为 source、formal input、Unit、candidate 或语言 judgment。

复用现有 `quality-check-transport` / `same_unit_bridge` receipt 和字段：`transport_member` 精确选择 archive 内 `formal/surface-transport-authority/<unit-id>.json`（原 comment 分支仍使用原 member），`transport_sha256` 绑定该完整 authority。Surface 分支的 `ordinary_ancestor_raw_sha256` 绑定按封存 codec template 导出的 ordinary ancestor response；若更早为 restricted transport，它仍只由既有 source mapping 派生，不冒充新的 Generation output。实际历史 raw、mapping 和 archive 的 path/SHA 另外完整冻结。其余 binding 冻结当前 reviewed Snapshot/candidate/validation/attempt、所选 batch manifest/result/input/raw/import/archive、新 template SHA、construction implementation/context；Surface reopen、transport、codec 与全部历史依赖进入 frozen dependency graph。新 export/current-check/import/process/retry 重建相同 proof；历史 reader 只用 frozen closure 重建同一 decoder，不能读取 latest Bundle 替代 imported archive。

`--record` 仍须独立授权、fresh 完整重建、write-once/no-overwrite；read-only preflight 无 mutation。新 revision 必须使用新 batch ID、显式 `--representation-bridge-id` 和自身 QC rating/finding。任何此前缺 capability 的零输出 batch 永久保留，不 retrofit metadata、刷新或 replay。现有 restricted current-validation comment bridge 分支保持原义。

restricted comment revision 之后，已有正式 B/C/D finding 可能需要普通 prose editing capability，而 formal source、input 与 Unit ID 均未变化。这不属于 replacement。共享 `quality-check representation-bridge` 默认只读，显式 `--record` 才在已有 `data/quality-check-reopens/<locale>/<bridge-id>.json` 写入 immutable schema-v1 receipt；新增 `authority_source=quality-check-transport` 与 `same_unit_bridge`，不建立另一套 candidate、attempt、Snapshot 或 QC lifecycle。

bridge 同时验证完整 registered package、原 Snapshot/results、逐 Unit rating/finding、显式 Reviewer evidence path/SHA、candidate/validation/attempt、原 import/archive/raw、restricted mapping 与其 current-validation authority、当前 source/input、glossary/full Unified Review 及 adapter/validator/projection implementation。旧/current restore 均按既有 artifact framing 精确得到同一 reviewed candidate。没有新的语言结论、candidate、Generation attempt 或 Unit ID；历史证据全部不改写。record 以前重新完整构建并 fresh 检查 dependency bytes；相同 ID 即使内容相同也拒绝重复写入，冲突不覆盖。

Page adapter 的 machine construction 只合成既有 source-owned mappings：普通 prose 和 token identity 来自原 invocation 冻结的 passed ordinary ancestor raw，绝不对 restored target Page 执行 Protect。原 restricted raw 的授权 teaching-body bindings 保留；原 source-selected ordinary teaching slots 使用该 raw 中已有 token correspondence。仍属于普通 protector opaque 区域的已审核 comment payload，只进入 private response scaffold 的 exact source coordinate，其余 source slots/结构不变。该 scaffold 不注册、不成为正式 source/input/TU/candidate。新 baseline 必须完整 restore 为原 reviewed bytes，任何非授权差异或不可唯一构造均失败；Generation 从不构造此 mapping。

adapter 通过 additive `SameUnitBridgeAdapter` 提供这个 source-owned operation；当前 Page adapter 支持上述组合。Example、未知 adapter、缺普通 ancestor、没有零 delta、缺失 mapping、非唯一坐标，以及需要尚未证明的 reference-definition offset 调整均 fail closed，不自动 fallback。已能由正常 decoder 执行的 ordinary predecessor 无需 bridge。

后续新 ordinary QC revision 显式传 `--representation-bridge-id`，exact-match receipt scope；manifest 只增加 optional `representation_bridge` path/SHA，仍保持 `revision_feedback_source=quality_check` 和逐 Unit 原 rating/finding。旧失败零输出 batch 不刷新、不 replay。Bundle 冻结 reviewed candidate、bridge template、旧 restricted raw 和原 restricted mapping；原 archive 与完整历史依赖由 immutable receipt 的 exact path/SHA 绑定，不复制为额外语言输入。current-check 独立重建并核对完整 dependency identity。原 early capability guard 保留，没有 recorded/current bridge 的受限 ordinary revision 仍失败。

import、process、retry、selected-candidate/Snapshot reader 共用相同 bridge response decoder，再由 canonical formal source validator 校验真实 candidate。revision-origin retry 继续绑定原 receipt/template、历史表示、QC feedback 与 predecessor；不会因 attempt 改变重选 mapping，也不扩大 sibling output scope。source/input/candidate/validation/raw/archive/mapping/QC/Reviewer/Snapshot/glossary/full Review/implementation/package 或 bridge bytes 漂移均拒绝。bridge 不是 QC/re-QC 或 A-only finalization 的替代。

bridge 的 current authorization 与 historical integrity 分开：新 export、Bundle/current-check、import、process 和 retry 必须验证完整当前依赖并重建；已通过候选与历史 A 的 reader 则绑定 receipt、frozen source/input/raw/archive/QC/validation 历史闭包，重建相同 source-owned response decoder，再用当前 canonical validator 判断候选资格。通用命令文档演进本身不改写历史 A；冻结历史字节漂移仍拒绝。Snapshot 写入前重新检查冻结 reference、selected response 与当前 validator identity，不能用历史 restore PASS 替代当前 validation。

### Approved-interval predecessor → ordinary QC bridge

同一 bridge 也支持已 imported/passed、随后取得独立 QC B/C/D 的 approved-interval predecessor。可选的 source-owned `ApprovedIntervalSameUnitBridgeAdapter` 消费原 archive 的完整 approved proof/codec：原 manifest/input、selection、reopen、semantic reconciliation、raw/validation/attempt 与 reviewed candidate 必须 exact 绑定。当前 implementation fresh 构造完整 approved semantic proof，与 frozen proof 做 typed equivalence；仅既有 code-owned implementation refs/identity 可 drift，非 implementation dependencies 不可漂移。普通 process/retry/revalidate/current-check 的 strict installed-proof gate 不变。

composition 沿原 inverse 的 source-owned boundary/token correspondence 还原 ordinary ancestor response，开放其普通 prose；approved whole parent 继续作为 opaque 值，内部新增 child 没有历史 target token correspondence，不搜索 target occurrence、不猜 mapping，也不 Protect(candidate)。Unit/source/formal input 不变；旧 approved inverse 与新 response template 必须分别 exact restore 同一 reviewed candidate，当前 canonical validator 必须 PASS。缺失/歧义坐标、source/input/codec/候选/provenance 漂移或未知 adapter 均拒绝。

复用现有 same-unit receipt 的 `transport_member`/SHA 绑定原 `formal/approved-interval-transport/<unit-id>.json`，冻结原 archive/proof 与 semantic dependency graph；不将旧 proof 标记 CURRENT，不执行历史 implementation，不创建新 schema/authority/recovery family。新调用完整重建、record 前 fresh 检查、frozen reader 的 exact reconstruction 及 Independent QC/A-only gates 保持不变。此 additive construction/currentness 变化使旧 first-real-use verification 失效；真实 preflight/record 前须重新核对 docs、implementation 与 direct CLI/contract tests。

## Current-validation predecessor representation transport

缺失原 ZIP 的 [`generation-bundle legacy-provenance` recorded authority](LEGACY_GENERATION_PROVENANCE_AUTHORITY_DESIGN.md#正式-implementation-command-contract) 是已实现的独立 historical-response provenance 层；json exact authority 已由维护者完成 first-real-use 与真实 record，见 [current evidence](CURRENT_VALIDATION_APPROVED_INTERVAL_MIGRATION_DESIGN.md#已记录-authority-与当前-case)。其 strict receipt 固定 single historical/current Unit、invocation、trusted import 与完整 surviving dependency graph，不提供 representation mapping、language approval 或 old Bundle CURRENT。当前本节 executable consumer 仍执行原 archive/codec gate，**没有 legacy 自动 fallback**。

[Approved-interval contract v2](CURRENT_VALIDATION_APPROVED_INTERVAL_MIGRATION_DESIGN.md) 明确未来派生 consumer 的 `NORMAL_ARCHIVE > EXPLICIT_RECORDED_LEGACY_PROVENANCE`：只有 archive 缺失时，才允许显式绑定 recorded authority path/SHA 并调用 `VerifyLegacyGenerationProvenance`、传 exact requested options；不扫描 latest/按 Unit 自动发现，不扩大 `historical-response/v1` allowlist 或六项 unknown facts。archive 后来出现时 fresh compare covered facts，冲突停止、匹配采用 stronger normal archive，保留 legacy receipt。该 provenance 层不替代独立 approved-interval representation authority。

v2 只设计 whole frozen interval 的 immutable opaque parent，保留 source/current children 的 identity/role/inventory/source coordinates，以独立 projection namespace 表达，绝不赋予普通 `P` token 新含义或建立 target atom mapping。全部 editable finding regions 必须经 machine-reproduced typed coordinates 证明与 parent 无交集；exact restore、active validation/current glossary/representation、独立 re-QC/A-only 均不可省略。selection/refs/proof 仅进入 derived Bundle graph，不写064 manifest；同批刷新仅为 architecture possibility，须待实现、32/32 readiness、新 first-real-use gate 后才可 export。本轮没有实现 capability，也不改旧 batch、Snapshot、reopen 或 reconciliation。

current-validation 的 revision authority 与 predecessor response codec 是独立的两种依据。原 immutable reopen 继续独占本轮 exact affected Unit、历史 A、Snapshot 和 machine finding；已有 Surface/QC bridge 的 immutable imported lineage 只能证明 predecessor 的表示。不得把本轮 feedback 改成 `quality_check`，不得伪造 B/C/D 或用 exclusive bridge receipt 替代 current-validation reopen。

普通 passed raw 能由 canonical frozen predecessor adapter 精确 restore 时，不需要 recovery。不能 restore 时，机器只接受已有 sealed Surface codec 或同 Unit 的 inherited bridge decoder；必须验证原 manifest/input、exact Snapshot candidate/validation/attempt、原 raw、import receipt/archive、原 codec 与其完整 dependency graph。Surface 的受限 response 使用既有 source-owned `BridgeSurfaceRevision` composition，已是 ordinary 的 inherited bridge 保持其 exact decoder。它们均须恢复逐字节相同的 reviewed candidate（仅沿用既有 single-LF framing），且 current Unit/source/formal protected input 与 predecessor 相同。新 protection、未知/歧义 mapping、不同 source/input/Unit 不能走此零 delta transport；保留 blocker，必要时另行建立正式 replacement lineage。不得根据译文出现位置、唯一字符串或中文语义猜映射。

能力表现为 Bundle 内派生的 `formal/current-validation-predecessor-transport/<input basename>.json`，schema 为 `go-learning/current-validation-predecessor-transport/v1`。它绑定 locale、完整 package identity/ordered scope、原 current-validation reopen path/SHA/identity、exact revision records 全集、Snapshot manifest 及逐 Unit candidate/validation/batch/attempt/raw、old/current representation/input identity、source-owned composed template/hash、实现和当前 adapter authority、完整历史 dependencies 与 recovery class。`formal/protected-candidates` 是已证明的 ordinary template；`formal/historical-protected-candidates` 保留 exact passed raw；所需 frozen dependencies 一并运输，historical glossary 仍按既有 immutable archive reader 解析。新 current-validation comment projection 如适用，继续独立授权 exact comment slots，并在 response decoder 选择上优先。

后续 independent QC 得到 B/C/D 时，现有 same-unit representation bridge 也可消费 exact imported archive 中的该 cvpred member。必须 fresh canonical reconstruction，并复用 Snapshot historical-PASS 的完整 typed semantic equivalence（只允许既有 implementation refs/identity 与 adapter authority drift）；source/input、reopen、scope、template、非 implementation dependencies 仍 exact。新 bridge 不再做 composition：复用已证明 ordinary-capable 的 decoder，把当前已审核 invocation 的 exact imported final response 作为新 template，必须 restore 同一 reviewed candidate 并通过 current validator；不能拿 cvpred 封存的更早 Surface/restricted raw 当普通 template。仍 restricted 的 decoder、互斥 representation members、缺少/被篡改的 archive/ref 均拒绝。receipt 与 frozen invocation reader 继续绑定 exact cvpred member/SHA、原 raw/import/archive、完整依赖、template/current authority；不新增 schema/authority，不改变 bridge exact-set 或已有零输出 batch。

这是现有 Generation Bundle 的 deterministic derived input，不是新的 receipt、pointer、language finding、QC 或语言 approval，也不新增 record/apply command。export、current-check、import/restore、process、retry 与后续 Snapshot 共享 response projection，返回/落盘前保留既有 fresh dependency reconstruction。拒绝失败结果、跨 invocation 的 PASS carry 或历史 grandfather bypass。工程 inspection API 可逐 Unit 报告 readiness，但报告本身不授予任何正式 authority；只有当前 canonical exporter 从上述 immutable/current inputs 重新构造同一 capability 才能消费。

若只是这种 transport 能力缺口，现有 immutable pending batch 不需改 manifest，可在 first-real-use gate 后重建同一 batch 的 Bundle；一个 Unit 未证明则整批 fail closed，无部分 Bundle。若 formal Unit/input/source 已变，不能借此同批刷新，须先证明并走已有 replacement/successor lifecycle。已有 Batch、raw、Snapshot、A、reopen、历史 Bundle 始终不改写。

| Formal command contract | 当前 contract |
| --- | --- |
| COMMAND | `generation-bundle export` / `generation-bundle current-check` |
| OPERATION | 从已有 current-validation revision graph 派生/独立重建 predecessor representation transport |
| INPUT_KIND | 已有 immutable batch + exact current-validation reopen + Snapshot + selected raw/validation/attempt + sealed imported representation lineage + current complete catalog/glossary/validator |
| INPUT_PATH_CONTRACT | `--batch-id` 为 batch identity；current-validation 必须一次显式 `--content-package`；`--output` 为新 ZIP 路径；`--bundle` 为 existing regular ZIP；repository authority refs 保持 repository-relative |
| MUTATION_EXPECTATION | export 只写新的 transport ZIP；current-check 无 mutation；不创建或改 receipt、batch、candidate、Snapshot、authorization |
| IDEMPOTENCE | 同一输入产生相同 bytes；既有 output 拒绝覆盖；不把 observed existing bytes 当某 invocation 的 provenance |
| CURRENTNESS_CHECK | 独立 canonical reconstruction、exact scope/hash/codec/implementation/adapter 与完整 dependency guard；有一项不可证明即拒绝 |
| PRIOR_OR_CAS | batch/reopen/Snapshot/input/raw/import 的 exact SHA bindings；ZIP publication 仍使用既有 write-new/no-overwrite 边界 |
| SUCCESS_STOP | export 后独立 current-check；其后在真实 Generation 前停止，transport 不代表语言审核通过 |
| FAILURE_STOP | 保留 batch、历史 receipt、raw、Bundle 与 logs；不生成部分 Bundle，不自动创建 successor 或改 authority |
| RECOVERY_REPLAY | 修复缺失的合法 proof 后重新完整检查；不盲重放、不伪造 import archive；publication provenance 不明时先诊断 mutation 状态 |
