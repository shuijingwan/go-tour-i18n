# ChatGPT 正式语言生成执行规范

全项目执行共同遵守[共享翻译单元生命周期与五角色协作](SHARED_TRANSLATION_UNIT_LIFECYCLE.md)：所有内容表面复用同一生命周期；页面差异限于适配器；传输安全与逐单元机器校验分层；先恢复、先复用，中文说明，交接只给一个执行者、动作和停止条件。本文中 Tour 的 present / Example 细节属于 Tour 适配边界，不另建 Learn/Docs 状态机。Learn/Docs 两批生成和机器校验全部完成后才初审，两批初审全部完成后集中修订。角色与工具分工不替代任何正式 machine authority 或语言质量 gate。

本文定义某个 locale 在开始正式 generation 前选定 `provider=chatgpt` 后，普通 ChatGPT **GPT-5.6 Sol + High** 使用完整 current Bundle 执行正式语言生成、维护者 Local terminal 默认执行 deterministic workflow 时的仓库边界。它与 [Codex 正式语言生成执行规范](CODEX_TRANSLATION.md) 并列；不新增 Translation Engine，也不改变任何 validation、Quality Check、Locale Surface Review 或 Production gate。

## 单 locale 固定会话角色

所有 Generation/Reviewer/Local terminal 阶段交接统一遵守 [正式阶段交接合同](WORKFLOW_HANDOFF.md)：默认提供完整 Local terminal 命令并消费合法 terminal evidence，再交接已真实生成的 current attachment；remote 例外遵守 Call-Zero-First；不得编造路径、hash 或 CURRENT。

一个 locale 默认维护一个长期 **Generation session**、一个与之独立的长期 **Reviewer session**，再由维护者 **Local terminal** 默认执行闭环内外全部 deterministic repository / shell workflow，与预计耗时无关；AI 提供完整命令并消费终态。这是协作职责规则，不是机器身份系统；不得为此新增 session 字段、receipt、schema、CLI flag 或 machine gate。

### Generation session

Generation session 负责所有产生或修订该 locale 语言内容的模型工作：

- locale glossary 的语言判断与制定；
- 公共 UI catalog、article metadata 与其他 locale-level 文案的生成或修改；
- TranslationUnit initial translation、revision，以及确实需要新译文的 `restore_failed` / `validation_failed` retry；
- schema v2 Course SEO localization，以及 Surface Review finding 所需的 refresh / revise description；
- Glossary Review、TranslationUnit Quality Check 或 Locale Surface Review finding 回流后的 replacement generation。

同一个 Generation session 可以按 `glossary → Glossary Review PASS → UI / metadata → TranslationUnit initial / revision → Course SEO localization → 后续 replacement generation` 长期连续工作，只要它始终承担 generation。它可以读取完整 source、target、glossary 与 reviewer finding；Course SEO generation 可以读取当前完整 Page source 与 ready target。读取这些材料不会破坏 generation 独立性，真正禁止的是同一 conversation/session 同时承担 generation 与 formal review。

Generation session 不得执行正式 TranslationUnit Quality Check、revision 后正式 re-QC、Locale Surface Review，或批准自己生成的输出。

Generation session 制定完整 glossary 后必须停在 Glossary Review 边界；只有 generation-independent Reviewer PASS 且维护者 Local terminal 记录并检查 current gate 后，才正式生成 UI / article metadata 并进入 TranslationUnit generation。若 Reviewer 返回 failed finding，Generation session 负责修订 glossary，并在继续下游 generation 前交回同一独立 Reviewer session 完整复审。

### Reviewer session

Reviewer session 负责该 locale 的正式语言审核：

- Glossary Review；
- TranslationUnit Quality Check；
- revision 后 re-QC；
- Locale Surface Review。

同一个 Reviewer session 可以先承担 Glossary Review，再承担 TranslationUnit Quality Check，之后继续承担同一 locale 的 Locale Surface Review。这三个 gate 的范围和 evidence 保持独立，但仓库不要求它们使用不同 reviewer conversation/session。前提是该 Reviewer session 从未参与该 locale 的 glossary、UI catalog、article metadata 或其他 locale-level language generation，从未参与 TranslationUnit initial / revision / retry generation，也从未参与 schema v2 Course SEO localization / revision generation。

Reviewer 发现问题时必须遵循：

```text
reviewer finding
→ Generation session 产生 replacement
→ Maintainer Local terminal 执行既有 deterministic lifecycle，AI 消费合法终态
→ Reviewer session re-review
```

Reviewer session 不得直接修改 candidate、生成 replacement translation 或 Course SEO replacement description 后再批准自己的结果，也不得因为自己审核过上一轮而自动批准 revision。只要它没有参与 replacement generation，就可以继续审核 Generation session 修订后的新输出。Validation passed、旧 A、Surface Review 结论或模型历史表现都不能替代当前正式审核。

