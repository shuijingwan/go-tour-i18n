# 共享翻译单元生命周期与五角色协作

本项目从开始就覆盖 Go Tour、Site shell、Learn、Docs 与未来内容包；Tour 只是最先完成的内容表面。所有正式翻译单元共用 `internal/i18n` 的生命周期，不按 surface 或代码目录建立平行状态机。本文规定新增执行；历史目录、批次、原始响应、审核、快照、finalization、promotion 和 Production evidence 不迁移、不改写。

## 生命周期与适配边界

统一顺序：翻译生成 → 传输与导入 → 逐单元还原 → 逐单元自动校验 → 失败单元精确重试 → 完整有效集合 → 候选快照 → 独立质量审核 → 精确修订 → 独立再审核 → A-only finalization → 内容包、Surface 与 promotion。

复用既有导出、Generation Bundle current-check/import/recover、`process`、`retry`、最新候选选择、Candidate Snapshot、Quality Check、revision、re-QC 与 finalization。`TranslationUnitAdapter` 只提供受保护输入、还原和结构/术语校验；Tour 保持既有实现，Learn/Docs 的 `PageAdapter` 接入同一调度与证据机制。一个物理页面就是一个正式页面翻译单元；内部槽位仅用于解析、源坐标保护、还原、影响分析和诊断。标题后缀、行内代码/链接绑定、Markdown 强调语法、教学代码注释、HTML/Markdown 块结构、合法软换行、保护标记成员、配对与页面边界继续严格校验。

传输层只负责 schema、输入/输出与 Bundle identity、真实 provenance、精确集合、缺失/多余/重复、UTF-8/BOM/EOF、安装路径、安全文件系统及不可覆盖。语义和结构失败发生在逐单元自动校验层，不能拒绝其余独立输出。批处理检查所有独立单元，一次报告总数、通过、还原失败、校验失败、精确 ID 和原因；单个单元内部遇到致命错误可以停止。

30 页中 29 通过、1 失败时，29 个候选及通过证据正式保留。失败页进入 `restore_failed` 或 `validation_failed`；正常 Generation Bundle retry 只选择该页。成功后合并既有结果得到 30/30；不得重新生成其余29页、引入局部接受 receipt 或另建 Site 重试分支。

多个失败 TranslationUnit 可以在同一机器失败状态下分别导出精确 retry Bundle，并由同一 Generation session 一次生成，随后按任意顺序导入和执行 retry。一个 sibling retry 成功不能使其他自身输入和失败状态未变的 Unit 的原 Bundle 失效；已有可信输出继续绑定模型实际读取的原 Bundle SHA、input identity 与真实 provider/model，不重新绑定新 Bundle、不重新调用模型。

`generation-bundle current-check --bundle <original.zip>`、result-pack、import 与 recover 共用同一 currentness 检查。retry 的完整 `formal/retry/result.json` 仍保留并验证：schema、batch/locale、精确有序成员、汇总计数与 canonical bytes 必须自洽，每条记录必须有当前 validation 或正式 retry validation history 支持，目标 Unit 的记录必须与当前汇总完全一致。仅允许有真实历史 evidence 的 sibling 状态变化；当前目标 validation bytes、失败状态、attempt、source、重新构建的 protected input、完整正式 manifest/inputs、glossary、authority、expected output/install path 仍严格绑定。目标 raw response 或 import provenance 已存在、Bundle 篡改或 provider/model 不匹配继续 fail closed。复用现有 schema，不新增 surface-specific receipt 或状态机。

## 现有结果沿用与证据完整性

historical integrity 与 current eligibility 正式分离，唯一共享 contract 见[当前翻译授权与 coupling inventory](CURRENT_TRANSLATION_AUTHORIZATION.md)。原 Reviewer A、Snapshot、finalization 和 provenance 不改写；原 accepted exact candidate 在当前 source/glossary/full Unified Review/adapter/validator 下按 TU 机械 re-attest。compatible 才 carry，affected/stale/unknown 精确 reopen；机器不产生新 A。合法 current checkpoint 之后，downstream 保留 original immutable hash checks，使用 current authorization，不再无限 replay pre-checkpoint compatibility/mutable projection。新 registered package 和 future locale 直接复用；旧 reader 仅兼容未建立 checkpoint 的历史、migration/recovery，不成为新 package 的默认 happy path。

