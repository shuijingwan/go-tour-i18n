# Site shell representation dependency re-attestation

这是独立的 dependency/currentness authority，**不授权 canonical representation mutation**。原 [representation successor](SHELL_REPRESENTATION_SUCCESSOR.md) receipt 与 pointer 永不改写；普通 `shell-representation-check` 仍拒绝 stale base receipt。current selection 只有在 exact additive re-attestation 验证成功时，才可恢复该已安装表示的 currentness。

## Eligibility 与绑定

schema 为 `go-learning/shell-representation-dependency-reattestation/v1`，plan schema 加 `/plan`。scope 仍是 all-locales / site-v2-shell / 一个 physical Homepage Page TU。

Preflight 从当前 exact representation pointer、content-addressed base receipt 与完整 predecessor lineage 重建。它验证原 receipt schema/self-identity/path/SHA，比较 current deterministic reconstruction 与 base 的 **全部非 dependency 字段**：source/foundation/adoption/heads/package、raw SHA、old/new ShellLanguageSource、representation identity、protected input、Page Unit、slots/roles/coordinates、released teaching bodies 均必须相同。不会通过去掉某些 prose/slot 字段来声称 equivalence。

只有原实现明确列于 `shellRepresentationParserPaths` 的 parser/contract 文件 SHA 演进可申请；path set、mode、absence 必须不变，其它 source/adoption/foundation/manifest/head/authority dependency 漂移均拒绝。parser bytes 改变并不自动获准：重建输出有任何 topology/content/identity 改变都拒绝，须另走 representation successor。没有 drift 时 preflight 为 `base_receipt_current=true, reattestation_eligible=false`，无需且禁止提交新 receipt。

