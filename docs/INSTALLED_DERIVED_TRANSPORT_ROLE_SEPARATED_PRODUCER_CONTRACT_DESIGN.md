# Installed derived transport：role-separated producer 合同设计

状态：**三项独立Review findings已修订，等待重新独立review；未实现，不授权真实 recovery。**
本轮仅补§5.2运算语义、§5.3/§7逐Unit D proof、§10 committed-result union及对应负例；
保留已获review认可的六角色、producer版本、CAS/pending/resume、first-real-use与walkthrough边界。
本设计是 [installed recovery v1](INSTALLED_DERIVED_TRANSPORT_STALE_RECOVERY_DESIGN.md)
中 producer/currentness interpretation 的显式 successor，不改变语言生命周期、D/R closure
或任何既有 artifact。当前 v1/v2 reader 对 `approvedResponseInputs` 的 BLOCKED 仍然正确。
本文没有批准该函数进入旧 allowlist，也没有宣称真实 preflight READY。
本文只规范后续实现；与旧设计中“whole declaration变化必须BLOCKED”相关的升级只在
producer3 + recovery v2 reader中生效，当前/legacy v1/v2行为不被本文即时替换。

基线：`df66a77eb2e1e0576ef761eb4d0ea98ac2f9ea9f`。
前置 diagnosis：`/tmp/go-tour-i18n-approved-response-inputs-mixed-role-attribution`；
其 exact historical/current bytes 与 semantic diff 是本设计的 motivating evidence，
不是未来 reader 可依赖的 `/tmp` authority。

## 1. 范围、版本与不变项

本设计适用于任意 locale、任意完整 registered package 的 installed approved-derived
transport。没有 locale、Unit、path/SHA compatibility override、force 或 caller role selector。
只支持现有 ordinary、deterministic-rebase、comment-transport、approved-interval decoder
variants；不新增翻译模式、target mapping、canonical representation replacement 或语言 finding。

- 旧 archive、proof、import receipt、raw、validation、result 和 supplemental byte authorities
  均 immutable。历史 validation PASS 继续是历史事实，不推导当前 Snapshot eligibility。
- 当前 reader 不改：仍逐 exact installed proof/recipe 比较、active validation、完整 provenance。
  本设计经实现和 gate 后，才可由新的 versioned reader 做角色归因；不在 v1 内偷换含义。
- `D` 仍是机器证明 direct derived-transport drift 的 Units；`R=U(B)` 仍是原 installed
  revision batch 的完整、有序 decoder context。Page-only、1..60、same locale/package、
  原 reopen/Snapshot/finding 全集、完整新 Generation 和禁止旧 raw replay 全部保持。
- 不进行 machine-only candidate reuse，不允许 31 old candidates + 1 successor 代替 full R。
  reopen 本身 stale 时 BLOCKED；不因此创建 reopen successor。canonical representation
  变化退出本合同，进入其原 successor lifecycle。
- [historical bytes authority v1](HISTORICAL_CONSTRUCTION_BYTES_FORMALIZATION_DESIGN.md)
  原样复用。它只提供 exact bytes，不授予角色、closure、equivalence 或 recovery action。

采用以下**显式新版本**；旧 schema 永远按原规则读取：

| 对象 | 新写入版本/用途 |
| --- | --- |
| producer family | 仍为 `go-learning/approved-interval-transport/v1`，`contract_version=3` |
| role projection | `go-learning/derived-transport-role-projection/v1`，完整 source/closure/evidence proof |
| role code core | `go-learning/derived-transport-role-code/v1`，单 role 的 implementation identity 输入 |
| role comparison | `go-learning/derived-transport-role-comparison/v1`，old/current attribution 与覆盖证明 |
| per-Unit dependency proof | `go-learning/derived-transport-unit-dependency/v1`，每Unit/variant的output-root闭包及exact D证据 |
| construction | `go-learning/derived-transport-construction/v2`，只绑定 construction roles 的 recipe |
| recovery authority | `go-learning/installed-derived-transport-recovery/v2`，绑定上列完整 comparison/projections |
| readonly plan | `go-learning/transport-recovery-plan/v2`，显式报告角色与 comparison 结果 |
| record pending | `go-learning/transport-recovery-record-transaction/v2`；不接管旧版 sealed owner |
| export pending | `go-learning/transport-recovery-export-transaction/v2`；不接管旧版 sealed owner |

这里的 producer version 3 不意味着 codec/namespace 3。现有 protected response codec、
canonical typed payload 与 Deflate level 5 transport policy不因此升级。

## 2. 最小 role model

固定六个 role ID；`C0/C1/V/L/E/O` 仅为本文简称，不是另一个可输入的 enum。

| role ID | 职责与输出 | 不得吸收的职责 |
| --- | --- | --- |
| `construction-context`（C0） | 从已认证 immutable language/representation facts选择完整 package、原 Snapshot/finding/candidate、source/protection 与 decoder context | new batch/session locator、安装状态、pending/ordinal、model provenance |
| `construction-codec`（C1） | 完整 protector/projection/typed E/O/parent/children/token/namespace/inverse recipe，包括 source-owned codec 的解释语义 | 外层 proof JSON 排版、ZIP method、receipt/attempt 状态 |
| `installed-proof-validation`（V） | archive/member/inventory/hash、frozen selection/recipe exact comparison、historical integrity/currentness predicates | 以 PASS 生成或修改 projection、替换不匹配 proof |
| `execution-lifecycle-validation`（L） | 新 model execution provenance、attempt/raw origin、import pending、process/retry、root/leaf/CAS eligibility | 以当前 provenance 补旧事实；把失败 candidate 当可恢复 PASS |
| `artifact-encoding`（E） | proof/manifest envelope encoding、Bundle inventory locators与外层 ZIP serialization | response codec 的 escaping/解释/还原；后者属于 C1 |
| `orchestration`（O） | 连接 typed role ports、调用顺序、collect/freeze evidence、dispatch与发布计划 | 任意改变 C 输入、丢 gate、隐藏 role call 或 mutable state |

只有 C0/C1 构成 construction producer identity。V/L/E/O **不是不重要或可忽略**：它们
有独立 identity、exact implementation evidence、compatibility proof 和 currentness gates，
全部进入完整 epoch/CAS/Bundle。它们自己的变化不能形成 D，也不能被仅有 C 的 PASS 洗白。

每个 IR node 的 `role_set` 必须非空；未知 role 拒绝。不存在 `mixed-but-safe` role。
O 可合法调用多个 role，但它是 composite wiring，**不是 construction entry point**。

## 3. Current entry points 与物理 ownership

producer 3 要求当前实现将 mixed wrapper 内的 pure construction 与 gates 分成独立函数。
允许保留 `approvedResponseInputs` 作为 O wrapper；它不得作为 C0/C1 root。本文规定的
逻辑 entry IDs 是 `context/from-language-facts`、`codec/projection-and-inverse`、
`proof/installed-exact`、`execution/installed-origin`、`encoding/envelope`、
`orchestration/response-inputs`。未来 code-owned binding 为每个 ID提供 exact Go symbol、
signature 和 source ref；函数具体命名不是授权条件，closure verifier 才是。

C0/C1 entry 只接受 immutable、typed construction facts/value 与 source-owned codec
witness；不得接收 unrestricted root/filesystem、Catalog function registry、receipt reader、
callback、环境变量或 mutable batch state。I/O由 V/L在 boundary完成，提供 exact事实与
独立 verification witness；C0 决定原语义 root 的选择，不能把 scope decision藏在 L。
新 batch/selection/model locator由 O/E绑定，不进入 construction facts。

现在各调用的角色分解如下；**这是需要由 closure proof验证的分解，不是新的函数 allowlist**：

| 当前调用/片段 | 分解 |
| --- | --- |
| `importedApprovedIntervalSelection` | V验证 exact receipt/archive/inventory/selection，L检查安装/pending；其中取 package/semantic scope 的纯字段选择导入 C0 facts，locator lookup不进入 C0 |
| `compareInstalledApprovedProofs` | V；其 exact comparison保持，不通过 C identity equality豁免 |
| `derivedRecoverySelection` | C0选择原 semantic scope；V/L认证 recovery child/epoch及 original selection；E/O写新 batch locator和 selection envelope。现有整函数不是 producer 3 的纯 root |
| `revisionGenerationInputsCore` | C0选择语义 inputs，C1构造 projection，V验证 language/representation facts，E/O生成proof/上下文 members；需拆 pure output与 gates |
| `applySurfaceTransportInWorld` | C1的 source-owned projection/inverse；V的 Surface/lineage/authority验证；O的连接，不能统称 construction |
| `recoveryGenerationInputs` | C0/C1重建完整 recipe；V/L检查 authority/current child；E/O冻结 authority、members、supplemental evidence及历史上下文 |
| `transportRecoveryConstructionMember` exact check | V；recipe内容来自 C1，是否与 installed member exact相同由独立 V负责 |
| `verifyRecoveryInstalledExecution` | L为主，内部 archive/frozen proof检查属 V；不属于 C |

`generationApprovedIntervalSelection` 上一轮的 v1 constructor ownership修复保持原解释。
producer 3 仍要对它的实际 calls/ports重做 closure；旧 allowlist 不自动成为新 proof。
任何新 implementation只拆函数而不证明旧→新 role mapping，仍不能使旧 invocation恢复。

## 4. Machine-verifiable closure：接受子集与闭合规则

采用固定 `go-learning/go-role-extraction/v1` 静态分析合同，**不是通用 Go semantic
 equivalence prover**。不执行/编译 recovered source，不调用 `go build`、plugin或旧 init。
解析、符号/类型约束检查和 bounded data/control/effect analysis只消费 exact source facts。
无法完成证明时 BLOCKED，而不是增加手工 safe annotation。

### 4.1 Source universe 与符号

1. source universe来自 verified old archive/declarations及其合法 surviving/supplemental
   bytes，或 current code-owned producer inventory。每个文件有 logical path/SHA/size；
   old缺失 ref/type/dispatch fact不从 current repo填入。闭包需要的 source即使不在旧
   十一项 root内，也必须能由既有 immutable graph exact绑定，不能任意扫描当前 sibling补齐。
2. symbol ID由 import path、decl kind、receiver exact type、decl name与完整 signature组成。
   imported alias、dot/blank import、method promotion、generic instantiation均先解析；
   无法唯一解析即拒绝。测试/build variants不能按本机默认选择。
3. 历史缺 build context时，只接受能证明对所有支持的相关语言/build语义相同的片段；
   例如无法确定 compiler directive、build variant或 loop-variable capture语义即 BLOCKED。
   不把当前 Go toolchain版本当历史事实。当前 producer 3显式冻结 build context。
4. 声明 inventory覆盖 reachable函数/方法、类型及字段布局、值/常量/全局初始化、compiler
   directives和所有影响绑定的 imports。未到达 C但属于改变的 bound file的 declarations，
   必须另外分类为 V/L/E/O；unreachable清单只可承载两侧IR未变化的非root声明。
   changed声明即使看似unused也必须有已知角色与完整proof，否则unknown并BLOCKED，
   不能靠“未调用”给任意新声明豁免。

