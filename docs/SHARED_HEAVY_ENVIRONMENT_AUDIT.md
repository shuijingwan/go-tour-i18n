# Shared heavy environment 消费检查

本文件记录 2026-10-29 的工程实现边界，不修改正式 CLI、schema、currentness、
mutation、CAS、write-once 或恢复合同。检查方式为静态入口/控制流检查和隔离 fixture，
没有执行下列真实 lifecycle command。

## 已共享的数据节点

所有 CLI 经 `run` 的 invocation-local `BeginMemoization` 进入已有 L1/L2。

| 节点 | 数据输入与结果 | 每次仍执行 |
| --- | --- | --- |
| Source Catalog | exact Tour bytes/manifest/options → Pages/Conditional/Examples | fresh tree、source/hash、manifest 对比及最终观察 |
| Global scope basis | exact frozen snapshot + fresh routes/source inventory/options → 完整 Global 基础图及 package identities | fresh live inputs、声明的 Global 对比、adoption/membership/current authority |
| Historical Page projection | frozen Documents + ordered descriptor + current source-owned Page data → 历史 descriptor 与 replacement bindings | fresh frozen reference/hash、调用方 Snapshot/QC/candidate/validation/provenance/currentness |
| Base package representation | exact snapshot/package/source inventory 或 fresh shell source → 整包 Documents/protection graph 与 raw map | membership 有序 predecessor 解析、source observation、adoption/current structured successor 检查 |
| Generation archive index | freshly read exact archive bytes → immutable member index | archive SHA、import/provider/model/attempt、inventory/raw/provenance |
| Workflow Page projection | 已观察的整包 Documents/raw + fresh Page registry → 有序 formal Page graph | package/adoption 选择、registry identity、原始 parser/protector；不可用时走 canonical reader |
| Ordinary bridge representation | 完整 source-owned Page/mapping/raw/reviewed bytes → ordinary template/ancestor/adapter 数据 | historical/QC/Surface 授权、exact restore、current/frozen dependencies；不保存 runtime decoder |
| Historical structured graph | exact frozen Document、raw source、已恢复 target bytes → 完整 source/target parser 与 protection graph | frozen source/hash/坐标、protected target 恢复、graph 的 structure/exact-set 比较、外层 QC/Surface/provenance/currentness |

各节点只保存确定性数据，不保存 PASS、authorization、QC carry、currentness 或 mutation
eligibility。缓存不可用或 codec 不可用均重算。未知/有运行时状态的 adapter、lossy codec
及未拆开的 mutable authority 路径继续原始计算。其他 package 与 runtime readers 从新
Catalog 绑定，不能由历史 Page 投影替换。

## 正式入口检查

下表的“剩余”不是新共享节点被绕过，而是尚未拆成完整 observed-input/pure-derivation
接口的上层证明。A：current corpus/source successors/package/current authorization；
B：effective QC、Snapshot re-attestation、same-unit/Surface bridge history；
C：closure/Integrated Surface/activation 的完整 scope 重建。分支可能仅在相关 representation
或历史 authority 存在时进入，不能把静态可达性当成所有 invocation 都实际调用。

另有入口 D：`WorkflowDocuments → PackageDocuments`。下表保留原 23 条路径的静态审计，
本轮只按共同 owner 增量复核，不把条件可达性计为 23 个独立性能问题。D 的三个 workflow
调用已改接 `sharedWorkflowDocuments`；两个受冻结实现身份约束的入口仍未迁移，详见下文。

| COMMAND | library entry | 共享消费方式 | 剩余 |
| --- | --- | --- | --- |
| quality-check representation-bridge | PrepareSameUnitRepresentationBridge | command Catalog + shared library builders | A/B |
| quality-check representation-bridge --record | PrepareSameUnitRepresentationBridge | 同上；保留第二次 fresh construction | A/B |
| retranslation export | ExportRetranslationBatch | command Catalog + resolver/library builders | A/B |
| generation-bundle export | ExportTranslationUnitGenerationBundle | command Catalog + resolver/archive builders | A/B |
| generation-bundle current-check | VerifyCurrentTranslationUnitGenerationBundle | 独立 canonical reconstruction 消费同一 builders | A/B |
| retranslation process | ProcessRetranslationBatch | readonly registration + resolver/archive builders | A/B |
| quality-check snapshot | CreateQualityCheckCandidateSnapshot | command Catalog + historical membership readers | A/B |
| quality-check preflight | BuildQualityCheckPreflight | command Catalog + QC evaluation | A/B |
| quality-check scope | BuildQualityCheckScope | command Catalog + QC evaluation | A/B |
| quality-check reviewer-bundle | ExportQualityCheckReviewerBundle | command Catalog + scope | A/B |
| quality-check reviewer-bundle-check | VerifyCurrentQualityCheckReviewerBundle | 独立 reconstruction + scope | A/B |
| quality-check record | RecordQualityCheckResults | command Catalog + scope/evaluation | A/B |
| quality-check record-batch | RecordQualityCheckResultBatch | command Catalog + scope/evaluation | A/B |
| quality-check finalize | FinalizeQualityCheck | command Catalog + full QC authority | A/B |
| quality-check authorization | RequireCurrentTranslationAuthorization | command Catalog + current assessment | A/B |
| site-content package-closure | FinalizePackageClosure / FinalizeComposedPackageClosure | shared source/closure readers | A/B/C |
| site-content surface-scope | BuildIntegratedSurfaceScope | shared closure/source/authority readers | A/B/C |
| site-content surface-reviewer-bundle | ExportIntegratedSurfaceReviewerBundle | current scope reconstruction | A/B/C |
| site-content surface-reviewer-bundle-check | CheckIntegratedSurfaceReviewerBundle | independent current scope reconstruction | A/B/C |
| site-content surface-record | RecordIntegratedSurface | current scope reconstruction | A/B/C |
| site-content activation-preflight | ActivationPreflight | LoadCurrent + closure/Surface readers | A/B/C |
| site-content activation-apply | ApplyActivation | fresh preflight + prior/CAS | A/B/C |
| site-content finalize | FinalizePackage | LoadCurrent + generation/review readers | A/B |

