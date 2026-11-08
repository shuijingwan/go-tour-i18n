# 正式阶段交接合同

全项目执行共同遵守[共享翻译单元生命周期与五角色协作](SHARED_TRANSLATION_UNIT_LIFECYCLE.md)：所有内容表面复用同一生命周期；页面差异限于适配器；传输安全与逐单元机器校验分层；先恢复、先复用，中文说明，交接只给一个执行者、动作和停止条件。本文中 Tour 的 present / Example 细节属于 Tour 适配边界，不另建 Learn/Docs 状态机。Learn/Docs 两批生成和机器校验全部完成后才初审，两批初审全部完成后集中修订。角色与工具分工不替代任何正式 machine authority 或语言质量 gate。

本合同适用于 Generation role/session、Reviewer role/session 和具备仓库终端能力的当前 AI execution environment。它统一阶段完成、失败与人工 handoff 的最终回复；不新增持久化状态，不替代 machine gate，也不扩大 AI 权限。

## Call-Zero-First 交接合同

multi-locale campaign 中 remote tool calls 是有限、共享且会随 locale 数量线性放大的资源。项目采用长期 **Call-Zero-First**：默认 **RDC / repository remote calls = 0** 是成本/风险默认值，**不是 zero-call KPI，不机械追求 0**。先复用 current conversation、current Bundle 与 sufficient terminal evidence；已有信息足够时不调用。需要只读 discovery 时，以总人工成本和风险选择路径，不因 RDC 可用就调用，也不按某个月额度或任务 duration 决定执行者。

标准 deterministic repository / shell workflow、正常 workflow 的 readonly inspection / currentness reassessment / contract verification、formal command execution 与 Git checkpoint/push 默认由 **Maintainer Local terminal** 执行，**Codex=0**，与预计耗时无关：export、current-check、hash、import、process、validation、Snapshot、scope、Reviewer Bundle、evidence save/check、QC record、finalize、authorization、package closure、activation、build、preview、browser verifier、publish、Production verifier 均如此；repository engineering writes 仍交获授权的本地 Codex，按正式 workflow 执行。已授权工程所必需的局部读取和 targeted tests 保留其任务边界。AI 提供完整可复制命令，包含已确认路径、ID、前置 gate、依赖顺序、结果检查和停止点。Coordinator / engineering 的 targeted readonly discovery 按下列规则选择；不把“Local terminal 也能读”作为强制转交手工 discovery 的理由。

合法 targeted readonly RDC 场景包括：

1. 安全执行下一步必需的 exact machine fact 不在 current conversation、current checked Bundle 或 sufficient terminal evidence 中，使用 `missing-machine-fact`。
2. 真实 FAILED / unexpected state / mutation=unknown 需要 diagnosis，使用 `failure-diagnosis`。复用已有 evidence 后仍缺事实时，优先一次合并 readonly RDC 读取所需 canonical docs + relevant implementation/tests，而不是先把纯读取的 `rg/sed/cat/find` 转给维护者；读取 tests 不代表运行测试。
3. 当前 conversation 未包含的 repo canonical docs / exact command contract 需要读取，且已有 Bundle / terminal evidence 也不足以满足该 discovery 时，使用正式合法理由 `canonical-doc-read`，不重复读取已有内容。
4. 一次 targeted readonly RDC 能明显减少维护者手工 discovery、重复粘贴或多轮只读命令时，允许并优先采用 RDC；按事实类型披露 `missing-machine-fact` 或 `canonical-doc-read`，不另造 reason。若 RDC 不可用或没有明显收益，再提供一个合并只读 Local terminal discovery command。
5. 维护者明确要求使用 remote tool，使用 `maintainer-request`；读取请求本身不授权写入。

对合法 FINAL PASS 做无意义 remote reverify、current Bundle 已完整仍重扫同一输入、对已知事实重复确认，仍然禁止。“再确认一次”“怕记错”、操作很短、AI 可自动完成或 RDC 可用本身都不是理由；减少**未知事实的只读 discovery**人工成本可以支持上述 bounded RDC，不能借“减少复制”重验已有事实或执行 mutation。

完整 current Generation / Reviewer Bundle 的 **Generation semantic RDC calls = 0，Reviewer semantic RDC calls = 0**；Coordinator 不为同一正式输入重新扫描 repo。用户返回以下终态，且 identity / SHA / counts / stop point 与合同一致时，直接作为当前 machine evidence 继续：

```text
===== FINAL =====
RESULT=PASS
```

除输出矛盾、缺项或之后发生相关 mutation 外，禁止 remote re-verification。相关变化先按下节 Historical PASS First 区分历史完整性与当前 eligibility，不把 implementation bytes drift 自动归为 semantic invalidation。这只是消费已执行的正式 machine evidence，不跳过下一阶段规定的 current-check、validation 或其他 gate；下一阶段的必要命令继续默认交 Local terminal 执行。

普通 workflow step 最多 **一个合并 readonly RDC call**，优先一次读取 canonical docs + relevant implementation/tests，以及必要的 raw/current-state facts，不拆成多轮 `rg/find/cat/sed/jq/stat/hash` discovery。第二 call **仅在第一 call 新暴露具体、阻塞安全推进的未知 fact 时允许**，不是固定预留的结果读取 call；不用 `start_process` + `read_process_output` 轮询可交 Local terminal 的任务。

**RDC readonly diagnosis 不等于 repository mutation。** readonly 授权不自动授权 write、Git mutation、activation、import、record、finalization、Production 或其他 lifecycle mutation。remote mutation 仍须维护者明确授权并遵守既有 workflow、角色与 mutation/recovery 边界；不能由缺事实、读文档或 diagnosis 推导写权限。

任一 step 实际使用 remote call 时，handoff 增加以下说明；它们是交接文本，不是 schema、state 或 receipt 字段。0 call 时无需增加噪音：

```text
REMOTE_CALLS_USED=<n>
REMOTE_CALL_REASON=<missing-machine-fact|failure-diagnosis|canonical-doc-read|maintainer-request>
```

