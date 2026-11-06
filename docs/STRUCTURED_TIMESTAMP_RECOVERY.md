# Structured Generation original timestamp 缺失恢复

本合同只处理已经真实完成、exact model input/output 和 provider/model/session 均可信，但原 `generated_at` 从未记录且无法从可信 evidence 恢复的 metadata loss。schema 为 `go-learning/structured-generation-timestamp-recovery/v1`；request、Generation-session evidence、result 分别使用 `/request`、`/generation-evidence`、`/result`。

正常 Generation 继续遵守 [语言语义 authority](STRUCTURED_GENERATION_SEMANTICS.md) 的真实 RFC3339 UTC 时间规则。普通 `structured-import`、`CheckStructuredAssetsBundle` 和 stale-Bundle `structured-recover` 的 validator、必填 provenance 和行为均不变。这里不是新的 Generation invocation，也不是用 metadata loss 兼容语言输入变化。不能将其与 `carry_forward` 组合。原 Bundle 满足普通 current-check 比较，或仅通过下述 exact shared re-attestation ancestor bridge；stale source/Document/protection/finding/representation/dependency authority 必须先按各自合同解决。

## 声明与信任边界

维护者独立核对原 Generation session 的 timestamp-unavailable 声明及 exact 三份 bytes：原 Bundle ZIP、原 StructuredOutput JSON、包含该 JSON 的原 output ZIP。request 必须包含 `schema`、`bundle`、`output`、`output_zip`、`generation_session_evidence`（均为 exact path/SHA256 Reference）和 `attested_by`（维护者 component identity，不得与 generation session 相同）。所有 path 必须 repository-relative、regular、无 symlink、输入不超过 32 MiB。原 output ZIP 必须只有一个 regular member `structured-output.json`，解压 bytes 与原 JSON 完全相同；不读取 ZIP mtime 作为时间证据。

Generation-session evidence 是 strict JSON，完整字段如下（占位符不能直接使用）：

```json
{
  "schema": "go-learning/structured-generation-timestamp-recovery/v1/generation-evidence",
  "original_generated_at_status": "unavailable",
  "generation": {
    "provider": "chatgpt",
    "model": "gpt-5.6-sol-high",
    "session": "<original-generation-session>",
    "generated_at": ""
  },
  "bundle_sha256": "<exact-original-bundle-file-sha256>",
  "bundle_identity_sha256": "<original-manifest-identity>",
  "output_sha256": "<exact-original-json-sha256>",
  "output_zip_sha256": "<exact-original-output-zip-sha256>",
  "declaration": "original generated_at was never recorded and cannot be recovered from trusted evidence"
}
```

evidence 由维护者忠实落地原 session 的已确认声明；不得要求新语言 Generation 来补时间。hash 证明 bytes/lineage exact，不能机器证明 provider/session 声明真实性或“从未记录”这一历史事实；维护者 attestation 承担与普通 provenance landing 相同的真实性责任。不能把任意 free-form prose 当作 currentness 或语言通过证明。

`original_generated_at_status=unavailable` 和空 `generated_at` 必须同时成立。`UNKNOWN`、估算时间、当前时间、message/download/landing/ZIP/filesystem 时间均拒绝，不能当成 original time。没有 `recorded_at` / `recovered_at`，identity 不依赖 wall clock。真实 provider/model/session 仍必须符合现有政策、完整 Unified Review 的 Generation session 与 generation/reviewer role isolation。

## Immutable authority 与下游

`StructuredGeneration` additive `timestamp_recovery` 绑定 immutable recovery receipt；`generation.generated_at` 保持空字符串，receipt 的 statement 明确 unavailable。旧带合法时间的 records 省略此字段，原 bytes/self-identity 与 reader 行为不变。普通 import 没有提供这个 reference 的入口。

receipt 保留 exact request、session evidence、原 Bundle、原 JSON、原 output ZIP、provider/model/session、维护者与原 manifest identity。archive 位置为：

