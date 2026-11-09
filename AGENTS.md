# Call-Zero-First（强制 gate）

RDC / repository remote calls 默认 **0** 是成本/风险默认值，不是 zero-call KPI，不机械追求 0。先复用 current conversation / Bundle / terminal evidence；Coordinator / engineering 若一次 targeted readonly RDC 能明显减少维护者手工 discovery、重复粘贴或多轮只读命令，应优先采用。安全推进所缺 exact machine fact、真实 FAILED / unexpected / mutation=unknown diagnosis、当前 conversation 未包含的 canonical docs / exact command contract、避免手工纯读取 discovery 或维护者明确要求，均可按[Call-Zero-First 交接合同](docs/WORKFLOW_HANDOFF.md#call-zero-first-交接合同)执行一次合并 readonly RDC，优先读所需 docs + implementation/tests；第二 call 仅限第一 call 新暴露的具体 blocker。deterministic workflow / lifecycle mutation 默认仍交 Maintainer Local terminal，repository engineering writes 交获授权的本地 Codex；readonly 不授权 remote mutation。完整 current Bundle 的 Generation / Reviewer semantic RDC calls = 0；合法 FINAL PASS 直接消费，除矛盾、缺项或相关 mutation 外禁止 remote 重验，禁止重扫完整 Bundle 或重复确认已知事实。所有 correctness、currentness、exact-set、provenance、machine validation、Independent QC、A-only、Surface、Production 与 HUMAN gate 保持不变。

# Codex 项目导航

全项目共享翻译单元生命周期、五角色协作、中文优先、唯一下一步、恢复/复用优先与工具成本边界，以 [共享翻译单元生命周期与五角色协作](docs/SHARED_TRANSLATION_UNIT_LIFECYCLE.md) 为准。不要因 surface 或 package 目录不同复制状态机；Learn/Docs 先两批生成与机器校验，再两批初审，之后集中修订。

正式阶段完成、失败、跨会话 ZIP 或维护者 Local terminal 交接时，统一遵守 [正式阶段交接合同](docs/WORKFLOW_HANDOFF.md)；AGENTS.md 只保留此导航，不复制详细合同。

**Historical Evidence / Current Eligibility Separation：**历史负责证明“过去发生过什么”；当前系统负责决定“现在能不能继续 lifecycle / 上线”。两者不得默认互相绑定。合法、完整且 identity / SHA / counts / stage / stop / provenance 一致的 historical `FINAL=PASS` 默认可信；当前资格仍由 current source semantic state、candidate、scope/membership、validator、必要 language authority 与 Production gates 决定。Historical reconstruction / replay 默认是 cold path，不是正常 lifecycle 的 hot path；implementation / exporter / decoder / producer SHA 或 code path 改变、无法重现旧 intermediate bytes、mutable pointer 合法推进、storage form 改变，以及 semantic bytes 未变的 checkout mode / umask / permission 变化，本身均不撤销 historical PASS，也不自动触发 reconstruction。**implementation drift ≠ semantic invalidation**。先找最近可信 checkpoint，再查明确 invalidation；无 invalidation 优先继续既有 lifecycle，有则只 reopen exact affected scope。historical path 的准入条件与全部 fail-closed 边界见 [Historical PASS First](docs/WORKFLOW_HANDOFF.md#historical-pass-first-与-recovery-decision-order)。

**稳定维护模式 / Infrastructure Admission：**新 infrastructure 不是默认禁止。正常允许修复导致错误 candidate/evidence/currentness、unsafe mutation/data loss 或 Production correctness/security 风险的 correctness bug，解决无更小合法路径的当前 Snapshot/QC/finalization/Surface/activation/preview/Production hard blocker，以及建设有现实跨 locale/package/多次 workflow 复用证据、能明显降低人工步骤、失败概率或 Codex/维护成本的 bounded capability。仍须优先 existing capability → smallest legal repair → bounded root fix；schema、authority、persistent receipt、recovery family、AST/IR、generalized decoder 不是黑名单，但新增前必须通过既有 A/B/C admission 并证明没有更小合法路径。speculative capability、architecture perfection、generalized historical replay、为单一 historical edge 建 recovery family/IR/generalized decoder，以及仅为统一、美观、去重或沉没成本而继续的大重构默认不做；细则见 [Stable Maintenance Mode 与 Infrastructure Admission](docs/WORKFLOW_HANDOFF.md#stable-maintenance-mode-与-infrastructure-admission)。

zh-CN 首次 Production 后默认进入稳定维护模式：fix-in-place / bounded change，不主动大改 workflow、统一全部历史 recovery 或重写稳定能力；只有多个真实 locale 重复暴露同类 blocker，才考虑抽象或结构性改造。第二、第三及后续 locale 的 success metric 是 lifecycle progression / Production completion，不是 infrastructure completeness；zh-CN 历史迁移不自动成为新 locale 的 historical recovery/reconstruction hot path。

**基础设施停止线：**遵守上述准入，先复用 checkpoint 和已有 capability。单一 blocker 约一个工作日未解、出现第二层 infrastructure blocker、fixture 又要求 production capability，或需要新 schema/authority、AST/IR、历史 implementation reconstruction 时，立即停止 implementation 并重新评估。其余触发条件、one-off capability 边界与恢复顺序统一见 [Historical checkpoint 与基础设施停止线](docs/WORKFLOW_HANDOFF.md#historical-pass-first-与-recovery-decision-order)。已有时间、代码、测试和 quota 投入不构成继续理由；允许保留现场停止，不把通用历史解释平台作为上线前置目标。

**Gate-chasing / operability 停止线：**修复一个 gate 后若只暴露另一个独立 historical/recovery/infrastructure gate，业务 lifecycle 未至少推进一个正式阶段，必须停止并执行 Complexity Reassessment，不自动逐层修复。同时禁止 route-chasing：至多验证一个明显更小、已有合法 contract 支持的替代路径；两条路径同根失败后不再找第三、第四个 workaround。多个 error 不等于多个独立 blocker；root cause 通过既有 infrastructure admission 且有 bounded repair 时，收敛到最小 root fix，普通实现或 targeted test 失败不自动换路，新的独立复杂度层才再次重评估；完整规则见 [Route Convergence](docs/WORKFLOW_HANDOFF.md#route-convergence)。`BUG_EXISTS` 不等于 `PRODUCTION_BLOCKER_REQUIRES_FIX`；只有后者才能继续占用上线前工程预算。正常 FAILED 后需连续多轮 exploratory commands 才能确定唯一下一步，是 workflow complexity warning；多 locale workflow 应保持 machine-decidable、bounded、single-next-step，不默认增加 command/wrapper/receipt/proof。本项目首先是 translation/localization workflow；完整 historical replayability 不是翻译质量或 Production correctness 的默认必要条件。这些原则不授权绕过 current authorization、validator、Independent QC/A-only、Surface、activation/Production 或 mutation=unknown 停止线，也不授权新 schema、authority、receipt、CLI、persistent state 或 framework。

**New/modified formal CLI first-real-use gate：**新增 command/subcommand/mutation mode，或修改 input/path/flag、stdout、currentness、idempotence、CAS/recovery contract 后，旧 verification 失效。第一次真实使用/交接前必须在实现完成后按顺序读取 current canonical command docs、implementation、direct CLI parsing/contract tests；三者一致并填完 [Formal Command Contract v3](docs/WORKFLOW_HANDOFF.md#formal-command-contract-v3) 交接字段，才能输出真实 mutation command。命令名和其他 CLI 已完成的 hardening 不能替代此 gate；contract drift/未知信息先 readonly discovery，fail closed。普通不影响 command contract 的 bugfix 不机械重触发。

Locale 采用版本无关的持续演进模型：v1/v2/v3 是 implementation/content-package milestone，不是语言治理 namespace；统一 corpus、glossary、shared assets 与增量 Surface authority 见 [Site v2 Workflow](docs/SITE_V2_WORKFLOW.md)。

Site v2 architecture、Content Scope、增量 content package、coverage-aware routing、glossary compatibility 与 upstream 扩展以 [Site v2 Architecture authority](docs/SITE_V2_ARCHITECTURE.md) 为入口；V2-D executable contract 见 [Site v2 Workflow](docs/SITE_V2_WORKFLOW.md)。Production machine identity 与内容完成状态分离。V2-C1 upstream sync 当前 deferred，只有维护者明确重开才执行；此前允许按正式 frozen baseline 完成 campaign / Preview / Production，不增加联网 drift gate。Site v2 coding 模型选择不改变下述正式 Translation / Reviewer 政策。

已有真实 structured Generation 因 execution authority 或 validator 演进遇到 stale Bundle 时，先按 [Structured carry-forward](docs/SITE_V2_WORKFLOW.md#structured-generation-stale-bundle-carry-forward)进行只读 preflight。保留原 Bundle/output bytes 与 provider/model/session/generated_at；语义输入 exact 且当前 validation PASS 才由维护者 Local terminal 执行 `site-content structured-recover --apply`，复用既有 StructuredGeneration/closure/Surface readers。不手改 output identity、不伪造新 invocation、不把任意 docs drift 当兼容。structured language-semantic authority 见[正式语义 contract](docs/STRUCTURED_GENERATION_SEMANTICS.md)。

当用户要求评估 locale 商业优先级或选择下一门语言时，Codex 必须先读取 `docs/LOCALE_ROADMAP.md`。用户未明确指定 locale 的新增任务也先按该路线图选择；单门 locale 的正式执行仍以 `docs/NEW_LOCALE_RUNBOOK.md` 为准。

locale 商业选择使用路线图当前冻结的 66-locale cross-project commercial execution order；canonical source 已覆盖的目标跳过，其余相对顺序不变。当前 Go generic `en` 仅在 commercial scheduling 层跳过 `en-US`，从 `zh-CN` 开始，不改变正式 Go source authority 或 65-locale Production scope。project-specific strategic override 只有维护者明确授权时才使用，不修改 base commercial order。

当用户要求新增一门语言、建立新 locale、准备新语言首次上线或评估新 locale 完成度时，Codex 必须先读取：

1. `docs/NEW_LOCALE_RUNBOOK.md`
2. `docs/GLOSSARY_REVIEW.md`
3. `docs/LOCALE_SURFACE_REVIEW.md`

新增 locale 的 TranslationUnit 翻译仍须继续遵守下列不可拆分输入规则；上述入口不能代替具体翻译规范。

当用户要求执行 retranslation 正式翻译、retranslation retry 的译文生成、revision batch 翻译或任何 TranslationUnit 翻译时，执行者在写文件前必须读取：

1. `docs/TRANSLATION_WORKFLOW.md`
2. `docs/TRANSLATION_TASK_SPEC.md`
3. `docs/RETRANSLATION_RUNBOOK.md`
4. 当前执行环境规范：Codex 读取 `docs/CODEX_TRANSLATION.md`；普通 ChatGPT + Remote Desktop Commander 读取 `docs/CHATGPT_LANGUAGE_GENERATION.md`
5. 当前 batch 的 `manifest.json`
6. manifest 列出的全部 `inputs/*`
7. `locales/<locale>/glossary.yaml`

manifest、全部 inputs 与 locale glossary 是不可拆分的正式翻译输入。不得因用户 Prompt 未再次提醒 glossary 而跳过它，也不得以聊天上下文代替仓库中的当前正式规则。

模型与 reasoning 由用户在对应 UI 中选择；Codex 后续新启动的正式语言生成默认使用 **GPT-5.6 Sol + High**（正式 provenance `gpt-5.6-sol-high`），普通 ChatGPT Generation 与独立 ChatGPT Reviewer 继续使用 **GPT-5.6 Sol + High**。本文件不负责切换 UI 中的模型或 reasoning。

## 执行职责与正式自动化

每个 locale 固定保持五个不得混用的职责角色：会话协调 / 基础设施架构、**Generation role/session**、独立 **Reviewer role/session**、Codex 仓库工程，以及维护者 **Local terminal**。在该 locale 开始正式 generation 前，根据当时真实的 ChatGPT 可用额度、Codex 5 小时额度与周额度、Remote Desktop Commander 月额度、历史实测成本和并发计划，选择 `chatgpt` 或 `codex` 作为 Generation provider。ChatGPT Generation 与新启动的 Codex Generation 默认均使用 **GPT-5.6 Sol + High**，Codex 正式 provenance 为 `gpt-5.6-sol-high`。已开始的 Generation 沿用已选模型；获批的 `mr-IN` Luna 批次边界切换保留为历史授权，不要求正在执行的 Luna batch 改用 Sol。generation 一旦开始，原则上 glossary、UI catalog、article metadata、其他 locale-level 文案、TranslationUnit initial / revision / retry generation、schema v2 Course SEO localization / refresh / revise replacement，以及 Locale Surface Review finding replacement 全程沿用同一 provider 与已选模型。只有获得维护者明确批准、到达安全批次边界且有额度或 provider/tool 的真实依据时才可切换模型；必须保留真实 provenance，不新增虚假 generation 记录，不弱化任何 gate。当前 `mr-IN` 的批次边界切换授权见 [Codex 正式语言生成执行规范](docs/CODEX_TRANSLATION.md)。provider/model selection 不新增 schema、receipt、machine gate 或 locale state 字段。

Reviewer role 继续固定使用独立的普通 ChatGPT **GPT-5.6 Sol + High** session，负责 Glossary Review、逐 Unit Quality Check、revision 后 re-QC 与 Locale Surface Review。同一个从未参与该 locale 语言 generation 的 Reviewer session 可以连续承担这些正式审核；各 gate 范围和 evidence 独立，reviewer finding 必须回到 Generation session 产生 replacement，Reviewer session 不得自行生成 replacement 后批准。Glossary Review 以 [Glossary Review 规范](docs/GLOSSARY_REVIEW.md) 为准；provider 为 `chatgpt` 时以 [ChatGPT 正式语言生成执行规范](docs/CHATGPT_LANGUAGE_GENERATION.md) 为准，provider 为 `codex` 时以 [Codex 正式语言生成执行规范](docs/CODEX_TRANSLATION.md) 为准。canonical English source-description review 是跨 locale 共享 authority，不默认归入单个 locale 的 Reviewer session，仍以 [课程页正式 SEO Metadata 规范](docs/COURSE_SEO_METADATA.md) 为准。

新 Quality Check 的单次 Reviewer model invocation / response 最多审核 60 TranslationUnits，Page / Example 分开；60 是实际审核质量边界，不得在同一次 response 内串联多个 `<=60` working set 绕过。下一组必须使用新的用户请求和新的 model invocation。TranslationUnit 初次自动 Page / Example export、显式 `--id` export、正常 Quality Check B/C/D revision，以及经 `quality-check surface-reopen` 授权的精确 revision，均使用默认且硬性上限 60，且 Page / Example 永不混合；当前正式课程仍只有 19 个 eligible Example，不为凑满批次扩大 scope。legacy Final Review 的独立 30-Unit 审核限制不变。

首次大批量 TranslationUnit Generation 仍由维护者 Local terminal 导出 current provider-neutral Generation Bundle ZIP 并交给 Generation session；模型在按 locale + batch 隔离的隐藏 staging 保存完整 TranslationUnit。具备相同本地文件访问能力时，默认用 `generation-bundle import --input-dir` 直接导入，不制作或下载 outputs ZIP。只有确需跨环境传输、目录不可访问或恢复时，才使用 outputs ZIP → `result-pack` → `import` 兼容路径。Generation ZIP 仍只是首次完整输入的 transport container；独立 Reviewer ZIP 继续由维护者上传。任何路径都不替代 validation、独立 QC、finalization 或其他 gate，也不代表已有双向自动附件传输。

新增 locale 的唯一默认调度顺序以 [新增 Locale 执行手册](docs/NEW_LOCALE_RUNBOOK.md#新增-locale-默认调度路径效率优先) 为准；各阶段文档中仍合法的中途 revision、retry 与 stale recovery 是兼容恢复路径，不反向改写历史 evidence，也不取代该默认路径。维护者 Local terminal 可访问输出目录时，默认从隐藏 staging 目录导入；若确需下载 Reviewer ZIP 或兼容 Result ZIP，仍按产品需要由维护者完成跨 ChatGPT / Ubuntu 或跨 session 的附件传递，不得声称已有无人工双向传输通道。

Generation / Reviewer 闭环内外的 deterministic repository / shell workflow，以及正常 workflow 的 readonly inspection、currentness reassessment、contract verification、formal command execution，默认均由维护者 Local terminal 执行，**Codex=0**，与预计耗时无关。这包括 export / bundle / result landing / import / process / validation / Snapshot / scope、Reviewer evidence save/check、current-check / record / finalize / authorization、locale-level replacement 落盘、Surface Reviewer bundle / record-a、package closure / activation，以及 Course SEO 结果落盘与正式 `assemble` / `refresh` / `revise`。AI 基于已确认的 authority、current Bundle 与 terminal evidence 提供完整可复制命令并消费终态；缺 CLI/schema/machine fact 或 canonical command contract 时，若一次合并 readonly RDC 能明显降低手工 discovery 总成本，则按文件顶部 gate 优先采用；不可用或无明显收益时再提供合并 Local terminal discovery command。正常 workflow 不因“短”而 remote 代跑，readonly discovery 也不授权 mutation。已授权 repo engineering 所必需的局部读取与 targeted tests 按对应任务边界执行。Generation / Reviewer 会话隔离与所有现有 gate 保持不变。

locale init、首次 bulk export / ZIP handoff、promotion、build、preview / browser verifier、publish、shared assets、Production / deploy / verifier、checksum / curl、search closeout 与最终 Git commit / push 同样默认由维护者 Local terminal 执行。当前 docs/code/production identity 已确认的 machine fact 应直接复用；用户返回 identity / SHA / counts / stop point 合法的 FINAL PASS 后直接继续，不 remote 重验。只有矛盾、缺项、相关 mutation 或真实 failure evidence 才调查具体未知事实，不重新设计已确认的 Production、广告、IndexNow、shared assets 或 verifier。

### 测试额度与执行边界（fail-closed）

Codex 是受限工程资源，优先用于 correctness bug、可复用 workflow blocker、activation / Production blocker 和有明确跨 locale 长期复用价值的工程；禁止顺手重构、unrelated fixture 清债、broad test cleanup 或无上线价值的 architecture perfection。以周总额度为口径，日常单日消耗目标约 **8%–12%**，单日 **hard cap 15%**，尽量长期保留约 **20% emergency reserve**；hard cap 不是 daily target，也不为用满目标而新增工作。达到 cap 或触发复杂度停止线即保留现场，按 [Codex Budget Rule 与 Production 优先级](docs/WORKFLOW_HANDOFF.md#codex-budget-rule)交接，不因额度恢复重启已无必要的 subsystem。

Codex 默认只能运行与本次实际修改直接相关、足以证明改动正确的最小 targeted tests。未经用户明确授权，Codex 不得将 targeted tests 自行升级为 package-wide、repository-wide 或 lifecycle-wide regression，包括但不限于 `go test ./...`、多 package 的广泛 `go test`、repository-wide regression、全量 browser regression、publish、deploy、verifier、`--all-live`、完整 Production lifecycle、完整 TranslationUnit lifecycle，以及已有正式 CLI/script 可由本地终端确定性完成的其他长时间机械流程。

targeted tests 已足以证明当前改动正确时，不得为了“更保险”“顺便确认”或“完整覆盖”追加全面测试。Codex 如认为确有必要扩大范围，不得直接执行；应先在最终总结中说明现有 targeted tests 为何不足、建议补跑什么，并明确该测试由用户本地终端执行。只有用户随后明确授权，Codex 才可执行更大范围测试。

测试边界按以下规则确定：

- docs / README / config 任务默认只运行 `git diff --check`，以及与修改内容直接相关的 parser、validator 或 focused unit test；不得因为涉及共享 package 就自动运行整个 package 或 repository 测试。
- 代码任务优先测试直接修改的函数或模块、与改动行为直接相关的既有 targeted test；必要时新增 focused regression test。不得机械运行整个 package test suite，除非该 package 本身很小、这是唯一合理的 targeted boundary，且执行成本明显低。
- targeted test 意外暴露 scope 外问题时，只保存最小真实 failure evidence，不继续扩大调查或追加测试；仅当用户明确决定暂缓该问题时，才按 Deferred Issue 规则记录。
- `go test ./...`、全量 publish、10-locale `--all-live`、browser regression、checksum / curl / verifier、build / deploy / assets verification 等确定性但耗时的测试与验证，优先交给用户本地终端执行。

额度优化不得降低正式质量门槛：ChatGPT Generation 与 Codex 新启动的正式 Generation 默认均使用 GPT-5.6 Sol + High（Codex 正式 provenance：`gpt-5.6-sol-high`）；QC 仍为 A-only，revision、validation、Production machine gate 与 HUMAN gate 均不得削弱。应节省的是不必要的 reasoning、重复检查、全面测试和机械执行。ChatGPT 与 Codex Generation 使用同一 provider-neutral deterministic Generation Bundle contract；独立 ChatGPT Reviewer 继续使用 GPT-5.6 Sol + High，并优先使用 Glossary、Quality Check 与 Locale Surface Review 的 deterministic Reviewer Bundle。bundle 只是绑定 current authority/input identity 的传输容器，不是 semantic authority、receipt 或语言质量 gate；当前性检查、result import、existing process/validation/record/finalize 等具体顺序以对应 stage 文档为准。附件完整且 current 时不再通过 Remote Desktop Commander 重复扫描同一批仓库输入。

Remote Desktop Commander 按 Call-Zero-First 成本/风险规则使用，不机械追求 0，也不因可用就调用。普通 workflow step 最多一个合并 readonly call 采集必要 docs + implementation/tests 与 exact facts；第二 call 只有第一 call 新暴露具体、阻塞安全推进的未知 fact 时才允许。不得用 `start_process` + `read_process_output` 轮询可交 Local terminal 的任务。实际使用 remote call 的 handoff 必须报告 `REMOTE_CALLS_USED=<n>` 与 `REMOTE_CALL_REASON=<missing-machine-fact|failure-diagnosis|canonical-doc-read|maintainer-request>`；0 call 时不增加这些字段。readonly diagnosis 不授权 write / Git mutation / activation / import / record / finalization / Production 或其他 lifecycle mutation，此边界不省略任何脚本内部 gate、receipt、bounded retry 或 HUMAN gate。

新增正式 CLI、validator、manifest/lifecycle logic 或可复用 machine workflow 时，优先使用 Go，并优先集成现有 `cmd/tour-i18n`。Shell/Python 主要用于薄的 OS、SSH、browser 或已有成熟 orchestration；已有稳定 sh/python 不因“统一语言”重写。不得用临时 Python/sed 代替需要 repository context 的正式实现。

### 性能优化的性价比原则

性能优化以工程性价比为准，不追求最低可能执行时间。对于低频 deterministic workflow：

- 优先修复明显、高收益的重复计算和算法瓶颈，优先采用简单的 invocation-local reuse；谨慎引入 persistent cache、复杂 memoization、LRU、cache isolation 或大规模 architecture refactor。
- 不得为了性能降低 correctness、freshness、stale detection、identity、hash validation、dependency mutation detection 或 fail-closed 保证。
- 执行时间达到实际可接受范围后，进一步优化属于可选项。原本约 5～10 分钟的低频 workflow，若已降低到约 1～2 分钟或其它实际可接受范围，通常可以停止继续优化。
- 如果继续节省少量时间需要明显增加实现复杂度、验证成本、维护负担或 correctness 风险，应停止；不得仅为了给低频命令再节省几十秒，就引入明显复杂的缓存架构。

性能任务的默认完成标准是 `REMAINING_HIGH_VALUE_OPTIMIZATION=none`，而不是 `REMAINING_DUPLICATE_WORK=none`。允许保留低收益重复计算，只要 correctness 不受影响，并且继续修复不具有合理工程性价比。

## 维护协作输出

除非用户明确要求英文，仓库 Markdown/docs 面向维护者的内容、Git Conventional Commit subject，以及 Codex 面向维护者的最终总结、failure 分析和下一步说明默认使用中文。代码标识符、CLI 名称、协议字段、原始第三方错误和必须保持的 source 文本不强制翻译。

Codex 已读取仓库当前 AGENTS/docs/code 后，不要求用户再次复制仓库已有正式规则。完成一次代码/docs 任务后，final response 默认简洁地只返回：实际修改文件、关键行为/决定、执行的测试及结果、未解决的真实 failure evidence（如有）、`git diff --stat`、`git status --short`。除非用户明确要求，不粘贴完整 git diff 或完整文件，不重述整个 Runbook，也不返回长篇“接下来让 ChatGPT 做什么”的 Prompt。

向维护者提供可直接粘贴到当前交互 shell 的命令时，不得在顶层修改可能持续污染或终止该 shell 的 shell options，包括 `set -e`、`set -u`、`set -o pipefail` 及 `set -euo pipefail` 等组合，也不得包含 `exit`、`logout`、`kill $$`。如确需 fail-fast 或临时 shell options，必须放入独立 subshell（例如 `( set -euo pipefail; ... )`）或独立脚本进程，使失败只结束子进程，不关闭或改变维护者当前交互 shell。

## Git 提交说明

仓库提交说明默认使用中文，并沿用现有 Conventional Commit 风格，例如：

- `fix: 修复……`
- `feat: 增加……`
- `chore: 完成……`
- `docs: 完善……`
- `refactor: 重构……`
- `test: 补充……`

除非用户明确要求英文，否则不得自行改用英文提交说明。提交前应参考近期 `git log`，保持与仓库现有风格一致。

## Deferred issue 记录

当 Codex 在开发、测试、部署或验收过程中遇到真实问题，且用户明确要求本轮暂不处理时，必须检查 `docs/DEFERRED_ISSUES.md`：尚未记录则新增条目，已有条目则更新本次上下文，不得因当前任务 scope 不处理而让问题只存在于聊天记录中。

只有已经真实发现并明确暂缓的问题适用本规则；普通 TODO、未来优化建议和架构设想不得写入。不得凭猜测补录历史问题，历史条目必须有可靠 evidence。
