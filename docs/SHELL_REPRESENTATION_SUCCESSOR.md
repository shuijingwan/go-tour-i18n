# Site shell canonical representation successor

本合同只授权 **raw source 不变时，canonical protected Generation representation 的显式演进**。source adoption、正文 replacement、翻译、semantic reconciliation、QC acceptance 与 activation 的 authority 继续独立。

## Scope 与选择

schema：`go-learning/shell-canonical-representation-successor/v1`；reviewable plan schema 在其后加 `/plan`。v1 scope 固定 `locale_scope=all-locales`、`site-v2-shell`、canonical physical Homepage `/`。表示是跨 locale 共享的 source authority，不能以某门语言的 judgment 改其他语言的保护规则。一个物理 Homepage 始终是一个 Page TU；textarea body 只是内部 slot。

当前注册的 prospective parser 为 `go-learning/shell-textarea-teaching-representation/v1`。它只允许由 active Go source comment classifier 证明的、scanner-owned textarea line teaching bodies 从 opaque gap 进入 language slots。既有 slots/identifier bindings 不变；raw/decoded coordinate 不精确、未知 grammar、重叠 span、未知 rule、没有真实 topology delta 均拒绝。机器 code、identifier、literal、directive、HTML 和 source content 不得注入或丢失。source round-trip 必须 byte-identical。`SafeTeachingCommentBodies` 与旧 extractor 不被修改，也不会因“新 parser 输出不同”而自动采用。

没有专用 current pointer，原 canonical representation 保持原样。current reader 只消费 `data/site-shell/representation-successors/current.json` 的单个 exact `Reference{path,sha256}`；不会扫描 receipts 选择 latest，也不把 orphan receipt 当 current。数组、重复 JSON member、非 canonical path、wrong schema、多个/矛盾 lineage 证明均 fail closed。

receipt 保存 source package、canonical raw path/SHA、当前 adoption 与 foundation refs、structured source heads、old/new 完整表示及各自 identity/protected-input SHA、old/new Page Unit IDs、精确 released ranges、parser/contract/file mode dependencies、predecessor ref/identity。表示 identity 绑定完整 source coordinates、slots、protection 与 role/encoding，不能只用 raw SHA 或 Unit ID。predecessor exact 连续、无 cycle 且 depth≤32；当前版本只注册上述 rule，不推测未来 parser 升级。相同输出不能创建第二个无意义 successor。

historical snapshot SHA 始终绑定；实体 snapshot ZIP 若按既有 frozen-manifest authority 重建而不存在，dependency 显式记录 `absent=true` 与 SHA256(empty)，由同一 invocation 的 absence guard 复验。不能创建 ZIP 迎合 checker，也不允许其它必需 dependency 缺失。Generation Bundle 通过独立 `TranslationRepresentationContextAdapter` 追加 current pointer、receipt 与本合同，不改旧 adoption 已冻结的 source-context provider bytes。

## Command matrix