- 原 Bundle：既有 `data/locale-language/<locale>/structured-bundles/<sha>.zip`。
- 原 JSON：既有 `data/locale-language/<locale>/structured-outputs/<sha>.json`。
- request/evidence/output-zips/receipts：`data/locale-language/<locale>/structured-timestamp-recoveries/<kind>/<content-or-identity-sha>.<json|zip>`。
- 最后 publication：既有 `structured-generation/<identity>.json`。

不 re-encode 原 JSON/ZIP，不修改原 Bundle identity，不删除或覆盖语言 output、历史 receipt 或 head。historical/current Generation reader 必须验证 recovery receipt self-identity、原 request/evidence hashes、三份 archives、strict original output 与完整 Plan/Expected，ZIP→JSON byte equality、provider/model/session 与 unavailable invariant。historical graph 将这些 refs 纳入 dependency guards；随后沿用 frozen source/registry/Review/representation reconstruction。closure/Surface 消费同一 Generation reader，缺少或篡改 recovery authority 一律失败，不把 unavailable 当 timestamp。

preflight/apply 仍通过同一 canonical exporter 完整验证 current graph，并执行 StrictJSON/output exact-set、current Document/slot/protected tokens、reconstruction、glossary、shared representation successor/current dependency re-attestation、exact revision finding 和 session isolation。metadata authority 只解决原时间 absence；不授权其它 asset，不扩大 Expected，不关闭其它 pending work，不 carry old A，不产生 QC/Surface PASS/closure/activation。后续继续原 lifecycle，不新造语言状态机。

## Exact shared re-attestation lineage compatibility

仅 `structured-recover-timestamp` 可在普通 Bundle 比较因 shared dependency re-attestation context 演进而失败时，建立 **exact ancestor** bridge；普通 current-check 和 generic `structured-recover` 不接受这个新例外。不能按目录前缀把 re-attestation context 全部忽略。

从 old/current 完全相等的 Plan 中的正式 shared Document dependency 找到唯一 package/asset/base successor。base receipt、representation identity、raw source、Document/slots/protected topology、finding、Expected、glossary、Unified Review、corpus 和全部真正 `StructuredSemanticAuthorities` 必须 exact。当前 canonical exporter 必须成功；另按既有 family 验证 current receipt/head 的全部 source/current dependency inputs，不能仅比较 representation identity 或 Plan identity。

old Bundle 必须携带其 head 和完整 Previous chain 的 exact path/SHA bytes；current Bundle 同样携带当前链。每个节点必须通过既有 family self-identity、same base/source/topology、dependency inventory 与 immutable receipt validation。old head 必须是 current 链中的 exact ancestor，且 old 完整链必须等于 current 链从该 ancestor 开始的完整 suffix。A→B→C 可接受 A→C；B、C 同为 A 的 child 时，B→C 拒绝。different base/asset/representation、missing/forged receipt、cycle/depth overflow、非 fresh current authority 均拒绝。

只有上述 proof 机械枚举并验证过的 changed head member 和两侧完整链 members，才可从本次临时比较副本中移出；remaining members 仍调用原 `compatibleStructuredBundles`，保留其全部 non-reattestation semantic comparison 和既有明确 execution classification。未知/多余 context、不相关 inventory/source/Document/finding/glossary/Review/semantic docs 漂移仍失败。没有 changed ancestor proof 时不能用此路径救活其它 stale Bundle。

receipt additive `currentness_bridge` 保存 `current_bundle`、`current_bundle_identity_sha256`、`reattestation_lineage`；每个 proof 保存 exact representation successor Reference、representation identity、old/current re-attestation References 及 `current_to_origin` 完整链。current Bundle 原 bytes 保存到既有 content-addressed structured-bundles archive，Generation 仍最后 publication。原 `bundle`、statement Bundle identity、原 StructuredOutput/JSON/ZIP bytes 与其 `bundle_identity_sha256` 全部保持真实 old model input，**不改成 current identity**。既有无 bridge receipts 字段省略，身份与行为不变。