所有 `BuildGlobal` consumer 已由唯一公开入口消费 shared Global basis；普通历史 Page
resolver 经 `sharedHistoricalPageProjection`。没有为 command 另建专属 cache。
所有 `CurrentPackageDocuments` consumer 自动消费 shared base representation，包括其
由 membership authority 选择的 predecessor。不能把“依赖 SHA 未变”直接当成完整
current authority PASS，更不能持久化 `readSameUnitBridge` 的成功结论。

## 共同 owner 增量复核

| GROUP | 共同入口 / 原 consumer | 本轮收敛 | 尚未覆盖 / 必须保留 |
| --- | --- | --- | --- |
| Workflow Documents（D） | WorkflowDocuments；Page 注册、workflow generation/review、shell successor | workflow.go 的三个调用经 sharedWorkflowDocuments，共用既有 package representation；formal Page projection 独立进程 cold=1 / warm=0，与 canonical 完整数据身份一致 | AttachPageTranslationUnits、BuildShellSuccessorAffectedScope 仍走 canonical reader |
| Current corpus/source（A） | CheckUnifiedCorpus；adopted Page ProtectedInput、current authorization、bridge reconstruction | recorded bridge 的 readonly construction 有显式 source-proof owner；同 phase current-source projection 首次构建 1 次，后续 0 次；新 phase 新 owner | current 判定不跨 invocation；record 保留 eager entry/final guards |
| QC / Snapshot（A/B/C） | qualityCheckEvaluation、AssessCurrentTranslation、closure/Surface scope | 消费 source、Global、package、archive、historical Page 和 historical structured graph；不增加 command cache | current/历史授权判定和冻结 native adapter 边界仍逐次验证，不持久化 evaluator/proof 的 PASS |
| Bridge representation（B） | readSameUnitBridgeForInvocation、source-owned ordinary factories | Catalog 私有 construction context 共享 source proof 和 pure representation；独立进程 representation cold=1 / warm=0；restricted-comment / Surface 构造与 canonical receipt bytes/SHA 相同 | frozen historical lineage、QC finding、Review、candidate/validation、import provenance 仍按原 reader 验证 |

record CLI 使用 eager 的 `BridgeRepresentationCatalog` 数据视图。原两次 complete fresh
construction、receipt bytes 比较、最终 dependency verification 和 write-once publication
顺序不变。record 没有在写入后才执行的 deferred guard；第二次 construction 不复用第一次
currentness verdict。readonly recorded bridge 的完整 proof context 在返回任何结果前关闭
aggregate guard，不供其他 invocation 使用。未知 adapter 保留原实现。

### 三项剩余入口的最终分类

1. `AttachPageTranslationUnits → WorkflowDocuments`。`translation_adapter.go` 属于
   receipt 的 ProjectionImplementationSHA256 及 PageAdapter authority 依赖；该函数内部
   直接选择 source Global、调用 WorkflowDocuments 并构造/注册完整 descriptor，不接收
   外部 Documents。WorkflowDocuments 又直接调用 units.go 中的 PackageDocuments/parser。
   三层都由现有实现身份绑定，没有可注入已派生 graph 的接口。外部替代注册器会复制正式
   注册实现；修改任一内部调用则改变冻结身份。本任务不采用这两种方式。
   **EXPLAINED_REQUIRED_REBUILD / required-by-frozen-identity**。
2. `BuildShellSuccessorAffectedScope → WorkflowDocuments`。`shell_successor.go` 自身
   在冻结 parser dependency 集中。该函数在已验证 completion/adoption 后直接读取全部
   Learn/Docs Page，并对 source exact-set/accepted targets 生成 affected scope，不接受外部
   Page graph。共同底层同样是冻结的 WorkflowDocuments/PackageDocuments。外部预构建
   不能替换内部调用；更改调用或底层函数会改变 successor authority identity。
   **EXPLAINED_REQUIRED_REBUILD / required-by-frozen-identity**。
