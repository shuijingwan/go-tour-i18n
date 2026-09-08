# Codex 项目导航

当用户要求新增一门语言、建立新 locale、准备新语言首次上线或评估新 locale 完成度时，Codex 必须先读取：

1. `docs/NEW_LOCALE_RUNBOOK.md`
2. `docs/LOCALE_SURFACE_REVIEW.md`

新增 locale 的 TranslationUnit 翻译仍须继续遵守下列不可拆分输入规则；上述入口不能代替具体翻译规范。

当用户要求执行 retranslation 正式翻译、retranslation retry 的译文生成、revision batch 翻译或任何 TranslationUnit 翻译时，Codex 在写文件前必须读取：

1. `docs/TRANSLATION_WORKFLOW.md`
2. `docs/TRANSLATION_TASK_SPEC.md`
3. `docs/RETRANSLATION_RUNBOOK.md`
4. `docs/CODEX_TRANSLATION.md`
5. 当前 batch 的 `manifest.json`
6. manifest 列出的全部 `inputs/*`
7. `locales/<locale>/glossary.yaml`

manifest、全部 inputs 与 locale glossary 是不可拆分的正式翻译输入。不得因用户 Prompt 未再次提醒 glossary 而跳过它，也不得以聊天上下文代替仓库中的当前正式规则。

模型与 reasoning 由用户在 Codex UI 中选择；当前推荐生产配置为 **GPT-5.6 Sol + High**。本文件不负责切换模型或 reasoning。

## 执行职责与正式自动化

Codex 用于正式 TranslationUnit translation / revision、需要重新生成译文的 retry、需要完整 repository context 的代码理解与修改、测试修复、docs/config 修改，以及复杂 failure evidence 分析。Codex 修改代码后可以运行证明修改正确所必需的最小 targeted tests，但不充当普通终端重复执行整个正式生命周期。

本地终端用于确定性执行：build、不会调用模型的 retranslation process / revalidate、status、validation、publish、deploy、verifier、checksum、curl、Git、assets，以及已有正式 CLI/script 已覆盖的其他机械步骤。当前 docs/code/production identity 已确认的 machine fact 应直接复用；没有新的 failure evidence 时，不重新探测服务器、不重新设计 Production、广告、IndexNow、shared assets 或 verifier。只有真实 evidence 显示基线变化时，才重新调查并更新正式 docs，避免为了“再确认一次”消耗 Codex reasoning quota。

新增正式 CLI、validator、manifest/lifecycle logic 或可复用 machine workflow 时，优先使用 Go，并优先集成现有 `cmd/tour-i18n`。Shell/Python 主要用于薄的 OS、SSH、browser 或已有成熟 orchestration；已有稳定 sh/python 不因“统一语言”重写。不得用临时 Python/sed 代替需要 repository context 的正式实现。

## 维护协作输出

除非用户明确要求英文，仓库 Markdown/docs 面向维护者的内容、Git Conventional Commit subject，以及 Codex 面向维护者的最终总结、failure 分析和下一步说明默认使用中文。代码标识符、CLI 名称、协议字段、原始第三方错误和必须保持的 source 文本不强制翻译。

Codex 已读取仓库当前 AGENTS/docs/code 后，不要求用户再次复制仓库已有正式规则。完成一次代码/docs 任务后，final response 默认简洁地只返回：实际修改文件、关键行为/决定、执行的测试及结果、未解决的真实 failure evidence（如有）、`git diff --stat`、`git status --short`。除非用户明确要求，不粘贴完整 git diff 或完整文件，不重述整个 Runbook，也不返回长篇“接下来让 ChatGPT 做什么”的 Prompt。

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
