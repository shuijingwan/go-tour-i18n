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