首次 complete-set re-attestation 与后续 mixed successor 分开：建立 current baseline 后，unchanged candidates 继承其 machine eligibility，合法 B/C/D finding / Surface reopen 授权的 exact revision delta 以原 Generation invocation/provenance、当前 input/source/glossary、restore/validation PASS 进入 successor Snapshot。baseline + delta 的 exact-set proof 在 current reader 中重新验证；任意 latest passed candidate 不得替换 baseline。delta 必须 pending independent re-QC，不 carry 原 A；独立 A-only finalization 后才可建立 complete current authorization。原 Bundle 缺 archive 的 recovery 只原样保留已导入 transport，不重新生成、导入或 process，具体字段与 gate 只在上述共享 contract 定义。

包级 SHA 变化不等于每个单元输入变化。旧/current 精确正式输入、原始源与路由一致，候选 bytes 不变，当前结构/术语校验通过且当前完整术语表独立审核通过时，才可以沿用。正式输入变化不得沿用。术语表 bytes 变化仍遵守既有 glossary compatibility authority，不能凭包级比较豁免。

历史 Page JSON 与共享逐文件输出的表示不同；正常 Generation Bundle receipt 不能如实表达原先模型读到的完整 JSON invocation。因此仅扩展既有批次 manifest 的可选 `generation_origin`，绑定完整原 Bundle 与 result 的不可变归档路径及 SHA-256；`ArchivedGenerationAdapter` 验证精确当前正式输入和原输出。未新增 schema、receipt 或生命周期。完整归档采用内容寻址，保留原30页及 recovery-8/2/1 的真实范围，机械导出的共享 manifest 不冒充原模型输入。末尾 LF 只作为共享输出传输 framing；不改写译文措辞或原结果文件。

`site-content generation-land` 是旧表示到共享导入的兼容适配器，不生成新语言、不创建 subset Generation Bundle。`site-content generation-import` 的旧 JSON 入口分开检查传输与执行共享 `process`。旧 Site 审核/修订/finalization CLI 停止新增记录，历史证据读取继续兼容。新页面一律使用 `retranslation`、`generation-bundle`、`quality-check --content-package learn-docs-v1`；内容包 closure 读取共享 A-only finalization。

## 固定调度与独立审核

`learn-docs-v1` 先第一批生成与机器校验30/30，再第二批生成与机器校验21/21，然后建立完整51页快照，进行两次独立初审。initial Generation / initial QC 的固定组1–30与31–51（future append 仍限31–60）不可合并；通用60上限不授权突破 initial 每组30限制。两批初审全部完成后，汇总全部 B/C/D，再集中精确修订及独立再审核。固定 membership 是 initial scheduling metadata，不是 TranslationUnit 的永久 partition：正式 B/C/D 与 Surface reopen revision 可跨 initial group，仍只处理 current authorization 的同 kind Unit、每次最多60，保留全部 identity、finding、transport、provenance 与 validation gate。glossary-stale recovery 本轮保留原固定分组；retry/recovery 仍只处理精确 failed/missing scope。只有真实失败恢复或正式当前性要求允许中途恢复。