### 4.2 Edges、effects 与 sneaky call

从 AST建立 typed CFG/def-use/alias图，而非扫描 call名称。必须枚举：

- direct与 cross-file/package calls；receiver method与 promoted method；函数值的完整
  points-to set；接口调用的完整 concrete targets；type assertion、type switch、closure capture；
- package values、constants、type definitions、read/write字段、global/init依赖；
- 数据依赖、branch/loop/return的 control依赖、defer/error/panic paths、mutation和 I/O effects；
- `Catalog.DerivedTransportCodecReader` 与 adapter的 source-owned binding。witness由已验证
  registry/schema和exact implementation facts派生，含所有 concrete types/methods；不能由 caller指定。

C0/C1 closure若能到达 V/L/E/O effect，或存在 unresolved interface/reflect/callback、
unmodelled external call、unsafe/cgo/linkname、dynamic load、并发/go statement、隐藏
mutable global/receiver/alias write，则拒绝。entry label不会覆盖这个结果。一个“纯 C函数”
偷偷调用 `verifyRecoveryInstalledExecution`，其 L/V calls与 filesystem/state effects必进入
closure，必须 BLOCKED；不能用返回值未使用、错误路径或 wrapper名字忽略。

标准库/Go builtin只可使用 extractor固定的、版本化 typed effect/intrinsic合同：纯
len/copy/append、完整 typed equality、byte/string操作等按真实读写/alias语义分析；
filesystem属 V/L，外层 JSON/ZIP属 E。codec interpretation的 strict decoding只可通过
已知 source-owned schema reader进入 C1。没有 summary的 intrinsic拒绝；不得给 repo helper
手写 opaque“pure” summary跳过源码。repo helper必须递归展开/验证，summary仅是缓存的
证明结果，绑定该完整 closed IR。任意 `reflect.Call`/runtime dispatch不支持；受支持的
`TypeOf`/typed-kind guard也必须满足固定 effect规则，否则拒绝。

算法固定点覆盖所有 targets与alias，不将动态“一次 trace没走到”当完整证明。循环保留
完整 CFG，不能缩 Unit footprint；无法证明 termination/effect边界或资源超界时拒绝。
一个 projection最多8192 source files、20000 declarations、200000 IR nodes、1000000 edges；
任一超界为 BLOCKED，不能截断。Bundle继续8192 members/134217728 logical bytes。

### 4.3 Typed ports、控制流与 composite wrapper

固定 port kinds：`construction-facts`、`construction-value`、`construction-status`、`verified-archive-value`、
`validation-result`、`execution-result`、`artifact-value`、`immutable-evidence-ref`。
每条跨 role edge记录 source/destination node、完整 port type、flow kind及 effect。
V/L只可将已验证事实经显式 boundary交给 C，不能借“verified”标签注入任意 data。
认证事实是 exact原 semantic inputs，选择哪一个原 Snapshot/package属于 C0。

O wrapper可以调 C/V/L/E，但必须证明：

- C的输入/输出 def-use完整，外层 locator不进入 C；
- V/L的 guard只控制是否接受/继续，不修改 C数据、共享alias、全局或构造返回值；
- 每个 required gate在所有成功出口之前支配该出口；error分支保持拒绝，不转成默认值；
- C返回值没有被 E/O非编码性质地改写；真实 response bytes绝不 normalise；
- role之间的顺序、guard及 effects全记录在 wiring IR。不能简单删掉所有 `if err`。

construction内部的合法性predicate/拒绝状态属于C，但诊断error文本属于E/O，不作为C内容。
producer3纯entry用typed construction-status（`accepted|rejected`）返回状态，诊断文本在外层
render；legacy bridge只能按完整return/error def-use证明这种区分，保留predicate/control
和所有诊断source evidence，不能根据error文本猜角色。若fmt/Stringer等无法证明是独立
诊断路径，或error值回流影响C数据，仍BLOCKED；不能把所有error分支一刀切忽略。

legacy mixed function允许**受限 structural projection**：分析完整 source graph之后，
按上述 proof分离 pure dataflow与独立 guard/wiring。不是给任意 AST subtree贴标签；
每个 statement、call、control edge及 return必须有 coverage。无法拆清一个 statement、
同一 alias跨 role写入、branch同时改 C值与 gate时仍是真正 mixed-role，BLOCKED。

仅支持固定的 nil/error guard、typed discriminator、局部赋值、可解析纯 call/loop及上述
已证明 port wiring。predicate只能从 exact frozen facts得到；例如旧 manifest中不存在
`transport_recovery`可证明相应 branch在该 historical domain不执行。inactive region仍须
完整解析、记录 role/effects/source coverage，并由独立 L/O route proof证明未污染成功路径；
不是任意 dead-code删除。未知 predicate、未知 fact、通过异常/alias跨 branch影响 C均拒绝。

当前 producer 3不得依赖这种 legacy mixed extraction来避免物理拆函数。它只用于读取
已有 v1/v2 facts。两侧最终均投影到同一 typed role IR；只比较相同 extraction版本和语义
comparison domain，不以 old/current batch locator区别生成不同 recipe。

### 4.4 Shared declaration

同一声明可以被多个 role共享，但完整 `role_set`与每个调用者必须列出，不能只列 C调用。
未改变的 pure shared helper可作为共同 closed dependency。global/shared mutable state拒绝。

shared节点发生变化时，默认 `UNCLASSIFIED_PROOF_MISMATCH`。只有 analyzer能证明各 role
使用的实例/regions独立（例如固定 typed discriminator的互斥分支、disjoint def-use/effect
集合，且所有 changed nodes均唯一归属一个 region），才能分别归因。一个 changed node
同时位于 C和non-C slice、同一共享 constant/类型布局影响多 role而无法分离时必须 BLOCKED；
不能将同一变化标两次便宣称已分离。shared helper的文件变而其实际 closed declaration/IR
未变，不构成 shared-node delta。

## 5. 精确 schema 与两种 identity：不可再把 file SHA塞回 construction

以下新对象全部 strict JSON：字段必填，unknown/duplicate/version/role拒绝；不用 RawMessage
作 typed identity。null仅出现在明确 nullable字段；集合空为 `[]`，非缺失/null。
所有 hash lowercase SHA256；size为非负 int64；file/member refs沿用安全 `{path,sha256,size}`。
仓库 refs必须 repository-relative、regular/no-symlink、exact hash/size；member ref只能在其
已验证 container inventory解引用。source artifact acquisition metadata不进入任何 role identity。

### 5.1 Role projection v1

固定字段顺序：

```text
schema, producer, extraction, subject, source_files, declarations,
entries, ports, edges, role_views, coverage, route_facts, identity
```

| 字段 | 完整结构/含义 |
| --- | --- |
| schema | `go-learning/derived-transport-role-projection/v1` |
| producer | `{family, contract_version, origin}`；origin=`frozen-explicit\|derived-legacy`；legacy真实version只能1/2，current为3 |
| extraction | `{id, version, implementation_refs, policy_identity}`；id=`go-learning/go-role-extraction`，version=1；implementation_refs为本次分析器完整实现的file refs集合，policy_identity为其固定规则/role/ports/intrinsics合同identity |
| subject | `{locale, content_package, predecessor_invocation, semantic_context_identity, domain}`；package/invocation复用recovery v1的完整closed类型；domain=`original-revision-context`，successor诊断仍绑定原 language context，而非新 locator |
| source_files | 有序 `SourceUnit`，字段见下 |
| declarations | 有序 `Declaration`，字段见下 |
| entries | `{entry_id, role_id, declaration_id, signature_identity, output_ports}`数组；entry_id为第3节六个逻辑ID及code-owned variant后缀，禁止 caller提供 |
| ports | `{id, kind, type_identity, fact_refs, obligations}`数组；obligations为固定enum集合 `exact-input\|immutable\|single-origin\|no-alias-write\|gate-dominates-success\|no-locator-flow` |
| edges | `{from, to, kind, port_id, dispatch_targets, effects}`数组；kind=`call\|method\|value\|type\|data\|control\|alias\|init\|port`；effects=`pure\|read-fact\|read-installed\|encode\|reject\|write-local\|publish`的集合；publish不得到达 C |
| role_views | 六个固定 `RoleView`，顺序为第2节表格顺序 |
| coverage | `{covered_declarations, unreachable_declarations, shared_regions, unresolved, coverage_identity}`；前两项为完整Declaration IDs集合；shared_regions为`{id,role_set,node_ids,separation_witness}`；unresolved必须为空；coverage_identity为前四项typed hash |
| route_facts | `{predicate_node, fact_ref, fact_selector, relation, value, affected_regions}`数组；selector仅已知schema reader的typed字段；relation=`is-null\|equals-enum\|equals-identity`；value为`{kind,text}`，kind=`null\|enum\|identity`；不是自由JSON pointer |
| identity | 仅此完整 projection的证据identity；置空后typed canonical hash |

`SourceUnit`固定 `{id, logical_path, raw_ref, declaration_origin, material_origin,
material_evidence_refs, build_context}`。raw_ref为`{container,member,file}` tagged ref：
archive来源container/member必填且file=null；repository来源file必填且其它null。
declaration_origin为 `{kind,historical,current}` tagged对象。kind=`historical-declaration`
时historical为`{archive,member,producer_version,declared_dependency}`，current=null；
原root的declared_dependency必须是已有声明，callee可用同一verified archive inventory的
exact RepositoryPath/member binding及调用边证明来源，此时declared_dependency=null并明确
不是新的adoption声明。kind=`current-contract`时current为`{producer_version,entry_contract_ref}`、
historical=null，entry_contract_ref是本次完整冻结的code-owned binding文件ref；当前source
不伪称已有historical archive。任一侧缺其必要origin直接BLOCKED。
material_origin=`verified-archive|surviving-repository|recorded-supplemental|current-source`，
最后一项仅current侧；evidence refs独立exact绑定。
SourceUnit.id仅hash `{logical_path,raw_ref}`；不包含后续build witness，避免自引用。

build_context为`{language_contract,goos,goarch,tags,variant_proof}`；旧缺事实的字段为null，
仅有第4节版本独立性 proof时允许；current侧完整显式冻结。variant_proof为typed proof ref，
不是“用当前平台”的字符串。跨文件 source缺少合法 origin同样 BLOCKED。

`Declaration`固定 `{id, symbol, kind, source_id, source_span, signature_identity,
raw_declaration_sha256, ast_identity, node_ids, role_set}`。symbol为`{package,receiver,name,
instantiation,signature}`；kind=`function|method|type|value|constant|import|directive|init`。
source_span为`{start,end,unit}`，unit仅`byte`、半开区间；严格绑定exact raw file。
raw_declaration_sha256对应该span原bytes；ast_identity来自固定parser的完整typed AST，
不以format.Node外观bytes代替role语义。decl ID hash完整symbol+source_id+span+rawSHA，
只作证据定位，**不得进入 construction role code core**。

