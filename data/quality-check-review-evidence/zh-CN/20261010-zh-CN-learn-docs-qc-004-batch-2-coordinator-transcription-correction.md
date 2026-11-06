# Coordinator transcription correction

locale: `zh-CN`
snapshot: `learn-docs-v1-qc-004`
review_id: `20261010-zh-CN-learn-docs-qc-004-batch-2`
role: 会话协调 / 基础设施架构
correction_kind: mechanical transcription correction
language_judgment_changed: no
rating_changed: no
finding_changed: no

正式 Reviewer 输出的 stable index 44 逐 Unit 段落与仓库 Candidate Snapshot 均对应以下 canonical Unit ID：

`126f2e430e4ad62ded97b90668a9621eb8f9b1b9bd9d316643afd1211cc6a9e6`

协调 session 首次将 Reviewer 输出保存为 Markdown evidence 并准备 CLI record 时，误将该 ID 转录为：

`126f2e430e4ad62ded97b90668a9621eb8f9b1bd9d316643afd1211cc6a9e6`

因此第一次 index 44 `quality-check record` fail closed，错误为：

`snapshot "learn-docs-v1-qc-004" has no unit "126f2e430e4ad62ded97b90668a9621eb8f9b1bd9d316643afd1211cc6a9e6"`

失败前已成功记录的结果保持不变；未盲目重放。随后从 immutable Snapshot manifest 读取 stable index 44 canonical Unit ID，使用 Reviewer 原始 rating=C 与原 finding 完成正式记录。

本说明不修改已保存 Reviewer evidence，不生成新的语言判断，不修改 candidate、rating 或 finding，仅补充机械转录错误的审计说明。