此处 mode 不变指 [repository-reproducible safety class](REPO_LOCAL_FROZEN_SOURCE.md#repository-reproducible-file-mode) 相同：普通 regular file 的 0664/0644 write observation 可等价，read/execute、file type、path/absence 和 SHA 不可放宽。mode-only 等价不进入 `changed_dependencies`。旧 receipt 的 `old_dependencies` 与 base envelope 仍逐字段原样绑定，self identity/physical SHA 按原 JSON 验证；仅已验证 receipt 的 typed current/validation modes 可与当前重建按类别比较，新 inputs 写 canonical mode。`validation_inputs` 新增统一解释器的 exact SHA；旧 checker SHA 或 validation inventory 失配仍 STALE，不能由 mode compatibility 自动 carry。后续使用现有 successor family 发布新的 currentness authority，原 receipt/pointer history 不改写，无新 schema 或语言/QC authority。

Re-attestation receipt 冻结 base pointer path/SHA、base receipt path/SHA/identity、source authority、完整 representation、released ranges、old/current exact dependencies、逐项 changed paths/SHA/mode，以及本机制 implementation/contract 的 SHA/mode。它不包含 locale candidate、QC 或语言判断。

Immutable receipts 位于 `data/site-shell/representation-reattestations/receipts/<identity>.json`；每个 base identity 有独立 pointer `data/site-shell/representation-reattestations/current/<base-identity>.json`。不扫描 latest、不按 path 缓存 PASS；其它 base 的 pointer inert。重复演进绑定 exact previous receipt，lineage 无 cycle 且 depth≤32。旧节点只作 hash-bound 历史，current head 必须与本次完整重建及当前 dependencies exact match。

## Formal Command Contract v3

新建/contract-modified command 的第一次真实使用，按 [first-real-use gate](WORKFLOW_HANDOFF.md#formal-command-contract-v3) 在实现完成后读取 current docs → implementation → direct CLI tests；三者一致并填完 handoff 字段后才能交接 mutation。下面是稳定合同，命令示例不授权真实 apply。

所有命令以前缀 `go run -mod=readonly ./cmd/tour-i18n site-content` 执行。

| COMMAND / mode | OPERATION | INPUT_KIND | INPUT_PATH_CONTRACT | MUTATION_EXPECTATION | STDOUT / SUCCESS_STOP |
| --- | --- | --- | --- | --- | --- |
| `shell-representation-reattest` | prospective-preflight | current installed authority（无 input flag） | none | none | plain `ShellRepresentationReattestationPlan`；停止于审阅 |
| `shell-representation-reattest --input <plan> --input-sha256 <sha>` | plan-inspection | re-attestation plan | repository-relative | none | `{plan,already_current,applied,mutation}`；停止于 inspection |
| 同一 input 加 `--apply --prior-pointer-sha256 <plan.prior_pointer_sha256>` | apply | re-attestation plan | repository-relative | known；publication 不明时 unknown | 同一 result wrapper；停止并交还 Coordinator，不开始语言 lifecycle |

CLI 无 `--locale`、`--package`、`--source`、`--parser`、`--output`、`--record` 或 `--prior-identity`；scope/base 从 exact current authority 推导。未知 flags/positional args 拒绝。无 input 时禁止任何其它 flag；有 input 必填 exact file `--input-sha256`，只在 apply 时允许且必须提供 prior-pointer flag。input 不能是 base plan、successor receipt 或 semantic review。

合法 input：`evidence/shell-dependency-plan.json`；非法 input：`/home/user/repo/evidence/shell-dependency-plan.json`，即使 absolute path 位于 repository 内仍拒绝。preflight stdout 可以先只读保存到 `/tmp`；由维护者另行授权落地为新的 repository-relative evidence、计算 exact SHA，不覆盖旧 plan。

Plan 包含 authority、planned receipt、current path、prior pointer SHA、eligibility/base-current 状态、`state=generation-required` 与 plan identity。authority 的 `changed_dependencies` 给 old/current Reference 与 mode；representation/source 均是 exact invariant，因此没有伪造的“representation changed”授权字段。

## CAS、idempotence 与 recovery

apply 绑定 exact plan file SHA 与 exact prior re-attestation pointer SHA（absence=SHA256(empty)，不是空字符串），同时由 plan 绑定 immutable base pointer SHA。shared `data/.glossary-corpus.lock` 排他锁下重新验证全部 authority/dependencies，使用第二独立 graph 在提交前复验，并执行 aggregate mutation guard。base successor pointer/receipt 不在写入集合内。

先保存 immutable receipt 并 fsync directory，再准备/fsync pointer temp；pending marker `data/.shell-representation-reattestation-pending` 绑定 input plan、prior、current path、result。再次 guard/CAS 后 atomic rename 新 pointer、fsync、verify installed bytes，最后删除 owned pending/lock。跨 invocation 的 preflight→apply 比较 repository mode class；bytes、mode class、absence、directory、pointer 或 source drift 均失败。单一 invocation 内 guard 对其观察到的实际 mode 仍严格复验，不因 write-bit compatibility 忽略并发 chmod。

同一 base + exact dependency set 已 current 时，不创建第二 receipt。新 preflight 要求使用原 exact plan inspection；原 plan inspection/apply 返回 `already_current=true, applied=false, mutation=none`。它只说明本次 no-op，不能证明其它 invocation 的 provenance。base-current 无 drift 的 plan 不可 apply。

Byte-identical immutable receipt 可以复用，不覆盖不同 bytes/identity。提交前留下的 exact orphan receipt inert；无 pending/lock 且 prior 未变、原 plan 仍 exact current 时，只有维护者另行明确授权才可重试。pending/lock 或 durability/rename 结果不明即 `mutation=unknown`，保留 marker/lock，停止 diagnosis；没有自动 repair/delete/replay。stale prior、tamper、missing/ambiguous lineage 或 representation/source drift 一律停止，不 fallback 到 current parser 的新 topology。

## Current selection 与下游

原 bound dependencies exact current 时无需 re-attestation。它们 stale 时，无 additive authority 继续 fail closed；有 exact current authority 才能选择 **同一个** base representation。Generation representation context 增加 re-attestation pointer/receipt，以冻结实际 currentness proof；新 invocation 重读全部 dependencies，不继承 mutable proof state。

Page Unit、protected input 与旧 successor identity 均不变；恢复的 Unit 仍是 `generation-required`，不继承旧 Page A。后续仍需正式 Generation → validation → Independent QC → A-only finalization。re-attestation 不产生语言 output、review、authorization、closure 或 activation。

`RepresentationSemanticReview`/`RepresentationReconciliation` 仍不能改变 canonical representation，也不能代替 dependency re-attestation。修改本机制合同/implementation 后，已有 re-attestation 自身同样必须通过 currentness；不能通过同名 schema 忽略 bytes drift。