3. QC / Surface history 组合层：新增共享的历史 structured parser/protection **数据图**，
   其完整两次 parse/protect 派生独立进程 cold=1 / warm=0。不是缓存重建是否通过。
   原 frozen Document/source 验证、target exact-set/UTF-8/protected 恢复仍在调用前执行，
   source/target graph 的 structure/count 比较仍在返回后执行。无法读缓存或 codec 不可用
   都重跑同一 canonical parser。**SHARED_DETERMINISTIC_DERIVATION**，其余 native
   验证按下表分类保留。

第三项的 A/B 边界如下，未重新审计 23 个 command：

| 共同调用链 | 每次 fresh 验证 A | 可复用的数据派生 B / 剩余原因 |
| --- | --- | --- |
| VerifyHistoricalQualityCheckFinalization → representation/history readers | Snapshot/QC/Reviewer A 起源、逐 Unit candidate/validation/attempt/result/import、冻结 refs 的最终重读 | 现有 Source/Global/package/archive/Page projection；effective rating/carry map 是接受结论，不进入 L2 |
| validateIntegratedSurfaceAuthority → validateIntegratedTUFindings / current scope | 独立 session、receipt/evidence/context SHA、finding exact-set、Snapshot artifact membership；current scope 必须重新比较 | 已共享数据节点；Surface current/failed/passed 判定不是可缓存数据图 |
| closureValidation.computeHistoricalStructuredGeneration → validateFrozenStructuredGeneration → reconstructFrozenStructuredRepresentation | Generation/Bundle/glossary/corpus/Review identity、frozen source/slot coordinates、protected targets、最终 references/current authority | **新增 historicalStructuredGraph** 保存完整 parser/protector Documents；不保存文件系统、Review 或 provenance 的验证结果 |
| machineReconciliationContext.buildUnit / historical bridge successor | 精确 predecessor A、当前/历史 source/input、raw/import/attempt、mapping/template/candidate equality 和 native validation | 既有 replacement representation、bridge representation、archive index；接受比较继续执行 |
| native frozen Shell/comment/Surface decoder | source-selected mapping 与 exact restore / frozen structure 对比 | 接口同时包含 native 验证与构造；shell_reconstruction.go、comment_revision_transport.go、same_unit_bridge.go 等已绑定 adapter/receipt implementation identity。现有 pure factory/data hooks 可共享的部分已共享；未暴露独立 graph 接口的冻结 native 验证仍须原样调用，不能以其前次成功替代 |

当前已知剩余项计为 **7 类 EXPLAINED_REQUIRED_REBUILD**，不是动态调用次数：

- **3 类 fresh authority 验证**：source/reference/hash/current pointers；QC/Surface/
  Review/adapter 的当前比较；历史 candidate/validation/attempt/import/provenance。
- **3 类 frozen identity 边界**：上述 Page 注册、shell successor、没有独立 pure graph
  接口的 native frozen adapter/transport 验证。没有修改这些冻结函数正文或 identity。
- **1 类 publication 边界**：第二次 fresh construction、receipt bytes 比较、最终
  dependencies 和 write-once/CAS。第二次不继承第一次 currentness verdict。

这三项已完成分类，**UNEXPLAINED_EXPENSIVE_REBUILDS_REMAINING=0**。这不表示所有
重复计算均已消失，也不代表所有 warm formal command 已达到时间目标。性能专项到此
收口；后续真实 lifecycle 记录 WALL_SECONDS 与 CACHE_STATE，warm/mostly-warm 超过
600 秒时只调查新的具体 heavyweight evidence，不自动重开全量优化。当前任务没有运行
真实 lifecycle 或修改任何正式文件。

## 回归边界

`TestSharedHeavyEnvironmentOwnership` 检查 canonical/derived kernel 只能由对应 owner
调用，并检查 CLI 保留 invocation scope。独立进程测试约束实际 full data derivation
cold=1、warm=0，另有 source/manifest/snapshot/representation 漂移和 codec 回退测试。
整包 representation 也有独立进程 cold=1 / warm=0 与 canonical Documents/raw 等价测试。
这些 guard 不禁止正式 currentness verification，也不声称全部 QC graph warm build 为零。
新增 guard 限定 readonly bridge observation lifetime 只能由 construction owner 设置，
并防止增加绕过共享 Workflow owner 的调用；上述两个冻结入口显式保留。bridge fixture
同时对比 canonical/shared planned receipt 和 recorded receipt，覆盖两次 fresh build，
并检查 source/QC/candidate/validation/archive/Surface/historical drift。clone fixture
为每个新 root 重新绑定 owner，不放宽 production 的 root isolation。

小型 fixture 时间不用于 production 的 300/600 秒门禁。实际 warm 性能留给后续真实
lifecycle 验收；本工程任务仍停在 before-generation-bundle-export，不代替维护者执行命令。