`RoleView`固定 `{role_id, semantic_contract, code, code_identity, bindings}`。
semantic_contract固定`go-learning/approved-role-semantics/v1`；code字段是下一节的closed
RoleCodeCore；code_identity必须等于code.identity。bindings为`{node_id,declaration_id,source_span,edge_ids}`数组，必须完整覆盖。

### 5.2 RoleCodeCore v1 与 source evidence分离

RoleCodeCore固定 `{schema, role_id, semantic_contract, entries, types, blocks, nodes, edges,
ports, intrinsic_contracts, identity}`；schema=`go-learning/derived-transport-role-code/v1`。
identity置空canonical hash即 `role_code_identity`。其类型/entry/port为第5.1节对象的**逻辑
值副本**，不带 source id、physical SHA、span、external acquisition或新 invocation locator。

#### 5.2.1 Closed operation payload（Finding 1 修订）

RoleCodeCore字段顺序补全为`{schema,role_id,semantic_contract,entries,types,blocks,nodes,
edges,ports,intrinsic_contracts,identity}`；其它identity/role约束不变。block固定
`{id,parameters,body,terminator}`：parameters为有序SSA值，body为执行顺序node IDs，
terminator唯一且必须属于该block。跳转实参与block参数逐位置/类型一致。

Node固定`{id,kind,result_types,operands,payload}`。kind是下表的closed enum，不再使用
缺operator的通用op记录。result_types为有序Type IDs；operands为有序
`{node_id,result_index}`（从0开始）的SSA引用，不能排序或去重。void是`result_types=[]`，
不是缺失类型。每个result index必须存在、type完全匹配；多返回值不能合并成单type。
所有CFG/body/operand/control/effect边都进入完整core和closure验证。
下表的signature、constructed_type、receiver_type、type_arguments/result_types均绑定Core
完整type图的Type IDs；symbol/method绑定§4.1的完整resolved symbol并与该signature一致。
不得把同名函数/方法、未实例化type参数或仅一个文本type名字当作已验证result binding。

payload是按kind选择的closed tagged struct：kind唯一决定其字段集，**下表列出字段全部
必填，其它字段全部禁止**。nullable只在表中明确允许时使用；没有“unknown但忽略”payload。
每个payload内部按表列字段顺序canonical typed `json.Marshal`；不允许通用map、RawMessage，
也不能靠node ID、source/AST、result type或operand值猜operator。

| kind | payload固定字段与操作语义 | operands / results与字段条件 |
| --- | --- | --- |
| `parameter` | `{entry_id,index}` | 无operand；一个result，必须是该entry/signature位置的type；method receiver先按signature显式列入 |
| `literal` | `{literal_kind,value}`；沿下文精确literal编码 | 无operand；一个与literal类型兼容的result；不能省略literal kind/value |
| `reference` | `{reference_kind,symbol}`；reference_kind=`function\|constant\|immutable-global` | 无operand；一个result，绑定完整resolved symbol/signature/type；global初始化闭包仍须完整；未知mutable global拒绝 |
| `field` | `{access,field_path}`；access=`value\|address`，field_path是完整有序field IDs | 一个base；一个field value或对应pointer result；嵌入/间接访问逐层解析，nil/bounds语义不能省略 |
| `index` | `{access,container_kind}`；access=`value\|address\|map-value\|map-value-present`；container_kind=`array\|pointer-array\|slice\|string\|map` | `[base,index/key]`；value/address一个result，map-value-present为`[value,bool]`；map缺项zero-value和数组/切片/字符串Go bounds/panic规则由固定type/operation合同绑定；不合法组合拒绝 |
| `slice` | `{form,has_low,has_high,has_max}`；form=`two-index\|three-index` | `[base,low?,high?,max?]`，出现顺序固定；flags与operand数量必须相等；three-index要求has_max=true，two-index要求false；一个确切slice/string result；其余Go类型/bounds约束必须满足 |
| `construct` | `{construction_kind,constructed_type,items,function,captures}`；construction_kind=`struct\|array\|slice\|map\|closure` | result为一个constructed_type；items和captures索引下述有序operands；function仅closure非null；非closure的captures空；不能用该kind隐含conversion/assert/new等其它操作 |
| `unary` | `{operator}`；固定enum见下 | 恰一个operand和一个result；operator与type规则独立核验，不能由type推operator |
| `binary` | `{operator,evaluation,right_block}`；固定enum见下 | `[left,right]`、一个result；有序，不能交换；短路与eager的CFG要求见下 |
| `call` | `{call_kind,callee,signature,type_arguments,variadic_mode}`；call_kind=`direct\|function-value` | direct的callee为resolved symbol，无callee operand；function-value的callee=null，operand0为函数值且points-to全集已知。之后参数按Go求值顺序，results严格匹配signature。variadic_mode=`fixed\|expanded`，区分f(xs)与f(xs...) |
| `method-call` | `{method,signature,type_arguments,variadic_mode,dispatch}`；dispatch=`{kind,receiver_type,targets}`，kind=`static\|closed-interface` | operand0为receiver，其后为有序实参；完整targets（有序集合）不可省略；receiver隐式取址/解引用必须已有显式IR，不在metadata外偷偷发生；所有targets结果类型一致且闭包闭合 |
| `builtin` | `{contract_id,contract_version,type_arguments,variadic_mode}` | 只接受extractor固定intrinsic合同；名字/version、全部type实参、求值/副作用/失败规则和参数/results均验证；未建完整合同的Go操作拒绝 |
| `assign` | `{assignment_kind}`；assignment_kind=`ssa-copy\|store` | ssa-copy为一个value operand、一个同type result；store为`[address,value]`、无result，必须有完整alias/effect顺序。`+=/-=/++/--`只能完整lower为相应operator节点+assign，不能抹operator |
| `phi` | `{incoming}`，incoming为有序`{predecessor_block,operand_index}` | operands与每个CFG predecessor逐一匹配；一个result，各incoming type相同；缺/重/额外predecessor或错operand index拒绝 |
| `branch` | `{true_block,false_block}` | 恰一个bool condition、无result；true/false顺序具有语义，不能排序；条件完整def-use/operator必须保留 |
| `jump` | `{target_block}` | operands按目标block参数顺序传入；无result；break/continue只可在label/target完全解析后lower为该kind |
| `loop` | `{condition_block,body_block,post_block,exit_block}` | 无operand/result；post_block可null，其它必须存在；这是完整CFG的结构anchor，条件/更新/进入/回边必须另有实际branch/jump与数据nodes，不能只存四个名称代替循环语义 |
| `range` | `{phase,source_kind,order,iteration_binding}`；phase=`init\|next`，source_kind=`array\|pointer-array\|slice\|string\|map`，iteration_binding=`per-iteration\|reuse-existing` | init一个collection operand、一个typed iterator-state result；next一个state operand、`[next-state,ok,key,value]` results（未使用也不能抹返回布局）。array/slice为index order，string为Go UTF-8/rune iteration，map为Go-unspecified order；order只允许与source_kind匹配。循环变量binding必须由历史language/build facts证明 |
| `return` | `{signature}` | operands按该signature返回值顺序；无result；named return必须先显式load/SSA resolution，不能省略数据/alias依赖 |
| `defer` | `{callee_kind,callee,signature,type_arguments,variadic_mode,dispatch,activation,capture,unwind}`；callee_kind=`direct\|function-value\|method` | callee/receiver/实参布局遵循call/method-call；capture固定evaluate-now，unwind固定go-lifo，activation绑定该逻辑function activation（entry+call-context），无result；完整exit/unwind、named-return/alias、panic/recover effects必须有可验证模型，否则本op BLOCKED |

defer的direct/function-value情形dispatch必须null，callee非null/null分别遵循call规则；
method情形dispatch必须非null并遵循method-call，receiver在operand0。activation是
`{entry_id,call_context}`逻辑frame key，不是PID/地址；capture/unwind除固定值外全部拒绝。
range.order固定enum=`index-ascending|utf8-rune|go-map-unspecified`，依次只适用于
array/pointer-array/slice、string、map。iterator-state type必须完整绑定source type与
iteration规则；不能用一个无参opaque state type掩盖source kind/语义。

`construct.items`固定有序`{item_kind,field_id,index,key_operand,value_operand}`：
struct-field仅field_id/value_operand非null；array-element仅index/value_operand非null；
sequence-element仅value_operand非null；map-entry仅key_operand/value_operand非null。
item_kind由construction_kind限制；其余字段为null，不允许混用。struct/array/slice省略项
的Go zero-value/type/length规则与所有显式items一起编码；slice长度等于实际item布局，
含index布局必须显式给出index并由array-element形式编码。items按原求值顺序，不按key排序。
closure的function是完整logical symbol；captures为有序`{symbol,mode,operand_index}`，
mode=`value|address`，必须与capture/alias图一致；closure只可在targets闭合时调用。

`unary.operator`只允许`plus|minus|logical-not|bitwise-not|address-of|dereference`，
对应`+ - ! ^ & *`，其Go类型、addressability、nil/panic与effect规则必须全部编码/检查。
receive等不在此enum的运算不得假装unary继续。`binary.operator`只允许：

```text
add subtract multiply divide remainder
bit-and bit-or bit-xor bit-and-not shift-left shift-right
equal not-equal less less-equal greater greater-equal logical-and logical-or
```

它们逐一对应`+ - * / % & | ^ &^ << >> == != < <= > >= && ||`；operator单独进入hash。
evaluation只有`eager|short-circuit`：只有logical-and/or可且必须short-circuit，right_block
必非null，right只可沿相应left条件求值，其结果到join/phi与完整CFG绑定；eager的right_block
必须null。相同operands/type下将`add→subtract`、`greater→less`或`equal→not-equal`
必然改变canonical payload与role_code_identity，不能通过推断、交换operands或source ref补救。

其他Go语义不能塞进通用node：conversion/type assertion必须用完整的versioned primitive
intrinsic（包含target type、单值panic/双值comma-ok区别等）或显式的等价lowering证明。
range特殊“不求值collection”的语言规则、整数/iterator/channel range、type switch、
conversion、defer/unwind等若当前extractor不能按完整合同编码，则整次projection BLOCKED；
不得省略node、operator、evaluation、隐式zero/panic行为后继续。map iteration必须另证
输出不依赖unspecified order；C不接受effect/确定性不闭合。此处补齐表示规则，不扩大
原来允许的Go/effect/role子集。

literal_kind=`string|bytes|integer|float|complex|boolean|nil`；string/bytes用base64保留原
byte序列，integer无多余前导零，float为约分`p/q`（q>0），complex为`pr/qr;pi/qi`，
boolean为`true|false`，nil为空字符串。所有literal、type/signature、targets、CFG顺序、
取值/失败/effect规则都必须进入对应typed core，不能做正文/代码/target normalization。
完整source/AST仍是evidence binding，**不是缺失RoleCodeCore语义的替代物**。每个payload
必须与已绑定source typed opcode及固定lowering一一对应；verifier须从exact AST独立重算
并核对该payload。source中`!=`却编码为equal同样INVALID，不能以hash自洽获得witness。
即使错误encoder令两个不同canonical cores得到相同声明identity，完整core equality/
source-to-IR核对仍须拒绝，而不是仅凭摘要相同宣布相等。

