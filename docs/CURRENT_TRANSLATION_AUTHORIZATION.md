# 共享 TranslationUnit 的历史完整性与当前授权

current package membership扩大但已存在多个 disjoint finalized predecessor时，closure可消费 [finalization composition](LEARN_DOCS_MEMBERSHIP.md)：每个原 scope分别完整 historical integrity与 current re-attestation，再以 current descriptor做 exact union。它不是合成 Snapshot、复制 A或新的 authorization checkpoint；任何 affected/stale/unknown继续拒绝。single-Snapshot的 complete scope合同不变。组合按 current registry顺序投影，但每个原 Snapshot Unit内的 index、batch/attempt、candidate/validation及独立 Reviewer provenance保持原坐标，不把历史 index改成新的 stable index。

本能力属于 `internal/i18n` 共享生命周期，不增加 Generation、Reviewer、QC 或 package 专属状态机。旧 Snapshot、Reviewer、finalization、compatibility、successor、Surface 与 Production evidence 保持原 bytes。

独立 Reviewer 对已记录 source-mismatch finding 的纠错使用 [Quality Check judgment correction](TRANSLATION_QUALITY_REVIEW.md#已记录-quality-check-judgment-的-source-mismatch-correction)。原 results 不改写；effective A 的来源是 exact 原 judgment 加原独立 Reviewer 的 immutable correction，不能由 machine re-attestation 自行产生。新 finalization 的既有 QC input references 同时绑定原 results 和 correction；historical integrity 验证两者 exact bytes/inventory，之后 current eligibility 仍独立执行 source/candidate/validator/scope gates。无 correction 的旧 schema、proof 和 checkpoint 不迁移。

## 实施前 coupling inventory

| 分类 | 现有 authority / 函数 | 耦合与处理边界 |
| --- | --- | --- |
| immutable historical evidence | Snapshot manifest、candidate、validation、protected input/raw response、QC results、finalization、archived glossary、Generation origin | 保留真实 A 的 exact source/candidate/attempt 与 Reviewer refs；历史 reader 不要求 current glossary 或 current mutable projection 等于过去 |
| historical mutable projection debt | `verifySuccessorFinalization`、`historicalCanonicalTargetsScoped`、`compatibleLegacyCompletion`、Course SEO recovery | 原 glossary/config/status/UI/SEO whole-file 重建属于 migration/recovery；不成为已建立 checkpoint 的 ordinary TU current oracle |
| current semantic authority | `RequireUnifiedGlossaryReview`、`glossarySemanticDelta`、`classifyGlossaryContext`、registered descriptor/source | 完整独立 Unified Review 仍 byte/corpus-current；逐 TU 从原 Reviewer glossary 到当前 glossary 做同一 semantic delta，不默认 compatible |
| current machine validation | `readSnapshotUnitRepositoryEvidence`、`validateUnifiedPromotionEvidence`、`ValidateTranslationUnitCandidate`、adapter | 当前 source/accepted candidate 必须 exact；原 input/provenance 完整性与当前 validator PASS 分开，parser/adapter contract 不明则 unknown |
| derived consumer freshness | `FinalizedTranslationPackageCandidates`、`CheckPackageClosure`、`checkPackageCompletion`、Integrated Surface carry | 使用同一 current authorization；TU checkpoint 不授权 structured assets、Course SEO、UI 或新 Surface judgment，各自 gate 保留 |
| runtime/public identity | Content Scope routes、Resolver、Production identity、Integrated Surface runtime refs | 独立 current authority；不纳入 TU language judgment，不改变 public identity 或65-locale Production lifecycle |

`VerifyQualityCheckFinalization` 当前先调用 `RequireCurrentGlossaryReview`，然后用 current evaluation 重新构建原 finalization；Snapshot reader 的 compatibility resolver 又依赖完整 current global inventory。`qualityCheckEvaluation.loadEffectiveQualityCheckResults` 递归 predecessor 并重新解析 glossary chain。它们把历史 A 的真实性与今日可用性混在一起。

`FinalizedTranslationPackageCandidates` 和 package closure 继承上述耦合；`finalizationMatchesLatestPromotionCandidates` 通过 current-glossary Snapshot reader 做纯 identity 检查，亦混入历史 glossary。`target_successor` 已区分历史事件与 current canonical，但 current path 仍扫描 current-glossary finalized snapshots、恢复 baseline targets 和逐 edge context。Course SEO history 已明确停止扩展，其旧 archive/recovery 不能作为 TU checkpoint 的 derived authority。legacy Tour checkpoint 另要求 exact-current Surface A 与 strict Course SEO，这些独立 gate 必须保留。`validateHistoricalIntegratedSurface` 已做 immutable judgment verification；`currentReviewedScope` 的 activation/CAS/public context freshness 是另一职责，不由 TU checkpoint放宽。

## 必要性与兼容边界

现有 finalization wire schema 表达当时的 A-only 与 carry lineage，不能表达针对今日 adapter、glossary、full Unified Review 和 complete scope 的重新机械确证。旧 compatibility artifact 绑定跨 consumer 的完整 inventory，不能作为 package-neutral、scope-local steady-state checkpoint。扩展这些旧 artifact 会改写历史或把 derived consumer 的 mutable state 再带回 TU 判断。因此新增 additive、immutable 的共享 current authorization；不修改旧 schema、不执行 mass migration。没有 checkpoint 的 legacy consumers 保留原严格 reader；新注册 package 使用同一能力，不按 package 名称选择政策。

历史完整性与当前授权分别提供 API。机械 re-attestation 不能产生 Reviewer A：只携带已 hash-bound、exact identity 相同的原 A；任何 source/candidate drift、affected、unknown 或 current validation failure 均不能建立 complete authorization。affected-only scope 回既有 revision/re-QC/successor finalization，不生成空 Reviewer response。

## Executable contract

`quality-check authorization --locale <locale> --snapshot-id <finalized Snapshot>` 对 registered package 按 finalization 的完整 ordered source coordinates 自动选择该 package，读取同一共享 current authorization，并输出完整逐 TU classification/counts 和原 checkpoint identity。显式 `--content-package` 必须与该 exact scope 一致；不能把 package Snapshot 交给默认 Tour canonical scope。native Tour 继续使用原 canonical installation diagnosis；package language authorization 与 canonical installation authority 分别保留各自 gate。诊断只读，不创建或改写 checkpoint。

- `VerifyHistoricalQualityCheckFinalization` 验证原 Snapshot/finalization exact-set、original Reviewer A、candidate/validation/attempt、archived glossary/full Review、protected input/raw 和必要的原 Surface/replacement/Generation refs。它不调用 current glossary gate 或 global compatibility inventory。optional Catalog 只提供 representation-owned historical finding/mapping reader，不把当前 package whole SHA 当成历史语言证明。
- `AssessCurrentTranslation` 按原 Reviewer 的 glossary 到当前 glossary 直接调用既有 normalization / impact algorithm。source exact unchanged、accepted candidate exact unchanged、当前 protector 可构造、当前 validator 通过和 visibility adapter 已知才可 carry；source drift 为 stale，semantic impact/validator failure 为 affected，unknown adapter/parser 为 unknown。每 Unit 绑定 current validation identity；complete scope 不能部分授权。
- `EstablishCurrentTranslationAuthorization` 保存 `go-learning/current-translation-authorization/v1` 到 `data/translation-authorizations/<locale>/<scope-sha>/<snapshot-id>/<identity>.json`。绑定 registered package authority、exact scope、原 A proof、当前 source/candidate/validation、glossary、完整 independent Unified Review、adapter/algorithm authority 与 self identity。只有全 compatible 才 immutable/no-overwrite 落盘；不改旧 wire schemas 或 locale completion。
- `CheckCurrentTranslationAuthorization` 校验 sealed flat immutable refs，并在当前 world 重验 exact candidate。保留原 Reviewer/validation/finalization 的 hash 检查；不执行 pre-checkpoint QC predecessor evaluation、global compatibility chain 或 mutable status/UI/SEO/runtime replay。原 Reviewer/finalization evidence 缺失或损坏仍 fail closed。新 glossary 必须重新 re-attest；corpus append 且 glossary byte-identical 时仍需新的 full Unified Review，但 unchanged package 的既有 checkpoint 可继续使用。

### checkpoint 中的历史 representation context

sealed historical proof 可能同时保存当时的可变 context pointer Reference 和它指向的 immutable receipt。两者不能混为一谈：后续合法 publication 可以替换 pointer，但不能改写 checkpoint 的原 path/SHA，也不能将今日 successor 当作原 evidence。flat historical reader 复用 Catalog 的 `HistoricalContextReader`；直接读取仍必须 exact SHA，只有 representation-owned 的明确 lifecycle proof 才能提供同一 frozen SHA 的历史 context bytes。

对于 [Shell dependency re-attestation](SHELL_REPRESENTATION_REATTESTATION.md) 的 repository-relative `data/site-shell/representation-reattestations/current/<base-identity>.json`，历史 reader 验证当前 pointer 的 canonical Reference、immutable base receipt 的 identity/content-address，以及完整 `previous` lineage 的每个 receipt hash、identity、同一 base/package/source/representation、允许的 dependency delta、cycle/depth 边界。只允许将这条 lineage 中某个 exact receipt Reference 按原 canonical JSON encoding 在**内存中**编码，并要求其 bytes SHA 与 checkpoint 原 pointer SHA 完全一致；不扫描 latest、orphan 或 sibling，不恢复、复制或覆盖任何文件。整个 proof 返回前重验观察到的 bytes/mode/absence，并拒绝 pending/lock、missing、tamper 或 ambiguous lineage。

这只验证历史 context，不声明当前 head/dependency 或语言仍有效；current source、glossary、完整 Unified Review、adapter/validator 与 candidate eligibility 继续独立验证。原 Reviewer、candidate、validation、finalization 仍按原 path/SHA 验证，不能使用 context reader 替代。没有上述精确 lifecycle proof 的其他 dependency 仍要求物理文件 exact SHA；immutable receipt 缺失或历史 pointer SHA 无法从合法 lineage 精确证明时 fail closed，禁止伪造旧文件或改用 successor bytes。

historical structured closure 消费 `ReadHistoricalTranslationIntegrity` 返回的完整 proof 时，使用 `ReadHistoricalTranslationReference` 重验该 proof 内的 exact Reference。dependency graph 分别保留 frozen path/SHA 与当前 physical observation：旧 SHA 仍由 canonical historical reader 的完整 immutable lineage 证明，当前物理 bytes/mode 则按 invocation 初始观察值监控。不能把已验证的历史 pointer 再交给普通 `bind`，要求它等于当前 pointer；普通 current `bind`/`readReference` 仍严格检查 physical SHA。缓存复用和最终 verify 都重新验证历史 context，不缓存 PASS；缺失 receipt、断链、wrong SHA 或 invocation 内 physical drift 均拒绝。不改旧 proof/receipt/pointer，不产生新的语言 authority。

`TranslationMachineAuthority` 包含 `internal/i18n/translation_authorization.go` 的完整文件 SHA。修复该 reader 会改变 current adapter authority 与 validation identity；即使所有 Unit classification 仍为 compatible，原 checkpoint 也不能因此自动成为 current。必须另按既有 current authorization 合同建立合法新 epoch；历史 proof、工程测试或 representation re-attestation 均不能代替该 gate，工程阶段不自动发布。

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check reattest \
  --locale <locale> --snapshot-id <accepted-finalization> --content-package <registered-package>
# 全 compatible 后，显式建立正式 machine checkpoint；不产生新的 Reviewer A：
go run -mod=readonly ./cmd/tour-i18n quality-check reattest \
  --locale <locale> --snapshot-id <accepted-finalization> --content-package <registered-package> --apply
```

Tour 和任何已安装 canonical target 的 registered scope 可加 `--apply --canonical`：另将 installed bytes 精确绑定到 accepted A candidate 的正式 EOF 投影。这个独立 current target binding 不把 current bytes 当成自己的 A 证明；原 A refs 和 full current Review 仍必需。后续 `RequireCurrentCanonicalQualityCheck` 优先使用同一个 shared checkpoint，不再为 current target 重放旧 successor event；native promotion/recovery records 仍用于 audit/recovery，不删除、不改写。没有此 canonical binding 的 language-only checkpoint不能冒充 installation authority。TU checkpoint 不授权 Course SEO、structured assets、UI、Surface 或 Production identity。

JSON 中 canonical refs 的正式字段是 `current_canonical_targets`（Go 字段 `Canonical`），不是 `canonical`。检查该数组的长度；不能因不存在的字段返回默认空数组，就判断 checkpoint 为 language-only。`--apply` 保存 language-only epoch；对同一 current state 重复请求为 immutable 幂等。`--apply --canonical` 每次 fresh 验证完整 current assessment 与 installed targets；已有 language-only epoch 不满足请求，必须保存具有不同 identity/path 的 canonical sibling，旧 epoch 原 bytes 保留。已有 exact canonical epoch 时返回该 canonical epoch，零新增 artifact。所有 sibling 的 sealed original proof 仍验证，corrupt sibling 不能被新 epoch 遮蔽。

native Tour 的 `quality-check authorization` 同样优先验证请求的 exact Snapshot 的 shared checkpoint。存在 current canonical binding 时输出其 canonical authorization；language-only checkpoint 仍 blocked。没有 shared checkpoint 时才走原 legacy promotion diagnosis，不要求已有合法 checkpoint 再通过旧 glossary-bound Snapshot discovery。

`FinalizedTranslationPackageCandidates` 是所有 registered package 的统一 current consumer；package closure、completion、Integrated Surface carry、completed-content loading 通过现有调用链复用它。QC evaluation 到达 sealed scope 时逐 TU re-attest，只继承 compatible A；affected/stale/unknown 保持 pending。registered package 首次迁移若尚无 checkpoint，也可从原完整 finalization integrity 使用同一逐 TU assessment规划 affected-only scope；不要求先建立无法全通过的 checkpoint，不写部分 authorization。原 B/C/D finding、exact revision、独立 re-QC、A-only 和原 initial batching 规则不变。未来新增 package 不扫描或恢复其他 package 的 QC history；adapter 可声明自己的 visibility 与 executable authority，未知表示绝不默认 compatible。Page adapter authority 按自身 type/method AST 与 parser dependencies绑定，无关 loader/helper 追加不改变旧 adapter contract。

### 首次 affected migration 的 Candidate Snapshot

`CreateQualityCheckCandidateSnapshot` 对 registered complete scope 先按既有 latest selection 比对完整 accepted candidate set；只有匹配原 finalized scope 才调用共享 historical integrity / current re-attestation。它不以 accepted candidate 默认替代当前校验：exact source/candidate、真实原 A、当前 full Unified Review 和每 Unit 当前 validation PASS 都必须成立，classification 只能是 compatible 或 affected。stale、unknown、validation failure、原 judgment 缺失/损坏均拒绝，不回退到旧 candidate。

旧 Snapshot 的 `validation_path`/SHA 表达原 passed attempt，不能真实表达今天的新 glossary、validator 和 semantic assessment；把新结果写回旧 validation 会改写历史，且 affected scope 不能保存为 complete authorization。因此只对新 Snapshot 添加可选 `current_reattestation`，直接嵌入既有 assessment wire format，不增加 schema version、平行 receipt 或新 QC 状态。旧 Snapshot/finalization 保持原 bytes。该字段冻结 current machine proof，允许 affected 进入独立 QC，但不是 current complete authorization，也不是 Reviewer finding/rating。当前 scope/preflight/Reviewer Bundle reader 重验该绑定和当前 assessment；历史 reader只检查其冻结结构。

此路径不调用旧 Generation-origin CURRENT glossary equality 或 global compatibility inventory。compatible Unit 可 carry 原 A；affected Unit保持 `quality_check_required`，不要求先有 replacement 或 `--glossary-stale` Generation。preflight 通过后按既有 JIT 合同只导出当前同 kind、最多60的 pending Reviewer Bundle；独立 Reviewer decision 之前不得生成 replacement。

### Current baseline 与 authorized revision delta

以上 complete-set first re-attestation 建立 current machine baseline。后续独立 QC 的 B/C/D finding 或正式 Surface reopen 可以授权 exact revision；latest complete set 此时由 unchanged baseline candidates 与 processed revision delta 组成，不再要求整套 latest candidates 等于某个 historical finalized Snapshot。

新 Snapshot 可另带 `current_revision`：绑定 predecessor manifest path/SHA、ordered exact delta、immutable finding/manifest/input/output/validation/provenance refs 和 self identity，`current_reattestation` 保留 baseline 的 sealed assessment。这是 existing Snapshot 的 additive machine proof，不修改旧 Snapshot，不增加 Generation/QC/finalization 状态机，不 mass migrate。当前 reader 重新验证完整 baseline、合法 revision authorization、当前 source/glossary/adapter/restore/validator、formal provider/model/input identity 与 exact processed bytes，再证明 baseline + delta 等于整个 Snapshot。stale/unknown、source drift、未经授权的 latest replacement、缺 finding、wrong predecessor、hash/input/provenance drift 或 machine failure 都拒绝；任意 passed batch 不构成 revision authority。

unchanged candidates 复用 current baseline eligibility，不回到 pre-checkpoint compatibility inventory。delta 只具有 machine eligibility，不能继承旧 A；successor scope 使用绑定的 predecessor，由独立 re-QC 审核 exact revised pending set。完成独立 A-only finalization 后，才建立最终 complete current authorization。分别区分 first re-attestation、baseline + authorized delta、successor independent re-QC 与 final authorization，不能用任一步替代后一步。

Candidate Snapshot 对同一 current-validation revision batch 复用既有 `prepareProcessOutputContext`：本次 invocation 只准备一次 finding-authorized response projection，逐 Unit 的 input/source/attempt/provenance、restore/candidate equality、automatic validation 仍完整执行。commit 前重新验证该 context 的全部 authority，并 fresh reselect ordered package membership，核对 source、candidate、validation、manifest/result/input、当前 glossary/full Review 与 validation 指向的实际 raw response（含 retry path）没有漂移。先前的 PASS 或已解析对象不能替代这些 disk checks；失败则清理本次 staging，不发布 Snapshot。没有跨 invocation cache，不改变 manifest bytes/schema、历史 A 或 write-once 语义，也不替代独立 re-QC。

普通 Quality Check B/C/D concentrated revision 同样沿用共享 invocation-local 边界。每次 command 新建的 registered Catalog 复用既有 Page replacement projection：每次仍读取并核对 frozen representation 的 exact bytes，只有完整 typed Catalog 与引用一致才复用纯计算结果；Shell adoption 或未知 adapter 仍走原 reader，不缓存 QC/currentness 的 PASS。若 reviewed predecessor 来自 current-validation batch，Generation Bundle 按 batch/Catalog scope 只准备一次既有 process output context，逐 Unit 仍保留自己的 rating、exact finding、candidate/validation/attempt 和 protected raw 绑定。返回前执行该 context 的完整 fresh authority reconstruction，再核对 QC lineage refs 与实际装入的 predecessor bytes；任何并发漂移均拒绝。没有跨 command cache，不改变 Bundle files inventory、aggregate identity、正式 schemas 或历史 A；current-check 始终在自己的新 invocation 中重建 canonical Bundle。

原 Generation invocation 是 provenance，不要求原 `authority/**` 与今天 docs 的 mutable equality；当前 source、glossary、formal input、predecessor candidate/finding 与机器 gate 仍严格重验。import receipt 的 Bundle/manifest/input SHA 只有配合实际原 transport bytes 才能核验。因此新 shared import 原样保存 Bundle 到既有 `data/generation-archives/<locale>/<sha>.zip`；旧已导入 batch 若未留 archive，可用 `generation-bundle archive-imported --bundle <original-imported.zip>` 对照现有 receipt 原样归档，immutable/no-overwrite，不重新 export/Generation/import/process。缺失或损坏原 Bundle/receipt 时 fail closed；禁止用今日 exporter 重建 bytes 冒充原 invocation。mixed proof 在 finalization integrity 中被 flatten/hash-bound，最终 checkpoint 的 ordinary consumers 不重放这些前置 mutable states。

Snapshot 的 archived candidate selector 同样区分原 invocation integrity 与 current eligibility：原 glossary 不同于当前 bytes 时，只有 sealed complete authorization 绑定的 exact candidate/validation/attempt、当前逐 Unit compatible assessment、完整 current Unified Review 和 unchanged formal input 才能使用 frozen-origin verifier。完整 scope 中其它 Unit 已有正式 revision 或 representation successor，不会使该 unchanged Unit 回到 current-only invocation glossary equality；latest processed selection 与 machine successor 优先级保持不变。formal input 真实变化、缺失或 stale authorization、affected/unknown 或当前 validation failure 均拒绝 carry，不改原 glossary/provenance，也不产生 Reviewer A。

## Current validation 的 affected-only reopen authority

`quality-check current-validation-reopen` 是既有 `quality-check-reopens` 的薄 authority 入口，默认只读，不创建第二套 revision/QC 生命周期。`authority_source=current-validation` 只授权 canonical validator 在 current re-attestation 中可重现的 `affected / current_validation_failed`；glossary-only affected、stale、unknown、缺失历史 A 或不完整 scope 均拒绝。它不伪造 Surface/Preview finding、QC B/C/D 或 representation replacement，也不把历史 A 改成当前语言质量结论。

必须显式提供且只提供一次 `--content-package <package>`、predecessor manifest SHA，以及完整 re-attestation JSON 的 path/SHA/identity；不隐式选择 package。合法 selector 分为：

- 显式 canonical Tour-native `tour-v1`：只接受 `TranslationScope` 为空的 native default complete Catalog。保留其完整 Page + eligible Example workflow ordered scope，经 `translationAuthorizationScope()` 重新证明 package identity、scope 与当前 source authority；不伪造 registration，不将任何已有 explicit scope（包括 partial、foreign 或人为枚举的 native IDs）清空或扩大。
- registered content packages（如 Learn/Docs、Site shell 与 future packages）：继续在 Catalog 副本上通过既有 `UseTranslationPackage()` 选择完整 descriptor scope；调用者已有 explicit scope 必须 exact-match。未知 non-Tour package 拒绝。

两者都必须与 frozen assessment 的 `package` 及 predecessor Snapshot 完整 ordered exact set 一致，不与 global locale inventory 比 count。每次 preflight、未来 record 和未消费的 export check 都在同一完整 package scope 重算 `AssessCurrentTranslation`，要求 compatible/affected counts、每 Unit source/candidate、原 A lineage、glossary/full Review、adapter/policy/current validation identities 与 evidence 完全一致；拒绝已存在的冲突 candidate/export。scope 自动来自全部 current machine failures，显式 UnitIDs 只能 exact-match，仍为同 kind、最多60，不能通过 route allowlist 或 subset 扩大/缩小 authority。native selector 不绕过 assessment/predecessor/package binding，不授权另一套 Tour reopen lifecycle。

future record 仍由明确授权的维护者调用既有 `RecordQualityCheckSurfaceReopen`，只写 `data/quality-check-reopens/<locale>/<id>.json`。additive `current_validation` 字段冻结原 re-attestation **exact bytes**、其原 path/SHA/identity、predecessor、authority dependencies、逐 Unit validation finding 和适用的 exact comment slots，并绑定完整 receipt identity。临时 evidence 只作为显式导入来源；消费后的历史 reader 检查 receipt 内冻结 bytes 与原 A/manifest/hash refs，不依赖 `/tmp` 持续存在，也不要求旧 failed candidate 重新成为 current passed。

既有 exporter 继续使用 `--surface-reopen-id`，并显式提供一次 `--content-package`。`retranslation export` 的 shared CLI dispatcher 复用上述 complete selector：native `tour-v1` 无需 registration，registered package 继续 exact descriptor 路径；禁止清空 explicit partial/different scope。此 flag 由 shared parser 提取，不归 export options parser，也不能按 flag 名推断 registry membership。recorded receipt 仍由原 current reader 全量验证，wrong package/reopen/predecessor 或不匹配的 affected set/order 继续拒绝。

exporter 保留历史 `previous_rating=A` 与 `revision_feedback_source=current_validation`，逐 Unit 携带自己的 machine finding；这不是新的 Reviewer A。只在该 authority 分支用 original-A evaluation 读取 predecessor，新 candidate 始终执行 active canonical validation，之后仍须 independent re-QC/A-only。preflight 不 record、不 export、不创建 authorization、Snapshot 或任何语言结果。

## 真实 integration proof 与停止条件

`zh-CN / learn-docs-v1 / learn-docs-v1-qc-010` 的只读 proof 完整检查49 Units：current machine validation 49/49 PASS、compatible36、affected13、stale0、unknown0。原 global inventory 缺 adapted scope 的 glossary mismatch 不再参与该 current assessment；13个 semantic affected 不能建立 complete authorization，不能让 Preview/Production 因此通过。机械输出不替代 Reviewer finding或产生 replacement；返回协调角色按 exact scope 进入既有语言闭环。本次不修改 glossary、candidate、Snapshot、QC、Surface 或 completion，不自动执行 revision。


### 同 Unit bridge 的 command-local reconstruction

bridge-backed export、Bundle current-check、result-pack/import 与 process/retry 共用已有 registered Catalog 的 invocation-local projection context。每次 command 都新建 context；只有 immutable Page adapter 与完整 typed Catalog 输入相同，才复用一次 canonical bridge reconstruction 派生的私有 response adapters。没有跨 invocation/global/persistent cache，不缓存 currentness 或 PASS。

每次消费仍重新读取 receipt、全部 current/frozen dependencies 并核对 SHA，验证完整 registered package ordered scope 与 source bytes、当前 glossary/full Unified Review 选择，以及当前 adapter/validator identity；返回前再次检查，任何 drift 均 fail closed。完整 canonical rebuild 已证明的 per-Unit QC rating/finding、Snapshot/candidate/validation/attempt、原 import/raw/transport 与 exact restore 可以由同 invocation 的调用方共同消费，不重复扫描同一 historical lineage；消费时仍重读原始文件并对照 exact binding。普通非 bridge QC/current-validation/Surface/initial 路径保留原读法。

历史 bridge receipt 与原 transport 实现的 SHA 不因性能优化重写；provider-neutral Bundle files、aggregate identity、raw/template mapping 与 serialized semantics 保持不变。已有 passed candidates 与 successor QC 的 frozen-history reader 仍按既有 immutable-history/current-eligibility 边界执行，不把 generation currentness 的完整重建错误地重放为历史 A 判断。

### Bridge revision 之后的普通 QC 连续性

bridge-backed candidate 经独立 QC 成为下一轮 predecessor 后，普通 B/C/D revision 沿该 Snapshot 选定的 batch/attempt 读取历史 response representation。原 bridge 只解释历史 raw bytes，不再授权新的语言修改；新 revision 的 rating、逐 Unit finding 与 exact scope 仍唯一来自新 Snapshot 的 QC。same Unit/source/formal input 的连续链无需再次 record bridge，也不向新 manifest 补写旧 bridge finding 或引用。replacement/machine reconciliation 与 restricted current-validation revision 继续使用各自的既有 reader，不自动取得普通 prose 编辑能力。

共享 resolver 重新验证选定 candidate、validation、manifest/result/input、原 import/archive 与 final raw 的 identity，使用原 bridge 的 source-owned decoder 证明 raw 精确恢复 reviewed candidate。后续 revision 直接编辑这个 passed raw；禁止重保护 candidate、猜 token mapping 或返回原 bridge 的初始 template。Bundle 的内部 `formal/representation-lineage/<unit-id>.json` 冻结原 receipt 与 exact dependency refs，属于 transport proof，不是新的 Unit、QC authority 或 lifecycle receipt。current-check、import、process、retry、Snapshot 和 current re-attestation 使用相同解释；retry 保留同一 historical decoder、protected baseline 与本轮 QC finding。

历史 decoder 与新 bridge authorization 在既有 command-local resolver 中使用不同复用域。每次仍 fresh 验证历史 receipt/frozen dependencies、所选 attempt 的 provenance 与 raw/candidate equality；只有纯 adapter reconstruction 在同 invocation 复用。新 revision 仍完整执行当前 source/input、package scope、glossary/full Review、validator 和 finding/currentness gates，输出或 commit 前重新核对依赖。历史 A 绑定实际 imported lineage proof 与 frozen history，不重放今天的 glossary/command policy；历史 representation proof 不授予当前 eligibility。缺失链、错误 receipt/ref、tamper、source/input delta 或任何 restore 差异均 fail closed。

历史 representation lineage 中的 `locales/<locale>/glossary.yaml` + frozen SHA 仅在 historical read domain 按既有 `readArchivedGlossary` 解析为 `data/glossary-history/<locale>/<frozen-sha>.yaml`，并重新校验 exact bytes。原 receipt、imported lineage path/SHA 与 transport bytes 不变；展平 historical proof 绑定实际 immutable archive ref，不能继续把 current path 当作旧 SHA oracle。archive 的 locale、文件名、SHA、路径安全必须一致；缺失、tamper、非法 scope 或 symlink 均拒绝，不扫描其他同 SHA 文件。除上文明确的 checkpoint historical context proof 外，其他 dependency 保留原 path/exact SHA 检查。current authorization、新 Generation 与完整 Unified Review 仍读取今天 current glossary，不使用此历史解析。