本策略不降低 correctness、currentness、exact-set、provenance、machine validation、Independent QC、A-only、Surface、Production 或 HUMAN gate，也不改变 shared lifecycle、角色隔离、正式模型政策、initial batching、Reviewer JIT、exact revision、恢复/复用、current authorization 或 structured lifecycle。

## Historical PASS First 与 Recovery Decision Order

本节收紧既有恢复、复用和工程投入原则，不新增 lifecycle、schema、authority family、receipt、CLI、persistent state、recovery subsystem、generalized reconstruction 或 orchestration framework，也不新增运行权限。不得修改已有合法 historical evidence；fail-closed、tamper/corruption detection、exact identity/hash、source semantic invalidation、current validator、current authorization、Independent QC、A-only、Surface、activation/Production gates 与 mutation=unknown 停止线均保持不变。本节不授权忽略机器失败或把未通过的 current gate 报告为 PASS。

### Historical PASS First

**Historical Evidence / Current Eligibility Separation：历史负责证明“过去发生过什么”；当前系统负责决定“现在能不能继续 lifecycle / 上线”。两者不得默认互相绑定。** Historical evidence 与 current eligibility 是两个独立问题。

合法、完整且 identity / SHA / counts / stage / stop / provenance 与正式合同一致的 historical `FINAL=PASS` 默认可信，证明当时的 Generation、validation、Reviewer、finalization 或 lifecycle 成果真实存在且未被篡改，直接复用已确认 evidence。Current eligibility 优先依据当前 canonical source semantic state、current candidate、current scope/membership、current validator、必要的 current language authority 与 current Production gates。历史 invocation 的合法性不等于下一 consumer 的 current eligibility；当前不再 eligible 也不自动否定过去的合法成果。新阶段不能继承未获得的 QC/A 或跳过它必须消费的 gate。

只有明确证据才可否定对应历史 evidence 的完整性，或阻断其在当前 scope 的继续使用：source semantic change；candidate/hash/provenance tamper；membership/scope mismatch；active validator 对当前候选真实失败且该 failure 是当前 lifecycle 必须消费的 gate；conflicting formal successor；formal artifact corruption；或明确的 post-checkpoint semantic invalidation。结论必须绑定实际受影响的 Unit、artifact 和 consumer，不把 ancestor 仅作为 immutable historical evidence 的读取等同于重新选择它为 current candidate。

**History 是 cold path。** Historical reconstruction / replay 默认属于 cold path，不是正常 current lifecycle 的 hot path。只有下列情形才默认允许进入 historical path，且读取/重建范围必须限于当前判断不可缺少的历史事实：

- explicit source semantic invalidation；
- candidate/hash/provenance tamper；
- formal corruption；
- membership/scope mismatch；
- conflicting formal successor；
- recovery 必需的历史事实缺失；
- 当前 safety/correctness gate 明确存在不可替代的 historical dependency。

以下变化本身不得否定 trusted historical PASS，也不得自动触发 historical reconstruction：current implementation 无法重建 historical intermediate bytes；exporter/decoder/producer implementation SHA 改变；current code path 与 historical invocation 不同；mutable current pointer 合法推进；storage representation/storage form 改变；filesystem checkout mode、umask 或 local permission environment 改变且 semantic bytes 未变；以及其它 current implementation drift。**implementation drift ≠ semantic invalidation**。环境变化若触发现行 safety gate，仍须按该 gate 停止并诊断，不能由 bytes 未变推导为当前安全。

不得仅因这些变化重做已可信的 Generation、改写旧 evidence，或直接启动 historical reconstruction。若既有 checker 因仅 reproduction drift 阻断，应先确认 consumer 合同、是否确为 Production 必需以及是否有最小合法路径；发现真实 bug 不自动授权继续修复。不得 force/ignore 机器失败，也不得把历史事实误报为 current-authorized。Formal Command Contract v3 的 first-real-use verification 仍独立适用于新/修改 command contract，其失效不等于旧内容 PASS 被撤销。

### Recovery Decision Order

任何 recovery/blocker 必须依次处理：

1. 从已有 exact evidence 找最近的合法 trusted checkpoint，保留原 identity、SHA、counts、stage、stop 和 mutation provenance。
2. 判断 checkpoint 后是否存在明确 semantic invalidation，区分 implementation reproduction failure、historical integrity 与当前 consumer eligibility。
3. 若无 semantic invalidation，优先从该 checkpoint 继续既有 lifecycle，不重做可信成果。
4. 若有，只 reopen 既有正式 dependency/context 合同确定的 exact affected scope，不人为扩大或缩小。
5. 优先使用已有 recovery capability，按现有 currentness、provenance 与 write-once/replay 合同恢复。
6. 只有前五项均无法合法继续时，才允许考虑新 infrastructure，且仍须通过下述 admission gate。

不能一看到 “current implementation cannot reproduce historical invocation” 就跳到 historical reconstruction。没有合法继续路径时保留现场、报告唯一 blocker 和停止点，不用新 infrastructure 掩盖未知 provenance，也不绕过现有 fail-closed gate。

### Infrastructure Admission Gate 与 One-off Capability Rule

新增 schema、authority family、persistent receipt、recovery subsystem、reconstruction framework、AST/IR/static-analysis framework、global/persistent cache、新 currentness/CAS mechanism、historical producer/emulator 或 generalized decoder/proof framework 前，必须显式重新评估最小合法路径和工程收益。在前述 recovery 顺序已耗尽合法路径的前提下，至少满足以下一项，才值得继续考虑：

- **A：长期复用收益。**明确会跨 locale/content package/多次 workflow 长期复用，累计收益显著；不能仅以“通用设计”推测未来价值。
- **B：实际安全风险。**不实现将造成实际 data loss、unsafe mutation，或 Production correctness/security 风险。
- **C：真实上线硬 blocker。**它是当前 Production/activation 的真实硬 blocker，且没有更小的合法路径。

