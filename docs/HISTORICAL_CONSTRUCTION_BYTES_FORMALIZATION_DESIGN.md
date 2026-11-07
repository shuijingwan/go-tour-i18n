# Historical construction bytes formalization 合同 v1

状态：**design complete；尚未实现，不授权真实采用或 recovery。** 本文定义独立的
supplemental historical evidence family，不复用 legacy Generation provenance authority，
不改变 [installed transport recovery](INSTALLED_DERIVED_TRANSPORT_STALE_RECOVERY_DESIGN.md)
的恢复资格或漂移判定。实现完成后仍须独立通过
[formal CLI first-real-use gate](WORKFLOW_HANDOFF.md#formal-command-contract-v3)。

## 声明范围与真实性边界

唯一 capability：`attest-exact-declared-historical-construction-bytes`。
它只证明：**发布的完整 byte sequence 与一个已验证 historical formal graph 中的既有
dependency declaration exact 一致。** 它不证明这些 bytes 原本位于旧 archive、来源
artifact 原本是 formal artifact、当前代码等价于旧代码，或任何语言、Generation、
representation、QC、finding、candidate eligibility 权限。

```text
OLD_ARCHIVE_MUTATED=no
OLD_RECEIPT_MUTATED=no
OLD_PROOF_MUTATED=no
```

v1 每个 authority 只绑定一个 installed initial invocation、一个 approved Unit、一个
logical implementation dependency declaration。共享相同 bytes 的其它 declaration
必须独立验证和 record；可以共享 blob，不能共享 scope 授权。此处 Unit selector 只定位
历史声明，不裁剪 transport recovery 的 D/R；原 recovery 仍要求完整原 batch closure。

`SHA256(recovered bytes) == old declared SHA256` 证明 byte identity。acquisition provenance
只解释如何取得这些 bytes，不能反向提供 declaration、role、producer version 或 scope。
旧 SHA 仅出现在聊天、日志、inventory、patch 中不够；必须先验证旧正式 declaration。
不执行、编译、动态加载 recovered Go，也不以重新实现、patch 应用、normalization 或
等价行为生成 candidate。known/declared reconstructed、equivalent、normalized、patch-derived
来源一律拒绝，即使 caller 声称已“恢复”。

必须诚实区分机器能力：hash 不能识别一个人是否伪造了 backup 的来源叙述。真实的既有
完整文件副本和 acquisition evidence 由提供者如实交付、维护者核对；来源虚报是 provenance
违规，不能被 hash PASS 洗白。机器必须验证下述 byte/graph/path gates，不能声称 hash
认证了编辑器、Codex session 或文件创建时间。

## Historical declaration 准入

不从 error text、路径前缀、current refs、mtime 或最近一次 receipt 推断 old facts。
默认 preflight 的 scope 由 explicit locale/package/batch/Unit 定位；caller 的
`logical-dependency-path` 只能在已验证 producer inventory 内 exact 选择一项，不能声明 SHA。

同一次只读 inspection 必须完成：

1. 显式选择完整 registered package；不使用 implicit Tour。验证指定 batch 的 manifest、
   fixed `generation-imports/initial-manifest.json`、其 locale/batch/task/attempt/input identity，
   从 receipt 的 exact SHA 定位 canonical `data/generation-archives/<locale>/<sha>.zip`。
   不接受外部 archive/proof/receipt override。所有磁盘 refs regular、non-symlink、exact SHA/size。
2. strict transport decode，沿用 bounded 8192 files / 134217728 logical uncompressed bytes、
   path/CRC/inventory/hash/expected-output/input-identity 验证；历史 Store/Deflate 均可读，
   不要求旧 installed archive 重新成为 pending-export CURRENT。
3. manifest 与 archive 冻结 manifest exact 一致；该 Unit 在 immutable ordered scope 中，
   source/formal input、initial installed raw/import provenance 与 attempt=1 binding 自洽。
   原 initial receipt 的完整 expected output set 与实际安装的 initial raws 必须 exact，
   不能以缺失其它 initial output 的 partial install 作为正式 invocation。
   只验证该 initial invocation 的 immutable facts；后来的 result/current validation/retry
   selection 不进入 declaration key，不以 mutable latest result 追认旧声明。历史失败不是
   本 byte authority 的翻译修复对象；不把历史 PASS 推导为当前 PASS。
   不完整或互相矛盾的 invocation provenance 拒绝。初次 revision 或 recovery-origin invocation
   只用各自已有合法 reader；unknown task/proof version 拒绝，不扩展 legacy unknown facts。
4. 验证该 Unit 的 frozen approved proof/member SHA/size、strict schema/mode/identity 及
   selection/reopen/Snapshot/package/Unit binding。package authority 必须能沿正式 registry/
   membership lineage 精确投影为该 invocation 的历史 descriptor，不能用 same-ID current
   descriptor冒充。初始 import archive 中的固定 member 是声明容器，不接受 caller 自由 member。
5. 用 code-owned versioned producer reader 取得 logical path、old SHA、role 和 version。
   有冻结 construction member 时，以其 strict producer declaration 为主，并验证与 approved
   proof 对相同依赖的声明一致；没有该 member 的 legacy approved v1，只能按既有固定
   producer declaration reader 派生角色/version，并显式记 `role_binding_origin=versioned-reader`。
   不能声称旧 JSON 有本不存在的 role/version 字段，也不能从今日文件填旧 facts。
6. v1 只接受 producer 中的 `approved-construction-implementation` 或
   `approved-construction-reader` implementation refs。ordinary source、日志、任意 JSON/docs、
   unknown role、同名但未声明的 Go file 不能注入此 family。未知 producer version、duplicate
   declaration、无法唯一归因的 selector、scope/role 冲突均 BLOCKED。
7. `declared_historical_size` 有明确旧声明时取其原值；否则为 JSON `null`。
   不用当前文件或 acquisition inventory 的 size 回填旧事实。recovered size 必须实测；
   SHA exact 且若旧 size 非 null 时也 exact。得到 byte material 后才能继续独立的旧
   construction 派生；本准入不要求先完成那个正因缺此 bytes 而阻塞的 construction 重建。

历史 graph 验证与 current transport eligibility 分离：这里不重新授予原 reopen、不检查
译文可否进入新 Snapshot，也不以当前 source/glossary 改变否认已验证的历史 byte declaration。
但缺失/损坏旧 graph、历史 package 不可合法投影、无法验证声明仍必须 BLOCKED。

## Acquisition boundary

允许 explicit **absolute external local regular file** 作为 adoption 输入；本版本不另设
repository staging/import mode。`--source` 与 `--acquisition-evidence` 必须位于 repository
之外，全部路径组件 non-symlink，禁止 `..`、隐式 search、URL、stdin、FIFO/device/socket、
directory。采用 OS path，禁止把 shell expansion 当 parser 功能。它们不是 formal refs。

source 必须是已存在的完整历史副本；v1 `source-kind` 为
`codex-artifact|editor-history|tmp-copy|backup`，其它值拒绝。evidence 是明确项目相关的
既有 acquisition artifact，例如保存该副本来源的 diagnosis/backup inventory；不能仅写
一句“我重建了”。它不替代 formal declaration。提供者提交事实来源、维护者核对完整副本
与 evidence 的关系；CLI 记录 source-kind、原 source/evidence absolute path、各 SHA/size 和
安全 file observation，保存 evidence 原 bytes。source-kind 字符串本身不赋予信任。

source 限 4 MiB，evidence 限 32 MiB，authority audit metadata 限 64 KiB；均 bounded read。
source/evidence 打开时使用 no-follow、fstat，并在读取前后确认同一 regular inode、size、
mode、link count、mtime/ctime observations、exact bytes/hash。v1 拒绝 link count != 1，
也拒绝与当前 logical dependency file 同 inode 的 alias。source 不得是 current worktree
文件或其 hardlink；该路径仍 exact 等于 old SHA 时可走既有 surviving-reader，不需要采用。
不能先把 current code/patch 复制到 `/tmp` 再谎报历史副本。

acquisition paths/metadata 必须是可 round-trip 的 UTF-8，无 NUL/控制字符；bytes 本体不作
UTF-8 或换行重编码。audit observation 的 device/inode/link_count 为 uint64，mode 为 uint32，
size 为非负 int64，mtime_ns/ctime_ns 为 int64；记录的是文件 observation，不声称旧 formal
source 原本有相同 mode/inode/time，也不加入 authority identity。

source 和 evidence 的路径只保存在 **audit metadata**，不作为 persisted formal ref，也不
允许后续 reader/check/resume（seal 完整后）再 dereference 它们。采用后 external 文件消失
不影响已提交 authority。所有可被后续依赖的 bytes、evidence 和 authority refs 都在 repository。

## Publication 与 exact schema

选择独立 `data/historical-construction-bytes/`，因为它存储 supplemental historical byte
evidence；不是旧 archive member、当前 source、Generation provenance 或 transport recovery
action receipt。固定 publication model（S、A、K 均由机器计算）：

```text
blobs/sha256/<S>.bin
authorities/<locale>/<A>/authority.json
authorities/<locale>/<A>/acquisition-evidence.bin
declarations/<locale>/<K>.json
transactions/<locale>/<K>.pending
transactions/<locale>/<K>.stage/**
```

这些路径均相对上述 family 根；不得写回 old archive/receipt/proof、当前 logical path，
不得产生其它 loose adoption file、mutable latest pointer、successor batch 或 Snapshot。
blob 为 exact raw bytes，不加 LF/header，不重新编码。authority/evidence/index write-once、
no-replace，发布文件固定 regular mode 0444；每次消费仍重新验证 SHA/size/type，不依赖只读
permission 宣称 immutable。多个 declaration 可以引用同一个 exact blob，不创建 hardlinks。

新增 authority schema：`go-learning/historical-construction-bytes/v1`。固定顶层字段顺序：

| 字段 | 完整含义 |
| --- | --- |
| schema | 上述固定 schema |
| capability | `attest-exact-declared-historical-construction-bytes` |
| statement | 下表的完整 typed identity core |
| identity | A；identity core 的 canonical SHA256 |
| recovery_source_kind | 实际四选一 acquisition kind；audit-only |
| recovery_acquisition_evidence | 完整 typed audit object；下述字段必填，audit-only |

`statement` 固定字段顺序与类型：

| 字段 | 类型/约束 |
| --- | --- |
| locale | safe locale identifier |
| content_package | `{id, authority_sha256, scope_identity_sha256, authority_references}`；历史 exact descriptor，refs 按 role/path/SHA 排序 |
| historical_invocation_ref | `{batch_id, task_kind, attempt, manifest, import_receipt, input_identity_sha256, provider, model, unit_id, unit_kind, source_sha256, input_sha256, lineage_references}`；attempt 固定初始 invocation 1，refs 绑定该 Unit 的 initial raw、原 Snapshot/reopen/representation lineage 的 immutable 历史事实，不引用 mutable result/current validation/latest retry |
| historical_archive_ref | repository file ref；canonical archive，不接受替换 archive |
| historical_declaration_ref | `{container, member, selector}`；container 必须等于 historical_archive_ref；member 是 typed ZIP member ref；selector 仅 `approved-proof.dependencies` 或 `construction.producer.implementation`，由版本化 reader 决定 |
| historical_proof_member_ref | `{container, member}`；同一 exact archive 内该 Unit 的 approved proof |
| logical_dependency_path | 已声明的 repository-relative logical source path |
| declared_historical_sha256 | 原声明 SHA，不从 acquisition 输入取得 |
| declared_historical_size | nullable int64；旧声明不存在 size 时必须 null |
| dependency_role | 上述两个 role 之一，从正式声明/版本化 reader 得到 |
| producer_contract | `{id, contract_version, role_binding_origin}`；origin 为 `explicit` 或 `versioned-reader`，无自由 role override |
| recovered_bytes_sha256 | S，必须等于 declared_historical_sha256 |
| recovered_bytes_size | 实测非负 int64；受界限约束；有旧 size 时也必须相等 |
| published_bytes_ref | exact repository file ref，路径仅可为上述 S 的 blob 路径 |

repository file ref 固定 `{path, sha256, size}`，path repository-relative、SHA lowercase
64-hex、size int64 >= 0。typed ZIP member ref 固定 `{path, sha256, size}`，path 是安全
member name，不能把 `archive.zip#member` 当 filesystem ref；其 bytes 必须在 container
inventory 中。selector 不接受 caller 传入 JSON pointer/index；按已知 schema 的固定 field
与唯一 logical dependency 选择，拒绝 missing/extra/duplicate/mismatched declarations。

所有集合 refs 的元素固定 `{role,ref}`；package authority refs 的 role 固定
`package-authority`，lineage refs 的 role 为 `initial-raw|predecessor-snapshot|original-reopen|
representation-authority|historical-provenance-authority`，由已知 graph reader 赋值，caller
不能提供。refs unique 且按 role/path/SHA 排序；必要 refs 不可缺，非适用 lineage 不虚构。
producer ID 固定 `go-learning/approved-interval-transport/v1`，contract_version 仅 1 或 2，
分别沿现有固定 legacy/explicit producer reader；unknown version 拒绝。

audit object 固定 `{acquisition_method, original_source_path, source_observation,
original_evidence_path, published_evidence_ref, observed_evidence_sha256,
observed_evidence_size}`。method 仅 `existing-whole-file-copy`；source_observation 固定
`{sha256,size,device,inode,mode,link_count,mtime_ns,ctime_ns}`。original paths 是不可
dereference 的审计字符串，不是 authority ref。published_evidence_ref 是本 A directory
下 exact `acquisition-evidence.bin` 的 repository ref，保存原 evidence bytes，不能裁剪/
改写 evidence 后冒充原 artifact。它的所有内容都不承担历史 declaration 授权。

所有对象 strict JSON：unknown field/schema/capability/version/enum、重复 key、超界数字、
unsafe ref、缺项均拒绝。不存在“unknown 但先继续”的 mode。所有 formal refs 都 exact
绑定 physical bytes；即使仅 JSON 排版变化，也不能绕过 ref SHA/size。

## 三种 identity 与唯一性

本 family 明确区分 identity 与 physical receipt SHA，不任意删除 JSON 字段后求等价：

1. **S = SHA256(raw recovered bytes)**，是唯一 blob identity；不同 logical declarations
   可共享 S，但不能借 S 取得其它 scope 的权限。
2. **K = SHA256(canonical DeclarationKeyV1)**。固定字段为上述 statement 中的 locale、
   content_package、historical_invocation_ref、historical_archive_ref、historical_declaration_ref、
   historical_proof_member_ref、logical_dependency_path、declared_historical_sha256、
   declared_historical_size、dependency_role、producer_contract，顺序与上表相同。
   它定位一个 exact declaration，不是按 content path 选 latest。
3. **A = SHA256(canonical IdentityCoreV1)**，固定结构为 `{schema,capability,statement,identity:""}`。
   使用对应 typed struct 的 Go `json.Marshal`，无缩进/末尾 LF；ordered arrays 保序，只有
   声明为集合的 refs 排序。blob path 仅依赖 S，因此无自引用；authority path 在 A 之后派生。

logical path、历史容器/member refs、role/version、历史 lineage 和 blob identity 都进入 A。
同一 exact historical declaration + exact bytes 在本 version 下得到同一 K/S/A。
source-kind、external paths、acquisition/evidence physical hash、file observations、wall clock、
PID/session path 不进入 IdentityCoreV1。它们不能产生另一种恢复权利。

audit object 是 authority envelope 的正式审计内容，完整受到 authority physical SHA/size
和 commit index 约束；**它不是 identity core**。不同 acquisition 路径可能产生不同
envelope/evidence physical SHA，但 A/K 不变，不能发布第二个 authority 或换路径再采用。
first committed envelope 保留第一次真实来源，不能以新 acquisition 覆盖其 audit。这里不将
physical ref SHA 伪装成第二个 capability identity；check 永远取 commit index 固定的原 envelope。

index 为 strict `go-learning/historical-construction-bytes-index/v1`：
`{schema,declaration_key,authority_identity,authority_ref,published_bytes_ref}`，没有 latest、
timestamp 或新的 scope。index 是唯一 commit point；其目标 A/K 必须由整个 statement 重算。
authority/blob 存在但没有该 exact index，不算已 adopted，也不能被 reader 自动消费。

## Future CLI：preflight / record / check

使用 `generation-bundle historical-construction-bytes`，与该 family 的 archive/member
取证职责一致，和 `generation-bundle legacy-provenance` 并列而不复用其 authority。
它不授权 retranslation successor，因此不塞进 `transport-recovery --record`。
以下只是未来 grammar，本轮禁止执行：

```text
generation-bundle historical-construction-bytes
  --locale <locale> --content-package <registered-package>
  --batch-id <installed-batch> --unit-id <unit>
  --logical-dependency-path <declared-repository-relative-path>
  --source <absolute-external-whole-file>
  --source-kind <codex-artifact|editor-history|tmp-copy|backup>
  --acquisition-evidence <absolute-external-evidence-file>
  --acquisition-evidence-sha256 <sha256>
  [--record --expect-identity <A> --expect-cas <token> [--resume]]

generation-bundle historical-construction-bytes --check
  --locale <locale> --content-package <registered-package>
  --authority <repository-relative-authority.json> --authority-sha256 <sha256>
```

每个 flag 唯一、非空，bool 只允许 bare flag；positional arguments、重复/unknown flag 拒绝。
`--record/--check` 互斥；两个 expect flags 仅 record 且全部必需；resume 仅 record。
check 不接受 batch/Unit/logical dependency/source/source-kind/evidence/expect/resume；
preflight/record 不接受 authority override。explicit package 必须到达该 command 的 production
parser/complete selector，不能依靠 startup Tour catalog。不能忽略多余 flag。

禁止 free old SHA/size/role/producer/member/receipt/archive、Unit list/limit、external proof、
force、allow-missing、ignore-drift、latest 等 override。新命令只验证 source，不调用 editor、
shell history reconstruction、patch、restore 或 model。

| Formal Command Contract v3 | 固定合同 |
| --- | --- |
| COMMAND | `generation-bundle historical-construction-bytes` |
| OPERATION | 默认 readonly historical declaration/acquisition preflight；record 发布 supplemental byte authority；check 验证指定 committed evidence |
| INPUT_KIND | existing formal historical declaration + external recovered whole-file bytes + acquisition evidence；check 为 exact recorded authority |
| INPUT_PATH_CONTRACT | source/evidence 仅 adoption boundary 接受 explicit absolute external regular files；logical dependency 与所有 persisted formal refs 为 repository-relative；original external paths 仅 audit 字符串 |
| MUTATION_EXPECTATION | preflight/check none；record known，仅 family blob/authority/audit/index 与其 sealed transaction；不修改任何 predecessor 或其它 lifecycle |
| IDEMPOTENCE | 同声明/bytes 的 K/S/A deterministic；preflight 可报告 ALREADY_RECORDED；duplicate record 拒绝，无 no-op adoption；resume 只完成同 sealed transaction |
| CURRENTNESS_CHECK | fresh historical graph、versioned declaration、exact source/evidence 与 conflict checks；check 只验证 historical evidence integrity，不宣称 current decoder/proof/语言资格 |
| PRIOR_OR_CAS | expect A + ephemeral CAS token；immutable old refs、decl/role/version、source/evidence observations、registered historical projection、verifier implementation contract、index/pending absence、transport root state；publication 前再次 fresh 验证 |
| SUCCESS_STOP | preflight 后审阅计划；record/check 后停止；不自动调用 transport recovery preflight/record/export 或 Snapshot |
| FAILURE_STOP | 非零且 typed blocker；保留所有已有 evidence，不能 fallback current code、修改旧 archive、猜 bytes 或发布 partial authority |
| RECOVERY_REPLAY | 先 readonly 归类 sealed pending 与 commit point；匹配且 CAS 成立才 resume 原事务；不能用普通重复 record 重放或换 acquisition 接管 |

stdout 固定 JSON `schema=go-learning/historical-construction-bytes-plan/v1`，含 mode/status、
locale/package/batch/Unit、完整 derived declaration、K/S/A、planned authority envelope/blob refs、
existing committed ref、source/evidence verification、CAS token、typed blockers、recorded/mutation。
不暴露不相关个人文件或整份源码。

plan 的固定字段依次为 `schema,mode,status,locale,content_package,batch_id,unit_id,
declaration,declaration_key,bytes_sha256,authority_identity,authority_path,planned_authority,
blob_ref,authority_ref,index_ref,acquisition_evidence_ref,source_verification,evidence_verified,
blob_verified,historical_integrity,transport_recovery_authorized,cas_token,blockers,recorded,
mutation,pending_path,allowed_next_operation`。mode 为 `preflight|record|check`；declaration 是
上述 DeclarationKeyV1 typed object，planned_authority 是完整 envelope。未知事实/不适用
refs/IDs/path/token/object 为 null，blockers 永远为 array，bool 不可用字符串代替。
source_verification 为 `verified-live|verified-sealed-input|not-required|unknown`；
content_package 是显式 ID，其完整历史 descriptor 在 declaration/statement 中。
`transport_recovery_authorized` 固定 false，`allowed_next_operation` 固定 null。
blocker 固定 `{code,detail}`，code 仅 `NO_DECLARATION|UNVERIFIED_HISTORICAL_GRAPH|WRONG_SCOPE|
UNSUPPORTED_PRODUCER|UNSUPPORTED_ROLE|INVALID_ACQUISITION|UNSAFE_PATH|SIZE_MISMATCH|HASH_MISMATCH|
CORRUPT_RECORDED_EVIDENCE|SUPPLEMENT_NOT_NEEDED|CAS_MISMATCH|DUPLICATE_RECORD|TRANSACTION_PENDING|
TRANSACTION_MISMATCH|ROOT_NOT_ADMISSIBLE|IO_FAILURE`。detail 仅 diagnostic，不参与角色推断。

- preflight `READY`：exit 0、完整 planned authority、A/path/CAS，recorded=false/mutation=none。
- `ALREADY_RECORDED`：exit 0，existing exact committed ref + check facts，planned authority=null、
  record identity/CAS=null、recorded=false/mutation=none。preflight 仍核对所给 source/evidence；
  external 已消失则直接用独立 check，不用 preflight 忽略传入的 flag。
- `BLOCKED`：非零、typed blockers，planned authority/record identity/CAS=null，mutation=none。
- record `RECORDED`：exit 0、exact authority/index/blob/evidence refs、recorded=true/mutation=known。
  重复 record 非零、recorded=false，不冒充 already-current mutation。
- check `VERIFIED_HISTORICAL_EVIDENCE`：exit 0、exact refs、historical_integrity=true、
  recorded=false/mutation=none、`transport_recovery_authorized=false`、allowed_next_operation=null。
  坏 ref/缺失/损坏/conflict 为 BLOCKED 非零。任何 check 均不读取原 external paths。
- publication 失败必须如实报告 `mutation=known|unknown`、pending/已知 commit refs 和停止点；
  不能输出 success 或可继续 recovery 的 operation。无法确认是否已写入时必须 unknown。

CAS token 是临时 inspection-to-record 防 TOCTOU observation digest，**不是 formal identity**。
它绑定 precise source/evidence flags、dev/inode/type/mode/nlink/size/mtime/ctime/hash observations、
old graph byte refs、registry projection refs、verifier implementation refs 和原 transport root/
index/pending observations。它可因 acquisition path 改变而不同，不进入 A/K/S 或长期 decoder
identity。record 必须用同一 flags/token；不是把可变 preflight plan 当作 authority。

## Record 的 CAS 与 sealed transaction

v1 首次 record 仅允许该 initial archive 的 transport recovery root link 和 `.record.pending`
均不存在；不介入已 recorded/exported/imported recovery 的 epoch。已有该 declaration 的
committed byte authority 后，check/reader 仍可使用，不因后续 transport root 正常推进而失效。
其它 phase 的新 adoption 不支持，BLOCKED；不隐式 supersede 或 patch 已记录 recovery authority。
尚无 recorded authority 但 archive/surviving source 已可合法提供 exact bytes 时，preflight/
record 报 `BLOCKED/SUPPLEMENT_NOT_NEEDED`，无 planned authority/identity/CAS，不新增冗余
采用权。已有 authority 的冲突/完整性仍按下文检查，不能用其它来源掩盖损坏。

record 取得与现有 retranslation exporters/transport-recovery record 相同的 repository
namespace 排他锁，并锁定本 K 的 pending owner；阻止同 root publication 与 batch/recovery
writer 并发。发布前 source/evidence 重新打开/读取验证，fresh 重算全历史 graph/A/CAS，
不采信旧 READY。source/evidence 改变、index 已占用、root 新增、其它 pending 或旧 ref 变化
均拒绝。尚未取得/写 pending 前拒绝 mutation=none。

pending 使用 strict `go-learning/historical-construction-bytes-transaction/v1`，固定 owner
（schema/K/A/CAS 的 typed digest）、原 flags 的 audit binding、preconditions、blob/authority/
evidence/index 的固定 target refs、各 exact sealed payload 和 seal。target 只能由 K/S/A
计算，不能由 caller 指定 transaction/stage 路径。serialized pending 限 64 MiB；未完整
seal 的 pending 不是可恢复事务。锁释放/PID 变化/mtime 老旧不代表 commit 或允许删除。

顺序：

1. fresh verification → exclusive pending → seal source 与 evidence 的完整实际 bytes、authority
   envelope、目标 index；fsync。未 seal 完整前 source 消失/变化即停止，不能猜缺项。
2. 在固定 stage 验证全部 sealed payload SHA/size/allowed inventory。sealed payload 是实际
   source 的精确副本，不由 patch 或 current source 生成。
3. blob no-replace 发布；已有 S blob 只能 regular/non-symlink/nlink=1、mode=0444 且 hash/size/bytes exact
   时复用，坏 blob 不覆盖。既有 loose blob 本身没有 authority；即使复用，仍必须完整验证
   external acquisition 和历史 declaration，再提交本 authority，不能只靠 blob 存在采用。
4. authority/evidence directory no-replace 发布；exact members/type/hash 均通过才继续。
   已有孤立 directory 无匹配 sealed pending 为 mutation=unknown，不能重写/领用。
5. index commit 前 fresh 再验证 old graph、historical package projection、implementation/CAS
   preconditions、root/pending 排他性及所有已发布 bytes；index no-replace 是唯一 commit point。
6. 重新验证 index→authority→blob/evidence 完整闭合后才报告 RECORDED；只清理本事务 pending/
   stage，不删除正式 bytes。commit 已证实而 cleanup 失败：mutation=known、保留 pending path、
   停止；commit 不确定：mutation=unknown。

publication 必须 fsync 新文件及其 containing directory；所有 blob/authority/evidence 已
durable 后才 commit/fsync index，随后才能清理 journal。恢复仍以 exact on-disk state 为准，
不能把曾返回 success 当成缺失 bytes 的替代证明。

record 开始、seal 完成之前，source 必须仍与 preflight observation exact 相符。seal 完成
之后 source 只是 acquisition audit，后续 publication 只消费已认证的 sealed payload，不再
依赖 `/tmp`；否则无法恢复“blob 已写后 source 消失”的可信成果。root/old refs CAS 在 commit
前仍必须成立，不因 source 不再读取而豁免。

fresh CAS 必须识别自身 transaction：只将 owner/seal/exact target bytes 全部吻合的本事务
pending/stage/已发布成员规范为原计划的“自身 publication progress”，不因自己刚写 pending
而误判，也不把其它 writer 的变更忽略。source/evidence observations 在 seal 前 live 重验，
seal 后固定为已验证 input snapshot；旧 refs/registry/implementation/root/index preconditions
在 commit 前仍 live 重验。CAS token 不按 publication 进度重发；不可拿新 token 接管旧事务。

`--resume` 只允许同 flags/A/CAS 的现存完整 sealed transaction：

| 中断/状态 | 唯一 continuation |
| --- | --- |
| preflight 后未 record | 重新 preflight；没有写权限预留 |
| source/evidence 在 record seal 前消失或改变 | fresh CAS reject；若已留下不完整 pending，unknown diagnosis，不换 source 自动重来 |
| blob 已发布，authority 未发布 | 核对 seal 与 blob；CAS 仍成立时仅完成原 authority/index，不能把 blob 当已 adopted |
| authority 已发布，index 未发布 | verify exact staged/published members；fresh CAS 后只提交原 index，不能重新分配 A |
| 完整 seal 后 external source 消失 | 可按同 owner flags 用 seal 的完整 bytes resume，不查找其它 external 文件 |
| index 已 exact commit，pending 尚在 | 验证完整原 commit 与旧 graph integrity后只清理本事务；不再要求 transport root 仍未推进，不重做 adoption |
| index 已存在且不同、部分文件不符、seal 损坏/owner 不符、无法确认 commit | mutation=unknown，保持现场，禁止 blind replay/cleanup |
| commit 前 old graph/registry/verifier/root CAS 改变 | 不 commit；保留 transaction/evidence，不能以新 expect/CAS 接管旧 pending |
| 完整 commit 后重复 record | write-once reject；用 check，不重写 audit、不生成第二 receipt |

resume 也取得同一排他锁；不接受自由 pending path、不允许 force、自动 rollback/reset/delete。
完整 commit 后的 historical check 不要求当时 acquisition inode/mtime 或 external path 继续存在。

## Transport recovery reader 接入与冲突规则

当前尚无此 reader 分支。未来 implementation/first-real-use 后，versioned historical reader
按 exact declaration context 采用以下固定顺序：

1. 已验证 old archive inventory 中包含该 exact byte sequence → archive source。
2. 旧 logical repository path 仍 regular/non-symlink、实际 bytes exact 匹配旧 digest/size →
   surviving dependency source，沿用已有规则，不需要采用 receipt。
3. 从完整 DeclarationKeyV1 计算 K，只读取唯一 canonical declaration index；验证其 exact
   authority physical SHA/size/A、同一个历史 graph/selector/path/role/version、published blob
   SHA/size/raw bytes 和 acquisition evidence integrity → supplemental recorded source。
4. 否则 BLOCKED；无 `/tmp`、source path、editor history、latest/mtime、prefix 或宽目录扫描。

有高优先级来源并不掩盖已存在的相同 K index/claim corruption：known recorded claim 必须
验证，任何同 ref 的不一致、index 指向其它 declaration、坏 blob、hash collision 下实际
bytes 不等都 BLOCKED。当前 logical path 已改为不同 digest 只是 surviving source **未命中**，
不是它声称提供了旧 bytes；不能因此阻止第 3 项，也不能把不同的当前 bytes 当冲突修好。
同路径但其它 digest/role 的 archive context 同样不是该旧 ref 的命中。多个完全一致的
合法来源按固定优先级选择，不能用 latest 选择；重复 authority/index 非法，不提供分叉选择。

reader 不编译/执行 blob，不把 published path 替换旧 logical path。old construction 使用
原 logical path/old SHA/role/version 与现在取得的 exact bytes；declared size 仍保留原有/null
事实。获得 bytes 的 origin、index/authority/blob refs 单独作为 supplemental resolution
evidence，不能加入旧 construction semantic fields 令其 identity 改变。

接入必须携带 verified invocation/member/producer 的 typed declaration context，返回
logical ref、exact bytes、origin kind 与 supplemental evidence refs。不能仅凭 path/SHA 在
全局 blob store 搜索授权；也不能把 supplemental bytes 塞进原 ZIP files map，再假称它们
通过了旧 archive inventory。原 archive inventory 保持独立、不可变；新来源是明确的
material resolver 分支，只用于 historical declaration，不为 current construction 的当前
source ref 提供替代。旧 reader 调用点缺少该完整 context 时继续 BLOCKED，不能省略绑定。

supplemental resolver/verifier 的 implementation refs 单独进入 recovery assessment/epoch
和新 Bundle 的证据 inventory，不能追加进已冻结 producer 1/2 的 old construction declaration，
也不能把 acquisition path/authority/blob locator 加入 old semantic dependency fields。
同一旧声明/bytes 从三种合法来源派生的 **old construction identity 必须相同**。这里不
声称修改 consumer implementation 不会令其它 current implementation/ZIP identity 改变；
已绑定 implementation SHA 变化仍按既有 currentness 规则处理，不提供 dual-canonical。

因此以下合同全部保留：typed construction identity、原 exact installed proof comparison、
drift predicate、fully classified dependency delta、encoding-only/unclassified BLOCKED，以及
current refs 不得替代 old refs。byte authority 不把旧 proof 标为 CURRENT，也不保证后续
transport recovery READY；可能继续发现其它缺失 bytes 或不允许的漂移。

未来 recovery assessment/current epoch 必须捕获实际采用的 index/authority/blob/evidence refs，
record 前重新验证；后续新 Generation Bundle 把这些 supplemental evidence exact 冻结进
独立 `formal/historical-construction-bytes/<A>/` inventory namespace，仍有 Files/hash/input
identity/current-check。不得补写旧 Bundle/receipt。改变 reader/Bundle currentness 的实现
必须重过 first-real-use gate；普通无该 evidence 的路径不获得任何例外。

## 实现验收与 fail-closed 边界

本轮不新增或运行 production tests。后续实现必须用 synthetic TempDir 与真实 `run(rawArgv)`
覆盖 default preflight、record、check、duplicate/empty/互斥 flags、external path boundary、
sealed interruption/resume；不得用 helper 冒充 CLI proof。必须验证：

- 旧正式声明缺失、只有日志/inventory SHA、free SHA/role/version、wrong path/package/batch/
  Unit/proof selector、unknown schema、duplicate declaration 均拒绝。
- wrong hash/old explicit size、current file alias、symlink/parent symlink、hardlink、non-regular、
  超限、已知/自述 reconstructed/equivalent/normalized/patch-derived input 均拒绝。
- external path/source-kind 变化不改变 K/S/A；第二 adoption 不能覆盖首次 audit；audit/evidence
  被篡改仍失败，不能因 audit 不进入 A 而免校验 envelope/index physical refs。
- old size 不存在时保持 null；actual size 不能伪装成旧声明；legacy role/version 的派生来源
  明示，不增加旧 archive member 或虚构 unknown 历史事实。
- record 前 source/inode/bytes、旧 refs、registry、implementation、root 状态或 CAS 改变失败；
  相同/不同 K writer、共用 blob、partial stage、坏/孤立 directory、wrong pending owner、
  committed cleanup、source 在 seal 后消失都按上表恢复，不自动 rollback。
- 优先级 deterministic；archive/surviving/supplemental 相同 bytes 结果一致；corrupt claim
  不被高优先级来源遮盖；没有 index 的 blob 不能被 reader 使用；不扫描 `/tmp`。
- supplemental 使用前后 old typed construction identity exact 相同，只有取得 bytes 的
  evidence 不同；缺失其它 facts、encoding-only、unclassified drift 继续 BLOCKED。
- bytes adoption 不创建 recovery authority/batch/Generation/Snapshot；普通 transport/retry/
  Snapshot/QC 路径无隐式采用，后续新 Bundle 独立冻结 supplemental refs。

## zh-CN 非执行性 walkthrough 与停止线

已知 `locale=zh-CN`、`package=learn-docs-v1`、`batch=chatgpt-zh-CN-064`、Unit
`b49d8e873a0f66f6803077eb534583a155d2bd209a032d7c00fba8fea856da5c`。
logical dependency 为 `internal/i18n/approved_interval_transport.go`；旧声明 SHA 为
`8aa14668d73e12233de6b9424deba65ee9fd0ec061a2fbf12c9c3eadd9df0e4d`。
已恢复的完整副本 size=23080，来自已知 codex-artifact 的 source-before file，regular/
non-symlink；诊断见 `/tmp/go-tour-i18n-historical-implementation-bytes-recovery-diagnosis`。
旧声明若没有 size，保持 null；23080 只作为 recovered size，不倒填旧证明。

当前 worktree bytes 的 SHA 为
`d215d7b6ac72280210b2fec43c52bfb36dadc9c1e3fb608d332a968b7d881726`，不能代替旧 bytes。
external source path 只在未来 acquisition boundary 使用，不能让当前 v1 reader直接读取它。
implementation 应在 synthetic graph 证明上述 selector、publication、check 与恢复合同，随后
维护者单独做 first-real-use gate；届时由 readonly preflight 实际派生 K/S/A/CAS 与 refs。

本文不预填 authority identity、实际 publication path、receipt SHA 或 transport recovery
identity，不采用真实 bytes、不执行真实 preflight/record/recovery、不创建 qc024。
**walkthrough 停在 implementation / first-real-use gate 之前。** 未来采用成功并独立 check
后也只消除一个 historical byte availability blocker；transport recovery 仍须另行 fresh
preflight，保留 full-R/new Generation/validation/Snapshot/Independent re-QC 全部 gate。