这里不含全图role_set：其它role新增caller不能改变原C的code identity。所有payload、
result_types、ordered operands、block/node执行顺序和完整edges均属于canonical core；
node IDs不能承担未声明的operator信息，非identity字段也不能藏运算语义。

`types`为`{id,kind,name,arguments,fields,methods,length,parameters,results,variadic,representation}`的闭合图；
kind覆盖primitive、array、slice、map、pointer、struct、interface、signature、named、type-parameter；
name为完整package/name，arguments为有序type IDs（map依次key/value，pointer/slice为element，
named/generic为完整underlying/实参）。array的length为非负int64，其它kind为null；signature
的parameters/results为分别有序的完整type IDs，variadic为bool；非signature两数组空且false。
fields为`{name,type_id,tag,embedded}`，methods为`{name,signature_type,receiver_kind}`，
receiver_kind=`value|pointer`；实际字段/参数/方法顺序及tag保留。无法解析的实例拒绝。
representation仅primitive非null，固定`{model,bit_width,signed}`；model闭合为
`bool|signed-integer|unsigned-integer|float|complex|byte-string|nil|untyped-integer|
untyped-rational|untyped-complex`。numeric具体machine type必须有已验证的正bit_width，
integer的signed与model一致；untyped、bool、string、nil的bit_width/signed为null，float/
complex的signed=null。named type沿完整underlying type绑定representation；不能只存一个
`int`名称省略其width，使不同overflow/shift行为得到同identity。缺historical build事实且
不能证明唯一表示时BLOCKED，不用当前platform补旧事实。
Core entries固定`{entry_id,role_id,signature_identity,output_ports}`，没有declaration_id；
Core ports固定`{id,kind,type_identity,fact_slots,obligations}`，fact_slots为完整原semantic
context的typed slot IDs，不是文件或新invocation refs；Core edges沿第5.1节结构但只用
logical node/port IDs。binding到exact bytes的关系全部由projection验证，不能由caller假定。
logical node ID用entry/CFG/def-use固定遍历序号；type ID为闭合type描述的canonical identity，
循环type用稳定SCC编号。edge IDs为完整typed edge hash。source span绝不充当logical ID。
旧Catalog/manifest容器不能直接伪装成新purefacts类型。legacybridge必须从已知schema字段与
完整source def-use证明typedport lifting，原容器完整类型/字段/alias证据仍在projection中；
任何未解释的field读取或写入拒绝，不能按currentfacts类型任意剔除old字段。
`intrinsic_contracts`为`{id,version,signature,effects}`集合，由extractor固定，不能从调用者
“pure”注释生成。RoleCode的edges/ports与projection对应，但用logical node/type/entry IDs，
排除physical refs；fact输入只用原semantic slot ID/type，其值由完整semantic context绑定。

IR canonicalization只有：解析语法无关的括号/注释、稳定SSA local编号、exact resolved
symbol与CFG order、以及已证明pure无递归透明helper的结构展开。函数物理拆分/局部变量
改名不能单凭version迁移制造D。复杂recursion保留封闭SCC；无法建立唯一common IR时
BLOCKED，不做general optimizer、模糊equivalence或constant folding正文。相同算法必须对
old/current独立执行；不允许观察current后选择旧subtree。

关键区分：

```text
projection_identity = hash(exact sources + declarations + role bindings + closure proof)
construction_producer_identity = hash(C0.role_code_identity + C1.role_code_identity)
```

前者绑定old bytes SHA、declaration identity、role、真实producer version、extraction版本。
后者只覆盖已证明的 C closure；完整file SHA、projection identity与non-C role identities
进入epoch/comparison，**不再回流到construction_producer_identity**。否则同file的V/L/E/O
变化仍会污染C，设计没有解决问题。

原v1/v2 construction identity保留原值，仅审计使用。上述新view明确标记derived，不声称
旧archive保存过role graph。origin producer version参与projection evidence，不自动参与
两侧可比的role code；相同固定comparison contract下的version bridge必须先验证，不能
因1/2→3字符串不同自动制造drift。

### 5.3 Construction v2、comparison与recovery authority v2

construction v2固定字段：`schema,producer,semantic_context,decoder_constructions,
construction_dependencies,construction_identity`。后四项中semantic/decoder原值沿用construction
v1的**完整typed对象**，不改变scope/payload字段或现有codec语义。producer为
`{family,contract_version,role_semantics,construction_roles,construction_producer_identity}`；
roles仅按C0/C1顺序含`{role_id,role_code_identity}`。construction_dependencies仅是C closure
读取的semantic slots/exact原authority依赖，不含file证据、model/session、新batch locator。
完整implementation/file/evidence由下列comparison与epoch保留，不能丢掉。
construction_producer_identity精确hash `{family,role_semantics,construction_roles}`；
contract_version=3由construction v2整体identity绑定，但不拿该数字与legacy版本直接比较。
semantic_context_identity为原完整typedsemantic_context的canonical hash，不裁剪R或finding。
新结构与legacy旧identity的比较只能通过下述共同projection view。

comparison v1固定 `{schema,extraction_contract,bridge,subject,old_projection,current_projection,
role_deltas,unit_comparisons,nonconstruction_checks,payload_check,coverage_check,classification,identity}`。

- extraction_contract固定`{id,version,policy_identity}`，必须与两侧projection完全相同；
  两侧本次extractor implementation_refs也必须相同，origin producer refs则各自保留。
- bridge=`{id,from_producer,to_producer,rule_version}`；支持的id仅
  `legacy-approved-to-role-separated`（1/2→3）与`role-separated`（3→3），rule_version=1。
- projection refs是exact member refs；role_deltas固定六项，每项
  `{role_id,old_code_identity,current_code_identity,changed_regions,changed_bindings,disposition}`；
  disposition=`equal|construction-dependency|supported-nonconstruction|blocked`。
  changed_regions按§5.3.2完整表示node/port/type/control等差集；changed_bindings记录exact old/newDeclaration refs对，
  可一侧null但必须证明新增/删除，不能把缺bytes当null。
- nonconstruction_checks为V/L/E/O四项 `{role_id,contract_id,contract_version,proof_ref,result}`，
  result仅`pass|blocked`；proof_ref为下节固定typed witness，不是free-form justification。
- payload_check/coverage_check为`{status,proof_ref}`；status=`pass|blocked`。
- classification=`current|construction-dependency-drift|encoding-only|nonconstruction-only|
  unclassified`；identity为该完整object identity置空后hash。

recovery authority v2复制v1全部字段/约束，schema改v2，在 `current_epoch` 后增加
`role_comparison` exact member ref；`construction`只指construction v2。direct drift每项
增加 `role_delta_refs`（`{comparison,role_id}`数组），comparison为完整memberref，role_id仅
能选择本comparison中C0/C1的construction-dependency项，不使用自由JSONpointer；旧
`derived_old_construction_identity`保留历史派生值，新增 `comparison_old_construction_identity`
和`comparison_current_construction_identity`用于共同role contract下比较，不能把旧v1
identity与新v2 identity直接比大小/不等。authority完整typed identity同时绑定这些refs。
其它allowed action、full R、historical integrity、final attempts与scope不改；逐Unit准入
必须同时有下节unit_dependency，不接受只引用整个C0/C1 delta。

#### 5.3.1 Per-Unit construction dependency proof（Finding 2 修订）

新增derived evidence schema `go-learning/derived-transport-unit-dependency/v1`，不新增人工
scope authority。对原batch stable order中的**每个Unit、两侧各一份**生成proof；approved
Units不可缺，ordinary/rebase/comment siblings也用对应variant的完整root，不能用batch-level
role delta或R coupling edges冒充它。旧archive没有此对象时只从已验证旧facts派生，标明
`side=old`，不补写archive。

固定字段顺序及类型：

```text
schema, side, subject, unit, variant, producer, extraction, projection,
semantic_slots, output_roots, dependency_edges, local_bindings, closure, closure_identity,
unit_construction_identity, identity
```

- schema固定上述v1；side=`old|current`；subject沿projection的完整typed subject。
- unit=`{id,kind,source_sha256,formal_input_sha256}`，必须与原manifest/recipe/正式Unit exact
  一致；kind保持既有Page-only约束，不接受caller提供的另一个Unit。
- variant=`ordinary|deterministic-rebase|comment-transport|approved-interval`，由strict recipe
  discriminator取得；不能凭文件名、token、语言内容或想缩D的意图推断。
- producer=`{family,origin_contract_version,comparison_contract_version}`，保留真实1/2/3，
  comparison_contract_version固定3；extraction=`{id,version,policy_identity}`，两侧必须相同。
- projection=`{ref,identity}`：ref为exact role-projection member ref，identity为该完整projection
  identity，且与subject/side/producer相符；不得只绑定一个batch role name。
- semantic_slots为有序`{slot,type_identity,value_identity,fact_refs}`；slot是下列closed key，
  value_identity=SHA256(该slot完整typed原值的canonical encoding)，fact_refs是确切原authority/
  source/representation事实，不含用于决定D的自由文字或implementationfileSHA。
- output_roots为按context、decoder顺序的两个`UnitOutputRoot`，二者均必需。
- dependency_edges为下述完整typed边集合；closure是机器推导的完整局部subgraph，不是caller
  自由挑的nodes。closure_identity与unit_construction_identity定义见下；identity为完整proof
  typed core的SHA256，identity字段置空，包含以上全部字段。

**Closed semantic slot。** `SlotKeyV1={kind,ordinal}`，kind只可为：

| 适用范围 | 固定kind及原typed来源 |
| --- | --- |
| 所有variant | `semantic-context`=完整原semantic_context；`unit-source`=该Unit/source facts；`formal-input`=完整原protected input；`reviewed-candidate`=原reviewed candidate与其identity；`protected-baseline`=完整response baseline；`token-inventory`=完整ordered tokens；`token`=其中一项 |
| approved-interval | `approved-representation`=原/当前representation pair；`approved-namespace`；`approved-parent`=whole parent/坐标/hash/不可变bytes binding；`approved-children`及`approved-child`；`approved-findings`及`approved-finding`；`approved-codec`=完整source-owned typed inverse payload；`approved-provenance`=原semantic/provenance roots |
| deterministic-rebase | `rebase-transport`=完整strict Rebase payload；`rebase-binding`=其完整source-owned ordered binding一项 |
| comment-transport | `comment-transport`=完整strict Comment payload；`comment-slot`、`comment-binding`、`comment-opaque-boundary`=其相应ordered inventory一项 |

