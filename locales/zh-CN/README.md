# zh-CN 语言空间

这里是第一阶段简体中文 `zh-CN` 的语言空间，目前只完成 scaffold，正式课程翻译尚未开始，也不存在已发布中文页面。

- 翻译单元是一个完整的顶层 `present.Section`，不采用句子级或多 text 槽位 JSON。
- 英文源来自固定官方上游基线，页面目录见 [`../../data/tour-pages.tsv`](../../data/tour-pages.tsv)。
- 页面状态见 [`status.tsv`](status.tsv)，当前 101 页全部为 `pending`。
- `page_id` 是状态和语言文件的持久关联键；route、上游位置或标题变化时不得自动更换 ID，规则见 [`../../PAGE_IDENTITY.md`](../../PAGE_IDENTITY.md)。
- 未来页面文件位于 [`pages/`](pages/)，公共 UI 文案位于 [`ui/`](ui/)。
- 术语记录在 [`glossary.tsv`](glossary.tsv)，当前仅有表头。
- 所有来源的完整页面候选都必须经过同一套自动校验，不允许绕过校验直接发布。
- 人工逐页校对不是强制发布门槛，但自动解析和结构校验必须通过。
