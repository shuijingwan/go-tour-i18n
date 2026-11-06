# Shared TranslationUnit conformance / future content 扩展审计

production 的新增 frozen Page package 注册与 structured 正文扩展入口见[Source content adoption](SOURCE_CONTENT_ADOPTION.md)；它们只提供 source/descriptor authority，仍沿用本合同的共享生命周期与独立语言 gate。

本文件是工程 conformance authority，必须与 [共享生命周期](SHARED_TRANSLATION_UNIT_LIFECYCLE.md) 和阶段规范一起读取。测试中的语言与审核结论均为 deterministic fixtures，不是正式模型 generation 或 Reviewer judgment。本轮冻结 zh-CN `chatgpt-zh-CN-036` attempt 2；不创建 Snapshot、不导出正式 Reviewer Bundle、不记录 QC、不 finalization、不 promotion。

## 审计方法与 invariant 字典

每个 phase 都检查 A–K；表中 `—` 表示该 phase 不产生相应对象，不能理解为豁免后续 gate。共同合同如下，phase 行记录具体绑定和差异：

| invariant | 每个 phase 的检查问题 |
| --- | --- |
| A identity | UnitID/kind/source SHA/path、predecessor、attempt、所选完整 package scope、Snapshot index 是否明确；不存在的 predecessor/attempt 不伪造 |
| B semantic authority | frozen source/context、全 batch inputs、完整 glossary、finding/authorization 是否完整；模型不可从聊天补输入 |
| C representation | source、protected input/raw response、restored candidate、reviewed predecessor、passed protected predecessor、validation 各自独立 |
| D transport | canonical 单 Unit/batch identity 与 transport identity 分层；所有 embedded/deduplicated bytes 都有 inventory/hash |
| E currentness | source/input/glossary/authority/target validation drift fail closed；只有已认证历史支持的 sibling result drift 可接受 |
| F recovery | partial successes 保留；连续 attempt；interruption 先 recover；pending/mutation unknown 禁止盲目重放 |
| G inheritance | execution retry 不覆盖 origin initial/revision/glossary-stale semantic task；QC/Surface authorization 不互换 |
| H evidence | Reviewer 对 Snapshot candidate SHA；Generation 编辑 protected predecessor raw；validation 绑定该 attempt 的 raw/restored candidate |
| I mutation | export/check/scope 只读；import 只安装 raw/provenance；process/retry 写 validation/candidate；record/finalize 各有唯一入口 |
| J scheduling | initial working sets 不能成为 revision/re-QC TU partition；Page/Example 同 kind、60 上限共享 |
| K neutrality | shared 状态机与 representation adapter 分开；Tour projection、legacy read、package discovery 明确标记 |

## phase × invariant × surface/package matrix（实现前审计）

下表的 Tour / Learn/Docs / future 列明确定位实现，`S` 表示同一个 `internal/i18n` 函数（不是仅同名测试）；`R` 是表示适配；`P` 是 package scope/discovery；`L` 是仅历史兼容；`U` 是尚未完成的工程证明。所有行同时适用上表 A–K，括号指出该行重点。审计初稿中的 U 必须在验收时如实保留或更新，不能仅凭 tests PASS 宣称等价。