仅`token|approved-child|approved-finding|rebase-binding|comment-slot|comment-binding|
comment-opaque-boundary`的ordinal为非负整数且必须遍历原inventory全体；其它ordinal必须
null。以上映射由variant/schema-owned reader固定生成，不能输入JSON pointer/任意slot名字。
semantic-context包含原package、ordered R、Snapshot/reopen/finding、glossary/policy等完整
字段；不得借per-Unit提取缩原editable footprint。parent/children/codec等重复覆盖同一事实
时必须exact一致。每种recipe全部typed字段必须映射到上述完整aggregate或元素slot，新增
schema字段无对应固定映射就BLOCKED，不允许遗漏后继续。列为slot不意味着它一定到达
该Unit每个output；实际影响范围仍由下述def-use/control closure确定。

**Root与edge。** `UnitOutputRoot`固定
`{id,kind,unit_id,variant,role_id,entry_id,output_port,result,type_identity,value_identity}`。
kind=`context|decoder`；context为C0、decoder为C1。id由typed `{unit_id,variant,kind}` hash
唯一派生。result为完整`{node_id,result_index}`，必须对应指定entry成功出口的该output_port，
其type/完整typed值分别与该Unit context或完整variant recipe匹配。多个出口必须在canonical
CFG用完整控制/phi汇聚，不能挑一次成功trace的return。无法重建该root就whole recovery BLOCKED。

DependencyEndpoint固定`{kind,payload}` closed union；每种payload仅允许以下字段：

| kind | payload |
| --- | --- |
| `role-node` | `{role_id,node_id,result_index}`；result_index只在zero-result control/effect节点为null，其它必须有效 |
| `role-port` | `{role_id,port_id}` |
| `role-entry` | `{role_id,entry_id}` |
| `role-type` | `{role_id,type_id}` |
| `role-intrinsic` | `{role_id,contract_id,contract_version}` |
| `role-edge` | `{role_id,edge_id}` |
| `role-block` | `{role_id,block_id}` |
| `semantic-slot` | `{slot}`，只接受SlotKeyV1 |
| `unit-root` | `{root_id}`，只指本Unit两个root之一 |

role_id只可第2节的construction-context/construction-codec两个literal IDs，C0/C1仅简称。
logical IDs必须在exact projection的对应RoleCodeCore中存在。其它
payload字段、跨Unit root、错variant、未解析output port均拒绝。
DependencyEdge固定`{source,target,relation,projection_edges,fact_refs}`，relation闭合为
`data|control|dispatch|alias|type|intrinsic|slot-binding|output-binding`；projection_edges为
有序`{role_id,edge_id}`引用，必须证明source node/result/port经相应关系到target slot/root
或中间dependency endpoint。slot-binding/output-binding即使没有现成Go call edge，也须从
完整typed port/return binding唯一重算，不能手写“Unit A使用C0”。fact_refs仅可支持已经
认证的slot值/branch事实，不授予删边权。每个alias写、实际callee/helper、type/intrinsic、
condition、loop/phi和output port的影响都必须在图中；未知或漏一条即不完整并BLOCKED。
每个entry parameter必须由本Unit已认证semantic slots/上游output ports唯一实例化；无法
解析哪个Unit/variant的参数流入该output时BLOCKED，不可只引用generic entry名称。

**确定性闭包与两层identity。** 从每个Unit的两个output roots沿上述边逆向到全部前驱，
对数据、控制、alias、dispatch/type/intrinsic依赖求least fixed point；不按source file边界
截取。使用所有可行路径，不使用一次运行trace。只有§4已有的immutable typed route facts
和完整inactive witness可排除某路径；不能用自然语言finding、关键词或target内容猜范围。

local_bindings为完整`{global_endpoint,local_kind,local_id}`数组：role-* global_endpoint属于
exact projection，semantic-slot/unit-root则属于本Unit proof中的对应对象，local_kind=`root|entry|block|node|edge|port|slot|type|intrinsic`；必须是闭包中的
一一映射，不容caller遗漏/重配。它进入proof identity，但不进入下列局部closure identity。
closure固定`{roots,entries,blocks,nodes,edges,ports,slots,types,intrinsics}`：nodes是完整相关
IR（含closed payload/operator/results）；slots仅含`{slot,type_identity,value_identity}`，
root/port及其它对象仅含其本地完整semantic结构，不复制physicalrefs、globalprojection IDs
或fact_refs。全部global引用由local_bindings一一重映射，不能通过剔除field改变payload。
按root kind、SlotKey固定
variant顺序、edge relation及ordered operand/CFG稳定遍历重新赋予**Unit-local** IDs；SCC按
该有序root遍历编号，集合按canonical IDs排序，实参/field/CFG等语义顺序不动。这样其它
Unit在batch graph插入node或global ID重新编号，不会伪造本Unit的语义dependency变化。

`closure_identity = SHA256(canonical typed closure)`。
`unit_construction_identity = SHA256({schema,unit,variant,semantic_slot_values,
output_values,closure_identity})`：schema固定本v1；semantic_slot_values是本closure中完整
`{slot,type_identity,value_identity}`；output_values为两个有序root的`{kind,type_identity,
value_identity}`。这两个局部identity**不含**projection physical ref/identity、整个batch
role_code_identity、fileSHA、origin producer数字、source span或acquisition metadata；否则
Unit A的changed region仍会污染Unit B。完整proof identity则绑定真实projection ref/identity、
producer/extraction版本、exact global→local node/port bindings及所有facts，供审计/CAS。
local ID重映射必须由原endpoint的一一对应产生并与edges完整核对，不能藉此删helper。

projection/producer版本和identity既被proof完整绑定，又与局部语义identity明确分层。
proof identity使用 `SHA256(canonical typed identity core)`（即上述全部proof字段、identity
置空）。audit-only acquisition path/kind/inode/time没有新增schema字段，不进入局部
closure_identity/unit_construction_identity，也不能决定D。完整proof则必须exact绑定其
projection/evidence physical refs；这些既有authority envelope的SHA会间接保护其原audit
bytes，不声称完整证据identity与物理容器无关。不得以audit不属construction语义为由
忽略已record envelope/声明的完整性，亦不得重写旧audit。

#### 5.3.2 Unit comparison与direct drift refs

comparison增加`unit_comparisons`，按完整原batch order排列，每项固定
`{unit_id,variant,old_dependency_proof,current_dependency_proof,changed_C_regions,
old_intersection,current_intersection,dependency_change,proof_mismatch,
unit_disposition,identity}`。old_dependency_proof/current_dependency_proof固定为
`{ref,identity}`，ref是完整Unit proof exact member ref；各项
identity置空canonical hash。unit_disposition只可`current|direct-construction-drift|
context-only-proof-drift|encoding-only|nonconstruction-only|blocked`。

C role delta的changed regions补全为`{kind,old_endpoints,current_endpoints}`数组；kind=
`node-payload|dependency-edge|port-binding|entry-output|type-binding|intrinsic-contract|control`，
endpoints只允许上述role-* endpoints；Unit/slot/root的差异由对应unit comparison绑定，
不能给batch role graph混入任意Unit。旧/新node对应须由共同typedIR结构/port绑定唯一证明，
不能把global ID重新编号当成真实changed region；对应不唯一时BLOCKED。这是
完整old/current canonical role graph差集，不是任取一组nodes；原changed_bindings保留
exact source evidence。projection/global IDs改变而完整局部结构不变，不算C semantic region变。

CRegionRef固定`{role_id,index}`，只可C0/C1，index是对应role_delta.changed_regions中的
canonical序号。unit comparison.changed_C_regions为相关CRegionRef的完整集合。
old_intersection/current_intersection固定为有序`{region,endpoint,root_paths}`数组：region为
CRegionRef，endpoint属于相应侧Unit closure，root_paths为`{root_id,edge_indices}`数组；
edge_indices是该endpoint到所列root的完整相关dependency-edge子图（包括SCC/loop边），
不是选一条trace，也不枚举无限循环walk。交集/可达root/有限子图均独立重算，不能填affected。
dependency_change固定
`{old_closure_identity,current_closure_identity,old_unit_identity,current_unit_identity,
explaining_regions}`；explaining_regions为本comparison中CRegionRef的完整有序集合。
只要dependency拓扑、type、output port或semantic root value变，都必须找出完整解释链；
仅有global role identity不等不够，无法解释的变化必须blocked。
proof_mismatch为`{old_proof,current_proof,exact_equal,differences}`；differences由strict
schema-owned reader穷尽列出，并绑定其old/current typed字段与exact source/projection refs。
differences为有序`{kind,slot,logical_dependency_path,role_id,old_value_identity,
current_value_identity,evidence_refs}`。kind只可`construction-dependency-reference|
construction-payload|envelope-encoding|nonconstruction-reference`：dependency-reference仅
logical_dependency_path非null，且由旧/新declared inventory唯一定位；payload仅slot非null，
必须是SlotKeyV1；encoding的slot/path/role_id全null；nonconstruction-reference的role_id
必须V/L/E/O、slot/path全null。两个value identity和exact evidence refs必须完整；不能使用
自由错误文本/JSON pointer推断影响。其它字段组合拒绝。

recovery direct drift记录不再只用`{comparison,role_id}`。每个direct Unit必有
`unit_dependency`=`{comparison,unit_comparison_identity,old_proof,current_proof,
affected_regions}`：old/current proof分别是该Unit dependency proof exact refs，affected_regions
必须是本Unit交集/解释链中CRegionRef的完整集合，不能自由只选一个region。原role_delta_refs只作分类导航，不能独立授权D。
缺Unit proof、错Unit/variant、extra/missing dependency或不完整交集使整次recovery BLOCKED。

canonical encoding统一Go typed struct `json.Marshal`，identity字段置空，无缩进/末尾LF；
只有声明为集合的数组按role/symbol/logical ID排序，Unit/token/child/slot/CFG/field顺序保留。
相同inputs/规则/epoch必有相同identity；无clock、PID、acquisition path、随机map顺序。
完整physical members仍受Files/SHA/size与exact ZIP current-check约束，不提供dual-canonical。

## 6. Historical v1/v2 projection与non-construction变化

旧version1/2 frozen proof/recipe先由原strict reader验证；优先级继续是 verified archive
> surviving exact repo bytes > exact committed supplemental authority > BLOCKED。
每次resolve root dependency都验证declared path/SHA/nullable size/role/version、invocation/
package/Unit和known conflicting claims。callee若只由旧archive inventory绑定，则只可消费
那个exact inventory member，不得借同SHA或currentfile凭空升级为新的supplemental声明。既有五份authorities原样使用；不重新record，不改K/A/S。
旧supplied bytes不注入ZIP files map；material origin/evidence独立记录。

新extractor从这些实际old sources构造projection；旧legacy缺role字段时仅由code-owned
versioned extraction规则产生derived view，不在旧对象补字段。两侧使用同一extraction
id/version/policy；不同版本不可直接比较，必须另行design/review一个显式bridge，本版不接受。
旧缺必要callee/type/dispatch/source/build事实就BLOCKED；“已经采用五份bytes”不承诺闭包足够。

