# Codex 项目导航

当用户要求评估 locale 商业优先级或选择下一门语言时，Codex 必须先读取 `docs/LOCALE_ROADMAP.md`。用户未明确指定 locale 的新增任务也先按该路线图选择；单门 locale 的正式执行仍以 `docs/NEW_LOCALE_RUNBOOK.md` 为准。

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

每个 locale 固定保持三个不得混用的职责角色：**Generation role/session**、与之独立的 **Reviewer role/session**，以及维护者 **Local terminal**。在该 locale 开始正式 generation 前，根据当时真实的 ChatGPT 可用额度、Codex 5 小时额度与周额度、Remote Desktop Commander 月额度、历史实测成本和并发计划，选择 `chatgpt` 或 `codex` 作为 Generation provider。ChatGPT Generation 与新启动的 Codex Generation 默认均使用 **GPT-5.6 Sol + High**，Codex 正式 provenance 为 `gpt-5.6-sol-high`。已开始的 Generation 沿用已选模型；获批的 `mr-IN` Luna 批次边界切换保留为历史授权，不要求正在执行的 Luna batch 改用 Sol。generation 一旦开始，原则上 glossary、UI catalog、article metadata、其他 locale-level 文案、TranslationUnit initial / revision / retry generation、schema v2 Course SEO localization / refresh / revise replacement，以及 Locale Surface Review finding replacement 全程沿用同一 provider 与已选模型。只有获得维护者明确批准、到达安全批次边界且有额度或 provider/tool 的真实依据时才可切换模型；必须保留真实 provenance，不新增虚假 generation 记录，不弱化任何 gate。当前 `mr-IN` 的批次边界切换授权见 [Codex 正式语言生成执行规范](docs/CODEX_TRANSLATION.md)。provider/model selection 不新增 schema、receipt、machine gate 或 locale state 字段。

Reviewer role 继续固定使用独立的普通 ChatGPT **GPT-5.6 Sol + High** session，负责 Glossary Review、逐 Unit Quality Check、revision 后 re-QC 与 Locale Surface Review。同一个从未参与该 locale 语言 generation 的 Reviewer session 可以连续承担这些正式审核；各 gate 范围和 evidence 独立，reviewer finding 必须回到 Generation session 产生 replacement，Reviewer session 不得自行生成 replacement 后批准。Glossary Review 以 [Glossary Review 规范](docs/GLOSSARY_REVIEW.md) 为准；provider 为 `chatgpt` 时以 [ChatGPT 正式语言生成执行规范](docs/CHATGPT_LANGUAGE_GENERATION.md) 为准，provider 为 `codex` 时以 [Codex 正式语言生成执行规范](docs/CODEX_TRANSLATION.md) 为准。canonical English source-description review 是跨 locale 共享 authority，不默认归入单个 locale 的 Reviewer session，仍以 [课程页正式 SEO Metadata 规范](docs/COURSE_SEO_METADATA.md) 为准。

新 Quality Check 的单次 Reviewer model invocation / response 最多审核 60 TranslationUnits，Page / Example 分开；60 是实际审核质量边界，不得在同一次 response 内串联多个 `<=60` working set 绕过。下一组必须使用新的用户请求和新的 model invocation。legacy Final Review、revision batch、Example export 与 explicit `--id` export 的既有 30 上限不变。

首次大批量 TranslationUnit Generation 仍由维护者 Local terminal 导出 current provider-neutral Generation Bundle ZIP 并交给 Generation session；模型在按 locale + batch 隔离的隐藏 staging 保存完整 TranslationUnit。具备相同本地文件访问能力时，默认用 `generation-bundle import --input-dir` 直接导入，不制作或下载 outputs ZIP。只有确需跨环境传输、目录不可访问或恢复时，才使用 outputs ZIP → `result-pack` → `import` 兼容路径。Generation ZIP 仍只是首次完整输入的 transport container；独立 Reviewer ZIP 继续由维护者上传。任何路径都不替代 validation、独立 QC、finalization 或其他 gate，也不代表已有双向自动附件传输。