| # / phase | A/B/C/H：identity、输入、表示与审核 bytes | D/E/F/G/I/J：transport/current/recovery/继承/写边界/调度 | Tour | Learn/Docs | synthetic future |
| --- | --- | --- | --- | --- | --- |
| 1 discovery/inventory | 完整 Page/eligible Example；package 来源与 route | 初始 inventory 不决定 revision | P `BuildSourceCatalog` | P `WorkflowDocuments` | U descriptor/fixture |
| 2 TU identity | `Catalog.Unit` 的 ID/kind/source；protected identity 不替代 source | 不按目录另建 attempt | S `Catalog.Unit` | S + R `PageAdapter` | U 同 S |
| 3 source freezing | manifest source SHA；Snapshot 引用 canonical source | current rebuild，不以旧 saved input 为 source authority | S export | 同 S | U 同 S |
| 4 protector input | protected text/tokens 与 source 分离 | adapter 只提供表示，不能选择 task | R present/Go protector | R Page coordinates | U 非 Markdown 表示 |
| 5 initial Bundle | 完整正式 manifest/inputs/glossary/authority | inventory + input identity；export 只读 | S `ExportTranslationUnitGenerationBundle` | 同 S | U 同 S |
| 6 output representation | raw 是 protected response，不是 candidate | exact files、UTF-8/BOM/single LF | S output checks | 同 S；`.article` 为 transport 后缀 | U 同 S |
| 7 import/provenance | provider/model、原 ZIP SHA、expected outputs | no-overwrite/pending marker；不批准语义 | S directory/Result import | 同 S；L archived JSON landing | U 同 S |
| 8 restore | raw attempt → restored candidate | 成功 sibling 保留；adapter 报结构错误 | S process + R | 同 S + R | U 同 S + R |
| 9 machine validation | candidate/source/glossary 与 validation | machine PASS 不产生 A | S `ValidateTranslationUnitCandidate` | 同 S + R structure | U 同 S + R |
| 10 restore_failed retry | exact failed ID + next attempt + current validation | S single retry；不重做成功 Unit | S `ProcessRetranslationRetry` | 同 S | U 同 S |
| 11 validation_failed retry | 同10，candidate 不作为新模型 input | failure origin 不变；连续 provenance | 同 S | 同 S | U 同 S |
| 12 interrupted Generation | expected files、恢复有效 raw bytes | CURRENT preflight；冲突 fail closed；staging 非 formal | S `RecoverGenerationOutputs` | 同 S | U 同 S |
| 13 complete candidate set | 全 scope latest numeric batch、passed evidence | 最新失败绝不 fallback；缺 Unit block | S `selectLatestRetranslationUnits` | 同 S + package scope | U 同 S |
| 14 Snapshot | source/candidate/validation SHA、final attempt；index 在 Snapshot 内固定 | full scope，不是 revision subset；仅写 manifest | S `CreateQualityCheckCandidateSnapshot` | 同 S | U 同 S |
| 15 initial QC | Reviewer source/current target 精确 bytes | initial 分组；完整 scope frozen | S reviewer exporter | 同 S + initial policy | U 同 S |
| 16 QC record | A/B/C/D、B/C/D finding、Snapshot SHA | no overwrite；Reviewer role；不生成 replacement | S `RecordQualityCheckResults` | 同 S | U 同 S |
| 17 B/C/D authorization | current predecessor 的 effective non-A + exact finding | 首审完成；kind/60；initial partition 不继承 | S `ExportRetranslationBatch` | 同 S | U 同 S |
| 18 exact revision Bundle | 三种表示 + Snapshot + finding/authorization | complete original batch inputs；replacement exact set | S Bundle | 同 S | U 同 S |
| 19 predecessor candidate | reviewed restored candidate path/SHA | 不读 latest；缺 bytes fail closed | S `revisionGenerationInputs` | 同 S | U 同 S |
| 20 protected predecessor | passed validation raw path/attempt → restore exact candidate | 不逆向保护 candidate；baseline inventory/hash | 同 S | 同 S + R restore | U 同 S |
| 21 revision import/process | protected replacement → raw/provenance → candidate | 不覆盖 predecessor；新 batch/attempt | S import/process | 同 S | U 同 S |
| 22 revision-origin retry | finding、两种 predecessor、原 source/full inputs 均保留 | execution retry/origin revision 分离 | S Bundle/retry | 同 S | U 同 S |
| 23 successor Snapshot | 全 scope 包含 changed + unchanged | latest selector + exact evidence；旧 Snapshot 不变 | S Snapshot | 同 S | U 同 S |
| 24 successor re-QC | current `quality_check_required` pending stable order | 同 kind/60；不继承 initial group | S reviewer selector | 同 S | U 同 S |
| 25 re-QC record | predecessor 首次持久化，后续 exact-match | 不补写/改换 lineage；changed bytes 要真实新 A | S record | 同 S | U 同 S |
| 26 A-only finalization | resolved A 覆盖全部 TU；bind result lineage + SHA | 纯 machine 决定；pending/non-A 拒绝 | S `FinalizeQualityCheck` | 同 S | U 同 S |
| 27 glossary-stale | current protector 与 saved input 真 drift；compatibility | 专门 reexport reason；不伪造 finding；既有 initial 分组保留 | S export/compatibility | 同 S + R protector | U 同 S |
| 28 Surface reopen | finalized A + exact formal Markdown finding/Unit IDs | 不改旧 A；immutable authorization | S `RecordQualityCheckSurfaceReopen` | 同 S | U 同 S |
| 29 Surface revision | reopen exact set + predecessor A + finding | 同 revision exporter；非旧 Site state machine | S export/Bundle | 同 S | U 同 S |
| 30 Surface retry | origin Surface revision + failed Unit | 同 machine retry；不失去 reopen authorization | S Bundle/retry | 同 S | U 同 S |
| 31 package closure | finalized complete TU + 必要 structured assets | post-QC exact projection，不再次 rating | Tour canonical/receipt | P `BuildPackageClosure`；L 旧 finalization | U generic descriptor boundary |
| 32 promotion | current finalized A-only selected candidates | shared authorization；输出位置是 representation/projection 差异 | S + R Tour canonical/status | shared finalization → package closure/activation | U 同授权，不写 Tour projection |
| 33 history/carry/current | immutable old source/candidate/validation/attempt/rubric/glossary lineage | exact matching A；compatibility 有证据；不抹历史 | S QC evaluation | 同 S | U 同 S |
| 34 JIT stale | Bundle 完整 scope + 本组 source/target | sibling record 改变 scope，未来预导 ZIP 正确 stale | S reviewer check | 同 S | U 同 S |
| 35 retry handoff | canonical per-Unit Bundle SHA/input identity/attempt | 当前每 failed Unit 一 ZIP，全 batch shared inputs 重复 | S single exporter | 同 S，无独立 aggregation | U shared aggregation proof |