旧mixed wrapper的structural extraction支持第4节closed port/guard grammar，必须逐node
coverage。`approvedResponseInputs`名字、某个SHA或真实batch不参与规则。无法提取的其它
locale/旧代码形态也BLOCKED。禁止用今日纯helper body补旧wrapper缺失语义。

Non-C changes不是自动compatible。comparison只允许以下固定witness类别：

| witness contract | 可接受条件；其它一律blocked |
| --- | --- |
| `role-ir-equality/v1` | 比较完整canonical RoleCodeCore：每个closed payload（尤其unary/binary operator）、类型/results、ordered operands、CFG与完整edges均相等；source/AST不能替代缺失语义；source envelope不同只作证据delta |
| `inactive-guard-route/v1` | 相同原semantic domain的exact immutablefact证明新增/删除region不可达；完整解析该region及effect，证明无alias/global/C数据污染；当前successor自己的active V/L gates仍全部执行 |
| `canonical-envelope-compatibility/v1` | E变化限现有明确的Store-readable/Deflate5-canonical与固定JSON typed envelope合同；验证双方inventory/CRC/logical bytes、当前canonical重建；不表示旧ZIP CURRENT，不支持其它encoder变化 |
| `pure-wiring-equivalence/v1` | O只有第5节透明helper展开/port连接变化；成功路径C数据、required gates、顺序/effects满足第4.3节完整义务 |
| `typed-construction-equality/v1` | payload_check必须证明完整原semantic_context与source-owned typed decoder payload相等，不比较任意RawMessage外观 |
| `closed-role-coverage/v1` | coverage_check证明所有source declarations、call targets、nodes/edges及shared regions完整，且unresolved为空 |
| `build-variant-invariance/v1` | 旧缺build facts时，只能证明支持variants的完整绑定/IR/effects相同；任何差异或无法覆盖均拒绝 |

这些proof固定结构为`{schema,contract_id,contract_version,old_role,current_role,
source_bindings,obligations,identity}`；schema=`go-learning/role-separation-witness/v1`。
source_bindings使用projection中的exactrefs；obligations数组元素固定
`{kind,nodes,edges,ports,facts,result}`，kind仅
`closed-ir-equal|inactive-region|no-cross-role-write|gate-dominates-success|exact-logical-payload|
canonical-current-envelope|same-port-wiring|same-semantic-context|same-typed-decoder|
complete-node-coverage|complete-dispatch-targets|separated-shared-regions|same-build-variants`，
result只能`pass|blocked`，证明由固定算法重算。
old_role/current_role为有序 `{role_id,code_identity}`数组，绑定对应projection中实际view；
仅build-variant witness可为空并改以完整source_bindings/variant facts证明，不引用projection
自身identity，避免hash循环。其它witness必须覆盖其所证明的全部roles。
不接受人手填“safe”、uninterpreted callback或只看一次current validation PASS。
`proof_ref`/variant_proof/separation_witness统一为exactmemberref：separation只可指向具有
`no-cross-role-write`和完整region coverage的witness；variant只可指向封闭旧build facts
或各支持variant相同IR的证明。不得用claim字符串代替可重算proof。

V/L predicate仅operator由`equal`变`not-equal`也使role core不等，不能获得role-ir-equality/v1；
AST/source refs再完整也不能把它判相等。语义未完整编码的core无资格申请任何equality witness。
V/L active规则变化、required gate减少、接受条件/attempt/provenance变化且不满足上述proof，
即使新机器检查恰好PASS也BLOCKED。Non-C role没有decl不表示其角色被建设性忽略；
legacy版本absence必须在bridge中作为真实旧事实记录，且有相应inactive/equality proof。
新recovery invocation的L gates不能从旧版本absence继承豁免。

对本案，新增recovery guard可尝试由旧manifest的`transport_recovery=null`生成inactive
route witness；这只分离其non-C影响，不生成D，不宣布整个wrapper兼容。旧/新所有其它
V/L/E/O delta仍需完整分类。任何一项不闭合，整个preflight BLOCKED、authority=null。

## 7. Drift predicate、encoding与D/R（逐Unit exact-set证明）

先完成原historical integrity、language/source/package/protection currentness、fresh完整
construction和角色闭包。原exact installed comparison E(u)保持，不以role相等让旧proof
重新CURRENT。版本、fileSHA或projection证据identity变化不单独形成D。

§5.3.1的两侧Unit proof必须覆盖**完整原batch**，先独立重算每个root及其least fixed-point
closure，再计算C region差集与逐Unit交集，不从batch-levelrolehash反推影响范围：

```text
Touched(u) = exact(old_unit_closure(u) ∩ changed_old_C_regions)
          ∪ exact(current_unit_closure(u) ∩ changed_current_C_regions)
          ∪ fully_explained_unit_dependency_change(u)

eligible(u) = 原v1所有准入/当前性/active-validation gates PASS
           AND E(u)
           AND 两侧Unit dependency proof完整且使用相同comparison contract
           AND nonconstruction_checks_all_PASS
           AND original_semantic_context_and_typed_decoder_payload_equal
           AND Touched(u) 非空且每条影响到output root的路径均可解释
           AND every_C_delta_uniquely_attributed
D = original_batch_stable_order({u | eligible(u)})
R = existing_ContextClosure_B(D)
```

交集包括node payload/operator、output port、type、control与dependency edge等完整影响，
不是仅比node名字。fully_explained_unit_dependency_change必须列出对应C regions和到该Unit
root的完整边/slot路径；单纯Unit proof physical identity不同不算。给定identities/refs/graph，
交集和D必须唯一；无法证明某Unit是否依赖变化时**整次recovery BLOCKED**，不能猜进/猜出D。

这是原“每Unit的已分类construction dependency drift”的缺失proof合同补全，不是换D定义：
旧草案只有batch角色delta，没有可验证地把它绑定到Unit，因此其D exact-set主张未成立。
现在把原predicate的`changed_C0_or_C1_closed_role_code`明确落实为该Unit的root/closure证明。
R仍只按原installed batch的context/coupling edges扩展；任何非空合法D在本v1 recovery
family都必须闭合为完整有序`U(B)`，不是新的per-Unit recovery范围。R edges不能用来决定D，
per-Unit construction edges也不能用来缩R。D为空时仍无可recordauthority。

本successor只补强`construction-dependency-drift` attribution，不以payload变化放宽原
semantic/decoder invariant。payload改变但角色claims未变，或变化没有独立原codec/
representation合同证明，仍UNCLASSIFIED，不在本合同造新inverse/target mapping。
producer升级、物理拆函数或source metadata变化本身不能形成construction-contract drift。

**两Unit判例。** A/B均为approved，E(A)=E(B)=true；comparison穷尽证明changed C region X
只在A的old/current dependency closure或其dependency-change解释链中，B的完整局部closure
identity和semantic/output root identity未变。则唯一direct-set是`D={A}`，绝不是`{A,B}`；
若无法证明B与X不相交则整次BLOCKED，不假报D。global C0/C1 identity变化不能替代这项证明。

若B的proof exact mismatch仅来自其仍冻结了同一implementation **whole-file ref**，而
该ref的完整变化已由comparison证明发生在X且不在B closure，则B标记为
`context-only-proof-drift`：它不是direct drift，也**不变成CURRENT**。对同一原batch，既有
context closure才把B纳入R，仍须新full-RGeneration；不能carry旧B candidate。这里没有
用R耦合证明D，也没有用另一Unit的drift掩盖unknown：B的所有mismatch字段必须exact归因。

B若还有独立JSON/envelope编码差异、未支持non-C变化或其它未解释字段，则保留其原
ENCODING_ONLY/UNCLASSIFIED blocker，整次不可record。只知道两者“proof不同”与A的rolehash
变化不够；必须具备完整逐字段mismatch解释和两侧Unit闭包证明。

| 完整比较结果 | preflight / authority |
| --- | --- |
| exact installed comparison全PASS且其它gate全PASS | CURRENT，D/R空，authority=null；禁止冗余record |
| 只有encoding/JSON/ZIP差异，没有任何Unit的C dependency变化 | ENCODING_ONLY_MISMATCH → BLOCKED，authority=null |
| 仅已分离V/L/O、source外观或整个C闭包未变，无合法D | NON_CONSTRUCTION_ONLY_MISMATCH → BLOCKED，authority=null |
| Unit有完整Touched路径且全部准入PASS | 该Unit进入D；随后按原context graph取得完整R |
| sibling的whole-file proof ref drift仅对应其closure外的已完整分类C变化 | 不进入D、不授CURRENT；若同batch已有合法D，仅由既有context closure纳入R |
| true mixed/shared/unknown、任一Unit依赖无法证明、覆盖缺项、未支持non-C变化 | 整次UNCLASSIFIED/BLOCKED，authority=null，禁止partialauthority |

漏掉实际helper、跨package codec、control/type/alias依赖均是incomplete proof，不是“该Unit
没有依赖”。C变化漏出D或non-C变化误入D时，重算的Unit comparison/closure必须reject。

## 8. Lifecycle、publication、CAS与旧版恢复

未来grammar不变，继续原 `retranslation transport-recovery` preflight/record/check 与
recovery-specific export/import/process/retry/Snapshot。**不新增role/projection override flags。**
version由canonical writer固定选3，不由caller选；旧authority readers按schema显式dispatch。

新增不可变members在同一个recovery authority write-once目录发布：

```text
roles/old-projection.json
roles/current-projection.json
roles/comparison.json
roles/unit-dependencies/<unit-id>/old.json
roles/unit-dependencies/<unit-id>/current.json
roles/witnesses/<witness-identity>.json
roles/sources/<source-sha256>.bin
construction.json
```

source member只是原bytes的exact transport副本，不是新的historical采用权；origin仍指向
原archive/surviving/supplemental declaration。source和witness名称由内容identity计算；
root/leaf link仍是exact ref的 `go-learning/transport-recovery-link/v1`，目标schema由新reader
显式dispatch；旧link/owner不改写，不能因新schema重建root或跳过排他性。
role projection内部仅以member refs引用，避免publication path自引用。全部进入authority
members/inventory、identity和publication sealed transaction；旧directory一字不改。

| consumer | 必须做的工作 |
| --- | --- |
| preflight | exact旧graph → resolve bytes → 独立old/currentprojection与closure → role comparison/两侧逐Unit dependency proofs/exact D → 原R及language/active-validation gates；诊断不发布 |
| record | fresh全部重建/CAS，固定schema v2 authority与所有members整体no-replace/sealed publication；expect identity仍是完整authorityidentity |
| check | read历史完整性与schema dispatch，freshcurrentprojection/analysis/epoch，保留原READY_FOR_EXPORT等状态/allowed-next mapping；从不自动执行next |
| successor export | full R finding-anchored新batch，authority显式exactref；batch ordinal沿现有max+1，不提前reserve；原task kind/manifestsemantics保留 |
| Generation Bundle | 冻结construction v2、完整两侧逐Unit dependency proofs及交集、所有role/projection/witness/source refs、原semantic full R及proof；`formal/transport-recovery/roles/`为独立namespace；Files/hash/inputidentity/canonicalDeflate5/current-check全部绑定 |
| import | 仍强制真实新execution object/evidence、exactoutput-origin全集、no-replace；不能复用旧raw/receipt；不得让roleproof代替modelprovenance |
| installed child process/retry | 从initialreceipt exactarchive重建package、construction与roleepoch；V/L完整执行；exactproof/recipe比较无fallback；retry仅失败Unit新invocation并保留同一recoverylineage |
| Snapshot | latestfull-R successor全部PASS且graph当前才可选；失败不回退旧B；outside-R按原current/A规则；re-QC由machine pending scope决定，不自动A/全R pending |

