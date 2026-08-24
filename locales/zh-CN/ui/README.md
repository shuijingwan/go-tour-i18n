# 多语言公共 UI 资源边界

公共 UI 文案与课程 TranslationUnit 正文分开维护。当前正式英文 source 是 `internal/tour/ui/en.json`，各 locale catalog 位于同一目录的 `<locale>.json`，并由 `internal/tour/ui/catalog.go` 嵌入和加载。

每个 locale catalog 必须完整覆盖英文 source 的全部 message key，并保持相同的 `plain` / `rich` kind；加载时会校验缺失键、额外键、kind 和 rich markup。正式 locale 不使用 English fallback。课程 candidate、status、review evidence 和 promotion 不属于 UI catalog。