## 隐藏 branch 分类与审计发现

搜索入口：`ContentPackage/learn-docs-v1/sitecontent/Tour/Adapter/TaskKind/PreviousSnapshotID/RevisionFeedbackSource/UnitID/WorkingSet/UnitKindPage/UnitKindExample`，覆盖 `cmd/tour-i18n`、`internal/i18n`、`internal/sitecontent`。

合法表示差异：Tour Section extraction / appengine projection、eligible Go comments、Page/Example 后缀与 token-order、PageAdapter 块坐标恢复与结构校验、Tour canonical article/status 安装、package files/routes activation。它们不能决定 QC 或 retry origin。`TranslationUnitAdapter` 只有 ProtectedInput/Restore/Validate；`ArchivedGenerationAdapter` 仅验证旧 invocation 的表示。

合法调度差异：Learn/Docs current 51 Page 固定 initial 30/21（历史49页为30/19），Tour仍为60/43/19；`RequireInitialTranslationWorkingSet` 与 `RequireTranslationInvocation` 分离。glossary-stale 保留已有 initial 分组是显式 authority；revision/re-QC 不继承它。Snapshot index 是 immutable Snapshot 内的顺序，不等于 source registry 的跨时间 append-only index（retired 页不会复用 registry index）。统一 package的历史组合见[同一 closure 的 finalization composition](LEARN_DOCS_MEMBERSHIP.md)，不产生新 A或另一套生命周期。

历史边界：schema-v1 Page manifest/result/validation decoder、legacy EOF、Final Review 30 限制、旧 Site JSON invocation archive/landing、旧 Site finalization 读取、旧 provider/model 真实 provenance。兼容读取不授权新增旧 Site QC/revision/finalization。既有中文 zh-CN artifacts 不迁移。

共8类记录：4类已有 WIP 修复（initial membership 泄漏 revision/re-QC；revision Bundle 缺 reviewed candidate；protected predecessor 缺失/表示混淆；retry execution task 丢 origin revision inputs），targeted tests 验证并保留；首轮修复3类（retry handoff 重复、package CLI whitelist/重复 post-finalization selection 收敛为 descriptor + shared authorization、historical glossary branch 绕过 adapter 强制 Present validator）。第8类 Integrated Surface → shared reopen authority gap 经维护者批准，以表示无关 artifact binding 收口，状态 **CLOSED**。8类已知 divergence 全部 CLOSED，未新增 package-specific 状态机。