currentness只绑定C的producer identity不代表child只验证C。child的完整authority/role epoch
改变仍可使child stale；不能因它是V/L/O变化就在installed decoder处忽略。读取/审计旧
v1/v2继续使用旧schema精确语义；不把它们标为role-separated CURRENT。新能力仅通过v2
recovery authority + producer3 successor链进入正式当前路径。

supersession语义不扩张：same root/locale/package/唯一leaf、无installedchild才可显式
supersede；已installed且全R PASS的child再次stale，以该child为新predecessor重走合法准入。
已installed失败child不满足old-PASS不能偷换root。旧未installed authority/零outputchild
保留；版本升级不许可覆盖或复用其Generation output。

CAS新增绑定全部role sources/decl/closure、逐Unit output/slot/dependency proof与exact交集、
extraction implementation/policy、projection/witness/comparison refs、build/dispatchfacts、V/L/E/O identities与currentgates；同时保留
原receipt/archive/manifest/result/finalattempt/raw/candidate/validation、currentpackage、
完整latestselections、原reopen/Snapshot/finalization/A、root/leaf/pending、D/R等CAS。
任何preflight→record变化均拒绝。record/export各自pending schema v2 seal这些新members；旧版pending不能
升级/重新领用，先按旧owner readonly诊断。matching resume不得换extractionversion/flags/
identity；commit已exact证明时仅清理原事务，unknown状态不blind replay。

record前中断重做readonly preflight；record后未export只check原authority；export后仅继续
唯一child。Generation已完成未import且Bundle仍CURRENT时复用真实可信newinvocation输出；
import后走原process/retry，不能reimport。validated后仅进入Snapshot preflight。
分析器/rolepolicy再次变化使epoch stale，按原phase/supersession边界恢复，不以“算法升级”
重演任何已完成invocation或删除历史evidence。

## 9. Existing historical authorities

现有 `historical-construction-bytes/v1` **不需要successor**，也不修改它的supported
producer1/2范围来偷偷兼容3。其byte identity、scope、acquisitionaudit和singledeclaration
准入不变；新extractor在消费侧读取，不能重新解释它为roleauthority。

future producer3 Bundle强制冻结其完整sourceclosure和projection事实；旧formalization
family不因此开放任意producer3文件注入。若未来producer3历史事实损坏/缺失，先BLOCKED，
不能用本设计给missingfacts补新权限。本轮不设计另一个adoptionfamily。

## 10. CLI交接与first-real-use

| Formal Command Contract v3 | 本successor的合同 |
| --- | --- |
| COMMAND | 原 `retranslation transport-recovery`；语法完全沿原设计，无新flag |
| OPERATION | readonly角色闭包/漂移preflight；expectidentity的write-oncerecord；exactauthoritycheck |
| INPUT_KIND | 显式installedpredecessor及完整registeredpackage；或exactauthorityref；不接受externalrole/source/proof |
| INPUT_PATH_CONTRACT | formalrefs全部repository-relative+exactSHA；source只从正式graph resolver取得，无 `/tmp` fallback |
| MUTATION_EXPECTATION | preflight/check=none；record仅新增既有recoveryfamily的v2write-oncedirectory与原link/pending；不修改前任、reopen、Snapshot |
| IDEMPOTENCE | typedcanonical身份deterministic；duplicate record拒绝；resume仅原sealedtransaction；不因projection重算生成新权利 |
| CURRENTNESS_CHECK | 原准入 + 两侧完整closure/roleprojection/bridge/non-Cwitness + exactinstalledproof + 当前epoch；不能由Crolehash替代 |
| PRIOR_OR_CAS | 第8节完整原CAS与新role/extraction/closure事实；root/leaf/pending排他与no-replace保持 |
| SUCCESS_STOP | preflight审阅计划；record/check依原allowed-next；本设计不执行nextoperation |
| FAILURE_STOP | typedBLOCKED，无可recordidentity/authority；保留现场，不force/忽略/partialscope |
| RECOVERY_REPLAY | 按原phase/owner/version恢复；未知pending保持，不能用新schema领用旧transaction |

### 10.1 Closed stdout/result union（Finding 3 修订）

所有有效mode的domain结果使用 `go-learning/transport-recovery-plan/v2` closed tagged union。
mode仍为`preflight|record|check`；新增operation=`preflight|record|resume|check`，只由真实
argv决定（resume仍是原record --resume flag，不是新command）。只允许
`(preflight,preflight)`、`(record,record)`、`(record,resume)`、`(check,check)`。
unknown discriminator、字段缺项/额外字段、禁止组合或下表语义不一致均是INVALID RESULT。

preflight/record/resume固定字段顺序：

```text
schema, mode, operation, status, locale, content_package, predecessor_batch,
direct_drift_units, recovery_units, direct_drift_count, recovery_unit_count,
blockers, role_assessment, planned_authority, identity, path, authority,
transaction, recorded, mutation
```

check固定字段顺序：

```text
schema, mode, operation, status, authority, child_manifest, child_batch, unit_id,
attempt, allowed_next_operation, historical_integrity, current_action_eligible,
blockers, role_assessment, transaction, recorded, mutation
```

所有字段必填；nullable只用于不适用/未能验证的值，不能用omission隐藏未知状态。
check保留原READY_FOR_EXPORT/EXPORTED/IMPORTED/RESTORE_FAILED/VALIDATION_FAILED/VALIDATED/
STALE/BLOCKED与allowed-next mapping；不适用的child_batch/unit_id/attempt为null。
flags/usage在domain进入前失败可以无JSON、非零退出，但绝无可供handoff的结果。

**CommittedAuthorityRefV2**固定`{path,sha256,size,identity}`：

- path是canonical repository-relative
  `data/translation-transport-recoveries/<locale>/<typed-identity>/authority.json`；
- sha256对**重新打开验证过的actual committed file bytes**求SHA256，不是planned serialization；
- size是同一actual file的实测int64 byte size，非负，不能缺失或由预计长度替代；
- identity由实际strict decoded authority重算typed identity，并与path、seal、root/leaf link的
  exact authority binding一致；actual完整members/inventory也必须重新验证。

只有authority directory存在不表示commit：还须原record contract规定的唯一root/
supersession link commit point已exact、durable发布并与本transaction一致。下表中non-null
`authority`都是这种**已重新验证的committed ref**。不能把planned_authority转型/改标签即用。
planned_authority是完整计划对象，identity/path可在READY前算出；它不是actual file physical
ref，不能替代发布后SHA/size。record成功后planned_authority=null。

`transaction`为nullable closed对象
`{schema,path,pending_state,pending_ref,owner,commit_ref}`：

- schema为本operation预期的record/export transaction v2 family，不声称损坏/旧版pending
  已符合它；path是原规则唯一计算的repository-relative pending path，禁止caller指定；
- pending_state=`present|absent|unreadable|unresolved`，如实报告最终观察；unresolved仅用于
  必要graph事实丢失而连expected path也无法安全计算的unknown恢复，此时path=null，绝不
  猜path或扫描latest；其它三态path必须non-null。pending_ref是安全bounded
  regular/non-symlink读取到的actual pending `{path,sha256,size}`。present且可安全读取时
  必须non-null；absent/unreadable/unresolved时null，不能提供想象的journal digest；
- owner仅在strict schema/seal/原args/CAS owner验证通过时non-null；否则null；
- commit_ref仅在actual原commit link完整重验后non-null，是其exact `{path,sha256,size}`。
  未commit/无法确认时null；不是authority ref，也不能替代顶层authority；
- 没有本次transaction或待报告的pending/progress时transaction=null。成功完成cleanup且
  pending不存在的RECORDED必须null。失败时只要本次曾进入publication、或目标pending/
  commit处于恢复/不确定状态，transaction必须non-null；可安全计算的expected path必须
  保留，否则显式unresolved/null，绝不把未知误报absent或伪造存在。

readonly mutation固定none，recorded固定false（不表示库里没有既有authority）。record/resume：

- `none`：本次未写入，且已证明没有本owner待恢复publication进度；如fresh CAS拒绝。
- `known`：本owner进度的所有已写/未写目标与commit状态均可exact归类，可能是已确认的
  partial pre-commit，也可能已commit而cleanup失败；**不是成功或CURRENT的同义词**。
- `unknown`：不能证明目标写入/commit/owner完整结果，包括损坏seal、不可读/冲突target、
  无法确认的既有pending；即使本次resume尚未新写也不能声称恢复现场确定无mutation。

recorded在record/resume结果中仅当本transaction目标已完整验证commit时为true；readonly
固定false。command exit/status、recorded事实与next-operation资格不得混淆。

| union case | exit/status | planned_authority；legacy identity/path | authority | transaction / recorded / mutation |
| --- | --- | --- | --- | --- |
| readonly preflight READY | 0 / READY | 完整计划及planned identity/path | null | null / false / none |
| readonly preflight CURRENT | 0 / CURRENT，D/R空 | 全null | null | null / false / none |
| readonly preflight BLOCKED | 非零 / BLOCKED，无可record identity | 全null | null | 观察到pending时报告transaction，否则null；false / none |
| record在mutation前失败 | 非零 / BLOCKED | 全null | null | 无owned进度时null；false / none；若发现未知pending按unknown case处理 |
| record已知partial、尚未commit | 非零 / BLOCKED | 全null | null | non-null且owner/已知phase可验证，commit_ref=null；false / known |
| record已commit、cleanup失败 | 非零 / BLOCKED | 全null | 必须为actual committed ref | non-null；commit_ref必须non-null；true / known；pending_state/ref按真实剩余状态 |
| record无法确认commit/进度 | 非零 / BLOCKED | 全null | null | non-null，未知owner/commit_ref为null；false / unknown；保留现场 |
| record成功 | 0 / RECORDED | planned=null；identity/path仅作为actual authority对应字段的镜像 | 必须non-null且四字段齐全、已post-commit验证 | null，pending已证明不存在；true / known |
| resume完成原sealed transaction或确认commit后完成cleanup | 0 / RECORDED，operation=resume | 与成功record相同 | 原transaction的同一actual committed ref，不重新分配identity/重写 | null；true / known |
| resume在无匹配pending且无本owner进度时拒绝 | 非零 / BLOCKED | 全null | null | 无target进度则null；false / none；不得重复record接管 |
| resume失败/阻塞且有已知partial或已commit进度 | 非零 / BLOCKED | 全null | 仅完整证明commit时non-null | non-null；分别复用known pre-commit或known post-commit行；recorded与commit一致 |
| resume的owner/seal/结果不能确认 | 非零 / BLOCKED | 全null | null | non-null；false / unknown；禁止blind replay |
| readonly check | 按原状态/exit规则 | 不存在计划字段 | 仅exact历史commit和authority都验证后non-null；无效/未commit则null | pending存在则如实报告transaction，否则null；false / none |

