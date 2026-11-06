# Structured semantic-input-expansion reopen

`go-learning/structured-input-reopen/v1` 是 machine authorization：只授权已完成的 exact structured asset 进入一次新的完整 asset Generation。它不是语言生成、Reviewer 判断、Surface finding 或 representation carry proof。

默认 preflight 只读，输出 deterministic JSON；只有维护者 Local terminal 显式使用 `--apply` 才保存 receipt：

```sh
go run -mod=readonly ./cmd/tour-i18n site-content structured-input-reopen --locale zh-CN --asset source:_content/learn/books.yaml
```

CLI 从 current contributor registry 与当前 completed package 自动确定 package、baseline closure、owner Generation、Surface 和 accepted target；不接受任意 JSON 或手填 SHA 作为 authority。当前 completion 可以是已有 delta closure，祖先依旧由现有 historical verifier 验证。

v1 仅允许 `semantic-input-expansion`。必须同时满足：

- historical completion、closure、Generation、Bundle、accepted output 和 Surface 完整性通过；locale/package/asset/path/target exact binding。
- raw source bytes、contributor identity、非 slot 的 Document context/dependencies 不变。
- 每个旧 Unit 的 ID、locator、source、source SHA、坐标和 protection metadata 全部 exact preserved；不允许 removed、changed、ambiguous remap 或只有 ID churn。
- current canonical parser 的正式 localizable semantic atoms 是旧集合的严格超集，至少新增一个 Unit；旧 output 无法覆盖 current exact slot set。
- current package、glossary、full Unified Review current；v1 不合并 baseline owner Generation 与 current glossary/Review 的漂移。

Receipt 绑定两侧完整 Document identities、structure/source/raw SHAs、Unit inventories、完整新增 atoms、accepted target SHA、baseline closure/Generation/Surface 和 current glossary/Review。Identity 不依赖当前时钟，路径为：

```text
data/locale-language/<locale>/structured-input-reopens/<identity(package,asset)>/<identity_sha256>.json
```

使用 strict JSON、self identity、content-addressed canonical path 和 immutable/no-overwrite writer。相同 bytes 重复 apply 为幂等操作；同一 current Document 的冲突 claims fail closed。旧 Document 的 receipt 仅是历史证据，不授权后续 parser/input；不允许 process-global eligibility cache。

无 valid receipt 时，Plan 保持 `stale / current exact source/input drift`，该 asset 不进入 Expected。存在 exact current valid receipt 时，只有该 asset 变为 `affected / generation-required / verified semantic input expansion`，Expected 包含其完整 target。其它 compatible assets 不进入新增 work set。

Generation Bundle 运输 receipt、完整 current Document/raw source、baseline/accepted output 和 predecessor context；仍要求完整 asset output，包括所有 current slots。内部 slots 不升级成 TranslationUnit。新 Generation 的 current checker 重算 receipt 与 assessment；历史 checker 则验证 frozen event-time 两侧和 archived receipt，不用未来 parser 推翻合法历史事件。

Receipt 不写 target、不补译新增 slots、不沿用旧 Generation 为 current，不提供 validation、A、Surface、closure、activation 或 Production 权限。没有 exact replacement Generation 时 closure 仍失败。后续继续沿用现有 Structured Generation → import/current validation → package closure → delta-aware Integrated Surface Review → activation，不能用 `structured-recover` 修补真实新增 semantic inputs。