initial QC、revision 后 re-QC 及其他存在顺序依赖的多组 QC，统一遵守[Reviewer Bundle 按需即时导出（JIT）](WORKFLOW_HANDOFF.md#qc--re-qc-reviewer-bundle-按需即时导出jit)：只导出当前下一次交给 Reviewer 的一组，禁止预导未来 Bundle。前一组正式 current-check、Reviewer evidence save/check 与 `quality-check record` 成功后，才基于新的 machine scope 导出下一组，并执行 `reviewer-bundle-check` 与 `sha256sum`。Bundle 绑定当前完整 QC scope，前一组 record 后预导的后一组 ZIP 被判 stale or non-canonical 是正确的 fail-closed 行为。initial QC 保留固定组；successor Snapshot re-QC 按 current `quality_check_required` pending、stable index 顺序、同 kind、最多60选择，不继承 initial partition，record 使用同一阶段语义（含已持久化 predecessor）。旧 A 仍仅按 exact identity / 正式 compatibility lineage carry-forward。每组保持独立用户请求/model invocation及 A-only 等全部 gate；两批初审完成后集中 revision 的顺序不变。

机器通过不等于 Reviewer A。完整快照和共享选择规则阻止第二批尚未有效时开始审核；共享 revision export 阻止首轮尚未完整覆盖时修订。新页面审核记录提供 `--reviewer-session`，并检查完整术语表审核绑定的 Generation/Reviewer 角色，禁止生成会话审核自身结果。历史审核记录字段缺省继续兼容。

普通修订沿用旧混合评级快照的逐单元 A，证据链绑定其既有审核结果；旧快照仍有 B/C/D 时本来就不能先做 A-only finalization。新修订快照完成独立再审核后按共享完整 A-only 门终结。没有新审核结果的零审核全沿用，仍必须绑定并验证不可变的前一 finalization；不能把这两类证据语义混为一谈。

所有 surface 的 Quality Check / re-QC / Surface finding revision 共用[精确修订任务语义](TRANSLATION_TASK_SPEC.md#精确修订任务语义)：逐 finding 处置，finding-anchored 且 locally complete，保护 source/technical identity 与合法结构，不授权整页重译。Generation 提交前完成[finding closure check](CHATGPT_LANGUAGE_GENERATION.md#generation-finding-closure-执行规范)，仅确认任务完成性，不承担正式审核或通过决定；Bundle 大小、既有调度、独立 Reviewer、A-only 与 validator 边界不变，不新增平行生命周期或状态。

Reviewer 对 B/C/D category 在完整 current TU 做 same-class scan，Generation 在 persisted category + bounded scope 内做 same-class complete sweep（examples 不默认穷尽授权边界，仅明确 exact-instance-only 时限于列举实例），successor re-QC 检查该 category 的 scope 内 closure；详细 completeness、歧义 fail-closed 与禁止无关润色的边界统一见[审核合同](TRANSLATION_QUALITY_REVIEW.md#bcd-finding-完整性与-successor-re-qc)和上述精确修订语义，不扩大任何正式 Unit/slot authorization。

共享 revision Generation Bundle 同时冻结 protected source、reviewed restored predecessor candidate 与正式 passed protected predecessor raw response；前者 candidate 用于 finding closure 对照，后者 raw response 用于实际编辑，Generation 输出仍是 protected raw response，复用 restore → candidate → validation。revision-origin machine retry 保留原修订语义和完整输入，不以 latest candidate、聊天记忆、逆向保护或仓库扫描替代。身份绑定、inventory/current-check 与恢复以[Generation Bundle contract](TRANSLATION_TASK_SPEC.md#deterministic-generation-bundle-transport)为准，QC 与 Surface reopen 不另建状态机。

Integrated Surface receipt 通过共享 exact source identity mapping 与 canonical candidate artifact（Snapshot、source、candidate、validation、attempt 的 path/SHA/identity）授权精确 TU reopen；artifact 的 Markdown、HTML、XML、Present 等表示由 TU/adapter 决定。独立 Surface authority reader 只检查原 receipt/session/scope/closure/evidence，不选择 Generation、retry 或 QC 策略。非 TU finding 保持 Surface/package blocker。授权后的 exact revision、origin retry、successor re-QC、A-only 与 package closure 继续走同一共享生命周期；具体当前性与 carry 边界见[Surface 定向纠错](TRANSLATION_QUALITY_REVIEW.md#qc-lineage-preflight完整重审与-surface-定向纠错)。新增 package 的注册、表示与 conformance 步骤见[扩展审计 authority](SHARED_TRANSLATION_UNIT_CONFORMANCE.md#adding-a-future-content-package)，不得为 package 修改共享状态机。

同一 FAILED Integrated Surface receipt 的多个独立 TU-language finding，在同 finalized predecessor、同 UnitKind、最多60及 exact artifact union 条件下，共用一个 aggregate Surface reopen 和一次集中 revision；逐 Unit finding 不合并、不泛化。兼容表示、授权与 exporter 规则仅由上述质量规范定义，沿用同一 Generation Bundle / revision / retry / re-QC 生命周期。

## 五角色与执行成本

| 角色 | 职责与边界 |
| --- | --- |
| 会话协调 / 基础设施架构 | 读取 authority、维护整体状态、决定唯一下一步、准备完整 Local terminal 命令并消费合法 machine evidence；不做正式语言判断、不自行改代码，真实工程任务交 Codex |
| 翻译生成 | 完整术语表、语言资产、初次生成、重试、修订与 finding replacement；不审核自身、不改基础设施或放宽校验器 |
| 独立翻译审核 | 统一术语表审核、逐单元初审/再审核与 Surface finding/decision；不生成 replacement、不改候选或代码 |
| Codex 仓库工程 | 架构、Go、解析器/结构保护器、CLI、schema、测试、docs、恢复基础设施；未经明确 provider 授权不生成语言，不承担正式审核 |
| Maintainer Local terminal | 默认执行所有 deterministic repository/shell workflow，与预计耗时无关；HUMAN gate、最终 Git；不承担语言判断 |

执行者遵守长期[Call-Zero-First 交接合同](WORKFLOW_HANDOFF.md#call-zero-first-交接合同)：RDC / repository remote calls 默认 0 是成本/风险默认值，不是 zero-call KPI；闭环内外 deterministic workflow 与 lifecycle mutation 默认由 Maintainer Local terminal 执行，与预计耗时无关。AI 提供完整命令并直接消费 identity / SHA / counts / stop point 合法的 FINAL PASS，除矛盾、缺项或相关 mutation 外不 remote 重验。完整 current Bundle 的 Generation / Reviewer semantic RDC calls = 0，Coordinator 不重扫同一正式输入。Coordinator / engineering 缺 exact fact、canonical docs / command contract 或需要 failure diagnosis 时，若一次 targeted readonly RDC 明显降低手工 discovery 成本，则优先合并读取所需 docs + implementation/tests；该读取不授权 mutation。调用预算、remote reasons 与第二 call 的具体 blocker 条件统一见[具体执行规范](CHATGPT_LANGUAGE_GENERATION.md#rdc-调用预算与回退)。代码工程仍只做获授权的直接相关 targeted tests 与 `git diff --check`，不扩大为全仓库、全浏览器或 Production 回归。

本地交接必须是可整段复制、完整路径/变量/ID/SHA/检查、无 placeholder 的命令；不得要求维护者推断 workflow。临时 fail-fast 放独立脚本或 subshell，不能污染或关闭当前交互 shell。

## 恢复、复用与表达

失败先查 staging、Bundle、原结果、import provenance、raw responses、validation、快照、审核及 pending marker；已有可信单元不重做，只处理缺失、还原/校验失败、真实 stale/affected。`mutation=unknown` 停止，禁止盲目重放写操作。staging 不成为长期正式 authority。

新增 schema、receipt、loader、恢复/重试/审核/终结状态机前，必须解释现有共享机制为何不能表达，并先评估 adapter、extension、shared helper、generic interface 和 existing evidence。只有真实语义或证据完整性缺口才允许新增；必须记录缺失、为何 adapter/extension 不足、必要性及全 surface 通用性，不能因为目录不同或泛化需要重构而复制。

维护者/用户说明、状态、失败分析、交接、docs、提交说明默认中文。第一次需要对照时用中文名称（English identifier），之后优先中文；CLI、schema/field、函数/类型、路径、locale、ID、SHA、model/provider、第三方错误和 canonical source 保留原文。采用“第一批页面机器校验：待收口”“独立质量审核：尚未开始”等摘要。

交接只给一个唯一下一步，写清执行者、动作、停止条件；存在架构分歧时唯一下一步是维护者决策，不能提前执行某个方案。已有完整 current deterministic Bundle 时，Prompt 仅强调角色、精确范围、任务、当前 finding/特殊缺陷、禁止项和输出，不重复仓库历史；模型仍必须完整读取 Bundle、manifest、全部 inputs、术语表与 authority。可预测 read/hash/status/check 默认合并为完整 Local terminal 命令，remote 例外只采集缺失必要事实，避免重复扫描与进程轮询；不能为节省 calls 而降低任何质量、provenance、独立审核、A-only、Production 或 HUMAN gate。
