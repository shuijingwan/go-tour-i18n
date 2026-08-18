# zh-CN 语言空间

这里是第一阶段简体中文 `zh-CN` 的语言空间。当前正式发布投影共 103 页，另保留 2 条条件源页面审计记录；正式状态为 `ready=103`、`pending=0`、`blocked=0`，103 个 canonical candidate 均已完成，zh-CN 已正式发布。

- 翻译单元是一个完整的顶层 `present.Section`，不采用句子级或多 text 槽位 JSON。
- 英文源来自固定官方上游基线，页面目录见 [`../../data/tour-pages.tsv`](../../data/tour-pages.tsv)。
- 页面状态见 [`status.tsv`](status.tsv)，覆盖 103 个正式发布页面；全部 103 页均为 ready。
- `page_id` 是状态和语言文件的持久关联键；route、上游位置或标题变化时不得自动更换 ID，规则见 [`../../PAGE_IDENTITY.md`](../../PAGE_IDENTITY.md)。
- canonical 课程译文位于 [`candidates/`](candidates/)；ready candidate 经 build / projection 生成正式语言内容，不使用历史预留的 [`pages/`](pages/) 作为正式译文维护目录；公共 UI 文案位于 [`ui/`](ui/)。
- 术语记录在 [`glossary.yaml`](glossary.yaml)。
- 所有来源的完整页面候选都必须经过同一套自动校验，不允许绕过校验直接发布。
- 人工逐页校对不是强制发布门槛，但自动解析和结构校验必须通过。

## ChatGPT 重译批次与术语

zh-CN 当前主要术语文件是 [`glossary.yaml`](glossary.yaml)。ChatGPT 开始翻译 zh-CN 批次前，应主动从 GitHub 仓库 `main` 分支读取该文件的最新版本，同时读取目标批次的 `manifest.json` 和其中列出的独立 `.article` 输入。用户无需在每个新会话中重新粘贴 `Exercise → 练习`、`channel → 通道`、`package → 包` 等已经存在于仓库术语表中的规则。

术语文件内各类规则沿用项目现有语义：`mandatory` 是强制译法，`preferred` 是优先译法，`terms` 用于统一术语，`forbidden` 是禁止使用的译法，`keep` 是保持原样的技术词或标识。README 只说明规则类型和文件位置，不复制完整术语表，避免双份维护。

一个批次可以包含多个页面（例如默认 10 页），但每个完整顶层 `present.Section` 仍是独立翻译单元，必须逐页输出，不得合并。若具体页面出现 glossary 尚未覆盖的新术语，ChatGPT 应根据完整页面上下文作出自然、准确、一致的翻译；是否将该术语补充进 glossary，留待后续单独决定。

正式约定使用 `main` 分支当前最新的 `glossary.yaml`，不要求批次绑定 glossary commit ID、glossary SHA 或保存术语表副本。需要追溯历史规则时使用 Git 历史，当前不另建术语版本绑定机制。