成功record/resume必须重新验证实际authority及commit后再构造结果；缺sha256、size、
错path/typedidentity或仅返回planned值，即使status写RECORDED也属于invalid result。
若输出编码/写stdout失败，命令不得伪造未commit；实际state/pending仍按原recovery诊断，
不能因未收到成功JSON重复record。已证明commit但cleanup失败必须返回上表known-commit
信息并停止；若pending仍在，只有匹配sealed resume可清理。若pending已移除但最终cleanup
同步失败，如实报告absent及已知commit，先readonly核对，不凭空重造pending来resume。

resume发现原exact commit已经完成时，返回的是**原seal/root-link所指的原authority实际
bytes ref**；不按本次重新生成的planned对象替换。完整cleanup成功才返回RECORDED。
confirm-commit不足以绕过仍未清理的pending、currentness或后续check。

**Downstream handoff。** 成功record/resume唯一正式ref是`authority.path/sha256/size/identity`。
保留并交接这四字段，先按原contract check，再使用该exactref export；不使用
planned_authority、legacy identity/path镜像或transaction.commit_ref作替代。
CLI grammar不新增size/identity flags：已有`--authority`/`--authority-sha256`（export使用
对应transport-recovery flags）从该committed ref投影取得，consumer必须从actual bytes
重算size/typedidentity，并验证完整commit链/currentness。domain/交接对象含四字段时必须
逐项匹配。check可在STALE时报告已验证的historical committed ref，但allowed-next=null；
pending或BLOCKED时同样不得export。planned ref无actual commit、缺SHA/size、错四字段或
只具备未commitdirectory时，typed handoff和check/export都必须拒绝。若之后actual commit
已完整存在，只能依据其重新验证的真实state取得权限，不能声称计划本身曾授权行动。

role_assessment保持nullable
`{old_projection,current_projection,comparison_identity,construction_changed_roles,
nonconstruction_status,coverage_status}`，另以comparison exact绑定完整unit_comparisons。
projection refs未发布时只表示计划member；changedroles只可C0/C1；statuses=`pass|blocked`。
分析未闭合可null，BLOCKED不得提供新的recordidentity，CURRENT无authority。
新增NON_CONSTRUCTION_ONLY_MISMATCH，unknown/mixed保持UNCLASSIFIED；check不自动执行next。


因此 `NEW_OR_MODIFIED_FORMAL_CLI=yes`、`NEW_OR_MODIFIED_FORMAL_CURRENTNESS=yes`，
即使flags不变也必须 `FIRST_REAL_USE_REVERIFY_REQUIRED=yes`。实现后重新按顺序检查
canonicaldocs → currentimplementation → true `run(rawArgv)` contracttests，再完成维护者
first-real-use gate；本设计、旧helpertests和历史PASS均不授权真实record。

## 11. Fail-closed矩阵与后续实现验收

| 情况 | 必须结果 |
| --- | --- |
| true mixed declaration changed，data/control/alias不可分 | UNCLASSIFIED；不得把wholewrapper加入C |
| unknownrole/declaration changed | UNCLASSIFIED；不能按filename/functionname推职责 |
| roleclosure缺helper/type/value/method/edge | BLOCKED；coverage不可partial |
| sharedhelper变化不可分离 | UNCLASSIFIED，即使其中一个caller是C |
| exacthistoricalbytes缺失或claimconflict | BLOCKED；不补currentbytes/外部fallback |
| historicalextractor/schema/versionunknown | BLOCKED，不尝试最近reader |
| currentprojection无法fresh重建 | BLOCKED，不用上次缓存PASS |
| encoding-only | ENCODING_ONLY_MISMATCH，authority=null |
| exactoldproof仍current且其它gatePASS | CURRENT；禁止冗余record |
| constructionpayload改变但roleclaims未变/解释不完整 | UNCLASSIFIED；不能隐藏realconstructiondelta |
| roledeclaration排除calledhelper | closure发现实际call/def-use，BLOCKED |
| cross-packagecodec或dispatchwitness缺失 | BLOCKED；普通Pagedecoder不能兜底 |
| preflight→record任意CAS变化 | CASreject，noauthoritypublication |
| old/current不同extractionversion直接比 | BLOCKED；本版没有隐式跨版本equivalence |
| filebytes变但roleclosure均相同 | 证据epoch变，不能形成D；根据变化归类encoding-only或nonconstruction-onlyBLOCKED |
| non-Crole变化误报D | comparison/directrole-ref校验reject，不能record |
| realC变化漏出D | perUnitdependencyedge/coverage完整性reject；不得假报CURRENT |
| C内部偷偷调用V/L/E/O、unknowncallback或mutableglobal | effect/dispatchclosureBLOCKED |
| oldwrapperprojection试图删除errorguard而无portproof | BLOCKED；guard/成功出口覆盖必须完整 |
| version迁移/物理拆函数本身被用来制造D | commonIR/bridge检查reject；不得比较不同schemahash不等来授权 |
| 其它非构造activeverification变更无支持witness | BLOCKED，即使currentvalidation刚好PASS |
| 忽略旧scope或缩R为directD | 原closuregate拒绝；不能选择mixed old/newR |

| operands/type相同，仅binary operator由equal改为not-equal，roleidentity却不变 | INVALID/BLOCKED；正确canonicalpayload必须不同、identity必须变，source/AST不能补缺operator |
| active V/L predicate的operator变化 | role-ir-equality/v1必须失败；不得把predicate改动宣称equivalent |
| 其它unary/binary/call/range等operation语义未完整编码 | projection BLOCKED；不能丢payload后继续 |
| A/B frozenproof均mismatch，仅A闭包含changedCregion且B局部root/closure未变 | exact D只能为{A}；whole-fileSHA不能把B推入D；B仍不自动CURRENT |
| 无法证明某Unit是否依赖changedCnode/port | whole recovery BLOCKED；不猜进/猜出D |
| Unitproof漏掉实际helper/call/data/control/type/alias依赖 | least-fixed-point重算发现不完整，BLOCKED |
| Unitproof只绑定batchrolehash/projectionhash而无output-root路径 | INVALID/BLOCKED；缺逐Unit证据 |
| record/resume声称成功却缺committed authority实际SHA/size | invalid result，禁止downstreamhandoff |
| planned_authority或planned identity/path被当committedref | typedhandoff拒绝；无actualcommit/错ref时check/export拒绝，不能依计划授予行动 |


未来tests必须包括：两个纯Cdeclaration的cross-filehelper闭包；source-ownedinterfacecodec
binding；sharedhelper不可分；偷偷调用executionvalidator；artifactencoder与activegate变化；
旧v1/v2mixedwrapper完整提取及negative；仅file/JSON变化；新producer3full-Rrecovery、retry、
Snapshot与raworiginlineage；equal/not-equal、add/subtract、greater/less的IR及activeV/L
witness拒绝；两approvedUnit仅一Unit闭包含changedregion的exact D；success/known/unknown/
resume的committedref union；trueCLI的preflight/record/check/resume。只跑相应targetedtests，
不借实现本合同重跑全部历史workflow。本轮design阶段不执行Gotests。

## 12. 真实zh-CN非执行性walkthrough

已知事实：locale=`zh-CN`，package=`learn-docs-v1`，predecessor=`chatgpt-zh-CN-064`，
原Snapshot=`learn-docs-v1-qc-023`；历史32/32validationPASS；直接目标Unit：
`b49d8e873a0f66f6803077eb534583a155d2bd209a032d7c00fba8fea856da5c`。
旧approvedproofstale、historicalbytes已正式采用，不重做采用或旧invocation。

1. future新reader读取原initialreceipt/exactarchive及既有supplementalbytes，分别分析old/
   currentgraph。现有五份blob足够与否由完整callee/type/dispatchclosure机器决定，不能预设。
2. `approvedResponseInputs`仍是Ocomposite：`verifyRecoveryInstalledExecution`→L/V；
   `importedApprovedIntervalSelection`→V/L与C0factboundary；revision/Surfaceprojection→C0/C1；
   frozenproof/constructionexactchecks→V；recoverymembers与evidenceassembly→E/O。
   旧manifest没有`transport_recovery`，可尝试证明新增recoveryguards的inactive-route，
   但该witness本身绝不进入D。
3. 真正需要检验的constructiondelta是**decodercontext的package/Snapshot选择**：
   `prepareSnapshotRevisionOutputs`从旧membershippredecessor evidencecatalog路径，转为调用
   `snapshotRevisionEvidenceCatalog`后使用原revision manifest的`PreviousSnapshotID`所冻结的
   完整package。前者callsite在`candidate_snapshot_outputs.go`，后者真实定义在
   `candidate_snapshot_membership.go`；不能仅分析同文件或虚构旧helper声明。
   它选择decoder要消费的完整原semanticcontext，归入C0的`context/from-language-facts`。
   machine必须从old/currentexactsource及正式package/Snapshotfacts证明此dataflow变化；
   其它constructiondelta也必须逐一完整归因，不能仅凭这个已知修复宣告wholefile安全。
4. future implementation必须为完整32Unit两侧各自重建§5.3.1 proof，从其output roots
   经typed slots、node/output ports和完整transitive edges确定闭包。必须严格证明目标Unit的
   C closure与已分类changed construction region相交/发生可解释dependency change，
   且其它31Unit的局部closure与semantic roots不具有这种变化，故不属于D；若任一Unit的
   归属不可证明则whole recovery BLOCKED。不能用同一Go文件SHA或batch role delta替代。
   只有这些per-Unit证据及原language/typeddecoder invariants、完整non-Cwitness/CAS/
   activevalidation全部PASS，才先得到conditional `D=1`；**之后**existingcontext/coupling
   closure得到 `R=32`。不将其余31Unit误标directdrift，也不允许carry旧candidate代替新full-R。
   因而future expected D=1/R=32仍是conditional walkthrough，绝非本轮READY声明。
5. 若缺historicalcallee/buildfacts、wrapper分离失败、activeV/L变化无法证明或其它delta
   unclassified，继续BLOCKED；本设计不承诺该真实case已可READY，也不执行preflight来验证。

没有预填successorbatch/reopen/authorityidentity、BundleSHA、output或Snapshot。下一步仅是
该contract的Maintainerreview；获准implementation后仍须独立first-real-use gate。
**停止于implementation/first-real-use/真实recovery之前；不record、不export、不Generation、
不创建qc024或QC。**