shared package descriptor 当前接完整 Page TU；standalone eligible Go Example 保持既有 Catalog inventory。新 source representation 使用 Page kind，不因 XML 等表示增加 kind/schema。新 kind 属于 TranslationUnit contract architecture review，不是正常新增 package。

`PackageDocuments` 的 Learn/Docs 56-file exact set、Page registry 1–60 与 frozen source ZIP，以及 `BuildPackageClosure` 当前已注册产品名单属于 source inventory / activation descriptor 的 frozen baseline。future source reader/parser/descriptor 可以 additive 注册，进入 shared authorization；其 output projection 不得另写 QC/finalization。旧 Site `ExportPackageGeneration` / review / finalization 函数保留为 legacy artifact interpretation/fixtures，当前 CLI 禁止新增；这些旧 tests 不作为 shared conformance proof。

## 实现后矩阵证据更新

上表 U 记录的是实现前历史缺口；下表逐 phase 更新最终证据。S 的含义仍为同一个 shared implementation，绝不是仅共享测试名称。`XML test` 位于 `internal/i18n/future_content_conformance_test.go`，Integrated receipt/release proof 在 `internal/sitecontent/surface_authority_test.go` 延伸同一 XML card fixture；均使用 temporary repository，没有生产 fixture 分支。最终 unresolved=0。

| phase | future 与 Tour / Learn/Docs 的最终同一路径证据 |
| --- | --- |
| 1 | `RegisterTranslationPackage` + test-only XML descriptor；Learn/Docs loader 也改用同一注册入口 |
| 2 | `Catalog.Unit`；descriptor 拒绝重复/shadow identity，无 new kind |
| 3 | shared export/source SHA + Snapshot；XML test |
| 4 | R XML card attributes/envelope；只实现 ProtectedInput/Restore/Validate |
| 5 | `ExportTranslationUnitGenerationBundle`；XML test initial |
| 6 | protected card raw → XML candidate，已有 `.article` transport 后缀不成为 parser identity |
| 7 | `ImportGenerationOutputDirectory`；fixture provenance，aggregate 测试核验原 ZIP SHA/input identity |
| 8 | `ProcessRetranslationBatch` 调用 R restore；XML test |
| 9 | `ValidateTranslationUnitCandidate` + shared archived-glossary dispatch；XML 实际 historical evidence branch 回归 |
| 10 | exact `ProcessRetranslationRetry` attempt 2；XML restore_failed |
| 11 | 同10；XML required heading 结构 validation_failed |
| 12 | `RecoverGenerationOutputs`；XML staging 完整 output recovery |
| 13 | `selectLatestRetranslationUnits` + 完整 scope；XML 三个 initial batches |
| 14 | `CreateQualityCheckCandidateSnapshot`；full 3-card scope，两个 final attempt 2 |
| 15 | shared Reviewer Bundle；XML 三个 initial working sets |
| 16 | `RecordQualityCheckResults`；fixture B/B/A，不是真实语言审核 |
| 17 | shared revision export；XML 两个 B 跨 initial groups |
| 18 | shared revision Bundle，XML test |
| 19 | `revisionGenerationInputs`；exact restored XML predecessor |
| 20 | 同19；protected card baseline 与 restored XML 独立；passed raw/attempt binding |
| 21 | shared directory import/process；XML test |
| 22 | shared retry Bundle；逐 predecessor bytes 对比，origin revision context 保留 |
| 23 | shared Snapshot；两个 successor |
| 24 | shared pending selector；2 pending/1 carry，然后1 pending/2 carry，不继承 initial partition |
| 25 | shared record；fixture independent new A |
| 26 | shared finalize；拒绝初始非A，两个 successor A-only PASS |
| 27 | shared source trace；XML protector 不随 glossary 改变，不制造虚假 stale；既有 glossary-stale targeted test PASS |
| 28 | **CLOSED**：同一 `RecordQualityCheckSurfaceReopen`；Integrated receipt reader + shared exact mapping/artifact binding，Tour carried A、Learn/Docs PageAdapter、XML adapter 都通过 |
| 29 | 同一 shared revision exporter/Bundle；receipt、reviewer evidence/context、predecessor artifact 冻结；不恢复旧 Site exporter |
| 30 | 同一 Surface-origin retry；XML 两个 failed siblings 用一个 aggregate，首个成功后原 sibling 仍 CURRENT；finding/baseline/receipt 完整保留 |
| 31 | `FinalizedTranslationPackageCandidates` + `BuildTranslationPackageClosure` exact registered descriptor/authority + latest/A-only proof；复用既有 PackageClosure schema，XML3文件与独立 Integrated PASS closure 通过 |
| 32 | shared package current authorization拒绝旧 finalized package；Integrated gate绑定新closure，旧failed receipt不能授权新invocation；fixture不注册public routes、不执行Production/activation |
| 33 | shared QC carry/currentness，XML test；historical glossary adapter 分派回归 |
| 34 | shared Reviewer checker；XML record 后原 Bundle stale；Learn/Docs fixture 另覆盖预导未来组拒绝 |
| 35 | `ExportRetryHandoff` / `ReadRetryHandoff` / `VerifyCurrentRetryHandoff`；Tour fixture、XML revision/Surface retry 走同一实现 |

