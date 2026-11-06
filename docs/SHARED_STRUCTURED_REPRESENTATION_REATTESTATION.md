# Shared structured representation dependency re-attestation

schema 为 `go-learning/shared-structured-representation-reattestation/v1`，plan schema 加 `/plan`。这是 shared structured family 的独立 dependency authority，`all-locales`；不授权 representation topology、source、owner、语言 output、PASS、closure 或 activation。Homepage family 的 schema、pointer 与 re-attestation 均不能代用。

## Bootstrap 与 invariant

输入为 exact installed shared representation successor receipt Reference，必须与其 package/asset keyed head 一致。bootstrap 不要求旧 dependency/currentness 已 PASS；先验证 base self-identity/path/SHA、完整 predecessor lineage、frozen Document/raw round-trip、真实 consumed Generation 的 frozen transport/registry/output/provenance，再独立重建今天的 source authority、registry 和完整 old→new topology。

package 比较使用 `loadSourceCurrent` 的 source-adoption/source-successor graph。membership 和 representation-bearing `Global` 必须先回到该 source graph；不删除 package 检查，不按 dependency 名字减字段。真实 SourceIdentity、source membership、adoption/foundation、structured successor、owner 或 registry 漂移继续拒绝。独立 Homepage representation projection 不属于 shared source package basis。

receipt 冻结完整 base（含旧 inputs 和 identity）、base pointer Reference、base receipt Reference、base representation identity；request、package、asset/path、raw SHA、previous identity、consumed Generation、rule、slot IDs、SourcePackage、RegistrySHA、SourceAuthority、Old、New、NewIdentity、Released 全部 exact invariant。当前 deterministic reconstruction 与任意一项不一致即停止，要求真正 source/representation successor；不使用目标语言判断证明 equivalence。

`old_currentness_inputs` 与 `current_currentness_inputs` 的 path set、mode、absence 必须相同且排序无重复。只允许 frozen profile 内、仍由 `sharedRepresentationParserPaths` 或 `sharedReattestationValidationPaths` 声明的 implementation/contract SHA 演进；corpus、archived Bundle 等非 implementation 输入仍 exact。沿用 Homepage pattern，禁止 inventory 增删；receipt 显式列出 `changed_dependencies`，`added_dependencies=[]`、`removed_dependencies=[]`。没有变化时 plan 为 base-current/ineligible，禁止 apply。未知文件不能进入 envelope。

新机制的 implementation、source-boundary reader、CLI/direct tests、canonical doc 另列于 `sharedReattestationValidationPaths`，冻结 SHA/mode。旧 lineage 节点仅作 immutable 历史；只有 active head 必须等于今天完整重建与 validation inputs。不能靠同名 schema 忽略实现 drift。

### Historical frozen profile 与 legitimacy gate

base 的 frozen profile immutable，不能将今天的 canonical path list 当作历史 base 的 inventory。current reconstruction 必须先证明 frozen profile 合法，再只读取其 exact paths 今天的 bytes/mode；不能静默排序修复、增删路径或接受任意混合子集。它仍要求 installed keyed head、immutable receipt identity/path/SHA、完整 predecessor/consumed Generation 和 source/topology proof，profile recognition 不能替代这些检查。

legitimacy 使用代码中追加式的 **canonical profile fingerprints**。从严格排序、无重复的 frozen inputs 中，移除两个独立校验的 mandatory paths：`data/glossary-source-corpus.json` 与 exact consumed Generation 的 Bundle path。对剩余完整 implementation path array 执行 UTF-8 `json.Marshal`（无空白、无尾换行），取 SHA-256；必须匹配已审定的完整 canonical epoch。此 fingerprint 不包含 locale、receipt identity 或目标语言，不允许任选 validation paths。原始 mode、absence、SHA 格式及 current physical files 另行严格校验；mode/absence 漂移、路径重排、缺项、foreign dependency 或 unknown addition 均 fail closed。

| Canonical epoch | Implementation paths 数量 | Fingerprint |
| --- | --- | --- |
| legacy | 15 | `f3d9fbe15ba2e3a474c98c5a46d04b1c3af869f8b5baa26b63a8d59dbd58f0f8` |
| extended-v1（最初 additive checker） | 23 | `11a0c0d22ffbc1f14a3d3ec72eef11574971e46486755469feaf3e3b3bfa9292` |
| language-audit-v2 | 37 | `71d3a48ee30b9570b7a0800535a865a6b3148559296645302ef88b6a0511d49b` |

每个完整 inventory 另含上述两个 mandatory inputs。新建 base 使用今天声明的完整 parser + validation profile，并同样要求已登记的 canonical fingerprint。以后扩大 creation profile 时，必须保留所有旧 fingerprint、追加审定的新 epoch，并同步 docs/direct contract tests；未登记的新 profile 也 fail closed。不能覆盖旧 fingerprint，不能从待验证 base 自动学习可信 profile。这是 reader 内的版本识别，不新增 artifact、registry lifecycle、schema 或 receipt field。