本 family 的首次真实使用及 contract 修改后再使用，必须先通过 [Formal Command Contract v3 first-real-use gate](WORKFLOW_HANDOFF.md#formal-command-contract-v3)：实现完成后读 current docs → implementation → direct CLI tests，填完 command/operation/input-kind/path/mutation 等交接字段。不能从 `-check` 名称推测输入。

表中命令均以 `go run -mod=readonly ./cmd/tour-i18n site-content` 为前缀；`<plan>` / `<receipt>` 是 **repository-relative path**，`<sha>` 是对应文件 exact SHA-256，不是 JSON identity。没有 `--output`，stdout 由维护者按交接保存。

| COMMAND / mode | OPERATION | INPUT_KIND | PATH CONTRACT | MUTATION | SUCCESS RESULT | STOP CONDITION |
| --- | --- | --- | --- | --- | --- | --- |
| `shell-representation-successor --parser <rule>` | prospective-preflight | parser-rule | no input path；stdout 可审阅落到 `/tmp` | none | plain `ShellRepresentationPlan`，schema 为 receipt schema 加 `/plan` | 审阅 plan；不自动 landing/apply |
| 维护者显式 plan landing（无专用 CLI） | plan-landing | plan | 新的 repository-relative evidence path | known：仅落地审阅 plan，不写 authority | landing bytes 的 exact SHA | 不覆盖旧 plan；尚未授权 apply |
| `shell-representation-successor --input <plan> --input-sha256 <sha>` | plan-inspection | plan | repository-relative | none | `{plan,already_current,applied,mutation}`，`applied=false` | 仅检查 plan currentness；不自动 apply |
| `shell-representation-successor --input <plan> --input-sha256 <sha> --apply --prior-identity <id> --prior-pointer-sha256 <sha>` | apply | plan | repository-relative | known；publication 结果不明为 unknown | 同一 result wrapper；新提交 `applied=true` | 成功后交还 Coordinator；失败停止，不 replay |
| `shell-representation-check --input <receipt> --input-sha256 <sha>` | receipt-verification | receipt | repository-relative | none | plain `ShellRepresentationSuccessor` receipt JSON | receipt 验证结束；不证明它是 current head |
| 原 exact plan 的 readonly inspection | already-current-inspection | plan | repository-relative | none | `already_current=true, applied=false, mutation=none` | 当前 binding 已存在；不能据此断言本次/先前曾执行 apply |
| pending/lock/I/O 不明后的人工交接（无自动 repair CLI） | recovery-diagnosis | plan/prior/result/pending evidence | 各原始 exact paths；不改写 | none：只读 diagnosis | 先确定 unknown mutation 的真实状态 | STOP；未完成 diagnosis 不 apply/replay/downstream |

**`shell-representation-successor --input <plan>` = plan inspection；`shell-representation-check --input <receipt>` = receipt verification。两者不能互换。**check 不接受 plan；plan inspection 也不消费 receipt。`--input` 合法示例为 `evidence/shell-representation-plan.json`；非法示例为 `/home/user/repo/evidence/shell-representation-plan.json`，即使该 absolute path 位于 repository 内仍拒绝。

required/forbidden flags：prospective preflight 只需 `--parser`，禁止 `--input`/`--input-sha256`/`--apply`/CAS flags；input modes 必须同时给 `--input`/`--input-sha256` 且禁止 `--parser`。只有 apply 可用且必须提供两个 prior/CAS flags；check 禁止 apply 与 CAS flags。所有 modes 仅允许既定 Homepage `--source _content/index.md` / `--package site-v2-shell`（可省略采用相同默认值），未知 flags/positional args 拒绝。具体 currentness、CAS、concurrent mutation、idempotence 与 replay/recovery 边界如下；任一验证失败即停止，不用 receipt-check 代替 plan inspection。

## Readonly preflight

在 repository root：

```sh
go run -mod=readonly ./cmd/tour-i18n site-content shell-representation-successor \
  --parser go-learning/shell-textarea-teaching-representation/v1 \
  --package site-v2-shell --source _content/index.md \
  > /tmp/shell-representation-plan.json
```

默认只读，仅打印 deterministic plan，不写 receipt/pointer/archive。plan 包含完整 authority、planned receipt path/SHA、prior identity/pointer SHA、affected package/route/count 与 `state=generation-required`。初次 prior pointer 不存在，用 SHA256(empty bytes) 明确表达 absence；不是任意空字符串。

维护者审阅 plan 后计算其 exact file SHA。以下命令重读 source/current pointers/dependencies 并重建 plan；不能以旧 observed SHA 代替 currentness：

```sh
go run -mod=readonly ./cmd/tour-i18n site-content shell-representation-successor \
  --input <repository-relative-plan.json> --input-sha256 <exact-file-sha256>
```

`--input` 必须是 repository-relative evidence path；上面的 `/tmp` 仅用于审阅输出，正式 inspect/apply 前由维护者按现有 evidence landing 规则保存为新的 repository-relative plan file，重新计算 landing 后的 exact SHA。不得覆盖历史 plan。stdout 为 `{plan,already_current,applied,mutation}`。初次合法 inspection 的 mutation=none；已安装的 exact 原 plan 显示 already_current=true，不构造第二份 successor。`already_current` 描述观察到的 binding，不证明 mutation provenance；与先前合法 FINAL 矛盾时停止并 diagnosis，不继续 downstream。

## Explicit apply / CAS

维护者确认完整 plan 后，使用 **同一份 exact input**，不重新猜 prior：

```sh
go run -mod=readonly ./cmd/tour-i18n site-content shell-representation-successor \
  --input <repository-relative-plan.json> --input-sha256 <exact-file-sha256> \
  --apply --prior-identity <plan.prior_identity_sha256> \
  --prior-pointer-sha256 <plan.prior_pointer_sha256>
```

apply 使用 shared `data/.glossary-corpus.lock`，重新加载 current source authority、精确 plan、predecessor 与 dependencies，并在提交前用独立 graph 再验证一次。file bytes/mode、source heads、directory membership、absence→presence 或 plan/pointer drift 均 fail closed。只写 content-addressed immutable receipt 与独立 current pointer，不修改 raw source、foundation、原 adoption、历史 receipt、candidate 或旧 A。byte-identical receipt 可以复用，不同 bytes/identity 不覆盖。

先保存 immutable receipt，再准备并 fsync pointer temporary file；记录 `data/.shell-representation-successor-pending`（绑定 plan/prior/result），最后 exact CAS + atomic rename pointer、fsync、验证 installed bytes 并清除 owned marker/lock。marker 以后发生异常返回 mutation=unknown，保留 marker/lock供诊断。不会盲 replay、自动回退 pointer 或删除未知 marker。

## Currentness、replay 与 recovery

```sh
go run -mod=readonly ./cmd/tour-i18n site-content shell-representation-check \
  --input <receipt.json> --input-sha256 <exact-receipt-file-sha256>
```

check 验证 receipt、当前 source/parser dependencies 与完整 lineage；它不发布，也不把历史 ref自动提升为 current。要证明当前 head 或 idempotent no-op，用原 exact plan 的 readonly inspection；无需再次 apply。current catalog 每次仍验证 source/adoption/foundation、receipt、自 identity、parser reconstruction、dependency SHA/mode 与 guard，没有 persistent/global proof cache。

- marker/lock 不存在、prior pointer 未变化且只有 exact orphan receipt：receipt inert；readonly 重建与原 plan byte-identical 后可由维护者明确重试，不重新生成另一个 authority。
- stale prior、wrong source/foundation、tampered/missing receipt、ambiguous/discontinuous lineage：停止；不要 overwrite、reset、改 pointer 或借 reconciliation 接受。
- pending marker、lock 残留、rename/fsync/I/O 结果不明：mutation=unknown，停止；核对 marker 指向的 exact plan、prior/result receipt、pointer bytes 和 dependencies，再由维护者决定既有 recovery 动作。没有自动 repair 命令。
- parser implementation/contract dependencies 改变会使 plan/receipt currentness失效；不能用版本名掩盖 bytes drift。未来 rule/dependency 演进须新增明确支持，不允许未经 authority 的 fallback。

相同 raw/source authority 与 exact 相同 representation 下的 parser/contract dependency 演进，使用独立 [dependency re-attestation authority](SHELL_REPRESENTATION_REATTESTATION.md)。它不修改或使 stale base receipt 本身通过 `shell-representation-check`，只在 current selection 中提供新的 hash-bound currentness proof；representation/Unit/protected topology 变化不得使用此路径。

## 下游 lifecycle 与 reconciliation 边界

采用 receipt 后，current Global 的 shell package **投影**增加 representation receipt contract dependency；原 source identity/raw bytes 不变，current package identity 与新 Unit inventory绑定。旧 package/Unit descriptor 保留为 historical；新 current scope 只含新 Homepage Unit，旧 Snapshot/A 不会自动匹配新 ID，也不能以包级 carry 代替语言 gate。新 Unit 是 `generation-required`，须走既有 Generation → restoration/machine validation → Independent QC → A-only finalization/current authorization，之后才可能建立新 closure/Surface。

本 authority 不创建 glossary Review、翻译结果或 QC PASS，不执行 corpus/glossary successor，也不把任何旧语言判断 promote 为新 Unit 的 acceptance。glossary/corpus eligibility 仍由其独立 canonical contract验证；若相应 projection 需要更新，必须另走既有正式 lifecycle。

`RepresentationSemanticReview` 的 APPROVE_RECONCILIATION 仍只批准 exact frozen prose correspondence，**不改变 canonical Generation representation**。`RepresentationReconciliation`、surface/current-validation reopen 仍只拥有各自既有 recovery/correction 职责，不能当此 receipt、不能写该 pointer。已授权 current-validation revision-only transport 与永久 canonical representation change 是不同 authority；前者不能使普通 initial Generation 编辑 opaque token。

本轮建立机制不代表真实 Homepage 已切换。真实 preflight 可以展示 prospective teaching slot，但只有未来 exact plan apply 才改变 current selection；不得把 preflight PASS 写成 source adoption、语言 PASS 或 activation。