新增 locale 的唯一默认调度顺序以 [新增 Locale 执行手册](docs/NEW_LOCALE_RUNBOOK.md#新增-locale-默认调度路径效率优先) 为准；各阶段文档中仍合法的中途 revision、retry 与 stale recovery 是兼容恢复路径，不反向改写历史 evidence，也不取代该默认路径。普通 ChatGPT + Remote Desktop Commander 可直接访问仓库/终端时，结果默认从本地隐藏 staging 目录导入；若确需下载 Reviewer ZIP 或兼容 Result ZIP，仍按产品需要由维护者完成跨 ChatGPT / Ubuntu 或跨 session 的附件传递，不得声称已有无人工双向传输通道。

Generation / Reviewer 闭环内部的短时 deterministic 操作默认由具备当前仓库与终端访问能力的 AI execution environment 自动连续完成，不逐步回交维护者。这包括 reviewer finding 后的小批 revision export / bundle / result landing / import / process / validation / Snapshot / scope、审核结果的 current-check / record / finalize、locale-level replacement 落盘、Surface Reviewer bundle / record-a，以及 Course SEO generation 结果落盘与正式 `assemble` / `refresh` / `revise`。自动执行只承担机械操作，不改变 Generation 与 Reviewer 的会话隔离，不替代人工语言判断，也不得跳过 current-check、exact-set、A-only、provenance、schema 或任何现有 gate；缺少终端能力、需要新的用户 handoff 或出现真实 failure / mutation-unknown 时才停下并交回维护者。

闭环结束后的成组 deterministic 操作由维护者 Local terminal 执行，包括 locale init、首次 bulk export / ZIP handoff、promotion、build、preview / browser verifier、publish、shared assets、Production / deploy / verifier、checksum / curl、search closeout，以及最终 Git commit / push。当前 docs/code/production identity 已确认的 machine fact 应直接复用；没有新的 failure evidence 时，不重新探测服务器、不重新设计 Production、广告、IndexNow、shared assets 或 verifier。只有真实 evidence 显示基线变化时，才重新调查并更新正式 docs，避免为了“再确认一次”消耗 Codex reasoning quota。

### 测试额度与执行边界（fail-closed）

Codex 默认只能运行与本次实际修改直接相关、足以证明改动正确的最小 targeted tests。未经用户明确授权，Codex 不得将 targeted tests 自行升级为 package-wide、repository-wide 或 lifecycle-wide regression，包括但不限于 `go test ./...`、多 package 的广泛 `go test`、repository-wide regression、全量 browser regression、publish、deploy、verifier、`--all-live`、完整 Production lifecycle、完整 TranslationUnit lifecycle，以及已有正式 CLI/script 可由本地终端确定性完成的其他长时间机械流程。

targeted tests 已足以证明当前改动正确时，不得为了“更保险”“顺便确认”或“完整覆盖”追加全面测试。Codex 如认为确有必要扩大范围，不得直接执行；应先在最终总结中说明现有 targeted tests 为何不足、建议补跑什么，并明确该测试由用户本地终端执行。只有用户随后明确授权，Codex 才可执行更大范围测试。

测试边界按以下规则确定：

- docs / README / config 任务默认只运行 `git diff --check`，以及与修改内容直接相关的 parser、validator 或 focused unit test；不得因为涉及共享 package 就自动运行整个 package 或 repository 测试。
- 代码任务优先测试直接修改的函数或模块、与改动行为直接相关的既有 targeted test；必要时新增 focused regression test。不得机械运行整个 package test suite，除非该 package 本身很小、这是唯一合理的 targeted boundary，且执行成本明显低。
- targeted test 意外暴露 scope 外问题时，只保存最小真实 failure evidence，不继续扩大调查或追加测试；仅当用户明确决定暂缓该问题时，才按 Deferred Issue 规则记录。
- `go test ./...`、全量 publish、10-locale `--all-live`、browser regression、checksum / curl / verifier、build / deploy / assets verification 等确定性但耗时的测试与验证，优先交给用户本地终端执行。

额度优化不得降低正式质量门槛：ChatGPT Generation 与 Codex 新启动的正式 Generation 默认均使用 GPT-5.6 Sol + High（Codex 正式 provenance：`gpt-5.6-sol-high`）；QC 仍为 A-only，revision、validation、Production machine gate 与 HUMAN gate 均不得削弱。应节省的是不必要的 reasoning、重复检查、全面测试和机械执行。ChatGPT 与 Codex Generation 使用同一 provider-neutral deterministic Generation Bundle contract；独立 ChatGPT Reviewer 继续使用 GPT-5.6 Sol + High，并优先使用 Glossary、Quality Check 与 Locale Surface Review 的 deterministic Reviewer Bundle。bundle 只是绑定 current authority/input identity 的传输容器，不是 semantic authority、receipt 或语言质量 gate；当前性检查、result import、existing process/validation/record/finalize 等具体顺序以对应 stage 文档为准。附件完整且 current 时不再通过 Remote Desktop Commander 重复扫描同一批仓库输入。

Remote Desktop Commander 是 ChatGPT 执行上述闭环内短机械操作和 failure diagnosis 的正式辅助工具，但不接管闭环后的维护者批量操作。`preview` + browser verifier、`publish`、`shared-assets-production.sh`、`first-production.sh`、`indexnow-closeout.sh` 等可能持续数十秒到数分钟的成组 deterministic 任务，由维护者本地执行；普通 ChatGPT + Remote Desktop Commander 只提供完整可粘贴命令并读取终态，不通过反复 `read_process_output` 轮询，也不自行执行 Production 或最终 Git commit / push。此协作边界不得跳过任何脚本内部 gate、receipt、bounded retry 或 HUMAN gate。

新增正式 CLI、validator、manifest/lifecycle logic 或可复用 machine workflow 时，优先使用 Go，并优先集成现有 `cmd/tour-i18n`。Shell/Python 主要用于薄的 OS、SSH、browser 或已有成熟 orchestration；已有稳定 sh/python 不因“统一语言”重写。不得用临时 Python/sed 代替需要 repository context 的正式实现。

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
