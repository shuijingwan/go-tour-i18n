# Approved-interval predecessor migration：contract revision v2

**状态：contract v2 已审定并已实现；本文保留原 migration 的 first-real-use 停止边界。** 已安装 invocation 后来 proof stale 的下一阶段以[独立 recovery 设计 v1](INSTALLED_DERIVED_TRANSPORT_STALE_RECOVERY_DESIGN.md)为准，目前只完成设计。本文的 whole-interval、provenance priority 与 typed nonoverlap 合同不变。工程测试不授权真实 selection、064 export/current-check、Generation 或任何 formal mutation；first-real-use 仍遵守 [Formal Command Contract v3](WORKFLOW_HANDOFF.md#formal-command-contract-v3)。

修订原因：旧合同把原 Generation archive 设为唯一 historical provenance 入口，同时将 legacy proof 视为未来可能。现在正式 [`generation-bundle legacy-provenance`](LEGACY_GENERATION_PROVENANCE_AUTHORITY_DESIGN.md#正式-implementation-command-contract) 已完成 design、implementation、first-real-use、readonly inspection 和 write-once record。应修正的是 approved-interval 对该独立 authority 的消费合同，不能再要求重走已完成的 legacy record，也不能由此直接开放 representation transport。

## 已记录 authority 与当前 case

以下身份与完成状态消费维护者本轮提供的正式交接，不重新运行 record 或 legacy inspection。后续真实 consumer 仍必须 fresh 调用 verifier；本轮的 formal inventory 只用于证明 bytes 未变，不是 provenance/语言资格重验。

```text
LEGACY_AUTHORITY_ID=6b6304a944e3abdda45deed5dfc0c00576529a5a6e37533f92ce4d3cbf5e6053
LEGACY_AUTHORITY_PATH=data/generation-provenance-authorities/zh-CN/6b6304a944e3abdda45deed5dfc0c00576529a5a6e37533f92ce4d3cbf5e6053.json
LEGACY_AUTHORITY_SHA256=17292b864444bb98746dd7b65d0eb3babaa7f1785c1b77fe31d325ced2916709
SCHEMA=go-learning/legacy-generation-provenance/v1
CAPABILITY=narrow-per-unit-historical-response-provenance
CONSUMER_CONTRACT=historical-response/v1
LOCALE=zh-CN
CONTENT_PACKAGE=learn-docs-v1
CURRENT_UNIT=b49d8e873a0f66f6803077eb534583a155d2bd209a032d7c00fba8fea856da5c
HISTORICAL_UNIT=99824b13c26ffe219230ad0171e93d850003d16aa9f780c2f06c41bc46845cb3
PREDECESSOR_BATCH=chatgpt-zh-CN-031
ATTEMPT=1
TASK_KIND=translation-unit-revision
ORIGINAL_GENERATION_BUNDLE_SHA256=6972502acc0ab83f6a9382b9bb6330cca212d74db56e6af29e204e895e37d542
ORIGINAL_ARCHIVE=data/generation-archives/zh-CN/6972502acc0ab83f6a9382b9bb6330cca212d74db56e6af29e204e895e37d542.zip
ORIGINAL_ARCHIVE_AVAILABLE=no
CURRENT_REVISION_BATCH=chatgpt-zh-CN-064
LANGUAGE_AUTHORITY=zh-CN-learn-docs-qc023-language-audit-current-validation
```

当前 json.md 的既有 diagnosis 是：historical P245 为 JSON literal，current P245 为 `MarshalerTo`，current `Marshal` identity 并非 historical P245；`Marshal` source occurrence=10，machine-bindable target occurrence=0，deterministic atom mapping impossible。token ordinal 不能衔接这些不同身份。本文不建立 `Marshal → 中文 substring` mapping；候选能力只考虑沿用既有 `APPROVE_RECONCILIATION` 的 **whole frozen prose interval** approval。

## Historical provenance source：有优先级的两条路径

```text
NORMAL_ARCHIVE > EXPLICIT_RECORDED_LEGACY_PROVENANCE
```

它们不是 caller 可自由选择的同权 alternative。algorithm 必须先按 receipt 中的 locale/original SHA 检查 canonical archive path；调用者不能指定另一个 archive 或强制使用较弱路径。

1. **canonical exact archive 存在：**验证 regular/no-symlink、exact SHA、strict ZIP inventory、import receipt 和所需历史 lineage，走 `NORMAL_ARCHIVE`。wrong SHA、corrupt ZIP、identity mismatch、无法读取或 archive 状态未知一律 reject，不能按“archive 缺失”降级。此处验证 historical archive-backed provenance，不声称旧 Bundle 今天 CURRENT，也不 replay/import 原 invocation。
2. **canonical archive 确认缺失：**只有 caller 显式提供 recorded legacy authority 的 repository-relative path + exact SHA，才允许进入 `EXPLICIT_RECORDED_LEGACY_PROVENANCE`。缺 ref/SHA 则 blocked。不扫描 latest legacy receipt，不按 Unit 自动发现任意 receipt，也不以 locale/batch 前缀或近似 path 选择 authority。
3. **legacy verifier 为强制入口：**调用正式 `VerifyLegacyGenerationProvenance(root, completeCatalog, exactRef, exactRequestedOptions)`，不能只读 receipt JSON 后信任其自称 identity/capability 或复用上次 PASS。`exactRequestedOptions` 从已验证的本轮 batch/reopen/Snapshot/reconciliation graph 独立构造，字段见下表；不从 caller 的 target mapping、路径相似性或 receipt 自身反向决定 requested scope。
4. **archive 后来出现：**即使旧 selection/Bundle 已冻结 legacy ref，也必须重新检查原 archive。保留该显式 ref 作 comparison/audit dependency，fresh 调用 verifier 比对全部 covered facts；conflict → fail closed，match → 强制以 `NORMAL_ARCHIVE` 为 provenance source，实际 API 返回 `normal-archive`。不得忽略 archive、删除 legacy receipt 或覆盖任何 archive。新请求不能借 legacy ref 把已存在的 archive 当成可选证据。
5. API 返回 `legacy-recorded` 时，consumer 只有在 canonical archive 确认仍缺失且所有 proof PASS 后，才能标记 `EXPLICIT_RECORDED_LEGACY_PROVENANCE`；API 返回 `normal-archive` 时必须标记 `NORMAL_ARCHIVE`，并完成 normal archive-backed proof。返回值/observed archive state 矛盾或 publication 前发生相关 drift 时停止重建，不冻结 stale observation。

当 archive 已存在且没有显式 legacy ref 时，normal path 无需发现或依赖任意 legacy receipt。当本次输入已显式绑定 legacy ref 时，该 ref 必须继续被验证，不能通过丢弃它掩盖 archive 与 legacy covered facts 的冲突。reference 的保留不意味着较弱 substitute 仍是 provenance 的必需来源。

| exact requested options 字段 | 独立绑定来源 |
| --- | --- |
| `Locale` / `ContentPackage` | caller 显式选择、machine 验证的完整 package；不从单 Unit 推断 package |
| `BatchID` / `UnitID` / `CurrentUnitID` | Snapshot 所选 predecessor batch、已记录 reconciliation 的 historical/current Unit mapping |
| `Attempt` / `TaskKind` | selected final validation/attempt、immutable batch task 与可信 import receipt；彼此 exact match |
| `ImportReceipt.Path` / `.SHA256` | exact selected invocation 的可信 immutable import receipt；不补写旧字段 |
| `OriginalBundleSHA256` | 同一 import receipt 记录的原 SHA；canonical archive path 由此派生 |
| `PreviousSnapshotID` / `SnapshotSHA256` | 当前 revision batch/reopen 指定的 predecessor Snapshot ID 与 exact manifest SHA |

verifier 必须 fresh 验证 receipt identity/path/SHA、requested options 全集、schema/capability/consumer contract、provider/model、batch/task/attempt、可信 import、Snapshot、完整 surviving dependencies 与 canonical original archive state。其独立 typed result/dependency refs 必须进入 derived proof，不能把旧 invocation、machine reconciliation materialized raw 与真实 imported response 混为同一个 output。

### `historical-response/v1` allowlist 与 explicit unknown facts

legacy provenance 只允许消费 receipt 中固定的 historical facts：

- `import-provider-model-task-batch-attempt`；
- `exact-imported-raw`；
- `formal-source-input`；
- `accepted-candidate-lineage`；
- `recorded-reconciliation`。

这些 facts 由 surviving immutable artifacts 重新证明，不表示已读取原 ZIP 成员。算法必须逐项声明所需历史事实及其验证来源；需求超出 allowlist 或 missing proof 即 reject，不扩展 receipt/旧 import 的语义。

以下六项仍为 explicit unknown，不能借 legacy receipt 推导：

| Unknown fact | 禁止的推导 |
| --- | --- |
| `original-generation-manifest-bytes` | 根据 GenerationManifest hash/size 逆推或创建 manifest |
| `original-member-inventory` | 由 aggregate InputIdentity 或 surviving refs 生成完整原 ZIP Files inventory |
| `original-authority-context-bytes` | 用今天 docs/context 冒充旧 member |
| `original-glossary-bytes` | 将独立 review-time/archived glossary 或 current glossary 等同于原 invocation glossary |
| `original-input-member-equality` | 从仓库存留 formal input 的 hash 推断已逐字节读过原 ZIP input member |
| `original-editable-predecessor-baseline` | 从 raw/candidate 反推旧 editable template 或缺失 codec |

若 approved-interval algorithm 需要其中任一事实，而当前 provenance 无法 machine-prove，则必须 fail closed。既有 machine receipt/semantic review archive 也不能改称原 Generation Bundle；不得合成 equivalent historical ZIP。source-owned saved mapping、surviving formal input 和 exact historical restore 是独立可验证材料，不能据此填补上述 unknowns。

## 独立的 approved-interval representation authority

保留四层职责：

| 层 | 权限与边界 |
| --- | --- |
| Language authority | 原 immutable `authority_source=current-validation` reopen，原 A/finalization/Snapshot、完整 ordered affected scope 和 exact machine finding 不变 |
| Historical import provenance | 原 import receipt，只记录真实 provider/model/batch/task/attempt/input/output identities，不重写历史 |
| Historical provenance consumption | normal archive-backed proof，或明确选择且通过正式 verifier 的 recorded legacy authority；只解决历史 response provenance |
| Approved-interval representation transport authority | 本文已审定的独立 Bundle-only capability，决定 whole approved interval 如何作为 immutable opaque parent；legacy PASS 不自动授权它 |

```text
legacy provenance authority
→ approved-interval representation transport authority
```

existing semantic receipt 仍只批准 whole-interval correspondence，不提供新增 protected atom 的 target-site mapping。新 transport 不产生语言 finding/rating/candidate、model invocation 或正式 receipt，不替换原 reopen，也不借 QC-only bridge 改写本轮 authority。

若 current Unit、source SHA、canonical formal input 与所选 Snapshot/M current side 相同，就没有新的 nonidentity replacement delta。旧 `OLD_ID → CURRENT_ID` replacement 已由 reconciliation 完成，不能重放它、注册 machine-only projection 为 canonical Page 或伪造 successor ID。其它真实 source/input/Unit topology delta 仍走既有 replacement lifecycle。

### Opaque parent 与 protected children

仅能把**整个已批准的 frozen prose interval**作为不可编辑 opaque parent，保留原 reviewed target interval 的全部 bytes/hash。不得拆成 target atoms、建立 `Marshal → 中文 substring`、使用 target-language inference、string uniqueness、fuzzy/nearest、token ordinal，或重新 protect candidate 猜 mapping。完整区间的 source coordinates/old anchors 只界定 parent，不证明任何 target child correspondence。

必须冻结 source/current protected children 的 **identity、role、完整 inventory、source coordinates** 与 representation identities。不得删除 children、取消 protection、改 classifier、重排 source sites 或把 children 重新绑定到猜测的目标位置。父区间内部 source inventory 的存留不等于 leaf mapping 已证明；若任一 gate 必须获得 leaf mapping，则本模式拒绝。

采用独立且严格 exact/unique 的 projection token namespace，与 normal `P` namespace 明确区分。新 token 只代表整个 frozen parent（及 decoder 明确声明的其它 opaque boundaries），不能赋予普通 `P` token 新含义。proof 必须声明 namespace/version、完整 response token inventory、每个 parent 的 exact source/current children、frozen target bytes/hash、decoder dependencies 与 inverse restore contract。parent 内的 children 由其 source inventory proof 保持，不能声称它们在 target 各有一个可见 `P` occurrence；区间外 normal tokens 继续维持原身份/绑定，所有 output tokens exact/unique，无 missing/extra/duplicate/cross-slot 例外。

### Finding nonoverlap：typed machine proof

必须在同一 current source/candidate/representation、validator/locale policy context 下 machine-reproduce 原 current-validation finding，exact 绑定原 reopen 的 machine finding identity；不得解析 finding 自然语言、关键词或 error 字符串来决定 editable scope。

令 `E` 为该 finding 的**全部** editable finding regions，`O` 为 opaque approved interval，要求：

```text
E ∩ O = empty
```

proof 必须列出 typed representation coordinates、coordinate space、offset 单位（byte/rune 等）、Unit/representation/hash bindings，以及 rendered/source/response 坐标间的确定性转换证据。比较应在同一已验证的 coordinate space 中进行；不能直接把 rendered HTML offsets 当 source 或 raw offsets。完整 required edit footprint（含修复所需局部上下文）都必须位于 O 之外。

任何 overlap、finding 需要修改 parent 内任一字节、坐标转换不唯一/未知、只剩自然语言描述而无可重现 typed regions、或无法穷尽所需 editable regions，均 reject。不能为采用 opaque parent 而冻结真正需要修改的位置，也不能通过缩小 finding scope 使交集变空。

### 其余必要 gate

- 明确 locale、完整 native/registered package/source authority、pending batch manifest path/SHA、原 reopen path/SHA/identity、完整 ordered language scope、Snapshot/finalization/A 和 selected Unit 的 exact finding。
- 所选 predecessor manifest/formal input/result、passed final validation/attempt、真实 imported raw、reviewed candidate，以及 applicable machine reconciliation/materialized raw/candidate/validation；各自 identity 不混用。
- 显式 semantic receipt、原 response、review Bundle/inventory/identity/SHA、**整段** source coordinates/old anchors/frozen target interval/hash。只沿当前 Snapshot 的显式 reconciliation/semantic refs 读取；不扫描 latest decision，不借另一个 Unit/interval 的 approval。
- 逐 Unit 按前述 provenance source contract 验证历史事实；legacy verification 不能代替 semantic approval、typed nonoverlap 或 decoder proof。
- current source、Unit/input/representation、current 完整 glossary/full independent Review、compatibility、locale policy、adapter/implementation identities 每次重验；不能以未知旧 glossary 或历史 A 豁免 current gates。
- 旧 raw 经有证据的 historical decoder byte-exact restore reviewed candidate；新 projection baseline 经其独立 decoder 也必须 byte-exact restore 同一 candidate（仅沿用既有 EOF framing）。不能用 candidate 反推缺失 decoder。
- 最终 response restore 后必须执行 **active canonical validation**，保留链接、代码、API/source identities、markup、glossary、结构和其它当前 machine gates，再走独立 re-QC/A-only。不允许使用 historical semantic projection 的 PASS 替代 active validation；validator 无法接受或任一 protection gate 未证即停止，不能放宽 gate 迁就本模式。

## Derived transport command contract v2（已实现，真实使用待 gate）

继续使用 `generation-bundle export` / `generation-bundle current-check`，不新增 publisher、receipt registry、record/apply 或第二套 lifecycle。selector、Bundle-only metadata 与 current-check reconstruction 已实现。本节不是本轮真实执行授权，也不构成 `COMMAND_CONTRACT_VERIFIED=yes`。

### 显式 selection input

新增 **export-only** flag：`--approved-interval-selection <repository-relative JSON>`，只出现一次，空路径或重复 flag 拒绝。不新增 `--allow-missing-archive`、force、target atom/occurrence、free-form mapping 或 legacy source override flag。

export 的既有 `--locale`、`--batch-id`、一次显式 `--content-package`、新 `--output` 仍须提供；current-check 仍以 `--bundle`、同一显式完整 `--content-package` 为入口。未提供 selection 时只走既有 ordinary transport 合同，即使仓库里存在 legacy receipt 也不自动启用本模式；缺 editable capability 仍拒绝。selection/proof 的未知 schema、mode 或 version 均 fail closed，不回退普通 decoder。

selection schema 为 `go-learning/approved-interval-selection/v1`，只含 `schema`、exact locale、batch ID、content package 和按 batch stable order 排列的 `units`。逐 Unit entry 仅有 `current_unit_id`、`mode=approved-interval-opaque`、可选 `legacy_provenance={path,sha256}`。未知字段、空/重复/乱序/越界 Unit、wrong locale/batch/package 和不适用 mode 一律拒绝。它只是显式消费请求，不是正式 authority/receipt，不提供语言批准、target coordinates 或 mapping；本轮不创建真实 selection。

selected Units 必须是原 reopen/batch exact ordered scope 中需要该能力的精确子集。semantic authority 从该 Unit 的 Snapshot reconciliation 显式 graph 取得；missing archive 的每个 Unit 都要有自己的 exact legacy ref。archive 存在时不能借这个 ref 选择弱路径；后来重现 archive 时，已冻结的 ref 仅保留作 fresh covered-facts comparison/audit。其它 Unit 按原合同处理，一个 Unit 未证明就拒绝整个 Bundle。

`current-check` **不接受新的 selection 文件或 override**：它从待检 ZIP 已冻结的 selection bytes、pinned refs 和 metadata 取出同一显式请求，在 repo 中重新验证 authority 并 canonical reconstruct；不能扫描 latest/同 Unit receipts 替换 frozen inputs。selection exact bytes 固定放在 `formal/approved-interval-selection.json`，作为有 inventory/hash 的 frozen request，entry 不声明 mutable `repository_path`。原 external selection 文件不作为长期 pointer；后续检查不依赖该文件继续存在。冻结请求本身不提供 authority，所有 refs/proof 必须 freshly machine-prove。

| 字段 | Contract v2 设计边界 |
| --- | --- |
| COMMAND | 既有 `generation-bundle export` / `generation-bundle current-check`；export 新增上述单个 selection flag，current-check 增加 frozen selection/provenance/proof 的独立重建语义；实现完成不代表真实使用授权 |
| OPERATION | 为显式 selected Units 派生/重建 `approved-interval-opaque` representation transport，按强证据优先选择 historical provenance，不推导 atom correspondence |
| INPUT_KIND | immutable batch/reopen/完整 package、Snapshot/A/selected final attempt/raw/candidate/validation、recorded machine reconciliation、semantic receipt/review Bundle/decision、current authority，以及 explicit selection 与必要的 recorded legacy reference |
| INPUT_PATH_CONTRACT | selection/evidence/authority refs 均 repository-relative、regular/no-symlink/no-traversal；`docs/evidence/selection.json` 合法，`/home/user/repo/docs/evidence/selection.json`、`../selection.json`、反斜杠路径非法；现有 `--output`（新 ZIP）/`--bundle`（已有 regular ZIP）路径语义不变 |
| PROVENANCE_SOURCE | `NORMAL_ARCHIVE` 优先；仅 canonical archive 确认缺失时允许 `EXPLICIT_RECORDED_LEGACY_PROVENANCE`；API spelling 为 `normal-archive` / `legacy-recorded`，返回 stronger source 必须采用；状态未知/冲突均 reject |
| LEGACY_PROVENANCE_INPUT | 每个 selected Unit 在 selection 中显式给 exact recorded path/SHA；调用正式 `VerifyLegacyGenerationProvenance` 并传 exact requested options；保留 immutable dependency refs、allowlist/unknown facts；不按 Unit/latest 自动发现 |
| SCOPE | 保留原 reopen/batch 的完整 ordered language scope；selection 只决定该 exact subset 的 transport mode，不缩小/扩大 finding、Unit 或 editable region authority；整批 readiness 必须全部 PASS |
| PROOF_SCHEMA | Bundle-only `go-learning/approved-interval-transport/v1` variant，位于 `formal/approved-interval-transport/<current-unit-id>.json`；冻结 mode/version、selection bytes/hash、provenance source/refs/result/dependencies、exact requested options、原 semantic/language authorities、whole parent/children/token inventory、typed E/O nonoverlap、old/new decoder/restore identity 和 current context；不修改旧 schema 含义、不冒充 receipt |
| AUTHORITY_SEPARATION | 原 current-validation reopen 独占 language revision authority；legacy 只供 `historical-response/v1` facts；semantic receipt 仍是 whole-interval approval；新 derived variant 单独承担 immutable parent 的 transport authority；四者不能互相替代 |
| MUTATION_EXPECTATION | export 只发布新 derived transport ZIP；current-check readonly；不创建 receipt，不写 batch manifest/input、reopen、Snapshot、candidate、locale metadata、旧 ZIP 或 legacy authority |
| IDEMPOTENCE | 同一 explicit request + 全部相同 verified inputs/observed provenance state 产生相同 canonical bytes；既有 output 拒绝覆盖。archive 后来出现可能改变 canonical graph，使旧 Bundle stale，不能 patch 旧 ZIP |
| CURRENTNESS_CHECK | 独立重建全 graph；fresh normal/legacy provenance、covered facts/完整 dependencies、source/Unit/input/package/representation、glossary/Review/policy/implementation、semantic approval、typed finding/nonoverlap、exact restores/active validator；拒绝复用上次 PASS |
| PRIOR_OR_CAS | exact batch/reopen/Snapshot/import/semantic/legacy refs 与所有 dependency SHA/size、original archive state、source/current identities；publication/返回前 fresh dependency guards 与 archive priority 重查；ZIP 原子 write-new/no-overwrite |
| SUCCESS_STOP | 本轮停止于 implementation/readonly proof；独立 first-real-use gate 后，才可在后续获准 export 后独立 current-check，仍停止于真实 Generation 前 |
| FAILURE_STOP | 任一 unknown/overlap/identity drift/missing authority/unsupported graph/active validation failure 均停止整批，无 partial Bundle、隐式 legacy fallback、自动 successor 或 authority 扩张；保留全部 evidence |
| RECOVERY_REPLAY | archive 缺失且有显式 recorded ref 时重新调用正式 verifier；无 ref 或需 unknown facts 则 blocked。archive 重现先比 covered facts，冲突停止，匹配走 normal；保留旧 transport/receipts，必要时按 canonical graph 刷新新 ZIP；mutation=unknown 先 diagnose，禁止盲重放 |

## 同批 064 与后续 transport graph

本合同将 **legacy reference 和 approved-interval proof 全部放在 derived Generation Bundle transport graph**，不向 064 manifest 增加引用、mode 或 authority 字段，也不回写其 formal input/reopen/Snapshot。实现沿用以下闭包：

```text
explicit selection bytes + recorded legacy reference（若需）
→ normal-priority provenance verification + independent approved-interval proof
→ derived Bundle Files / aggregate identity / response codec metadata
→ independent current-check canonical reconstruction
→ import 验证同一 graph，并封存实际新 invocation Bundle
→ process / retry 从真实 imported Bundle 重建同一 decoder/provenance graph
→ validation / Snapshot 绑定同一 selected response 与不可变依赖
```

selection bytes 冻结在 Bundle input；proof/refs、token namespace 与 decoder dependencies 全部进入 Files inventory/hash、aggregate identity 和 current-check。普通 import/process/retry/Snapshot 不获得通用 missing archive bypass；它们只能消费实际新 imported Bundle 中显式 mode 且完整通过验证的闭包。后续 retry 必须保留原 revision semantics/selection/approved interval 与 provenance refs，不换 authority、不丢 lineage。历史 raw 仍独立冻结；machine-only materialized raw 不伪装成旧 Generation output。

因此这里可作**架构可能性**判断：

```text
SAME_BATCH_064_REFRESH_POSSIBLE=yes
SUCCESSOR_REVISION_BATCH_REQUIRED=no
```

`no` 仅指本合同的 transport dependency 设计本身不要求 successor，不是已获准真实同批 export。实际执行前仍须完整32/32 representation readiness proof、first-real-use gate 和正式 export/current-check；readonly readiness 不能授权真实 export。若实现发现必须回写 batch manifest，则违反本合同，必须停止并重新审议；旧064 immutable，只能在另行批准的 successor contract 下继续，不能原地修改，也不猜 successor ID。

## v2 migration 的原始停止点

legacy authority 已真实 record，仍需由每次后续 consumer fresh 验证；approved-interval contract v2 已审定，实现后仍须独立 first-real-use gate。执行顺序：

```text
legacy provenance authority（已 record）
→ approved-interval representation transport authority
→ all 064 representation readiness（32/32）
→ new/modified CLI first-real-use gate
→ Generation Bundle export/current-check
→ Generation
```

该 migration 的原始下一阶段为 **`current-validation-approved-interval-first-real-use-gate`**。任何新 selector、metadata schema 或 current-check/reader contract 变化都使旧 verification 失效；第一次真实使用前，必须重新按 current docs → implementation → direct CLI parsing/contract tests 顺序核对。现有 legacy CLI 的 first-real-use 不替代 approved-interval consumer 的 gate。这是原先 selection、Generation Bundle export 和 Generation 之前的边界，不要求重做后来已经完成的 invocation；installed stale recovery 另见下节设计链接。

### 实现与恢复边界

普通 predecessor capability 保留在 `current_validation_predecessor.go`，不依赖 approved types；`current_validation_approved_interval.go` 只承担显式 selected Units 的只读 overlay。projection namespace 为 `GTI18N_AI1/v1`，whole parent 与区间外 frozen boundaries 只由新 namespace 表示，普通 P tokens 不获得新含义。Markdown parser-owned block 与 canonical rendered HTML 必须 byte-exact，再把 fresh audit 的 typed rune/HTML byte coordinates 保守转换为完整 candidate block bytes；无法完整转换或任意 overlap 即拒绝。

export 冻结 selection exact bytes，current-check 的内部 reconstruction 只消费 ZIP member；external 文件随后移除不影响检查。import 仍封存真实新 Bundle 与真实 provider/model。process/retry/Snapshot 从该 batch 显式 initial import receipt 的 exact archive 取 selection，重新构造 proof 并与冻结 proof exact 比较；unknown/missing/extra proof 不回退 ordinary decoder。retry 不提供新 selection 时自动继承已导入的同一请求；显式提供不同 bytes 则拒绝。archive 状态、dependency bytes 或 implementation 改变后旧 Bundle stale，不 patch 旧 ZIP、receipt 或 batch。

### 已安装 invocation 的 proof stale：现有 recovery 边界

本节澄清现有拒绝边界，不新增 recovery authority 或执行授权。已 import/process 的
invocation 后来发生 decoder/proof dependency drift，历史 validation PASS 仍是历史事实，
但不证明该 proof 今天 CURRENT。`compareInstalledApprovedProofs` 必须继续 exact 比较；
即使 observed drift 仅为 implementation SHA，也不能据此自行声明 semantic equivalence。

| 既有机制 | 不能用于本情形的原因 |
| --- | --- |
| same-batch Bundle re-export | [Task Spec](TRANSLATION_TASK_SPEC.md#deterministic-generation-bundle-transport) 只允许未安装正式 raw/import 的同批刷新；已有 raw 时拒绝 |
| exact retry | 仅处理 restore/validation failure，且必须保留 initial archive 的 selection/proof；不是更新 proof 的新 invocation |
| revalidate | 重验同一 attempt 前仍须通过原 decoder/provenance preflight；不能刷新 installed approved proof |
| successor revision batch | 仍须正式 revision authority 和 exact Unit scope；不能仅凭 stale error 复用已消费 reopen 或导出其任意子集 |
| representation successor | 需要真实 representation delta 与其正式 authority；implementation drift 不授权伪造 Unit/source identity 或重放已完成 reconciliation |
| current-validation reopen successor | 当前 builder 要求完整 affected set、frozen A baseline 与 current latest selection 一致；改 reopen ID 不解除 conflicting current candidate/successor gate |
| machine-only candidate reuse | 现有 representation reconciliation 不是任意已安装、尚未重新 QC 的 model response 的 proof-renewal authority；不能制造新 model provenance |
| Generation invocation replay / `generation-bundle recover` | recover 只恢复尚未 import 且 Bundle CURRENT 的中断 staging；import、archive、raw 或 result 已安装后不能重放 |

Snapshot 当前按 selected batch 重建完整 process output context，并比较 initial archive
中的完整 approved-proof inventory。因此“只创建一个 successor、保留其余旧 batch Units”
也不是已有的免责路径：只要旧 batch 仍有 selected Unit，完整 context 可能仍消费其中
未被选中的 stale proof。没有新的、明确限定 authority 与 proof 消费范围的合同，不能
删 proof、跳过未选 Unit 的 comparison，或声称其余 Units 已恢复 CURRENT。

当前 production 尚无 installed-approved-proof-stale recovery capability。已完成的
[独立 recovery 合同设计 v1](INSTALLED_DERIVED_TRANSPORT_STALE_RECOVERY_DESIGN.md)
选择原 batch完整decoder-context closure、全范围successor revision、新Language Generation，
不carry该batch内未直接drift的candidate；不新建reopen或canonical representation successor。
该设计尚未实现/first-real-use，不改变上表当前拒绝行为。真实恢复必须先按该设计验证
已消费 language authority的显式继承、完整context closure、latest selection和immutable
invocation provenance；不能直接
提供可执行 recovery command、猜 batch ID 或改用未创建的 Snapshot。旧 raw 不得作为
新 invocation output；若希望确定性复用，必须另有正式 equivalence/materialization 合同，
并保留原 provider/model/Bundle/raw 身份，不能把机器重验记成新的 Language Generation。