TranslationUnit Quality Check 与 revision re-QC 的单次 Reviewer model invocation / response 最多审核 60 TranslationUnits；Page / Example 分开。60 是实际审核质量边界，不得通过在同一次 response 内串联多个 `<=60` working set 绕过；下一组需要新的用户请求和新的 model invocation。每组必须逐 TranslationUnit 审核，不得抽样。首次 122-Unit locale 的 full QC 推荐使用 Page stable index `1-60`、Page stable index `61-103`、Example stable index `104-122` 三组；revision re-QC 按实际 pending scope 分组，每组最多仍为 60。

### RDC 调用预算与回退

长期执行策略以[Call-Zero-First 交接合同](WORKFLOW_HANDOFF.md#call-zero-first-交接合同)为准：默认 **RDC / repository remote calls = 0** 是成本/风险默认值，不是 zero-call KPI，不机械追求 0。先复用 current conversation、current Bundle 与 sufficient terminal evidence，不以任务 duration 或月度剩余额度决定执行者。完整 current Generation / Reviewer Bundle 直接从附件或本地解压内容完整读取；**Generation semantic RDC calls = 0，Reviewer semantic RDC calls = 0**，Coordinator 不为同一 source/candidate/glossary/finding/manifest/authority 再次扫描 repo。

所有 deterministic repository / shell workflow 默认交 Maintainer Local terminal：AI 根据当前 conversation、checked Bundle 与合法 terminal evidence 提供完整可复制命令，保持依赖顺序和 fail-closed gate。用户返回 `===== FINAL =====`、`RESULT=PASS`，且 identity / SHA / counts / stop point 合法时直接继续；除矛盾、缺项或相关 mutation 外禁止 remote re-verification。下一阶段规定的正式 current-check 与 validation 仍默认由 Local terminal 执行，不把消费已有 PASS 当作省略 gate。

Coordinator / engineering 可为安全推进所缺的 exact machine fact、真实 FAILED / unexpected state / mutation=unknown diagnosis、当前 conversation 未包含的 canonical docs / exact command contract，或维护者明确请求使用 targeted readonly RDC。若一次合并读取明显比要求维护者手工执行纯读取 discovery、重复粘贴或多轮回传更高效，允许并优先采用 RDC；failure diagnosis 优先一次读所需 docs + relevant implementation/tests，而不是先转交 `rg/sed/cat/find`。若 RDC 不可用或没有明显收益，再提供合并只读 Local terminal discovery command。这不允许因为“能用 RDC”就调用，不允许重验合法 FINAL PASS、重扫完整 current Bundle 或反复确认已知事实。

普通 workflow step 最多 **一个合并 readonly RDC call**，优先一次读 canonical docs + relevant implementation/tests（只读，不运行测试）与必要 current-state facts，不拆 calls。**second call 仅针对第一 call 新暴露的、具体且阻塞安全推进的未知 fact**；不预留异步结果读取，不用 `start_process` + `read_process_output` 轮询可交 Local terminal 的操作。实际使用 remote call 的 handoff 必须报告 `REMOTE_CALLS_USED=<n>` 与 `REMOTE_CALL_REASON=<missing-machine-fact|failure-diagnosis|canonical-doc-read|maintainer-request>`；`canonical-doc-read` 是正式合法 reason，减少未知事实 discovery 的人工成本按 `missing-machine-fact` 或 `canonical-doc-read` 披露；0 call 不增加字段。

RDC readonly diagnosis 不等于 repository mutation，不自动授权 write、Git mutation、activation、import、record、finalization、Production 或其他 lifecycle mutation。deterministic workflow / lifecycle mutation 仍默认由 Maintainer Local terminal 执行，repository engineering writes 仍交获授权的本地 Codex，remote 写入须有维护者明确授权；角色隔离和正式 workflow 边界不变。

RDC 不可用不阻断 Bundle-local 语言工作或正常 Local terminal workflow；已有可信 outputs/result/staging 优先复用，不重做。repository engineering 仍交获授权的本地 Codex CLI，不改变 Generation / Reviewer 角色隔离、正式模型政策或任何 gate。sequential QC/re-QC 仍[只导出当前下一组](WORKFLOW_HANDOFF.md#qc--re-qc-reviewer-bundle-按需即时导出jit)，previous record 成功后才导下一组，禁止预导未来 Reviewer ZIP。

### 闭环机械操作与 Local terminal

维护者 Local terminal 默认执行初次 bulk、revision 与 retry 的 export / Bundle current-check / hash / import / process / automatic validation，以及 Snapshot / scope / Reviewer Bundle、evidence save/check、QC record / finalize / authorization、locale-level replacement 落盘、Surface Reviewer bundle / record-a、Course SEO description 落盘与正式 assemble / refresh / revise、package closure / activation。ChatGPT 负责 Bundle-local 语言工作、完整命令和终态解释；具备 RDC 或终端能力本身不授权自动 import/process/validation。“短”不改变默认执行者。

Generation session 按既有 locale + batch 隔离 staging contract 成组产出完整 TranslationUnit；每个文件先完整写入临时文件再 rename 为 staging 成品。维护者 Local terminal 完成 exact-set 检查后默认执行 `generation-bundle import --input-dir`，不制作或下载 outputs ZIP；仅确需跨环境传输、目录不可访问或恢复时保留 outputs ZIP / Result ZIP 兼容路径。随后执行正式 process / retry / automatic validation，每批通过后按原停止点等待维护者明确“继续”。目录与 ZIP transport 都不改变 provenance、attempt 或质量 gate。

locale init、promotion、build、preview、browser verifier、publish、shared assets、Production / deploy / verifier、checksum / curl、search closeout 与最终 Git commit / push 同样默认由维护者 Local terminal 执行；不按耗时划分闭环内外。ChatGPT 提供可直接粘贴命令时，不得在当前交互 shell 顶层启用 `set -e` / `set -u` / `set -o pipefail`；需要 fail-fast 时放独立 subshell 或脚本进程。所有脚本内部 machine gate、receipt、bounded retry 与 HUMAN gate 保持不变。

Legacy Locale Surface Review 输入默认由 Local terminal 使用 `surface-review reviewer-bundle` 生成、完成 current-check 并上传给独立 Reviewer。完整 current Bundle 的附件读取满足本次正式输入要求，不再通过 RDC 扫描同一 repo context；任何正式输入或 Bundle authority 真实变化都须 fail closed，按现有 checker 结果重导。附件模式不允许抽样、跳过 Course SEO full-context coverage，亦不改变独立 Reviewer 与 finding → Generation 回流。

Site v2 Integrated Surface 使用独立的 `site-content surface-reviewer-bundle` / `site-content surface-reviewer-bundle-check`，不能用上述 legacy ZIP 替代。Local terminal 只需提供 current scope input，或 locale + prospective closures；标准跨 locale 命令矩阵见 [Site v2 Workflow](SITE_V2_WORKFLOW.md#integrated-surface-reviewer-bundle)，Reviewer 输入与原 receipt 输出合同见 [Integrated Surface contract](INTEGRATED_SURFACE_REVIEW.md)。fresh exact assets 的完整 source/restored target 必须全部在附件中，carried 与 Tour context 仅作 integration comparison，不重新给 carried rating。semantic RDC=0。缺项是 INPUT_GATE failure，不是语言 failed receipt，也不授权 reopen/revision；修复输入后使用新的 Reviewer invocation/review_id。Bundle 通过 current-check 后才交独立 GPT-5.6 Sol High Reviewer；正常 `surface-record` 与后续 gates 保持不变。

canonical English source-description extraction / review 是跨 locale 共享 authority，不归入单个 locale 固定的 Generation / Reviewer 配对；只有其 authority 真正 stale 时才按[课程页正式 SEO Metadata 规范](COURSE_SEO_METADATA.md)执行。provider-neutral 选择、持续使用和例外切换仍由[多语言翻译流程](TRANSLATION_WORKFLOW.md)与[新增 Locale 执行手册](NEW_LOCALE_RUNBOOK.md)规定；不因 Call-Zero-First 改变任何正式语言模型或角色政策。无论 Generation provider 为 ChatGPT 还是 Codex，Reviewer 继续使用独立 ChatGPT **GPT-5.6 Sol + High** session。

## TranslationUnit 正式输入与 export

ChatGPT TranslationUnit 执行仍以 [翻译任务规范](TRANSLATION_TASK_SPEC.md) 为 provider-independent contract。每个 batch 的以下三部分是不可拆分的正式输入：

1. `manifest.json`；
2. manifest 列出的全部 `inputs/*`；
3. `locales/<locale>/glossary.yaml`。

开始写任何 raw response 前，生成 session 必须完整读取三部分；不得用聊天上下文、上一批输入或 glossary 摘要替代仓库当前字节。

ChatGPT batch 必须显式导出为：

```sh
go run -mod=readonly ./cmd/tour-i18n retranslation export \
  --locale <locale> \
  --generator chatgpt \
  [--unit-kind page|example] [--limit <count>] [--id <unit-id> ...]
```

未提供 `--batch-id` 时，`--generator chatgpt` 选择 `chatgpt-<locale>-NNN`；Codex 是兼容默认并使用 `codex-<locale>-NNN`。两种 prefix 共享同一 numeric namespace，latest export、Candidate Snapshot 与 promotion 都按 numeric suffix 判断，不按 prefix 字典序判断。显式 `--batch-id` 保持既有兼容语义；generator 不写入 manifest，也不构成可由机器证明的模型 provenance。

新增 locale 的首次 Page batch 仍使用 60-Page 基线，Examples 独立。初次自动 Page / Example、显式 `--id`、正常 Quality Check B/C/D revision 与 Surface reopen 精确 revision export 均使用默认且硬性上限 60，Page / Example 永不混合；当前正式课程仍只有 19 个 eligible Example，不为凑满批次扩大 scope。retry 范围不变。

若 candidate 生成后 glossary 已经合法修改并重新取得完整 current Glossary Review PASS，只有正式 protector 在当前 glossary 下重建出的 protected input 与 latest processed batch saved input 字节不同的 Unit 才进入 glossary-stale recovery：

```sh
go run -mod=readonly ./cmd/tour-i18n retranslation export \
  --locale <locale> --generator chatgpt \
  --allow-reexport --glossary-stale \
  --id <unit-id> [--id <unit-id> ...]
```

该模式不是 QC revision、Surface reopen、retry 或 revalidation，不使用 stale `--previous-snapshot-id`，也不允许 unaffected Unit。其 Generation Bundle task kind 为 `translation-unit-glossary-stale-recovery`；ChatGPT 仍按当前完整 glossary 与新 protected input 生成 replacement，并以 `provider=chatgpt`、`model=gpt-5.6-sol-high` 走普通目录 import/process。input identity 未变化的 candidate 保持不动。随后必须建立 full Snapshot，并因 glossary identity 变化执行完整 QC；旧 A 不 carry-forward。

initial Page / Example bulk export 后由 Local terminal 生成 provider-neutral Generation ZIP 并 handoff；ChatGPT 完整读取一个附件即可取得 manifest、全部 inputs、完整 glossary 与当前 authority，无需再通过 Remote Desktop Commander 逐文件读取。译文写入 `/tmp/.go-tour-i18n-generation/<locale>/<batch-id>/` 隔离目录；首次 Page / Example、revision 与 retry 都可使用同一目录 import。以下未带 package selector 的 export/current-check 示例仅用于 native/default Tour scope；要求显式 selector 的 current-validation invocation 仍须按其合同传参：

```sh
go run -mod=readonly ./cmd/tour-i18n generation-bundle export \
  --locale <locale> --batch-id <batch-id> \
  --output /tmp/<locale>-<batch-id>-generation.zip
go run -mod=readonly ./cmd/tour-i18n generation-bundle current-check \
  --bundle /tmp/<locale>-<batch-id>-generation.zip
```

registered content package（例如 `learn-docs-v1`）必须从 retranslation lifecycle 延续同一显式选择：`generation-bundle export` 与 `generation-bundle current-check` 两次独立 invocation 都须传 `--content-package <registered-package>`。不能从 batch、receipt、路径或历史 invocation 反推 package scope；省略 selector 不会自动选择 registered package，而会按默认 Tour scope 检查并对 scope mismatch fail closed。Learn/Docs 的 canonical 示例：

```sh
go run -mod=readonly ./cmd/tour-i18n generation-bundle export \
  --content-package learn-docs-v1 \
  --locale <locale> --batch-id <batch-id> \
  --output /tmp/<locale>-<batch-id>-generation.zip
go run -mod=readonly ./cmd/tour-i18n generation-bundle current-check \
  --content-package learn-docs-v1 \
  --bundle /tmp/<locale>-<batch-id>-generation.zip
```

首次 122 Unit 的默认调度固定为 60 Page、43 Page、19 Example 三个 initial batch；每个 batch 使用一次 model invocation，正式落地并取得 automatic validation evidence 后停止，等待维护者明确“继续”。不得在一次 response 串联多个 initial batch。完整顺序与合法恢复边界见[新增 Locale 执行手册](NEW_LOCALE_RUNBOOK.md#新增-locale-默认调度路径效率优先)。

ChatGPT 在隔离 staging 中成组保存完整 TranslationUnit，并检查 expected filename exact set、protected token、glossary、明显未翻译文本、无解释/fence 与 single-LF。维护者 Local terminal 具备输出目录访问能力时执行以下命令；ChatGPT 不 remote 代跑：

```sh
go run -mod=readonly ./cmd/tour-i18n generation-bundle import \
  --bundle /tmp/<locale>-<batch-id>-generation.zip \
  --input-dir /tmp/.go-tour-i18n-generation/<locale>/<batch-id> \
  --provider chatgpt --model gpt-5.6-sol-high
```

该命令直接检查目录，不创建 Result ZIP。确需跨环境传输时，outputs ZIP 仍只是原始输出传输包，先由 `result-pack` 生成正式 `kind=go-tour-i18n/generation-result-bundle` Result ZIP，再使用 `import --result`；历史 Result ZIP 也继续兼容。

`generation-bundle import` 对目录与 Result ZIP 共用 current canonical bundle、全部 input identity、locale/batch/task/attempt、provider/model、exact-set、路径、UTF-8/no-BOM/single-LF、protected restore、glossary 与 machine candidate 检查，并以同一已验证字节安装。导入会将既有 `GenerationResultBundleManifest` 持久保存为 provenance 记录，不新增 schema；记录 provider、model、Generation Bundle SHA-256、input identity、attempt 与输出 hashes。import 不等于 `retranslation process` / `retry` 写出的 automatic validation evidence，更不等于 Quality Check；每批通过 automatic validation 后仍停下等待维护者明确“继续”。

Reviewer ZIP 继续由维护者上传给独立 Reviewer。只有确需兼容 outputs ZIP / Result ZIP 或 Reviewer ZIP 时，才涉及当前产品所需的附件上传/下载；本流程没有无需用户操作即可跨 ChatGPT / Ubuntu 或跨 session 双向传送附件的通道。

恢复执行前先核对 `raw-responses/` 或 retry attempt、generation import provenance、`result.json`、`validation/`、Snapshot 与 QC evidence。若正式输出已经 import，直接从缺失的 process / retry 或后续 evidence 继续；若 automatic validation evidence 已 current，则不得覆盖输出、重复 import 或重放 validation。若产品消息流/模型 invocation 在 import 前中断且正式输出尚不存在，不得立即让 ChatGPT 重做整个 batch：先对仍 CURRENT 的 Generation Bundle 运行 `generation-bundle recover`，把旧 hidden staging、临时单文件和可识别的 interrupted tar/tar.gz artifact 作为一个或多个 `--source`，只把通过 filename、UTF-8/EOF、protected restore、glossary 与 machine candidate 检查的完整 Unit 写入新的 recovery staging；archive 尾部损坏不抹掉此前独立验证通过的完整成员。随后只生成命令列出的 `missing` Unit，补齐 exact set 后再正常 import。CLI 的 stale、conflicting valid candidate、exact-set 与 no-overwrite failure 都是保护信号，不是要求清空可信成果后重来。

### Generation finding closure 执行规范

`translation-unit-revision` 及其 machine retry 的完整 CURRENT Bundle 必须同时包含三层表示：protected source/input、`formal/candidates/<input basename>` 中的精确 reviewed restored candidate、`formal/protected-candidates/<input basename>` 中的 passed predecessor protected raw response，以及 Snapshot/validation、finding authorization、glossary 和 authority，详见[Generation Bundle contract](TRANSLATION_TASK_SPEC.md#deterministic-generation-bundle-transport)。实际编辑从 protected predecessor baseline 开始；finding closure 对照 restored candidate。输出必须是保留全部 tokens exactly and uniquely 的 protected raw response，不得直接输出 restored candidate或自行重建 token mapping。不得从聊天记忆恢复、通过 RDC/repo scan 另取 latest candidate，或从 English input 整页重译替代 exact revision。缺项时正式 FAILED 且不生成 outputs；修复合同后，无正式输出时重导同 batch，有失败 evidence 时仅走下一连续 attempt 的单 Unit retry，保留旧 failure evidence。`task_kind=translation-unit-retry` 不丢失原 revision semantic inputs 或 finding closure 责任。

若 CURRENT Bundle 携带 `formal/comment-revision-transport`，`formal/protected-candidates` 是机器从精确 reviewed candidate 派生的 revision-only 编辑投影，仅开放 current-validation receipt 授权的 teaching-comment bodies；原 passed raw 位于 `formal/historical-protected-candidates`。使用投影中现成的 body 文本与 token bindings，不重建 mapping、不编辑 opaque boundaries、不改其它代码/注释/正文。输出仍是同一个 TranslationUnit 的 protected raw response；机器 restore 全 Page 后执行 active validator。三层输入和历史原 raw 都保留，细节见上述 Bundle contract；这不授权 Page representation migration、整页重译或新的 comment TU。

Quality Check、re-QC 或 Locale Surface Review finding 进入 TranslationUnit revision 后，Generation 必须按[精确修订任务语义](TRANSLATION_TASK_SPEC.md#精确修订任务语义)逐 Unit、逐 category 完成 closure。**finding-anchored / locally complete 不等于 enumerated-example-only**：Reviewer examples 是 evidence，不默认构成 exhaustive revision boundary；只有 persisted finding 明确标为 **exact-instance-only** 才仅修列出的具体实例。不得只机械修改 listed examples、最严重或最显眼的一项就视为 Unit 完成。

1. 完整对照正式输入中的 source、current candidate、persisted Reviewer finding、glossary 与 protected/structure requirements，识别每个独立 defect category、exact predicate 与 bounded scope。实际 candidate defect 必须修复；source 原貌、protected content、合法结构移动或 validator / Reviewer 误判必须区分，不能一律当作译文缺陷。
2. 先修明列实例，再在**同一 current Unit、同一 finding category、同一授权 scope**内做 **same-class complete sweep**，一并修复其他明确匹配 predicate 的实例；它们未被逐项列出不等于越界。scope 明确为当前 Page 全文时，扫描全文；scope 限于 section/table/comment 区域或 exact instances 时，保持该限制。“local”不是只看 examples 附近几句；同时核对修复直接影响的句段，确保局部完整。
3. 可重复的 category 包括不当 inline code/API/identifier 与目标语言正文粘连、同型标点空格、bounded section 内同一术语错误、同一表格列的同型语义错误或 teaching-comment 局部问题；它们是解释 completeness 的非穷举示例，不自动成为本轮 finding。涉及技术语义时，重新核对 source 的分类、条件与流程关系，不凭 finding 增补语义。不同类别分别处理，不能合并成“改好全文”。
4. 不得借 sweep 修改其他类别、一般措辞偏好、无关标点/style、普通润色或已正确内容，更不得整页重译。predicate/scope 不清楚或无法可靠判断某处是否同类时，fail closed：不猜、不扩大范围，保持不确定处原文并报告 ambiguity / blocker。canonical source、code/identifier/link/directive、protected literal/token、machine semantics 与合法结构继续保持；合法词缀、软换行和允许的完整结构移动不应被机械消除。未授权 slot 或 opaque/protected span 不因同类问题而开放；缺合法 editable location 时报告 representation blocker，不自行 reprotect candidate 或重建 token mapping。对保持项明确 source/protected/structure 依据，疑似 validator 误判按下节诊断，不改写 Reviewer finding/evidence。
5. 每个 revised Unit 提交前必须完成 **finding closure check**（任务完成性检查）：重新对照 source、revised candidate、persisted finding 与 protected/structure requirements，确认每个 category 的明列实例及授权 scope 内其他明确同类实例均已修复，或有明确依据保持，并检查直接受影响的局部完整性。未关闭或仍有 ambiguity 的问题不能视为完成。检查使用当前完整 Bundle 内的正式材料，遵守 Bundle-local semantic work、语言 semantic RDC calls 默认 0 的边界。
6. finding closure check 不是正式 Quality Check，也不是 self-review：Generation 不得给自己 A/B/C/D、决定通过、替代 Independent Reviewer 或伪造 reviewer evidence。修订仍输出完整 Unit，处置说明不得混入 raw response；后续 machine validation、独立 re-QC / Surface re-review 与 A-only finalization 全部照常执行，不新增 closure schema、receipt、状态或机器 gate。

Transport Bundle 大小与 semantic completion contract 分开：一个正式 Revision Generation Bundle 仍可包含现有 export 上限与 Page / Example 分离约束内的当前完整 revision scope，不因 closure 规则拆成每 3–5 Page 一个 ZIP，也不新增 Generation invocation size 上限。正式 B/C/D / Surface reopen revision 可跨 initial group；Learn/Docs 固定 batch 调度仅约束 initial Generation / initial QC，revision re-QC 使用[共享 pending 调度](SHARED_TRANSLATION_UNIT_LIFECYCLE.md#固定调度与独立审核)。未来若研究 invocation size，须另有 benchmark/evidence；finding closure 不改变 Reviewer invocation 上限或 Reviewer Bundle JIT。

这些例子不成为 hard validator 新规则：部分形式可能来自 source/protected content，当前 evidence 只支持加强 revision closure contract，尚不足以建立可安全泛化、无误报的 machine rule。未来若多 locale 重复出现，可另行依据 evidence 评估 advisory lint / precise validator；现有 validator 与 gate 保持不变。

### Validator 冲突的主动诊断

已完成的 structured Generation 因 execution authority drift 失去旧 Bundle currentness 时，保留原 outputs 与真实 provider/model/session/generated_at。按[Structured carry-forward](SITE_V2_WORKFLOW.md#structured-generation-stale-bundle-carry-forward)由维护者 Local terminal 运行正式 preflight / apply；Generation session 不修改 `bundle_identity_sha256`，不伪造新 invocation，也不因运输/执行说明更新而重新生成。`structured-language/v1` 的语义依赖以[正式 structured language contract](STRUCTURED_GENERATION_SEMANTICS.md)为准，执行说明不覆盖语言语义；真正 source/slot/glossary/semantic authority drift 仍 fail closed。

长期 ChatGPT Generation session 在目标语言自然句法与 validator 冲突、或怀疑 validator 漏检时，必须按 [翻译任务规范](TRANSLATION_TASK_SPEC.md#generation-执行者主动诊断-validator) 主动完成证据诊断，而不是把所有报错归为译文错误或为过 gate 机械改写自然译文。诊断至少核对完整 source/candidate、全部 protected token、restore/validation evidence，必要时核对真实 Go Present 解析/渲染或 Go 源码语义，并明确区分翻译缺陷、非法结构/代码变化、受保护内容破坏、validator 误报/漏报及 session 自行增加的过度预检。

普通 Page 允许完整 inline-code pair 与完整链接连同自然语言按目标语言语序整体移动，同时保持 link label-target 绑定和全部结构关系；不得对 Page 额外施加 Example 才有的全局固定 token 顺序。Example 仍严格保持 token 顺序、完整 Go 文件和非注释机器语义，只翻译正式允许的 teaching comment 自然语言。

ChatGPT Generation session 可以保存完整复现、positive/negative fixtures 与确定性最小修正方案，优先提出通用修正；只有充分证据证明通用方案不适用时才提出窄化的 locale-specific 规则。它不得自行修改生产 validator 或共享校验规则，必须停在维护者批准边界并交给获授权的仓库级 Codex 任务。不得用不自然空格、删句、伪造 retry、重译已审核通过成果或降低任何语言 gate 绕过疑似误报。只有正式批准并实施规则修复、restore 已成功且原 candidate 已证实有效时，才使用正式 `retranslation revalidate`；真正的语言质量问题继续走 revision 与独立 re-QC。

## 兼容恢复的安全写入

以下是目录与 Result ZIP importer 均不可用时的既有兼容恢复边界，不是正常 remote 执行授权。默认给维护者完整 Local terminal 命令；只有维护者明确要求 remote 写入时才适用于 RDC，并披露 remote calls/理由。首次翻译或 revision 不得逐个文件直接建立正式 `raw-responses/`。在同一 batch filesystem 内执行以下顺序：

1. 确认正式 `raw-responses/` 尚不存在，在 batch 内创建隐藏 staging directory，例如 `.raw-responses.staging/`。
2. 将整个 batch 的每个 Unit 完整输出写入 staging；Page 使用对应 `.article`，Example 使用对应 `.txt`。
3. 对照 manifest 核对 expected filename exact set 与 count，并逐文件检查 protected token、glossary、明显未翻译自然语言、无 Markdown/code fence/解释，以及恰好一个结尾 LF。
4. 只有全部检查通过后，才在同一 batch 内将整个 staging directory rename/move 为 `raw-responses/`。

任何检查失败或执行环境中断时，隐藏 staging 不是正式 raw response，必须保留为未完成工作或安全清理；不得留下部分 `raw-responses/`。正式目录已存在时不得覆盖或合并，必须先查明当前 batch 状态。

Retry 使用同一原子提交思想：先将完整内容写入目标 Unit retry 目录内的隐藏 staging file，核对它满足当前连续 attempt 编号、文件名、protected token、glossary 和 single-LF contract，再 rename 为正式 `attempt-NNN.article` 或 `attempt-NNN.txt`。不得覆盖既有 attempt、跳号或伪造 provenance。

手工直接安装到正式 `raw-responses/` / retry 路径，只作为目录与 Result ZIP importer 都不可用时的兼容恢复路径，默认由 Local terminal 执行；RDC 写入还必须有维护者明确要求；不得与同一 attempt 的 `generation-bundle import` 混用或覆盖既有文件。标准目录 import 使用独立隐藏 staging 作为输入，再由 CLI 安装验证过的字节。无论哪种 transport，`retranslation process` / `retranslation retry` 继续是 candidate、validation evidence 与 attempt lifecycle 的正式 fail-closed authority。

## 非 TranslationUnit 语言资产

这些资产按以下输入边界生成或修改：

- 首次制定 locale glossary：先遵循 [术语治理政策](TRANSLATION_TERMINOLOGY.md) 与 [术语制定指南](TERMINOLOGY_GUIDE.md)，并读取制定术语所需的正式 English/source context；尚未建立完成的完整 locale glossary 不是其自身的前置输入。
- 修改已有 glossary：必须读取完整当前 glossary 与相关正式 source/context。
- 生成或修改 UI catalog、article metadata 等其他 locale-level 语言资产：必须读取对应完整 source/context 和已经建立的完整 locale glossary。

首次 glossary 制定或任何使当前 Glossary Review coverage stale 的修改，都必须按 [Glossary Review 规范](GLOSSARY_REVIEW.md) 由 Reviewer session 完整审核。UI / article metadata 的正式 generation 排在 PASS 之后；Glossary Review 不因这些资产已经存在而自动批准。

所有资产仍须保持现有 key、kind、placeholder、markup、schema 和技术 identity。不得给 `glossary.yaml`、`internal/tour/ui/<locale>.json` 或 `article-metadata.json` 增加 provider/model/generation 字段。Glossary 继续是该 locale 的正式术语 authority，这些资产也继续由现有 validator 与 Locale Surface Review 审核实际内容。

这些 generation 输入优先一次性导出：

```sh
go run -mod=readonly ./cmd/tour-i18n generation-bundle locale-export \
  --locale <locale> --task glossary \
  --output /tmp/<locale>-glossary-generation.zip

go run -mod=readonly ./cmd/tour-i18n generation-bundle locale-export \
  --locale <locale> --task locale-assets \
  --output /tmp/<locale>-locale-assets-generation.zip

go run -mod=readonly ./cmd/tour-i18n generation-bundle locale-export \
  --locale <locale> --task surface-replacement --review-id <review-id> \
  --output /tmp/<locale>-surface-replacement-<review-id>.zip
```

`glossary` 包含完整 locale identity、English/source corpus 与 terminology authority；`locale-assets` 在 current Glossary Review 后包含完整 UI/article source、当前 target 与 glossary；`surface-replacement` 还绑定指定 reviewer evidence 和 current full Surface Review package。开始或记录结果前用 `generation-bundle locale-check --bundle <zip>` 确认 current。此 contract 只运输生成上下文和预期 deliverable，不直接覆盖正式 locale 资产；默认由维护者 Local terminal 按现有 validator/lifecycle 正式落盘，Reviewer finding 仍须返回 Generation session 后再独立复审。

首次 schema v2 Course SEO localization bundle 默认由维护者 Local terminal 从正式 working tree 生成、检查并完成实际 ZIP handoff：

```sh
go run -mod=readonly ./cmd/tour-i18n course-metadata localization-bundle \
  --locale <locale> \
  --output /tmp/<locale>-course-seo-localization-generation.zip
go run -mod=readonly ./cmd/tour-i18n course-metadata localization-bundle-check \
  --locale <locale> \
  --bundle /tmp/<locale>-course-seo-localization-generation.zip
```

ZIP 的 `course-seo-localization.json` 对 Catalog 全部当前 Page 逐页序列化 canonical English description、完整 English Page source、完整最终 ready target、source/source-description/target identity，并绑定 current canonical source-review authority、完整 glossary 与 `locale.json` identity；`formal/source-descriptions.json` 同时保留 canonical asset 的原始正式字节，`formal/` 还包含 glossary 与 locale identity；`authority/` 封装 ChatGPT/Codex generation、Course SEO、workflow 与 `AGENTS.md`。`manifest.json` 为全部文件提供 SHA-256。只要 bundle 来自未变化的正式 working tree 且 manifest/hash 完整，Generation session 直接完整读取附件，不再通过 Remote Desktop Commander 分批读取同一 103-Page context。ZIP 只优化传输，不新增 Course SEO schema、receipt 或 stale identity，也不允许跳过任一 Page；Codex 使用完全相同的 bundle contract。

schema v1/v2 refresh 与 schema v2 revise 使用同一独立 maintenance bundle，仍由既有 CLI 写正式 asset：

```sh
go run -mod=readonly ./cmd/tour-i18n course-metadata generation-bundle \
  --locale <locale> --task refresh \
  --output /tmp/<locale>-course-seo-refresh.zip

go run -mod=readonly ./cmd/tour-i18n course-metadata generation-bundle \
  --locale <locale> --task revise \
  --page-id <page-id> [--page-id <page-id> ...] \
  --finding data/locale-surface-reviews/<locale>/<review-id>.md \
  --output /tmp/<locale>-course-seo-revise-<review-id>.zip
```

refresh 复用正式 refresh 的 exact stale-subset selector，不允许调用者另选 Page。v1 base 的 Bundle 包含每个 selected Page 完整 English source、完整 ready canonical locale target、完整 current glossary 与原 base entry/description/identity，沿用 `course-seo-description-v1` semantic scope；不迁移 schema、不要求或绑定 canonical source-description asset/review。v2 继续以 canonical English description 为唯一 semantic-scope authority。revise Bundle 仍要求 current schema-v2 base、显式 Page subset 与 repository 内 reviewer finding，不扩大到 v1。开始语言工作前完整读取自包含 Bundle，不通过手工 repo scan 拼上下文；`generation-bundle-check` 对 v1/v2 同样重导逐字节比较、验证 inventory/hash 与 exact stale set。下载的 strict `page_id`/`description` JSON 仍交给现有 `course-metadata refresh` / `revise`，模型不得输出 hash/route/schema/provenance；真实 provenance 由调用者明确记录。

schema v2 Course SEO localization 允许并推荐为每个当前 Page 提供 canonical English description、完整 English source、完整最终 ready canonical locale target、完整 locale glossary、locale identity 与 `course-seo-localization-v2` constraints。canonical description 是唯一 semantic-scope authority；source/target 只用于技术语义核对、正文术语一致性、自然表达和实际内容对齐，不授权重新摘要、增删语义或重选重点。同一 session/batch 可处理多个 Page，但每页必须只用自己的 canonical description 决定 semantic scope，不得跨页补充、混合或推断语义。普通 ChatGPT 的真实 provenance 固定记录为：

```text
provider=chatgpt
model=gpt-5.6-sol-high
```

生成 session 只产出 `page_id → localized description` 输入，不得直接手写正式 `course-metadata.json`。维护者 Local terminal 默认把 strict descriptions JSON 正式落盘，并运行现有 `course-metadata assemble` / `refresh` / `revise` CLI 机械生成资产、记录真实 provenance 并通过现有 gate。canonical English source-description 使用当前共享 authority；普通 locale 生成不重新生成它，也不改变其 review gate。

Course SEO revise 对明确 subset 可额外向 generation session 提供 current localized description 与独立 reviewer finding，两者只用于定位和修复语言质量问题，不能取代 canonical description 的 semantic scope，也不得把 finding 当作增加新语义的依据。生成 replacement 后默认由维护者 Local terminal 运行 `course-metadata revise`，机械更新正式资产与真实 provenance，再交由独立 Locale Surface Review session 复审。