## Integrated Surface finding authority：CLOSED

维护者已批准补齐该 edge，并要求 canonical candidate artifact 不固定为 Markdown。原 gap 是 Integrated receipt 使用 source path，而直接 reopen 只读历史 Markdown/UnitID；旧 `surfaceRevisionPages` 属于已停止新增的 Site workflow，不能恢复它。

最终链：原 immutable Integrated receipt path/SHA + review id/decision/locale/session + scope identity + reviewer evidence hash + closure/runtime references → explicit language finding/affected artifacts → `Catalog.ExactSourceTranslationUnit` → Snapshot canonical candidate artifact → 同一 shared reopen authorization。`SurfaceCandidateArtifact` 复用 `QualityCheckSnapshotUnit`，包含 UnitID/kind、source path/SHA、candidate path/SHA、validation path/SHA、selected batch、attempt、stable Snapshot index，另绑定 SnapshotID。它不解析文件后缀，也不把 protected raw response当canonical candidate。Tour Present、Learn/Docs Markdown/HTML、future XML由原TU/adapter产生各自表示。

映射以source path + UnitKind + frozen source SHA为完整source identity，恰好一个结果且UnitID等于receipt显式scope。Tour同一物理article中不同section由各自source SHA区分；相同identity重复仍拒绝。0 match、多个match、未知ID/kind、source/candidate/validation/Snapshot drift均fail closed；不读route/latest猜Unit。

在既有v1 receipt/scope中增加可选 `class` / `artifacts`、`source_path` / `unit_id` / `candidate_artifacts`；现有hash/finalization字段复用。必需性：旧Tour TargetSHA是可见context projection，不能可靠派生完整canonical candidate；旧finding只有ID/prose，无法判断TU语言缺陷还是navigation等问题。新增字段只明确审核对象与finding分类，不bump schema，不产生新状态机/receipt类型。旧receipt缺这些字段不推断TU授权；历史直接Surface evidence路径保留兼容。`reviewer_markdown`保留原JSON字段名，仅表示Reviewer evidence reference，不限定TU artifact表示；fixture使用独立`.txt` evidence。

`class=translation-unit-language`且explicit artifacts属于scope中的exact candidate才可授权。已carried A也可由新receipt明确指出语言缺陷，但不因此获得新的TU A；unaffected A继续carry。UI/structured/navigation/sitemap/canonical/runtime/ads/coverage/package integration等finding保持Surface/package blocker，不生成伪UnitID，不进入TU revision。多个Unit必须explicit列出，exact-set不自动扩展邻近path。

record reopen和首次revision export检查current scope/closure与当前finalized selected candidates。export后进入原invocation的冻结证据合同：每次读授权仍核对原receipt/evidence/context/source/predecessor hashes；不因自身revision sibling成功重算旧closure/latest后误判另一个retry stale。新的invocation必须再次current-check，原failed receipt在candidate/scope变化后拒绝。Generation Bundle与revision-origin retry内嵌原authority references，不要求复制/重新命名Reviewer evidence。成功后新full Snapshot→exact re-QC→A-only→新PackageClosure→新的独立Integrated Surface PASS；旧failed receipt不被改写或变成PASS。