re-attestation 始终保持三者分工：`old_currentness_inputs` 是 base 原 frozen inventory；`current_currentness_inputs` 是同一 exact profile 的今日 SHA/mode；`validation_inputs` 独立绑定今天完整 re-attestation mechanism/checker authority。新的 checker paths 只能通过新 receipt 的 `validation_inputs` 绑定，不能扩张旧 base 的前两者。新建 base 则同时在自身 inputs 中绑定其创建时完整 checker profile。future validation-path growth 不使已识别的历史 base 永久失去 bootstrap 能力，但 implementation drift 仍须合法 re-attestation，不能自动 carry。base/source/representation topology 变化继续要求既有 successor contract。

## Selector、history 与语言边界

base representation keyed head 保持不动。新 selector 为 `data/site-shell/shared-representation-reattestations/current/<base-identity>.json`，严格为单一 canonical Reference；receipt 在同 root 的 `receipts/<identity>.json`。不扫描 latest、mtime 或排序选择。绑定 exact previous re-attestation，禁止 cycle/depth>32、其它 base/head、重复/ambiguous JSON。每个节点仍绑定同一完整 base，不能成为第二个 representation successor。

current reader 验证 source/topology 后比较 exact dependencies：原 inputs 仍 current 时无需 additive authority；发生允许的 SHA drift 时，必须存在 exact current re-attestation，否则 fail closed。Bundle 冻结实际使用的 re-attestation pointer/receipt lineage；Document 仍指向原 base receipt，representation identity 与语言 delta 不改变。历史 Generation 依据其 frozen Bundle/Document/raw 解码，不读取今天 re-attestation 来重判旧语言。affected asset 仍需 Generation；其它 structured assets 不因 dependency re-attestation reopen。

## Formal Command Contract v3

前缀为 `tour-i18n site-content`。首次真实使用前必须重新执行 [first-real-use gate：docs → implementation → direct CLI tests](WORKFLOW_HANDOFF.md#formal-command-contract-v3)。工程测试只可在隔离 fixture mutation，不授权真实 plan/inspection/apply。

| 字段 | prospective | inspection | apply |
| --- | --- | --- | --- |
| COMMAND | `shared-structured-representation-reattest` | 同左 | 同左 |
| OPERATION | readonly bootstrap/preflight | readonly reviewed-plan inspection | dependency receipt + 独立 keyed head publication |
| INPUT_KIND | installed shared successor receipt | re-attestation plan | exact reviewed plan |
| INPUT_PATH_CONTRACT | repository-relative regular base file | repository-relative regular plan file | 同 inspection |
| REQUIRED_FLAGS | `--base`, `--base-sha256` | `--input`, `--input-sha256` | inspection flags + `--apply`, `--prior-pointer-sha256` |
| FORBIDDEN_FLAGS | input/apply/prior flags | base/apply/prior flags | base flags |
| STDOUT_KIND | plain SharedRepresentationReattestationPlan | `{plan,already_current,applied,mutation}` | 同 inspection |
| MUTATION_EXPECTATION | none | none | known；终态不明为 unknown |
| IDEMPOTENCE | deterministic；已经 re-attested current 时要求检查原 plan | exact installed original plan返回 already_current | exact original installed plan no-op，不造第二 receipt |
| CURRENTNESS_CHECK | immutable base/selector/history、完整 source/topology invariant、dependency exact-set 与 SHA/mode | 独立重建 exact plan及 head | 两次独立重建 + aggregate mutation guard |
| PRIOR_OR_CAS | observe exact re-attestation head/absence | exact plan prior 与 current selector | exact plan file SHA、base head Reference、prior re-attestation head SHA；absence=SHA256(empty) |
| SUCCESS_STOP | 审阅 plan，不自动 inspection/apply | inspection 结束 | 返回维护者，不进入语言阶段 |
| FAILURE_STOP | 无新 authority，停止 diagnosis | stale/tamper/错误 kind 即停止 | unknown 保留 marker/lock，禁止 blind replay |
| RECOVERY_REPLAY | 无 mutation | 原 exact plan | 无 pending/lock、prior 未变且原 exact plan 仍 current 的 orphan 仅经维护者另行明确授权重试；unknown 先只读 diagnosis |

不接受 locale/package/source/parser/output/request/prior-identity、positional 或未知 flags。显式 `--apply=false`、空的 forbidden flags 同样拒绝。SHA 为 exact file bytes。合法 path 如 `evidence/shared-dependency-plan.json`；absolute（即使 repo 内）、traversal、symlink 均拒绝。base receipt 与 re-attestation plan 不互换，不接受 Homepage authority。

apply 使用 shared `data/.glossary-corpus.lock`。锁内重建两次并验证全部已观察 source/refs/dependencies/base head/prior head；先 write-once immutable receipt并 fsync，准备/fsync head temp，再写 pending `data/.shared-structured-representation-reattestation-pending`（plan、prior、head、result）。guard/CAS 后 atomic rename、目录 fsync、installed head/receipt verification，最后删除 owned marker/lock。base receipt/head 永远不在 write set。

durability/rename/postcommit 结果不明返回 `mutation=unknown`，保留 marker/lock；所有 shared current readers 与其它 representation publication 遇 pending 同样停止。无自动 repair/delete/replay；exact orphan inert，不扫描它作为 current authority。