historical reader 读取 frozen old/current archives 重新验证上述 proof 和 immutable chain refs，不要求当时的 mutable current head 今天仍停在同一 receipt；合法 descendant advance 不改写历史。current downstream reader 还必须执行原有 current source/Document/representation/current head/dependency checks。missing/tampered recovery authority、immutable receipt 或 current authority 仍 fail closed；bridge 不是缓存 PASS，不授权新 topology、language A 或其它 lifecycle mutation。

这次只扩展 timestamp recovery currentness/receipt lineage，不修改普通 checker、generic recovery 或 shared dependency authority contract。工程实现不得以放宽 bound dependency inventory 的方式避免 re-attestation。

## Formal Command Contract v3 / first-real-use

| 字段 | readonly preflight | explicit apply |
| --- | --- | --- |
| COMMAND | `site-content structured-recover-timestamp` | 同左 |
| OPERATION | 验证原 bytes/currentness 并计算 immutable publication refs | 重做全部 preflight，再 publication archives/authority/Generation |
| INPUT_KIND | timestamp-loss request JSON；不是 Bundle、Generation receipt 或 stale recovery plan | 同一 exact reviewed request |
| INPUT_PATH_CONTRACT | request 及其全部 refs 必须 repository-relative regular path；`tmp/loss-request.json` 合法，absolute/traversal/symlink 非法 | 同左 |
| REQUIRED_FLAGS | `--request`、`--request-sha256` | 同左加 `--apply` |
| FORBIDDEN_FLAGS | 不接受 bundle/input/output/locale/package/provider/model/session/generated-at/CAS overrides 或 positional args | 同左 |
| STDOUT_KIND | `/result` JSON：authority/Generation exact References、validation count、`applied=false mutation=none` | `applied=true mutation=known`；全体 exact artifacts 已存在则 `already_current=true applied=false mutation=none` |
| MUTATION_EXPECTATION | none；不创建目录或 receipt | 仅上述 immutable content-addressed write set，最后写 Generation；不改 head 或 completion |
| IDEMPOTENCE | deterministic，无 current time；已有结果仍重做 currentness | exact replay no-op；不同 bytes 永不覆盖 |
| CURRENTNESS_CHECK | canonical current graph、普通 Bundle 比较或本合同唯一的 exact shared re-attestation ancestor bridge；全部 source/language/representation/dependency/finding gates、original output validation、role isolation 和 input byte/mode guards | 同左，apply 不消费旧 PASS cache |
| PRIOR_OR_CAS | 无 mutable head，none；request SHA 是 reviewed input guard | none；沿用 structured-recover 的 immutable no-replace publication，先检查全部 destinations，无任意 replacement |
| SUCCESS_STOP | 审阅 preflight；不自动 apply | 仅 metadata lineage + Generation evidence，停止在 downstream validation/closure 之前 |
| FAILURE_STOP | 无新 authority；保留原 inputs，禁止填假时间 | publication error 可能留下已知 content-addressed archives，停止，不 rollback/覆盖，不宣告 import 成功 |
| RECOVERY_REPLAY | 修复 exact blocker 后重新 readonly preflight | 先只读核对 exact planned paths/bytes 与 currentness，再由维护者明确授权 exact replay；不 blind replay，不换 output/provenance |

复用既有 `SaveArtifact` 的 regular-parent、file sync、no-replace link、same-bytes idempotence；没有 selector、CAS pointer 或自动 activation。存在 partial archives 不等于成功，Generation 最后保存并 post-read 验证。未知文件类型、destination conflict、input 漂移均停止。

这是 modified formal CLI currentness / recovery contract，旧 first-real-use verification 不得 carry。第一次真实使用必须重新按 [docs → implementation → direct CLI/contract tests](WORKFLOW_HANDOFF.md#formal-command-contract-v3) gate。工程测试只在 isolated fixture publication，不授权真实 preflight/recovery/import。若工程变更触及 installed representation dependency authority 已绑定的 implementation bytes，该 currentness gate 仍必须拒绝；先按既有 dependency re-attestation 合同审计，不能用 timestamp recovery 绕过。
