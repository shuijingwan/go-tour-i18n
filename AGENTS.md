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

模型与 reasoning 由用户在对应 UI 中选择；Codex 与普通 ChatGPT 的当前正式语言生成配置均为 **GPT-5.6 Sol + High**。本文件不负责切换模型或 reasoning。

## 执行职责与正式自动化

普通 ChatGPT 的 locale 工作分为两个不得混用的长期角色：**Generation session** 负责 glossary、UI catalog、article metadata、其他 locale-level 文案、TranslationUnit initial / revision / retry generation，以及 schema v2 Course SEO localization / revision；与之独立的 **Reviewer session** 负责 Glossary Review、逐 Unit Quality Check、revision 后 re-QC 与 Locale Surface Review。同一个未参与该 locale 语言 generation 的 Reviewer session 可以连续承担这三类正式审核；各 gate 范围和 evidence 独立，reviewer finding 必须回到 Generation session 产生 replacement，Reviewer session 不得自行生成 replacement 后批准。Glossary Review 以 [Glossary Review 规范](docs/GLOSSARY_REVIEW.md) 为准，完整职责以 [ChatGPT 正式语言生成执行规范](docs/CHATGPT_LANGUAGE_GENERATION.md) 为准。canonical English source-description review 是跨 locale 共享 authority，不默认归入单个 locale 的 Reviewer session，仍以 [课程页正式 SEO Metadata 规范](docs/COURSE_SEO_METADATA.md) 为准。

本地终端负责 deterministic lifecycle：locale init、glossary-review record / check、retranslation export / process / retry process / revalidate、status / validation、Candidate Snapshot、quality-check scope / record / record-batch / finalize、promotion、Course SEO assemble / refresh / revise 的机械 CLI、canonical/source/current checks、build、surface-review export / record-a、preview、browser verifier、publish、Production、deploy、verifier、checksum / curl、Git、assets、search closeout，以及已有正式 CLI/script 已覆盖的其他机械步骤。Generation session 产生 Course SEO refresh / revise 所需的新 description 文本，本地终端才执行对应 CLI mutation。当前 docs/code/production identity 已确认的 machine fact 应直接复用；没有新的 failure evidence 时，不重新探测服务器、不重新设计 Production、广告、IndexNow、shared assets 或 verifier。只有真实 evidence 显示基线变化时，才重新调查并更新正式 docs，避免为了“再确认一次”消耗 Codex reasoning quota。

### 测试额度与执行边界（fail-closed）

Codex 默认只能运行与本次实际修改直接相关、足以证明改动正确的最小 targeted tests。未经用户明确授权，Codex 不得将 targeted tests 自行升级为 package-wide、repository-wide 或 lifecycle-wide regression，包括但不限于 `go test ./...`、多 package 的广泛 `go test`、repository-wide regression、全量 browser regression、publish、deploy、verifier、`--all-live`、完整 Production lifecycle、完整 TranslationUnit lifecycle，以及已有正式 CLI/script 可由本地终端确定性完成的其他长时间机械流程。

targeted tests 已足以证明当前改动正确时，不得为了“更保险”“顺便确认”或“完整覆盖”追加全面测试。Codex 如认为确有必要扩大范围，不得直接执行；应先在最终总结中说明现有 targeted tests 为何不足、建议补跑什么，并明确该测试由用户本地终端执行。只有用户随后明确授权，Codex 才可执行更大范围测试。

测试边界按以下规则确定：

- docs / README / config 任务默认只运行 `git diff --check`，以及与修改内容直接相关的 parser、validator 或 focused unit test；不得因为涉及共享 package 就自动运行整个 package 或 repository 测试。
- 代码任务优先测试直接修改的函数或模块、与改动行为直接相关的既有 targeted test；必要时新增 focused regression test。不得机械运行整个 package test suite，除非该 package 本身很小、这是唯一合理的 targeted boundary，且执行成本明显低。
- targeted test 意外暴露 scope 外问题时，只保存最小真实 failure evidence，不继续扩大调查或追加测试；仅当用户明确决定暂缓该问题时，才按 Deferred Issue 规则记录。
- `go test ./...`、全量 publish、10-locale `--all-live`、browser regression、checksum / curl / verifier、build / deploy / assets verification 等确定性但耗时的测试与验证，优先交给用户本地终端执行。

额度优化不得降低正式质量门槛：TranslationUnit 正式翻译仍使用 GPT-5.6 Sol + High，QC 仍为 A-only，revision、validation、Production machine gate 与 HUMAN gate 均不得削弱。应节省的是不必要的 reasoning、重复检查、全面测试和机械执行。

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