Tour、Learn/Docs、XML使用同一reader、mapping、RecordQualityCheckSurfaceReopen、revision/Bundle/retry/QC/finalization；只有inventory/context和artifact表示不同。test-onlyscope builder只提供隔离descriptor inventory，调用同一scope composition/receipt validation/release gate；不增加future-package workflow分支，不污染public registry。XML two-sibling Surface aggregate retry和unaffected A carry均有确定性fixture证明。

## Shared Retry Handoff transport

`go-tour-i18n/retry-handoff/v1` 是通用 transport manifest，不是 formal retry schema/receipt。已有 Generation Bundle 无法只嵌入一次 shared bytes：其完整 ZIP SHA 本来就是逐 Unit provenance 的一部分。此 transport envelope 用 `objects/<sha256>.txt` 保存相同字节一次，members 按 immutable batch Unit 顺序绑定 index、UnitID、attempt、expected output、原 single ZIP SHA、input identity 与原 manifest object。原 manifest Files 的每个 hash 解析到一个 object；shared 与 per-Unit bytes 的区分只按 hash，不猜 semantic authority。

完整性检查重建每个原 ZIP、核对 exact SHA/inventory，并重建 aggregate 做逐字节 canonical 比较，拒绝缺/额外对象、错误 ordering/metadata/output/identity。aggregate 与 reconstructed input 总量均 bounded；不增加 formal transport safety limit。

CLI：`generation-bundle retry-handoff --bundle <single.zip> [--bundle <single.zip> ...] --output <aggregate.zip>`；`retry-handoff-check --bundle <aggregate.zip>` 逐 constituent 调用原 checker。`retry-handoff-unpack --bundle <aggregate.zip> --output-dir <new-local-dir>` 原子且 no-replace 地恢复各原 ZIP，文件名为原 ZIP SHA；这是完整性 extraction，**不宣称 CURRENT**。

handoff 前 aggregate 必须全部 constituent CURRENT。模型完整读取 member manifest 对应的全部 objects（formal inputs/glossary/authority/predecessor/validation），只生成各 expected output。记录 provenance 和 import/recover/result-pack 仍只接受重建的原 single ZIP，provider/model 与 attempt 各自独立；aggregate 不能传给 formal importer。一个 member 导入后 aggregate fail-closed stale；未处理 sibling 继续用原 single ZIP checker 和 authenticated historical result，不重导/重新绑定模型输入。aggregate stale 后仍可 integrity-unpack 原 handoff，但不能作为新的 CURRENT invocation。

实测 Tour fixture：3个原 ZIP合计532,188 bytes，一个aggregate 196,795 bytes；乱序consume后所有 sibling 仍独立有效，provenance继续绑定各原SHA。XML revision/Surface retry 也经过同一 aggregate实现。本轮未为已经13/13 PASS的正式zh-CN batch导出新的retry ZIP。

## Tour 与 Learn/Docs retry transport 历史证据

`73fe4058` 引入 Generation ZIP 时的 `generation_bundle.go` 已以 optional 单一 `UnitID`、单一 next attempt 建立 retry，完整 batch inputs 一并嵌入；没有 multi-retry exporter。当前 exporter 延续同一机制，Tour 与 Learn/Docs 都调用它。`0d54da81` 的 ChatGPT 规范明确支持直接向每 Unit retry 路径安全安装；当时没有 importer。`0b88e108` 加入本地目录 import，也没有改变 canonical single retry identity。

因此不存在 Tour aggregation 未被 Learn/Docs 复用的证据。旧本地路径无需为每 Unit 上传附件，是可证明的环境/transport 差异；不能从这些实现证明用户某次历史 invocation 的附件数、失败数或 batch size。当前13个 retry ZIP 的 repeated authority/glossary/full inputs/predecessor 属于 transport duplication，不是 Learn/Docs 独立状态机。canonical per-Unit identity 是现有 attempt/provenance/currentness 合同；聚合仅在 handoff 层去重，不改它。

