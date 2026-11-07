# Installed derived-transport proof stale recovery 合同 v1

状态：**design complete；已实现，真实操作仍须独立 first-real-use gate。** 本文定义跨 locale、跨 registered
content package 的窄恢复能力。下述 schema、CLI 和 reader 分支是实现所遵守的合同；
synthetic verification 不授权真实 preflight/record/export 或 recovery。下一阶段另做
[first-real-use gate](WORKFLOW_HANDOFF.md#formal-command-contract-v3)。

## 状态、范围与确定结论

historical invocation validity 与 current transport validity 分开保存。可信 import
receipt、exact archive、真实 provider/model/raw、当时 restore/validation PASS 是不可变
历史事实；current reconstructed proof 与 frozen proof 不同后，该 invocation 不因历史
PASS 而获得新 Snapshot eligibility。恢复不撤销历史事实，也不把旧 proof 标成 CURRENT。

v1 接受 `approved-interval-opaque` 的 installed **current-validation revision**：它必须
来自完整 registered package，原 batch 的所有 Unit 已最终 restore/validation PASS，且
canonical source、formal Unit/input、package membership、原 language/semantic authority、
glossary/full Review、locale policy 均仍合法。普通 initial、其它 finding family、未知
derived mode、不完整 provenance、语言/source/glossary drift 不借用这一能力；unsupported
graph 明确 reject。此限制按 capability，不按 locale、文件名或 package ID。

| 决定 | v1 合同 |
| --- | --- |
| Recovery authority | 新的 `installed-derived-transport-recovery/v1` immutable successor authority，显式继承原 language authority |
| Recovery scope | **per-original-batch 的完整 decoder-context dependency closure**；不按单个报错 Unit裁剪 |
| 原 batch 内未直接 drift 的 candidate | **不 carry**；新 invocation 必须覆盖整个 closure |
| batch 外 candidate | 保持原 shared selector/currentness/QC carry 合同，不获得额外豁免 |
| Successor revision batch | **required**；新 batch、新 invocation，旧 batch 原地不变 |
| Reopen successor | **not required**；新 recovery authority 授权再执行，原 reopen仅提供不可变 finding/范围，不隐式重用其已消费 export 权限 |
| Canonical representation successor | **not required**；仅 derived transport/decoder/proof epoch 改变，不能伪造 source、Unit ID 或 replacement |
| Language Generation | **required**；v1 不提供 machine-only replacement/materialization 或跨 invocation output reuse |
| 旧 raw 作为新 invocation output | **forbidden**；字节能 restore、active validation PASS 均不能生成新的 model provenance |

这个 scope 不是因为 ZIP 大、batch size 或执行方便。当前
[`prepareSnapshotRevisionOutputs`](../internal/i18n/candidate_snapshot_outputs.go)
按 batch 调用 [`prepareProcessOutputContext`](../internal/i18n/retranslation_process_context.go)，
后者经 `approvedResponseInputs` 重建整个 manifest，并用
[`compareInstalledApprovedProofs`](../internal/i18n/approved_interval_transport.go)
核对完整 proof inventory。每个 selected response 因而连到同一个完整 context，里面包括
其它 Unit 的 approved proof。v1 保留这个 authority/decoder 原子边界，不新增删 proof、
裁剪旧 manifest 或绕开 unselected-proof gate 的 reader。ZIP aggregate hash 本身并不
证明语言不可分；**现有共享 context 的消费合同**才决定本版 closure。

令 `B` 为显式指定的原 installed batch，`U(B)` 为其 immutable manifest 的 ordered Unit
集；`D` 是按下文typed predicate证明construction/dependency drift的approved Units；`R` 是沿上述
共享 context 反向依赖得到的 recovery closure。当前 v1 中，只要 `D` 非空，
`R = U(B)`。`D` 与 `R` 必须分别报告，不能把未直接 drift 的 Unit谎报成 proof drift。
不跨原 invocation 扩到 package；若 closure 还依赖不能独立验证的其它 invocation，则
blocked，不擅自 union 更多 batch。整个 `R` 必须同 kind、最多60且 exact-match 原 batch
及其原 reopen 的完整 language scope；不能按 initial 30组重新切分，也不能手选子集。

三个scope分别固定为`DIRECT_DRIFT_SCOPE=D`、`RECOVERY_AUTHORIZATION_SCOPE=R`、
`SUCCESSOR_GENERATION_SCOPE=R`。其中`R-D`的“不复用”只指不能在successor Snapshot中
直接carry旧invocation candidate；其历史candidate、当时validation PASS及raw/provenance
没有失效或被删除。新的Generation仍按每个Unit原有finding边界工作；R增至完整context
不增加任何Unit的可编辑范围，也不把32 Page变成自由重译任务。

## 两层 authority 与准入证明

原 reopen 不变，不创建 `current-validation-reopen` successor。其 original A/finalization、
Snapshot、逐 Unit finding/category/授权范围仍是唯一 language revision authority。新
recovery authority 仅授予在 current transport 下对 `R` **重新执行同一 semantic task**
的一个 successor invocation；它不是新 finding、A、QC、package completion 或 missing
archive bypass。普通 reopen/export reader 的 consumed/conflicting-candidate gate不变。

`REOPEN_REUSE_CURRENTNESS`要求原reopen的identity、Snapshot/finalization/A、semantic/
finding refs、complete scope及当前typed finding predicate均合法；fresh原baseline assessment
必须重现相同language授权，不允许原reopen自身stale。已消费的export权、原Snapshot不再
是latest candidate这两个执行状态由新recovery authority显式处理，不伪称原普通reopen
builder重新CURRENT，也不忽略其它stale原因。版本化producer必须区分language-authority
dependencies与derived-construction dependencies；原language authority的implementation/
dependency约束发生无法独立证明合法的变化时拒绝，不能一律归为transport drift。

`REPRESENTATION_SUCCESSOR_REQUIRED=no`以canonical protected representation identity、
source/input/Unit及其formal保护语义不变为条件。若这些identity真正变化，退出本合同，
进入既有representation successor/replacement lifecycle；不能把它作为decoder epoch更新。

recovery preflight必须在同一 invocation 内完成下列证明，并在 record前全部重做：

1. 显式 `locale/package/B`，加载 current complete registered descriptor及其 exact authority、
   ordered Unit/source identities。按原 import receipt 的 SHA读取 canonical archive；
   regular/no-symlink、exact bytes、strict bounded ZIP、完整 manifest/inventory/hash/
   provider/model/input/output identity均通过。保留 Store 可读、Deflate可读和现有
   `8192 members / 134217728 logical bytes` 上限，不能以历史可读声称旧 Bundle CURRENT。
2. 原 batch manifest、最终 result、逐 Unit validation、raw、candidate及连续 retry histories
   全部精确匹配。每个 `R` Unit的最新 processed selection仍来自 `B`，最终 status=passed；
   不允许以“前一个 PASS”替代 latest failure。所有最终 attempt及相应 initial/retry import
   receipt/archive必须存在、合法。原 proof只通过其 strict frozen codec/闭合 inverse
   contract验证历史 restore，不执行 archive 中的任意旧 Go代码。不能从 candidate反推
   codec，不能以本恢复能力补 legacy unknown facts。
3. 读取原 Snapshot、finalization/A、reopen及 semantic/legacy authority的 exact refs。
   原 reopen按 immutable historical reader验证；不调用要求“原 Snapshot等于当前 latest”
   的 builder再伪造一次 reopen。另做新的 typed recovery assessment，分别绑定原 A
   baseline与已安装 `B` selection，fresh重现原 baseline的完整 finding范围；不得新增、
   遗失或换 finding。恢复准入仅在 `R` exact-match 原已授权集合时成立。
4. 当前 source、formal input/Unit/package/representation identity、完整 glossary/full Review、
   policy与原 language/semantic scope保持。每个旧最终 candidate执行 current active
   canonical validation并 PASS，结果只是恢复准入 evidence，不是新 Snapshot eligibility。
   新语言/结构/保护缺陷、未知历史事实或其它 stale原因为独立 blocker。
5. fresh构造整个 `B` 的 current decoder/proof graph，不把 error message、文件名、mtime
   或手选 ID当分类依据。对 strict typed proof/manifest逐字段比较，产出 `D`、dependency
   role/path/old-current SHA diff和闭合 read graph。v1仅接受 registered approved decoder/
   projection implementation及其声明的 transport dependencies所致 drift；若同时有 source、
   language authority、glossary/policy或未分类 dependency drift，则 blocked。
   exact member mismatch只是currentness失败信号，不足以进入D。必须按下节construction
   predicate证明真实producer implementation/dependency或typed construction差异；纯
   JSON formatting、key order或无语义影响的proof encoding差异不属于可授权recovery。
   旧v1 plain refs的role只按versioned producer声明和typed authority字段还原，不能凭path
   名字或error子串猜；原proof的ref顺序/bytes不重写。无法分类则BLOCKED，不是CURRENT。
6. 新 proof必须独立通过完整 provenance priority、semantic approval、whole opaque parent、
   children identity/role/inventory/coordinates、typed finding nonoverlap、exact inverse
   restore与当前 validator gate。旧 proof不能覆盖 fresh结果。旧 proof仍 current时
   `D=empty`且原exact comparison通过，才报告CURRENT并禁止record；fresh proof不能构造时是BLOCKED，不能
   把无法验证等同于已证明 stale。
7. 计算完整 `R`，冻结 current implementation/dependency epoch、full package selection和
   filesystem latest-state/CAS observations。任何 Unit已有不同 current successor、未知
   pending、overlapping recovery或不完整 scope，整次停止，不发布 partial authority。

本版不声称支持一个原batch只剩部分latest Units的状态：若`R`中已有另一份正式successor，
拒绝覆盖那个后来的finding/语言结果，不能把它重新拉回原finding。absence/currentness
不确定也是BLOCKED，不为了使scope等于原batch而制造新的affected分类。

诊断重建必须复用现有严格builder的readonly construction阶段，取完整manifest和archive
冻结selection，返回typed old/current proof graph；不能调用要求“尚无raw”的pending exporter/
current-check后把其installed-state错误当成transport drift。可以将共享construction与原
strict comparison明确分层，但process/retry/Snapshot仍调用原exact comparison且reject。
诊断所得current projection不能直接交旧invocation的runtime decoder使用，不能借此让
旧raw在新的proof下进入Snapshot；此处唯一后续权限来自record的新full-R recovery authority。

## Immutable authority 与身份

新增 schema：`go-learning/installed-derived-transport-recovery/v1`。仅支持
`capability=renew-approved-derived-transport-invocation`，strict JSON，未知字段/version/mode
拒绝。authority必须包含下表字段；所有 external refs包含 repository-relative path、SHA256、
size，regular/no-symlink/no traversal，不使用 latest receipt discovery。

| 字段 | 精确定义 |
| --- | --- |
| schema / capability / locale | 固定 schema/capability，safe locale |
| content_package | ID、current authority SHA、scope identity、complete ordered Unit/source identities、membership/registry refs |
| predecessor_batch / invocation | exact manifest/result refs、original task kind、initial Bundle SHA/input identity/import receipt、真实 provider/model；逐 Unit最终 attempt/raw/candidate/validation和相关 retry import/archive refs |
| original_language_authority | 原 reopen path/SHA/identity、原 complete scope、逐 Unit finding、原 Snapshot/finalization/A及原 semantic authority refs；不新建 finding |
| direct_drift_units | stable order的 `D`、旧 proof member/path/SHA/identity、fresh reconstructed proof member/SHA/identity、typed drift分类与 exact dependency diff |
| recovery_units | 完整 `R`、context root/edges、从 `D` 到 `R` 的机器 closure证明；另列 `R - D`，不伪称它们直接 drift |
| current_assessment | 两个 baseline的 typed assessment、active validation、new proof gates及完整 immutable refs；complete package审视不等于 package-wide regeneration |
| current_epoch / construction | complete read graph、source/protector/adapter/validator/decoder/policy/Review/implementation identities及 bytes/hash observations、new transport construction contract |
| allowed_action | 只允许一个 same-locale/package/provider/model 的 full-`R` successor revision export；`new_language_generation_required=true`、禁止旧 raw replay、禁止扩大 finding |
| supersedes | optional exact prior recovery ref和其 original root/无 installed successor证明；规则见下文，不扫描 latest |
| identity | 所有 typed字段与 proof member inventory 的 canonical JSON SHA256，排除 identity自身及 publication path；不含 wall clock、random ID或未选定 successor batch ID |

authority和 assessment/proof blobs整体作为 write-once目录发布：
`data/translation-transport-recoveries/<locale>/<identity>/authority.json`。
directory内 member refs为相对该目录的固定 member name + SHA/size，identity先绑定这些
相对 member refs，再派生 publication path，避免 self-hash/path循环。仓库其它引用该
authority 时使用完整 repository-relative path + exact SHA/identity。
identity沿用现有typed-struct `json.Marshal` 后SHA256的规则：identity字段置空、固定字段
顺序，无缩进/末尾LF参与hash；Units按formal stable order，去重refs按role/path/SHA排序，
arrays不得重排为不同scope。transaction owner/临时stage名不进入authority identity。

Receipt的 fresh reconstructed proof 是**原 batch请求在 current epoch下的诊断重建**，
不是未来新 invocation 的冻结 proof。另用显式 typed `construction` 绑定 invocation-locator-independent
recipe：source/Unit/package、original finding/semantic/provenance refs、mode/namespace、
whole-parent/children、typed E/O、tokens/inverse codec、glossary/policy/validators和完整
dependency read graph。未来新 batch/selection/receipt locator只作为新 invocation binding；
不得拿它与原 diagnostic proof逐字节相等。不得用任意 JSON字段剔除或模糊 normalization
来比较 recipe，typed字段必须穷尽且固定版本；原 installed proof comparison完全不变。
`construction`按原Unit的受支持decoder variant封装：ordinary完整protector/input/token
inventory、既有deterministic rebase的source-owned exact mapping、comment transport的
完整slot/binding/opaque boundaries，以及approved transport的whole-parent/typed E/O/
inverse codec。只能使用既有variant的严格构造器；不能为R-D临时生成target mapping，
不能把ordinary Unit伪称含approved proof。每个variant的完整read graph共同进入epoch。

### Construction identity 与可授权 drift predicate

`construction`固定为typed `go-learning/derived-transport-construction/v1` object，不是
对proof任意删字段后的JSON。其`construction_identity`为该固定结构identity置空后按上文
canonical typed serialization求SHA256；**只有invocation locator独立，semantic scope不独立**。
原有`scope-independent`表述不适用：package、ordered R、finding/parent/edit边界都必须绑定。

| 固定字段组 | 进入construction identity的内容 |
| --- | --- |
| schema / producer | construction schema；producer ID及contract version；approved v1使用其已识别的`go-learning/approved-interval-transport/v1` family，不以本次JSON外观猜producer |
| producer_implementation | versioned、code-owned producer declaration；确实参与response construction/decoder选择的implementation refs及role/path/SHA/size；由这些refs确定implementation identity |
| semantic_context | locale、complete package authority/scope、ordered R、source/formal-input/representation identities、原Snapshot/reopen/finding/semantic/provenance roots、glossary/full Review/policy/validator refs |
| decoder_constructions | 按Unit stable order的variant及完整typed payload：ordinary/rebase/comment/approved的严格字段，含protected-response namespace/version、tokens、bindings、parent/children/typed E/O及inverse codec |
| construction_dependencies | producer声明的完整construction read graph及dependency identities；不把共享context拆成按需subset |

以下分别绑定在authority/invocation inventory中，**不冒充construction内容**：新旧batch/
selection/recovery receipt locator、model invocation locator、proof外层JSON的缩进/key order/
物理member SHA、ZIP encoding及其它仅artifact packaging的配置。模型额外历史上下文仍进入
完整Bundle input identity；不因它不属于decoder recipe而丢掉inventory/hash/current-check。
protected-response codec自身的escaping/namespace/解释语义属于construction，不能混同于
proof envelope的JSON排版。construction本身的固定canonical hash序列化不接受caller选择。
嵌套codec/binding payload必须按已知version strict decode为完整typed值，不能直接将
`json.RawMessage`排版bytes计入construction hash。仅schema声明为集合的dependency refs
采用固定排序；ordered Units/tokens/children/slots保持其定义顺序。正文、代码、parent及
response模板的实际UTF-8 bytes/offset/hash一律不做Unicode、空白或target-language normalization。

旧proof若没有显式construction字段，versioned reader只可从**已验证的旧archive/proof及
其声明refs**派生`derived_old_construction_identity`，并标明这是本次诊断的派生值，不声称
旧record曾保存该字段。schema/codec/producer版本或角色无法唯一确定、缺必要旧dependency
bytes时BLOCKED；不得拿current refs替代旧refs、填补legacy unknown facts或创建“等价旧ZIP”。
旧v1 family到producer declaration的映射必须固定在支持该version的reader中，不能由
selection、自由plan、path前缀或错误文本提供。

缺失的 exact implementation bytes 已从本地历史副本找到，也不自动取得 reader authority。
[Historical construction bytes formalization v1](HISTORICAL_CONSTRUCTION_BYTES_FORMALIZATION_DESIGN.md)
设计独立的 supplemental byte evidence：只绑定既有正式 declaration，不 retrofit 旧 archive。
该能力尚未实现/first-real-use；当前 reader 不允许从 `/tmp` 或外部 source fallback。
未来接入时严格沿 archive → surviving exact repository bytes → exact committed supplemental
authority → BLOCKED 的顺序，声明 key 唯一定位且冲突 fail closed。它不改变本节的 typed
identity、drift predicate 或新 invocation/完整 R 要求；正式采用后也须另行 recovery preflight。

令`E(u)`为原strict frozen/current proof member exact比较失败，`I_old/I_cur`为上述
construction identities，`P_old/P_cur`为其producer implementation identities。可授权的
direct drift仅为：

```text
eligible(u) = historical_integrity_pass
           AND all_language/source/package/protection_currentness_gates_pass
           AND fresh_complete_construction_pass
           AND E(u)
           AND (I_old != I_cur OR P_old != P_cur)
           AND difference_is_fully_classified_approved_construction_dependency
D = stable_order({u | eligible(u)})
```

implementation/dependency identity变动是正式currentness变动，**不等于已经证明译文或
decoder行为发生语义变化**；报告必须分别使用`construction-dependency-drift`或
`construction-contract-drift`，不得统称“bytes不同所以semantic drift”。每个difference
都必须有versioned producer/role和新旧ref证据。仅artifact JSON/encoding依赖变化不进入D；
一个source file兼具encoding和construction职责而无法唯一归因时同样BLOCKED，不因同file
包含constructor就把全部SHA变化升级为恢复授权。不提供force-classification开关。

若exact member bytes不同，但typed construction、producer及其construction dependencies
全部相同，则报告`ENCODING_ONLY_MISMATCH`：旧installed exact gate仍失败，preflight为
BLOCKED、recovery authority=null、禁止record；留给独立canonical encoding diagnosis。
不得自动重Generation、patch ZIP/receipt，也不得通过normalized-JSON equality让旧proof
重新CURRENT。若尚不能证明“仅encoding”，则`UNCLASSIFIED_PROOF_MISMATCH`并BLOCKED。
任何未分类或其它scope mismatch都禁止从剩余eligible Units发布partial authority。

只有原exact comparison也PASS且无其它currentness问题才报告CURRENT。新successor Bundle
须在recorded epoch按同一versioned producer及typed construction deterministic重建，再绑定
其新invocation locators；current-check仍独立核对完整inventory、input identity和canonical
ZIP bytes。没有dual-canonical、JSON normalization shortcut或对旧installed comparison的豁免。
新recovery Bundle把typed construction与producer declaration固定保存为
`formal/transport-recovery/construction.json`，列入完整Files/hash/input identity；installed
reader同时strict比较这一contract member和原approved proofs。这里的`E(u)`包括其所属
已冻结construction contract的exact失败，不能只检查旧proof文件而漏掉新producer binding。
旧普通Bundle没有此member时仅可按上述versioned派生规则诊断，不能补写旧archive。

## Command contract：authority preflight / record / check

选择 `retranslation transport-recovery`，因为它授权已安装 batch的 successor execution，
不是 Generation ZIP重新打包，也不是 Quality Check finding/reopen publisher。以下是正式
grammar，**实现阶段禁止对真实现场执行**：

```text
retranslation transport-recovery
  --locale <locale> --content-package <registered-package>
  --predecessor-batch <installed-batch>
  [--supersedes <repository-relative-authority.json> --supersedes-sha256 <sha256>]
  [--record --expect-identity <preflight-identity> [--resume]]

retranslation transport-recovery --check
  --locale <locale> --content-package <registered-package>
  --authority <repository-relative-authority.json> --authority-sha256 <sha256>
```

所有 flag唯一、非空；`--record`/`--check`互斥；record必需 expect identity。
check不接受 predecessor/supersedes/expect override；preflight/record不接受 authority override。
`--resume`只允许record模式且必须存在匹配的sealed pending，不接受自由transaction path。
禁止 `--unit-id`、`--id`、`--limit`、`--batch-id`、`--reopen-id`、external selection/proof、
force/ignore-drift/allow-missing flags。package selector严格取完整 registered scope，
不接受 native Tour隐式 scope；不新增 fake registration。

| Contract字段 | 行为 |
| --- | --- |
| COMMAND | `retranslation transport-recovery` |
| OPERATION | 默认 readonly preflight；`--record`发布新 immutable recovery authority；`--check`验证指定 authority的 lineage与当前 phase eligibility |
| INPUT_KIND | complete current package、显式 installed batch及其 exact imported graph；check只消费 exact authority，不消费自由 plan/proof作为 authority |
| INPUT_PATH_CONTRACT | refs为 repository-relative；`evidence/recovery.json`形式合法，绝对路径、`../`、反斜杠、symlink非法；identifier不是 path |
| MUTATION_EXPECTATION | preflight/check=none；record=known，仅新增 authority/proofs和其唯一链索引；不创建 batch/reopen/Snapshot、不改旧 bytes |
| IDEMPOTENCE | 同一 observed inputs得同一 identity；重复record拒绝，不作为 no-op；已经CURRENT只可 readonly报告，不可制造recovery receipt |
| CURRENTNESS_CHECK | 准入1–7全部 fresh重建；record发布前第二次检查；check按显式 authority与 child phase验证，不以旧preflight PASS代替 |
| PRIOR_OR_CAS | expect identity、原 archive/import/manifest/result/attempt/raw/candidate/validation、source/package/epoch、完整 latest selections、root/supersession leaf及无 pending/重复 owner |
| SUCCESS_STOP | preflight后审阅 D/R；record后停止于 successor export前；check后停止于报告的唯一 continuation，不自动执行 |
| FAILURE_STOP | 任一缺项/unsupported/unknown/分类或closure错误、CAS变化立即停止；无partial authority、无普通decoder fallback、无自动扩大scope |
| RECOVERY_REPLAY | 保留旧 invocation和所有旧/新 receipt；恢复已记录 continuation，不重放 import、不重写 proof；pending处理见事务/中断合同 |

preflight stdout固定 JSON：`schema=go-learning/transport-recovery-plan/v1`、`status`、
`locale/content_package/predecessor_batch`、`direct_drift_units`、`recovery_units`、counts、
typed `blockers`、`planned_authority`及 identity/path，`recorded=false, mutation=none`。
`status=READY`时 exit 0并有完整 planned authority；`CURRENT`时 exit 0、D/R为空、
planned authority=null；`BLOCKED`时 nonzero、只报告已证事实，不提供可record的identity。
这里CURRENT只指installed derived context无需本恢复，不表示已安装的initial Bundle又
成为pending-export CURRENT。D/R为空表示没有本次recovery action，不抹掉其原Unit inventory。
record成功 JSON另含 exact authority ref、`status=RECORDED, recorded=true, mutation=known`。
record遇CURRENT、重复或CAS失败 nonzero，无新authority；mutation状态如实报告。
check输出 `status=READY_FOR_EXPORT|EXPORTED|IMPORTED|RESTORE_FAILED|VALIDATION_FAILED|VALIDATED|STALE|BLOCKED`、
exact child refs、`allowed_next_operation`、`recorded=false, mutation=none`；前三种及
RESTORE_FAILED/VALIDATION_FAILED/VALIDATED只表示已核对真实phase，STALE/BLOCKED nonzero。所有结果都
区分 `historical_integrity` 与 `current_action_eligible`，不能把旧 invocation标CURRENT。
old proof已恢复CURRENT或D不再等于记录值时，旧authority的action eligibility为STALE；
不能用check自动撤销、重写或再次执行旧授权。

`allowed_next_operation`取固定command-family operation：READY_FOR_EXPORT为
`retranslation-export-with-transport-recovery`；EXPORTED为
`verify-or-export-child-generation-bundle`（已有可信CURRENT ZIP/staging则沿其phase继续，
不要求重做Generation）；IMPORTED含exact task/unit/attempt，分别为
`retranslation-process`或`retranslation-retry`；两个FAILED状态在无已imported next attempt
时为`generation-bundle-export-exact-retry`，已有next receipt/raw则先归类IMPORTED；
VALIDATED为`quality-check-snapshot-preflight`，仍需caller提供新unused Snapshot ID。
STALE/BLOCKED为null，只提供typed blocker/diagnosis。preflight/record/check本身绝不代跑
allowed operation，外部Generation完成状态必须有真实当前Bundle/执行evidence支持。

## Successor export、Generation 和 provenance

现有 `retranslation export` 的 recovery 专属 mode：

```text
retranslation export --locale <locale> --content-package <registered-package>
  --generator <same-provider>
  --transport-recovery <repository-relative-authority.json>
  --transport-recovery-sha256 <sha256>
  [--resume]
```

该 mode禁止 caller提供 `--id/--unit-id/--unit-kind/--limit/--batch-id`、
`--previous-snapshot-id/--surface-reopen-id/--representation-bridge-id`、
`--glossary-stale/--allow-reexport`。`R`、kind、原 Snapshot/reopen、finding和batch origin
全部从已验证 authority取得；原普通 export路径不变。新 batch ID仍由
`scanRetranslationBatches` 的 locale数值 ordinal max+1分配，全 provider共享ordinal、
新目录 no-replace；record/preflight不预留或猜 ID。每个 recovery authority只可消费一次
export，已存在 child则拒绝重复并报告其 ID/phase，继续该 child。

新 manifest保留原 `previous_snapshot_id`、`revision_feedback_source=current_validation`
及 exact finding/reopen refs，并增加 strict `transport_recovery` ref及
`reexport_reason=installed_derived_transport_stale`。这是新 invocation，不 retrofit旧064。
这个 mode由新 authority独立验证继承权限，不通过关闭原 reopen的 latest-baseline gate
实现。无新authority时现有九个拒绝边界测试必须继续PASS。

新 Bundle使用独立 `task_kind=translation-unit-transport-stale-recovery`，注明原 semantic
task为 current-validation exact revision。完整 `R` 的 source/formal input、原 Snapshot
reviewed candidate、原 finding、current glossary/full Review、最新 current proof/template
都是正式 semantic/representation inputs。原已安装 candidate仅作为另一个有 exact
refs的“历史已完成修订”上下文，不能替换原 finding baseline或成为新的 Reviewer A。
Generation沿原 finding做 locally complete revision，已正确内容应保留，不授权整页重译。
新的task kind必须仍执行完整revision/predecessor/finding/approved-proof构造，不能因为
不等于原`translation-unit-revision`字符串而跳过任何输入。新batch从attempt 1开始；
原batch最终attempt 2等记录只作为明确的历史lineage，不改号、不删除、不当作新attempt。

Bundle冻结 authority、其 proof inventory和 current recipe，沿 `formal/approved-interval-selection.json`
冻结由 authority显式 mode请求和新 batch ID唯一派生的 selection；不接受外部 selection
override或latest discovery。新的完整 proof由 current canonical builder独立生成，重新绑定
new manifest/selection/recovery ref，验证 typed recipe一致。Bundle/current-check exact ZIP
comparison、Deflate-5、bounded reader、normal archive priority/explicit legacy verifier均不变。

**新 Language Generation必须覆盖整个R。** 旧 raw只作为明确标注的历史 evidence，
不能复制进新的 output目录或改 receipt冒充新 output。import仍要求真实新 provider/model、
new Bundle SHA/input identity、attempt=1和exact outputs。此 mode还必须记录可核对的
`recovery_generation_execution`：新 session/invocation reference、实际输入 Bundle SHA/
input identity、provider/model、output hashes与原始执行 evidence ref；由 Generation role
提供真实 evidence、Maintainer核对后随 import持久化。旧 execution/receipt、只有旧 raw
路径或无法核对的新执行声明拒绝。它是新 import provenance的必填扩展，不是 machine
materialization receipt，也不声称 ZIP/hash可独立证明模型执行；沿用正式角色/执行证据
的真实性边界。真正新 invocation允许产生相同字节，不能为通过“不同hash”而强迫改词。
v1不借恢复切换provider/model，必须与原batch的真实已选provider/model一致且符合当前
独立模型政策；若无法合法使用，明确BLOCKED，先解决独立selection authority，不伪造原provenance。

execution object使用新 schema `go-learning/transport-recovery-generation-execution/v1`，
固定 `kind=model-generation`，包含locale/package/new batch/task/attempt、recovery authority
ref、new Bundle SHA/input identity、provider/model、真实session/invocation reference、
stable ordered output path/hash/size、原始执行evidence refs及canonical identity。所有字段
strict/exact，不能由旧import receipt或旧raw自动生成；CLI不新增declare-replay/force模式。
import在其既有 receipt中保存该object和evidence archive refs，作为同一pending/no-replace
事务的一部分；不把它伪装成新QC或新language authority。
reference指可核对的真实执行/response artifact身份，不要求人工环境伪造provider API
request ID；UI实际暴露的ID才按事实记录。执行evidence须明确本次实际输入和实际模型输出，
只写“我已重新生成”的自声明或重新封装旧receipt不足以通过new-execution gate。
execution object的`invocations`为实际新model calls的stable ordered列表，每项绑定上述
Bundle/input/task/attempt、provider/model、session/response evidence及其产出的output子集；
aggregate output set必须exact覆盖本次ExpectedOutputs，每个output有唯一origin。
同一formal attempt在中断后可以由多个真实新call补齐missing，不伪造成一个provider call，
也不因此重做已经可信的完整Unit；所有call必须消费同一CURRENT Bundle和已选模型。

v1 recovery输出仅使用现有directory import，新增且强制如下输入：

```text
generation-bundle import --bundle <new-generation.zip>
  --input-dir <exact-new-output-directory> --provider <provider> --model <model>
  --generation-execution <repository-relative-execution.json>
  --generation-execution-sha256 <sha256>
```

该mode不接受`--content-package`，从new Bundle的exact recovery/selection恢复完整package；
两个execution flags必须同时存在、唯一且非空，仅允许recovery task及其retry；其它task
禁止传入，以免静默获得新语义。execution JSON与evidence refs严格绑定，不接受override。
provider/model、execution object和import参数必须一致。input directory可为absolute-external
或repository-relative新staging目录，execution/evidence refs必须repository-relative。
目录不能是任何旧正式raw/retries目录，其成员不能与旧formal raw共享inode/hard link，
现有symlink/non-regular/exact-set限制保留。已复制到新staging的旧bytes也不能在缺少真实
新execution evidence时通过；文件hash相同不能单独区分合法重生成与复制，测试/交接不得
声称能用hash证明模型执行。伪造execution declaration属于provenance违规，不能被machine
PASS洗白。v1明确拒绝recovery Bundle的`result-pack/--result`路径，防止丢掉execution binding；
其它task的两种import及既有flags保持原义。retry若真实产生新response，也必须提供与其
attempt/retry Bundle精确绑定的新execution evidence。

| Modified command contract | OPERATION / INPUT / PATH | MUTATION / IDEMPOTENCE | CURRENTNESS / CAS / STOP / REPLAY |
| --- | --- | --- | --- |
| `retranslation export --transport-recovery ...` | 新建唯一full-R successor；input为exact recovery authority；refs repository-relative，locale/package/provider为identifiers | known，仅新batch；重复child拒绝；沿用原export stdout并追加`transport_recovery=<path> identity=<identity> predecessor_batch=<B> units=<R-count>` | fresh authority/epoch/closure/latest CAS与no-replace；成功停在Generation Bundle export前；失败保留stage，只允许同transaction resume |
| `generation-bundle export/current-check` | source为new child manifest/authority；ZIP output/input沿既有路径规则；禁止external recovery/selection override | export只写新ZIP，check none；output不覆盖；保留既有stdout和exact ZIP currentness | independently reconstruct new proof+recipe+recovery binding；导出/check后停在Generation前；仅未installed且同current inputs可重导，不patch旧ZIP |
| `generation-bundle import --generation-execution ...` | recovery directory import；execution/evidence repo-relative，Bundle/output-directory沿既有external规则 | known，新raw/receipt/archive/execution evidence；no-replace，无replay/no-op；保留`Generation result imported: ...`并追加`recovery_identity=<id> generation_execution_identity=<id>` | current new Bundle、new execution、exact output set/restore、provider/model、authority phase CAS；成功停在process/retry前；pending先diagnose，既有import safety与new evidence同事务 |
| `retranslation process/retry` | installed child，scope只从exact imported graph取得；不增package/recovery override flag | known，现有candidate/validation/result及attempt-history；不改原B；保留原human/`--json` contract | fresh new proof/authority/epoch/execution provenance；失败不切authority；成功停在新Snapshot前；retry只处理failure且继承同proof |

任何新增execution输入、stdout字段、currentness或resume mode都必须纳入implementation后的
true `run(rawArgv)` tests与first-real-use command matrix，不能只用domain/helper测试。
这些gate必须在共享domain层生效；不经CLI的内部export/import/process入口也不能在缺少
recovery authority或execution evidence时处理该新task，不能仅靠parser提供安全性。

import、process、validation均只消费新 child的完整 current graph；先验证新recovery ref和
epoch，再按现有流程安装/restore/active validate。新批失败仅按原 restore/validation
retry规则重试该 Unit，冻结同一 recovery/selection/proof，attempt连续；不能用 retry更新
recovery epoch。所有 initial/retry receipts、archives、raw、validation保持真实provenance。

authority的phase verification必须区分以下状态，不能在安装后调用“pending initial
export CURRENT”来判断它是否历史合法：

- record/export前：全R的latest processed selection为原B，所有原bound bytes及epoch/CAS
  exact；没有child。record后同一事实只能授权一次export。
- child已export、未import：唯一child manifest exact引用authority，scope/source/finding及
  typed construction一致；latest仍为原B；CURRENT Bundle检查适用，未installed时才可
  使用preimport staging recovery。
- child已import、未process或process失败：actual new initial/retry receipt/archive/execution
  evidence合法，installed decoder/proof fresh重建exact；不再要求它作为pending initial
  Bundle CURRENT。当前status控制process或连续retry，不能重放initial import。
- child全R validation PASS：全R由该child合法产生，所有selected attempts/provenance及
  current proof合法，方可首次recovery Snapshot。phase推进不改变receipt identity，也
  不把child result/validation的合法更新错认为原B的CAS变化。
- 此后新的QC/Surface revision：必须从恢复后的合法Snapshot沿其独立authority继续；该
  recovery receipt只提供historical lineage，不永久要求未来所有latest Units来自同一child，
  也不能将未来最新failure回退到它。若此时partial latest graph再次stale而不满足本合同
  全R准入，则明确BLOCKED，不能覆盖后来的语言authority。

## Snapshot、latest selection 与 re-QC

shared selector仍按最大 processed numeric batch ordinal逐 Unit先选latest、再验证。
new child export-only不替换旧结果；其任一 failure/identity mismatch不能退回旧PASS。
完整 `R` 已被新 child合法处理且全部PASS后，Snapshot才可选其全体 Units；不能选择
`R` 中一部分新、一部分旧来绕过任何失败。外部package Units仍按原规则验证。

Snapshot从selected child manifest的 exact recovery ref验证 original/child/full-R lineage，
冻结新 receipt、proof、selected attempts/input/raw/candidate/validation refs。旧 `B` 已无
selected Unit时不构造其 current decoder context，历史 provenance仍保留exact archive；
若旧 `B` 仍有一个 selected Unit，则原 full-proof gate照常拒绝stale graph。
不修改 `compareInstalledApprovedProofs`，不提供 scoped comparison例外，不发布unaffected
carry receipt，不把旧 batch内31个candidate标成重新attested eligibility。

恢复不是QC。Independent re-QC scope仍由新 Snapshot与实际 predecessor的 exact candidate/
source/validation/attempt和真实 A lineage决定：没有新 A 的 `R` 都是pending，不因候选
字节相同、旧validation PASS或recovery receipt自动carry A。`R`外只有原完整currentness
和existing exact A carry均PASS才carry。不能只因直接drift数是1就把re-QC限制为1。

## 事务、CAS 与中断

record采用 staged authority/proofs、fresh全量复核、no-replace发布immutable目录。每个
原 initial archive SHA在 `data/translation-transport-recoveries/<locale>/roots/` 有一个
write-once root link；显式supersession在 `supersessions/<prior-identity>.json`有唯一
write-once next link。link严格绑定 authority path/SHA/identity和root，不按mtime挑receipt。
root/next link是record commit point；目录已发布而link未提交时不是可消费authority。
发布跨路径期间保留`.record.pending` sealed transaction（owner、prior/目标、各SHA、stage
状态）；任何pending阻止该root继续mutation，不能用孤立目录当success。
索引采用 strict `go-learning/transport-recovery-link/v1`：`kind=root|supersession`、
locale、root invocation SHA、optional exact prior ref、exact next authority ref、identity；
不能声明language或currentness。pending采用本family版本化transaction记录，固定stage
inventory和hash，不接受caller编造的自由path。已验证commit但cleanup失败时报告
`mutation=known`及pending path、停止continuation；不能谎报未record后重做。

export使用同一authority的exclusive`.export.pending`，冻结待分配batch、manifest/input
hashes和prior/CAS。batch no-replace发布后验证manifest唯一且exact引用本authority，
才清除pending；registered batch manifest是其child binding，不新增可随意改写的latest
pointer。root/leaf排他锁与全局batch ordinal/no-replace共同阻止双writer和不同provider撞号。
resume也必须取得同一排他锁，不能与活跃writer同时推进；不能按mtime或“PID似乎已死”
删除pending。锁释放不代表事务commit；sealed pending和exact on-disk refs才决定恢复phase。

record/export各自的 `--resume`专属恢复形态仅允许同一原参数和 sealed transaction：
不重新生成identity、不分配另一个batch。先readonly检查owner/每一已发布/未发布path和
prior/current facts；若commit point已存在且字节完全一致，确认原commit并只清理本事务
staging；未commit且全CAS仍成立，只完成原计划的剩余步骤。其它情形保持现场，报告
`mutation=unknown`并停止diagnosis；不自动删除、覆盖、rollback正式artifact或换ID重放。
普通重复record/export仍拒绝，不能借resume接管没有匹配pending的既有状态。

| 中断/变化 | 唯一合法 continuation |
| --- | --- |
| preflight后、record前 | 无authority；重新preflight并使用新expect identity。旧PASS不授权write |
| record后、export前 | exact ref check当前epoch；current则首次export，不重新record |
| successor batch已创建、Bundle未导出 | 继续唯一child；按current inputs导出/check新transport ZIP，不能改manifest/authority |
| Bundle已导出、Generation未完成 | exact Bundle仍CURRENT则保留完整Unit及真实execution evidence，仅恢复/生成missing并补齐execution列表；不重做可信完整输出、不换epoch |
| Generation完成、未import | Bundle仍CURRENT且无formal outputs时复用同一真实invocation staging；可用现有recover仅恢复完整/missing outputs，不重新生成可信完整输出 |
| import后、validation失败 | 保存receipt/archive/raw；同epoch按现有process/retry流程，绝不reimport；旧attempt/history不删除 |
| validation PASS、未Snapshot | 重验新child/current proof后创建新Snapshot；不重做Generation或import |
| authority/implementation/dependency再次变化、尚无installed child | 拒绝旧authority继续write；只允许显式supersedes前一leaf、证明child不存在或只有零formal outputs的unused export；原authority/旧零输出child/已有staging全部保留，新leaf授权新batch，不改旧manifest |
| 已installed且全R PASS的child再次stale | 以该latest child作为新的显式predecessor，重走本合同；不能回到更老B重新覆盖它 |
| 已installed child失败且proof又stale | 不满足本合同old-PASS准入，停止于原failed-invocation blocker；不伪造PASS或用本合同创建另一候选 |
| pending / mutation=unknown | 先readonly核对sealed transaction与实际commit，再按上述限定resume；无法证明则保持现场停止 |

supersedes只在same locale/package/root、explicit prior path/SHA、leaf唯一和无installed
child时可record。新leaf仍须完整证明D/R与新epoch，不是修改旧receipt。旧leaf/child
一旦被supersede就不能作为下一write的authority；已有historical bytes仍可审计。
再次变化可能使旧pending Generation不可复用；无跨epoch等价恢复合同，不能把其输出
迁移到新invocation。可信且CURRENT的成果按原phase继续，不能无理由重做。

## 必须保持的负例与实现验收

| 负例 | Required fail-closed结果 |
| --- | --- |
| old proof仍current | readonly CURRENT，无receipt/child；record拒绝 |
| 仅JSON排版/encoding变化、construction与producer相同 | ENCODING_ONLY_MISMATCH，BLOCKED且不可record；旧exact comparison仍拒绝，不误报semantic drift |
| 非approved transport原因或unknown dependency | BLOCKED，不借recovery改变语言/source/glossary authority |
| D漏项 / R不完整 | typed diff/closure不相等，整次拒绝 |
| 加入R外Unit | 非context成员拒绝；R-D为机器推导的coupled Units，不是手加“unaffected” |
| 原provenance缺失、archive错误、proof inventory被裁剪 | reject，不fallback、不补历史ZIP/receipt |
| old validation非PASS | reject本合同；保留原failure，走其原恢复/诊断边界 |
| wrong package / Snapshot / reopen | exact ID/SHA/ordered scope/finding不等即拒绝 |
| preflight→record dependency/CAS变化 | expect identity与fresh graph不等，不发布authority |
| implementation再次变化 | 原authority/action stale，按上表处理，不ignore drift |
| duplicate record / duplicate child | write-once拒绝；只有匹配pending可resume原事务 |
| 新invocation提交旧execution/raw作为output | provenance/input identity/new-execution evidence不匹配则拒绝；不提供copy/import replay mode |
| Snapshot选择旧staleB或mixed old/new R | reject，不回退、不缩scope、不inherit历史PASS |

implementation必须保留现有9个旧路径拒绝测试，再新增synthetic positive：approved
revision installed PASS → decoder epoch drift → typed D/R preflight/record → sole full-R
successor →真实raw argv export/import/process及当前proof → Snapshot preflight PASS；
同时验证所有原archive/receipt/raw/validation/result byte-exact保留、latest failure不回退、
re-QC pending不被缩为D、record/export pending/CAS/supersession及新execution provenance。
本轮不增跑这些测试，不以design COMPLETE声称implementation/first-real-use PASS。

## zh-CN 非执行性 walkthrough

已知 locale=`zh-CN`、package=`learn-docs-v1`，原Snapshot为
`learn-docs-v1-qc-023`，installed revision为`chatgpt-zh-CN-064`，直接drift Unit为
`b49d8e873a0f66f6803077eb534583a155d2bd209a032d7c00fba8fea856da5c`。

在本合同中，这给出 `D=1`，而不是自动授权单Unit successor。future preflight读取原
batch完整32-Unit decoder context并验证closure；若所有准入均PASS，则`R=32`。
064内其余31个candidate保留为历史成果，不作为当前Snapshot的carried candidate。
新successor完整替换这32个selection；current完整51-Page package中的其它19个Unit
只走原latest/currentness/A-carry，恢复不授权其重译。新re-QC仍由真实pending scope决定，
不能宣称只需review json一个Unit。

不预填successor batch/reopen ID、receipt identity或output SHA。不创建qc024，不执行
preflight/record/export/Generation/import/process/Snapshot/QC。**本walkthrough停止在
first-real-use gate之前；不提供真实mutation命令或身份，须独立通过该gate。**

## v1 实现绑定

implementation 使用固定的 producer declaration：legacy producer contract version 1
按原 approved family 的十一项 implementation refs 派生旧 construction；version 2
额外绑定 `transportRecoveryImplementationPaths` 的闭合集合及 source-owned typed codec
reader。两者都绑定 role/path/SHA/size。version 1 缺少 version 2 新增 binding 是明确的
旧版本事实，本身不足以授权 recovery；必须同时存在原 exact proof 失败和已完整分类的
construction dependency delta。已冻结 version 2 的后续读取必须验证其原 implementation
bytes，不能再以 version 1 的缺省规则处理。

新导出的 approved Bundle 在原 Files inventory 中补充尚未包含的 exact dependency bytes，
以供将来的 versioned historical construction reader 使用；这不会补写旧 archive。旧 archive
只有 digest 时，可读取仍与该**旧 digest** exact 相等的现存 immutable bytes；若发生变化的
旧 dependency bytes 已不可取得，结果为 BLOCKED，不能拿 current refs 填空。source-owned
codec reader strict 解码完整 inverse/children 后参与 typed identity；JSON key order、缩进与
等价 escaping 不构成新 construction，实际 target/source bytes 不做 normalization。

record 与两个 exporter 共用现有 repository directory 的排他 namespace lock，不增加
可变 ordinal pointer；record/export 仍分别保留上述 sealed pending、root/leaf CAS 与
no-replace commit。resume 重新核对 sealed members 与正式原 inputs，未知/损坏 pending
不可被普通重试接管。CURRENT recovery leaf 无新 epoch 不可 supersede。import 在同一
pending transaction 中安装 execution object/evidence、raw 和 receipt，并在 publication
前再次核对 recovery CAS；中断后的 pending 阻止 process/retry，不产生隐式 replay。

check 不以 result 的 PASS 字段代替 installed graph/active validation。它从 exact initial
archive 重建 selection/construction/执行 provenance，验证 retry receipt 与 exact archive，
VALIDATED 时再确认完整 R 的 latest candidate 与 active validator。Readonly decoder
重建与 record/export action eligibility 分开：已全 R PASS 的历史 child 可以作为后续独立
QC/Surface 的历史输入，但不因此重新取得 export 权限或豁免 Snapshot selection。