一次性 historical recovery、one-off command 或 synthetic fixture 默认不足以通过 gate。预计只使用一次或极低频使用的 capability，默认限于 readonly diagnosis、invocation-local helper、bounded one-off proof、小型局部 patch；未明确通过 gate，不得增加 persistent framework、generalized schema、authority family、long-lived reconstruction subsystem 或 broad abstraction layer。

评估依据写入既有交接文本即可，不新增 checkpoint/receipt 格式。gate 通过不扩大当前任务授权或取消更窄的停止线；不能把获准局部 patch 自动升级为新 subsystem。

### Mandatory Complexity Reassessment 与 Sunk Cost Rule

满足任一条件，立即停止当前 implementation 并重新评估：

- 单一 local blocker 已累计消耗约一个工作日仍未解决。
- blocker 产生第二层 infrastructure blocker。
- 为测试 recovery 又需要新的 recovery capability。
- 需要新 formal schema/authority。
- 需要 AST/IR/static analysis。
- 需要 historical implementation reconstruction。
- synthetic fixture 本身开始要求新的 production capability。
- 实际 translation/lifecycle progression 长时间停滞，而 infrastructure 工作持续增长。
- 修复一个 gate 的直接结果只是暴露另一个独立 historical/recovery/infrastructure gate，业务 lifecycle 未至少推进一个正式阶段。

**Gate-chasing stop-line：**出现 `trusted checkpoint → historical decoder blocker → historical pointer blocker → storage-form blocker → historical corpus blocker → another recovery blocker` 这类链条时，立即停止并执行 Complexity Reassessment，不能解释成“继续逐个修复即可”。一个真实 bug 存在不等于它必须在当前 Production 前修复：明确区分 `BUG_EXISTS` 与 `PRODUCTION_BLOCKER_REQUIRES_FIX`，只有后者才能继续占用上线前工程预算，且仍须满足当前任务授权与 Infrastructure Admission Gate。当前 gate 未通过时仍停止该 consumer；停止继续工程不代表 gate 已通过。

非绑定案例：近期一个 trusted validated checkpoint 进入 Candidate Snapshot 时，连续暴露 historical decoder、filesystem mode、Shell representation、storage form、historical mutable pointer 等 gate，未发生 translation/source semantic invalidation。该经历说明 gate-chasing 必须触发复杂度停止；它不是新的 formal lifecycle，不为任何具体 candidate 或 command 授予 eligibility，也不绑定特定 locale、SHA 或路径。

重评估必须回到最近 trusted checkpoint、明确 semantic invalidation、现有 capability、最小合法路径和 admission gate；不能继续沿 blocker 链自动扩张。已经投入的时间、代码量、测试量、Codex quota 都不是继续 subsystem 的理由。当前最短合法路径已不再需要它时，允许保留现场并停止，不为证明旧投入值得而补完。可信成果保留；任何回退仍须遵守 exact own-hunk 与 formal artifact 不可改写边界，不能广泛清理 worktree。

### Codex Budget Rule

Codex 是受限工程资源。正常 deterministic workflow、readonly inspection、currentness reassessment、contract verification、formal command execution、Git checkpoint/push 默认交 Maintainer Local terminal，Codex=0；Call-Zero-First 的 bounded readonly discovery 例外和已授权工程的必要局部读取仍按原边界执行。

Codex 优先用于 correctness bug、reusable workflow blocker、activation/Production blocker，以及有明确跨 locale 长期复用价值的工程。禁止顺手重构、unrelated fixture 清债、broad test cleanup 和无上线价值的 architecture perfection。

以周总额度为口径，日常单日消耗目标约 **8%–12%**，单日 **hard cap 15%**，尽量长期保留约 **20% emergency reserve**；同时遵守平台 5h/weekly 实际限额。hard cap 不是 daily target，不为用满目标增加工作。达到 cap 即保存可信现场并停止 Codex 工程，确定性步骤交 Local terminal；额度恢复不是继续已无必要 subsystem 的理由。预算只约束投入和分工，不降低 validator、独立 QC/A-only、currentness、provenance、Production 或 HUMAN gate。

### Translation Throughput / Production Priority

本项目首先是 translation/localization workflow，主要目标是完成 **canonical source → Generation → validation → Independent QC → finalization → Integrated Surface → activation → preview → Production**，各阶段仍消费原有必要前置条件。若 translation/lifecycle progression 少于 infrastructure 工作，视为 warning，必须检查基础设施投入是否仍服务最短合法路径。

项目目标不包括要求 current implementation 完整重放所有 historical implementation states。完整 historical replayability 不是 translation quality 或 Production correctness 的默认必要条件。历史机制主要服务 audit、recovery、tamper/corruption detection、trusted output reuse 和 avoid duplicate work；不得让 historical machinery 本身成为正常 current workflow 持续前进的默认前置条件。当前 safety/correctness gate 的不可替代 historical dependency 仍按上述 cold-path 条件处理。

当前 zh-CN Site v2 优先完成：**Candidate Snapshot → Independent QC → finalization → package closure → Integrated Surface → activation → preview → Production**。按既有 canonical 前置关系和全部 gate 推进；不把建设能解释所有历史 implementation 状态的通用翻译平台作为上线前置目标。本节不授权立即执行其中任何真实 workflow。

### Workflow operability warning

Maintainer 或 Coordinator 在一次正常 FAILED 后，若需要连续多轮 exploratory commands 才能确定唯一下一步，应视为 workflow complexity warning，先重新评估当前路径。不能默认通过更多 command、wrapper、receipt 或 proof 解决 operability 问题。尤其多 locale workflow 必须尽量保持 machine-decidable、bounded、single-next-step；本原则不授权新增 orchestrator、schema、registry 或 framework。

## Formal Command Contract v3