## Adding a Future Content Package

1. 定义 frozen scope/inventory、source hashes、retirement/move mapping、route/context；不联网漂移正式 baseline。
2. 用 `Catalog.RegisterTranslationPackage` 注册 descriptor，`AuthoritySHA256` 绑定当前Content Scope package identity，再用 `UseTranslationPackage` 选择完整 scope；production CLI 的 source loader 在 `sitecontent.translationPackageLoaders` additive 注册。scope 不能静默扩大 Tour 默认 inventory，也不能把 initial WorkingSet 当永久身份。
3. 选择或实现 `TranslationUnitAdapter`；只解析/protect/restore/structure validate，保持 canonical source 与 protected output 各自表示。共享 lifecycle 解释 task/attempt/predecessor/QC。
4. 建立结构 positive/negative fixtures，以及 source/input/route drift、EOF、token membership 等表示测试。
5. 运行 shared conformance targeted tests，覆盖 initial/retry/QC/revision/revision-origin retry/re-QC/A-only/Surface；只在临时 repo 写 fixture evidence。
6. corpus/glossary 独立审核与 package source authority 完成后，进入既有正式 locale workflow，沿用相同 Generation/Reviewer session 与 provenance policy。post-QC projection 必须调用 `FinalizedTranslationPackageCandidates`；TU-only descriptor可复用 `BuildTranslationPackageClosure`，mixed structured assets继续组合既有structured gate，不复制latest/current/A-only selection。Surface receipt mapping/currentness使用既有shared reader + TU/adapter identity，正常package author不重写reader或reopen。release仍需独立Integrated Surface PASS；test-onlyXML proof不表示发布了真实package。

正常新增 package 通常不得修改 `generation_bundle.go`、`generation_retry_currentness.go`、`generation_revision_inputs.go`、`retranslation_process.go`、`retranslation_retry.go`、`candidate_snapshot.go`、`quality_check_results.go`、`quality_check_bundle.go`、`quality_check_finalization.go`、`quality_check_reopen.go` 的 state-machine semantics。若必须修改，先解释现有 TranslationUnit contract 无法表达的真实语义/证据缺口，触发 architecture review；不得为新 package 新建 schema、receipt 或平行 workflow。

## 验收记录

`result: PASS`。共8类 divergence：4类已修WIP经验证；首轮修复3类；最后Integrated Surface authority bridge已按维护者批准收口。全部CLOSED，unresolved=0。futureXML Integrated Surface/package closure为temporary fixture证明，不执行真实locale语言/发布阶段。

首轮targeted checks：`internal/i18n` 14个明确命名的测试（含subtests）PASS；`internal/sitecontent` 的3个 SharedPage targeted tests PASS；`cmd/tour-i18n` 的4个Generation/import/recovery/handoff flags tests PASS。

最后closure targeted checks：原XML完整lifecycle、结构/registration、legacy Surface reopen、protected revision两类failure/retry、QC lineage/A-only、retry aggregate两个测试共8个命名i18n测试PASS；新增inventory-wide artifact lookup与runtime reader不改变catalog identity的2个focused tests PASS。Integrated Surface三表示闭环、14类fail-closed负例PASS，最后artifact lookup调整后只重跑直接涉及的exact-scope/首次export scope-stale负例PASS。既有structured Surface exact revision兼容测试PASS；正式49-page descriptor/mapping focused regression PASS（隔离fixture，不写正式locale）。CLI3个指定flags/import tests PASS；其首次启动遇Go cache只读，按sandbox规定重跑后PASS。没有残留test failure；未运行package-wide/repository-wide/browser/build/preview/publish/Production测试。

`git diff --check` PASS。最后closure正式状态 fingerprint 再核验覆盖48,085个文件，changed=0；`chatgpt-zh-CN-036` 保持13/13 restore与validation PASS，最后一个Unit保持passed/attempt=2。`formal_state_mutation: none`。当前未启动 zh-CN 后续语言阶段。`project_instruction_delta: none`。既有dirty WIP保留；未stage/commit/push。
