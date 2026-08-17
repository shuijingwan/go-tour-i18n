# zh-CN pages 目录说明

`pages/` 是早期规划保留的目录，当前不承担手工维护正式译文或 production 输入的职责，暂时保留但不存放课程页面。

当前 zh-CN 内容模型如下：

- 翻译单元是一个完整顶层 `present.Section`，不采用句子级或多 text 槽位 JSON。
- canonical candidates 位于 [`../candidates/`](../candidates/)，页面状态由 [`../status.tsv`](../status.tsv) 维护；当前为 103/103 ready。
- build / projection 从 catalog、locale status 和全部 canonical `ready` candidates 生成正式语言内容；缺少 ready candidate 时失败，不回退到英文或旧译文。
- production release bundle 使用生成后的正式内容，不依赖 `pages/` 作为手工维护的最终译文集合。

页面继续以持久 `page_id` 关联 catalog、状态和 candidate；route、上游位置或标题变化时不得自动更换 ID。完整规则见 [`../../../PAGE_IDENTITY.md`](../../../PAGE_IDENTITY.md)。