本节是 formal command 的通用交接标准，具体命令合同仍归各自 canonical docs；不新增 receipt schema、持久化 registry 或运行权限。集中字段说明见 [Package closure](SITE_V2_WORKFLOW.md#package-closure-canonical-command-contract)、[Surface scope](SITE_V2_WORKFLOW.md#surface-scope-canonical-command-contract) 与 [Shared re-attestation plan fields](SITE_V2_WORKFLOW.md#shared-re-attestation-plan-field-semantics)；其它实例见 [Activation](SITE_V2_WORKFLOW.md#activation-canonical-command-contract)、[Integrated Surface](SITE_V2_WORKFLOW.md#integrated-surface-reviewer-bundle) 与 [Shell representation command matrix](SHELL_REPRESENTATION_SUCCESSOR.md#command-matrix)。不要求机械迁移所有旧命令。

### New/modified formal CLI first-real-use gate

`NEW_OR_MODIFIED_FORMAL_CLI` 包括新 command/subcommand、新 mutation mode/`--apply`/`--record` 路径，以及 input type、path validation、flag semantics、stdout/result schema、currentness、idempotence、prior/CAS、concurrent mutation、recovery/replay semantics 的变化。**影响这些 contract 的 implementation changed → previous command-contract verification cannot be carried。**普通不改变 contract 的 bugfix 不机械重触发。

Coordinator/engineering 在实现完成后、第一次真实使用或 handoff 前，必须按顺序重新读取：

1. current canonical command docs；
2. current implementation（含 parsing、dispatch 与实际 reader/publisher）；
3. direct CLI parsing/contract tests。

canonical repo docs/code 是 authority，direct tests 是可执行 contract evidence。三者不一致时 **fail closed**，先修 contract drift；不能选择“更合理”的一方继续。读取 tests 不授权运行 broad regression。过去别的 formal CLI 已 harden、旧 session 记忆或相似命令经验，均不能跳过此 gate。无相关 contract 变化且有 current verified evidence 时继续复用，不要求每次 invocation 重读。

**命令名不能推导 operation。**`*-check`、`*-preflight`、`*-inspect`、`*-record`、`*-apply` 的名称不能代替 operation/input-kind/mutation contract；禁止用其它 family 的行为类比。尤其 Shell `shell-representation-successor --input <plan>` 是 plan inspection，`shell-representation-check --input <receipt>` 是 receipt verification。

每个 command/mode 的 canonical contract 必须无歧义给出以下信息（字段可沿用各文档的既有名称，不强制统一表格）：

| 信息 | 必须说明 |
| --- | --- |
| COMMAND / OPERATION | exact family、mode 及真正操作 |
| INPUT_KIND / INPUT_PATH_CONTRACT | plan/receipt/closure/bundle/snapshot 等类型，以及 repository-relative 或 absolute-external 等路径合同 |
| REQUIRED_FLAGS / FORBIDDEN_FLAGS | 必填、互斥、禁止的 flags，以及 input-derived scope |
| STDOUT_KIND / SCHEMA | plain plan、receipt、result wrapper 等 exact 输出及字段含义 |
| MUTATION_EXPECTATION | none/known；写入的 exact scope，失败时 unknown 边界 |
| CURRENTNESS_BEHAVIOR / IDEMPOTENCE | 重算/校验哪些 authority；exact replay 与 no-op 条件 |
| PRIOR / CAS / CONCURRENT_MUTATION_BEHAVIOR | 不适用明确 none；适用时 exact 字段、absence 表达及 mutation 检测 |
| SUCCESS_NEXT_STEP / FAILURE_STOP | 成功唯一下一步/停止点，失败不得继续的边界 |
| RECOVERY / REPLAY | 保留哪些 evidence、可重试条件、禁止覆盖/自动修复的状态 |

路径不能只写 `repo-local`、`local file` 或 `input file`。实际要求 repository-relative path 时必须明确如此，并给正反例：`evidence/foo.json` 合法；`/home/user/repo/evidence/foo.json` 非法，即便 absolute path 位于 repository 内。其它路径类型按其实际 contract 写明，不能为文档方便放宽 safety validator。

第一次真实使用 new/modified formal CLI 的 Coordinator handoff 必须填完以下模板；这是 workflow 文本，不是正式 receipt schema：

```text
COMMAND_CONTRACT_VERIFIED=<yes|no>
COMMAND=<exact command family>
OPERATION=<exact operation>
INPUT_KIND=<exact input kind>
INPUT_PATH_CONTRACT=<repository-relative|absolute-external|none|exact other contract>
MUTATION_EXPECTATION=<none|known>
IDEMPOTENCE=<exact replay/no-op semantics>
CURRENTNESS_CHECK=<exact checks>
PRIOR_OR_CAS=<none|exact fields and absence semantics>
SUCCESS_STOP=<exact stop>
FAILURE_STOP=<exact stop>
```

任一信息无法由 current docs/implementation/tests 确定时，`COMMAND_CONTRACT_VERIFIED=no`，**禁止输出真实 mutation command**；先按 Call-Zero-First 做 bounded readonly contract discovery。first-real-use gate 是 workflow hard gate；focused regression 固定模板必填字段和 Shell concrete matrix，不声称能自动证明人已阅读或穷举所有 CLI。

### stdout 与 mutation provenance

`already_current=true` 只表示观察到 machine state 已绑定 expected result，不能推出本次 invocation 执行了 apply，也不能证明之前 handoff 的 mutation provenance 正确。`recorded=false`、`applied=false`、`mutation=none` 必须按当前 invocation/stage 的 contract 解释，不可跨阶段推导。machine state 与先前合法 FINAL 的 mutation/provenance 声明矛盾时，**STOP → mutation provenance diagnosis**，不得继续 downstream；不能用 idempotent no-op 掩盖未知历史写入。

### Wrapper / verifier 与 mutation provenance

Shell/Python wrapper 只能按 current canonical command/schema 验证字段；command name、flag name 或历史调用顺序不授权推导 schema 未规定的关系。operation、input kind、stdout schema 与正式 artifact bytes 分别核对；不得给现有 command 增加 wrapper 自创的 acceptance gate。

| 错误推导 | canonical 解释 |
| --- | --- |
| `structured_baseline_closure must equal previous successor closure` | must equal the exact frozen baseline selected by canonical closure reconstruction；[successor 运输 current exact delta set](SITE_V2_WORKFLOW.md#frozen-structured-baseline-与-successor-delta)，不按输出先后自动串联 baseline |
| `changed_dependencies must contain every changed re-attestation validation file` | currentness dependency drift and `validation_inputs` are separate fields/scopes；[两组 field semantics](SITE_V2_WORKFLOW.md#shared-re-attestation-plan-field-semantics) 不允许用 `CHANGED_DEPENDENCY_COUNT` 推断所有 validation bytes |

wrapper `POSTCONDITION` failure 不自动意味着 formal command failure。若 formal artifact 已成功发布，先保存 command exit/stdout、wrapper failure、artifact/inventory 与 provenance，再诊断真实 mutation；不得删除已发布 artifact 或盲重放。交接按以下含义报告，不能仅从 shell 最终 exit code 分类：

| 报告值 | 必须支持的事实 / 停止边界 |
| --- | --- |
| `MUTATION=none` | 同一 exact formal scope 的 before/after 完整 path set 与 bytes inventory 证明无新增、无删除、无 byte change；仅 command/wrapper FAILED 或缺 stdout 不能证明 none |
| `MUTATION=known` | 成功 publication 或可精确证明的 formal writes；报告真实写入 scope。repository bytes 已证明的 known mutation 不会被 wrapper verifier FAILED 覆盖，也不等于所有 downstream gates PASS |
| `MUTATION=unknown` | terminal state/evidence 无法证明真实写入终态；preserve/diagnose，禁止 replay 或 downstream mutation，不能猜 none |

`known` / `unknown` 的失败都保留原 formal evidence；修 wrapper 或补 diagnosis 不授权重写 receipt/closure、清理现场或自动重新执行 lifecycle。idempotent exact replay 的可能性不代替 mutation/provenance diagnosis。

### Formal artifact mutation accounting

声明 `FORMAL_ARTIFACTS_CHANGED=no` 必须有同一 exact scope 的 before/after **完整 path set 与 bytes** 对照：existing paths unchanged、new paths=0、removed paths=0。仅证明 N 个旧文件未变不足以覆盖新增文件。交接报告：

```text
FORMAL_ARTIFACT_PATHS_ADDED=<count>
FORMAL_ARTIFACT_PATHS_REMOVED=<count>
FORMAL_ARTIFACT_BYTES_CHANGED=<count of retained paths with changed bytes>
```

三个 count 全为 0 且 inventory scope 完整、读取成功，才能报告 unchanged；缺失任一侧 inventory/读取失败为 unknown，不得当空集。可复用 `contentidentity.CompareArtifactDigests` 比较已有完整 SHA-256 inventory；它只比较 exact path set/digests，不扫描或推测 scope，不是第二套 artifact inventory framework。48646 个旧文件不变加 2 个新 formal paths 仍是 mutation。symlink/非 regular file 等读取异常必须停止，不伪装成删除或 unchanged。

## Bundle 冻结输入与工具回退

revision Generation handoff 的完整性包括被审核的 predecessor candidate，而不仅是 protected English inputs 和 finding；候选必须由 exporter 按 predecessor Snapshot 精确冻结，见[Generation Bundle contract](TRANSLATION_TASK_SPEC.md#deterministic-generation-bundle-transport)。缺 candidate 的 Bundle 应 FAILED 且不生成 outputs；无正式 raw responses/import 时优先修复合同后重新导出同一 batch，不新建 batch、不手工 patch ZIP。本段不授权绕过 current-check 或覆盖已有 evidence。

revision/revision-origin retry handoff 还必须区分 restored candidate 与可编辑 protected predecessor baseline：前者对照 finding，后者来自正式 passed validation 绑定的原 raw response/attempt；Generation 输出始终为 protected raw response。machine retry 保留原 revision semantic inputs，不因 `task_kind=translation-unit-retry` 丢失。已 import/process 的 restore/validation failure 保留原 attempt 全部证据，按失败 Unit 导出下一连续 attempt 的单 Unit retry Bundle，不覆盖、不重做成功 Unit。

正式 Generation / Reviewer Bundle 完成对应 current-check 并上传到目标模型 session 后，是本次 invocation 的冻结完整正式输入。Bundle 已包含的 source、candidate、glossary、finding、manifest 与 authority 必须从上传附件或其本地解压内容完整读取；Generation 与 Reviewer semantic RDC calls = 0，Coordinator 也不得再次扫描同一 repo 输入。缺项或 currentness 真实变化时 fail closed；Coordinator 必要的 readonly diagnosis 按上节执行，正式检查、补齐或重导仍默认交 Local terminal，不通过 RDC 补读来绕过完整 Bundle 输入边界。transport/landing 与 machine validation 本身不是 remote call 理由。结果落地或记录前仍须由正式 checker 执行规定的 current-check，并直接消费其合法 terminal evidence。

Local terminal 是正常 deterministic workflow 的默认执行者，不只是 RDC 不可用或额度耗尽时的回退。ChatGPT Generation / Reviewer 使用完整 current Bundle 完成语言工作；会话协调提供完整可复制命令并消费维护者回传终态，repository engineering 仍交获授权的本地 Codex CLI。RDC 不可用不改变这一默认路径，不降低任何 gate；已有可信 outputs/result/staging 优先复用，不重做。

structured-assets 不使用普通 TU `generation-bundle recover`。已完成真实 Generation 的 stale Bundle 按[Structured carry-forward](SITE_V2_WORKFLOW.md#structured-generation-stale-bundle-carry-forward)交给维护者 Local terminal：先 `site-content structured-recover` 只读 preflight，再对同一原输入、原 output SHA 与原 provenance 明确 `--apply`。交接区分 preflight PASS 与正式 evidence 已落盘，报告原/current Bundle identity、原 output SHA、9/9 等实际 validation counts 与 immutable generation evidence path；不能修改原 output 的 Bundle identity。任何语义漂移或当前 validator failure 均停止，不重做可信输出，不覆盖历史失败。

ChatGPT + RDC 的成本/风险默认值、readonly discovery 选择与调用预算见[具体执行约束](CHATGPT_LANGUAGE_GENERATION.md#rdc-调用预算与回退)。下文 deterministic 命令均默认由 Local terminal 执行；Reviewer Bundle 仍按 JIT 规则逐组导出。

## Reviewer Markdown evidence 格式

QC、re-QC 等新 Reviewer Markdown 必须先写入正式路径之外的临时 draft，再通过同一格式检查以原始字节、原子且不可覆盖地保存到正式目标：

```sh
go run -mod=readonly ./cmd/tour-i18n review-evidence save \
  --input /tmp/<review-draft>.md \
  --output <formal-review-evidence>.md
```

`surface-review evidence-scaffold` 已创建的正式待填写文件不另行复制或覆盖；Reviewer 完成填写后立即只读检查，`PASS` 后才执行正式 `surface-review record-a`：

```sh
go run -mod=readonly ./cmd/tour-i18n review-evidence check \
  --file data/locale-surface-reviews/<locale>/<review-id>.md
```

Glossary Review 如保存独立 Markdown，也使用 `review-evidence save`。两条命令只检查非空、UTF-8/BOM、LF、行尾空白和文件末尾换行格式，不修改 Markdown 内容、不作语言审核。格式失败时，只修复尚未被正式 receipt 绑定的当前 draft 或待审核文件；不得修改历史 evidence，不得自动重新审核。正式目标已存在时 `save` 必须失败，尤其不得覆盖已被 receipt 绑定 SHA-256 的 Markdown。

本文件是 Generation / Reviewer Bundle 的共享 authority。此 authority 修改后继续使用任何在途 ZIP 前，必须先执行该 bundle 对应的正式 current-check；只有检查真实判定 stale 时才按实际受影响范围重新导出。不得仅因本节新增格式规则而重译、重新审核、修改已有 A 结论或改写 finalization。

## Integrated Surface Reviewer handoff

Integrated Surface 使用独立标准入口：`site-content surface-reviewer-bundle --input <current-scope.json> --output <reviewer.zip>`（或 `--locale <locale>` + repeatable `--generation <package-closure.json>`），随后 `site-content surface-reviewer-bundle-check --bundle <reviewer.zip>`。完整模板见 [Integrated Surface command matrix](SITE_V2_WORKFLOW.md#integrated-surface-reviewer-bundle)。不得复用旧 `surface-review reviewer-bundle` 代替 Integrated payload。handoff 报告 Bundle path/SHA/bytes、scope identity/SHA、fresh/carried 实际 counts、CURRENT 结果与停止点；附件内直接读取完整 source/target、glossary、provenance 与 integration context，semantic RDC=0，carried 不重新 rating。输入不足返回 INPUT_GATE failure，不保存 language failed receipt 或 reopen；新输入使用新的 Reviewer invocation/review_id。正常 export/check/record 仍由 Maintainer Local terminal 执行，合法 FINAL PASS 直接消费。

Surface PASS 后的执行交接使用 [Activation canonical command contract](SITE_V2_WORKFLOW.md#activation-canonical-command-contract)：preflight 的 `--input` 是 package closure，stdout 为 ActivationPlan JSON；apply 只消费本包 fresh preflight 的 exact `prior_sha256`。该合同记录 flags、Surface binding、authority 顺序、真实 zh-CN 范例及失败恢复，不以 receipt 充当 closure，也不使用 transition 中的 coverage 猜顺序；交接止于 preflight 前时不得执行示例。

## QC / re-QC Reviewer Bundle 按需即时导出（JIT）

任何存在顺序依赖的多组 QC / re-QC，只导出**当前下一次将交给 Reviewer 的一组**，禁止预导未来 working set 的 Reviewer Bundle。此规则统一覆盖 initial QC、revision 后 re-QC、Learn/Docs 固定工作集及其他存在顺序依赖的 QC working sets。

`quality-check reviewer-bundle` 包含导出时的当前完整 QC scope，不只绑定本组 Unit。前一组 `quality-check record` 后 machine scope 改变，提前导出的后一组 ZIP 会被 `quality-check reviewer-bundle-check` 正确判定为 stale or non-canonical；即使后一组 Unit bytes 未变，也不能沿用该 ZIP 或放宽 checker。

每组按以下顺序执行：

1. 基于当前 machine state 执行 `quality-check preflight`，确认 predecessor、carry-forward、完整 scope 与本次 working set；只为当前下一次 Reviewer invocation 导出一组 Reviewer Bundle，执行 `quality-check reviewer-bundle-check`，PASS 后计算 `sha256sum` 并交接。
2. 独立 Reviewer 完成本组后，执行正式 current-check，并按本文格式要求完成 Reviewer evidence 的 `review-evidence save` / `check`；这些检查通过后才执行本组 `quality-check record`。任何失败均停止，不导出下一组。
3. 只有本组 `record` 成功后，才基于新的 machine scope 为下一组重复上述导出、`reviewer-bundle-check` 与 `sha256sum`。下一组必须使用新的用户请求和新的 Reviewer model invocation，不得在同一次 response 内串联多组。

Learn/Docs `learn-docs-v1` 的 initial Generation / initial QC 固定组 `1–30` / `31–51`（30+21 Page；future append 仍限 `31–60`）不可合并；第一组完成上述正式记录后才导出第二组，两批初审全部完成后才集中 revision。正式 B/C/D 或 Surface reopen revision 不继承 initial partition。successor Snapshot re-QC 按 current `quality_check_required` pending、stable index 顺序、同 kind、最多60选择；多组仍逐组 JIT，禁止预导未来 ZIP。Bundle selector 与 record 均使用解析后的 predecessor lineage 区分 initial QC / re-QC，不用原 batch membership 限制后续修订。

该调度减少无效 ZIP、重复 current-check/export 与 RDC/模型 calls；currentness、exact-set、A-only、Reviewer invocation 隔离及所有既有 gate 均保持不变，不新增 schema、state、receipt 或 CLI。

## 必填交接内容

current re-attestation/checkpoint 的交接另须区分 historical A 与 current machine eligibility，按[共享当前授权 contract](CURRENT_TRANSLATION_AUTHORIZATION.md)报告 exact compatible/affected/stale/unknown IDs、current validation 和是否真实落盘 authorization。affected 或 unknown 时不声称 current-authorized，不执行语言 replacement；返回协调角色处理 exact scope。checkpoint 不重写历史，也不代替 full Unified Review、独立 QC 或其他 consumer gate。

需要跨会话、交给维护者 Local terminal、等待明确继续、进入 HUMAN gate，或因失败停止时，最终回复依次提供：

1. 当前事实：locale、stage、`PASS` / `FAILED`，batch / review / Snapshot ID，真实完成数量，以及正式 evidence path 或 machine output。
2. 唯一下一步：根据当前 machine state 和正式 scope 指定一个动作、执行者与停止条件。有多个恢复选择时先说明阻塞，不猜测选择。
3. 附件 handoff：需要跨会话 ZIP 时，由 Local terminal 真实导出并执行 current-check，AI 消费合法 terminal evidence 后给出真实绝对路径、`sha256sum`、目标 session/role 和完整指令。尚未执行时先提供完整 Local terminal 命令，不 remote 代跑；禁止把占位路径、推测 hash 或未运行的检查写成事实。
4. Local terminal handoff：提供可直接粘贴的完整命令、前置 gate 与完成后的检查方法；不得在当前 shell 顶层设置 `set -e`、`set -u`、`set -o pipefail`，不得包含 `exit`。
5. 暂不能导出时：说明缺少终端、附件通道、current input 或其他真实原因，并给出正确的导出、current-check 与 SHA-256 命令，不能只写“请上传 ZIP”。
6. 失败交接：记录已完成事实、失败 stage、mutation 为 `none` / `known` / `unknown`、最小安全恢复路径。`mutation=unknown` 时禁止盲目重放写操作。
7. remote 调用披露：本 step 实际使用 remote call 时，报告 `REMOTE_CALLS_USED=<n>` 与 `REMOTE_CALL_REASON=<missing-machine-fact|failure-diagnosis|canonical-doc-read|maintainer-request>`；0 call 时省略。
8. TranslationUnit 语义任务指令：Reviewer handoff 必须提醒完整 current TU fresh judgment、B/C/D finding 前的 same-class scan，以及 category、可定位实例或 exact predicate + bounded scope；successor re-QC 还须检查 persisted category 的 scope 内完整 closure。revision Generation handoff 必须提醒 finding-anchored / locally complete 包含同一 category + bounded scope 内的 same-class complete sweep，examples 不默认穷尽修订边界，仅明确 exact-instance-only 时限于列举实例。禁止其他类别、无关润色及扩大正式 Unit/slot 或 protected representation 边界；不能确定是否同类或 scope 不清楚时 fail closed，保持不确定处并报告 ambiguity。细则分别见[Reviewer completeness](TRANSLATION_QUALITY_REVIEW.md#bcd-finding-完整性与-successor-re-qc)与[精确修订语义](TRANSLATION_TASK_SPEC.md#精确修订任务语义)。

Generation 遇到自然句法与 validator 冲突或疑似漏检而需要维护者决策时，失败交接还必须附上完整 source/candidate、protected token identity、真实 restore/validation evidence、预期 parser/render 或 Go 语义，以及至少一个 positive fixture 和保护现有边界的 negative fixtures；同时明确归类为译文缺陷、非法结构/代码变化、受保护内容破坏、validator 误报/漏报，或额外过度预检。只交付具体最小提案，不自行修改生产 validator。维护者未批准规则改造前不得以 retry、硬改自然译文或降低 gate 绕过；获批修复实施后，确定性规则误报按[共享 Revalidate 合同](RETRANSLATION_RUNBOOK.md#4-revalidatevalidator-修复后重验或恢复同一-attempt)重验原 candidate，或从仍 current 且原 invocation/provenance 精确绑定的同 attempt raw response恢复 `restore_failed` Unit。历史 failure 原 bytes 必须归档，不重新 Generation；真实译文缺陷仍停止并决定 exact retry。

闭环内外的正常 deterministic repository / shell workflow 默认均由维护者 Local terminal 按完整命令执行；AI 消费合法 FINAL PASS 后直接按既有顺序给出唯一下一步。附件上传、维护者决策、明确“继续”及 HUMAN gate 的停止边界保持不变；真实 FAILED / unexpected state / mutation=unknown 才进入 diagnosis，remote 调用遵守本合同必要性白名单并披露次数/理由。

## ZIP 交接模板

以下 `/tmp/<...>`、`<sha256>`、ID 与数量均为文档占位符，不是已生成附件：

```text
locale: <locale>
stage: <stage>
result: PASS
identity: batch=<batch-id> snapshot=<snapshot-id>
completed: <actual>/<scope>
evidence: <repository evidence path>
next: 把 <absolute-zip-path> 上传到独立 Reviewer session；完成该 ZIP 的一次独立 invocation 后停止。
bundle: <absolute-zip-path>
sha256: <sha256>
current-check: PASS
task: 对完整 current TU 做 fresh judgment；B/C/D finding 前做 same-class scan，尽量收全已观察实例，或给 exact predicate + bounded scope；successor re-QC 检查原 category 在 scope 内完整关闭，同类残留标为 same-class residual。
```

revision Generation handoff 在相同的真实 Bundle identity/path/SHA/current-check 与唯一停止点之后，使用以下对应角色指令；不得合并 Reviewer 与 Generation 职责。这些是交接文本，不是新增 machine 字段：

```text
task: 从正式 protected predecessor baseline 做 finding-anchored revision；locally complete 不等于只改列举实例。修复明列实例，并在 persisted category + bounded scope 内做 same-class complete sweep；仅明确 exact-instance-only 时限于列举实例。
boundary: 不扩 Unit/slot authorization，不改 protected 内容，不做其他类别或一般性润色，不整页重译；边界或同类归属不确定时保留不确定处并报告 ambiguity / blocker。输出仍为 tokens exact/unique 的 protected raw response，不重建 token mapping。
stop: 完成本次 exact revision 输出后交回既有 validation 与独立 re-QC；Generation 不给自己 rating 或通过决定。
```

只有命令尚不能执行时才交付下列占位命令，并明确标注“尚未执行”：

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check reviewer-bundle \
  --locale <locale> --snapshot-id <snapshot-id> \
  --previous-snapshot-id <previous-snapshot-id> \
  --start-index <stable-index> --limit 60 \
  --output /tmp/<locale>-<snapshot-id>-qc-<stable-index>.zip
go run -mod=readonly ./cmd/tour-i18n quality-check reviewer-bundle-check \
  --bundle /tmp/<locale>-<snapshot-id>-qc-<stable-index>.zip
sha256sum /tmp/<locale>-<snapshot-id>-qc-<stable-index>.zip
```

## 真实流程示例

以下路径和 hash 仍是明确的占位示例；实际回复必须替换为已核验事实。

### Initial Generation validation PASS 后等待继续

```text
locale: <locale>
stage: initial TranslationUnit batch validation
result: PASS
batch: <batch-id>
completed: 60/60 validation passed
evidence: data/retranslation-runs/<locale>/<batch-id>/result.json
next: 维护者确认“继续”后，由 Local terminal 导出下一批 Generation Bundle。
stop: 等待维护者明确“继续”；此前不启动下一次 model invocation。
```

### 每组 QC / re-QC record 后即时导出下一组

维护者 Local terminal 按上文 JIT 规则完成本组正式 current-check、Reviewer evidence save/check 与 `quality-check record`；AI 直接消费合法 terminal evidence，只有 record 成功后才提供下一组 export / `reviewer-bundle-check` / `sha256sum` 的完整 Local terminal 命令。交接给出真实 ZIP 绝对路径、hash、stable indexes 和目标 Reviewer session。Tour 首次默认顺序固定为 Page `1–60`、Page `61–103`、Example `104–122`；Learn/Docs initial QC 保持固定组 `1–30` / `31–51`，revision re-QC 使用上述共享 pending 调度。每组使用新的用户请求/model invocation，不提前导出后续组。

### Revision 后导出 pending Reviewer ZIP

正常 Quality Check B/C/D revision 与 Surface reopen 精确 revision batch 默认且最多包含 60 个 TranslationUnits，Page / Example 分开，并只包含正式 finding 授权的精确 scope；legacy Final Review 的独立 30-Unit 审核限制不变。

新 Snapshot 建立后先执行：

```sh
go run -mod=readonly ./cmd/tour-i18n quality-check preflight \
  --locale <locale> --snapshot-id <new-snapshot-id> \
  --previous-snapshot-id <previous-snapshot-id>
```

只有 predecessor、carry-forward 与待审数量符合实际 revision 身份，才按 JIT 规则导出当前下一次交给 Reviewer 的一组 pending re-QC ZIP，并执行 `reviewer-bundle-check` 与 `sha256sum`；不得一次预导全部 pending working sets。交接停止于独立 Reviewer 的新 invocation，不自动把 replacement 评为 A；本组正式 current-check、Reviewer evidence save/check 与 `quality-check record` 全部成功后，才基于新的 machine scope 导出下一组。

Glossary 变化前按 [Glossary Compatibility](GLOSSARY_COMPATIBILITY.md) 归档原 bytes；新完整 Glossary Review CURRENT PASS 后生成精确 compatibility evidence。protected-input stale 使用 `retranslation export --allow-reexport --glossary-stale`，只导出有真实 input drift 的同 kind Unit，不伪造 QC/Surface finding。新 batch 普通 import/process 后建立当前 full Snapshot；交接列出实际 carried A 来源/compatibility chain、affected/pending exact-set，不能再笼统宣称全量重审或全部自动继承。

### Surface Review PASS 后移交 preview

`record-a` 与 `check-a` 均通过后，AI 不启动 preview。交给维护者以下完整命令，并以 `PREVIEW SURFACE ACCEPTANCE: PASS` 为停止条件：

```sh
go run -mod=readonly ./cmd/tour-i18n preview --locale <locale> --http 127.0.0.1:0
# 另一个终端使用上条命令打印的真实端口：
scripts/verify-preview-browser.py http://127.0.0.1:<port>/ <locale>
```

### 额度耗尽恢复

```text
locale: <locale>
stage: <stage>
result: FAILED
completed: <actual>/<scope>
last current evidence: <path>
failure: generation quota exhausted before <not-started unit/batch>
mutation: none
next: 额度恢复后由原 Generation role 从 <exact batch/staging/evidence> 继续；先核对 raw responses、result.json、validation、Snapshot/QC，禁止重复 import 或覆盖已有输出。
stop: 等待额度，或等待维护者批准安全批次边界上的 provider/model 变更。
```

额度耗尽不降低模型、A-only QC、current/exact-set、provenance 或任何 Production gate。

## 同 Unit representation bridge 的交接边界

restricted transport 的 ordinary QC capability failure 先按 [bridge Runbook](RETRANSLATION_RUNBOOK.md#same-unit-restricted-transport--ordinary-qc-bridge)交接 readonly preflight。报告 exact source/input/Unit identity、reviewed candidate SHA、原 Snapshot/QC evidence、逐 Unit 原 rating/finding、old/new representation identity、planned receipt path/SHA 与 `recorded=false, mutation=none`。工程 proof 不等于正式 receipt；Coordinator 单独授权 record 后才提供 Maintainer Local terminal 的 explicit `--record`。旧零输出 failure、raw、candidate、validation、QC 和 finalization 原样保留；record 后仍只允许新的 finding-anchored ordinary revision，不能重放旧失败 batch、替代语言 QC 或让 Generation 重建 token mapping。
