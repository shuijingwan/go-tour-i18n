# zh-CN 语言空间

这里是第一阶段简体中文 `zh-CN` 的语言空间。当前正式发布投影共 103 页，另保留 2 条条件源页面审计记录；已有 8 个页面完成 candidate 定稿，其余页面仍为 pending。

- 翻译单元是一个完整的顶层 `present.Section`，不采用句子级或多 text 槽位 JSON。
- 英文源来自固定官方上游基线，页面目录见 [`../../data/tour-pages.tsv`](../../data/tour-pages.tsv)。
- 页面状态见 [`status.tsv`](status.tsv)，覆盖 103 个正式发布页面；`welcome/1` 至 `welcome/5`、`basics/1` 至 `basics/3` 已 ready，其中 `basics/3` 为 Attempts=1。
- `page_id` 是状态和语言文件的持久关联键；route、上游位置或标题变化时不得自动更换 ID，规则见 [`../../PAGE_IDENTITY.md`](../../PAGE_IDENTITY.md)。
- canonical candidates 位于 [`candidates/`](candidates/)，未来正式页面位于 [`pages/`](pages/)，公共 UI 文案位于 [`ui/`](ui/)。
- 术语记录在 [`glossary.yaml`](glossary.yaml)。
- 所有来源的完整页面候选都必须经过同一套自动校验，不允许绕过校验直接发布。
- 人工逐页校对不是强制发布门槛，但自动解析和结构校验必须通过。
